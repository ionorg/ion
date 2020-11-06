package avp

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	iavp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

// sfu client
type sfu struct {
	ctx        context.Context
	cancel     context.CancelFunc
	config     iavp.Config
	mu         sync.RWMutex
	client     string
	onCloseFn  func()
	transports map[proto.RID]*iavp.WebRTCTransport
}

// newSFU intializes a new SFU client
func newSFU(addr string, config iavp.Config) (*sfu, error) {
	log.Infof("connecting to sfu: %s", addr)

	ctx, cancel := context.WithCancel(context.Background())
	return &sfu{
		ctx:        ctx,
		cancel:     cancel,
		client:     addr,
		config:     config,
		transports: make(map[proto.RID]*iavp.WebRTCTransport),
	}, nil
}

// getTransport returns a webrtc transport for a session
func (s *sfu) getTransport(rid proto.RID) (*iavp.WebRTCTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.transports[rid]

	// no transport yet, create one
	if t == nil {
		var err error
		var sub *nats.Subscription
		mid := proto.MID(uuid.New().String())
		if t, sub, err = s.join(rid, mid); err != nil {
			return nil, err
		}
		t.OnClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			log.Infof("transport close, rid=%s", rid)
			if _, err := nrpc.Request(s.client, proto.ToSfuLeaveMsg{
				MID: mid,
			}); err != nil {
				log.Errorf("send leave msg to sfu error: %v", err.Error())
			}
			delete(s.transports, rid)
			sub.Unsubscribe()
			if len(s.transports) == 0 && s.onCloseFn != nil {
				s.cancel()
				s.onCloseFn()
			}
		})
		s.transports[rid] = t
	} else {
		log.Infof("transport exist, rid=%s", rid)
	}

	return t, nil
}

// onClose handler called when sfu client is closed
func (s *sfu) onClose(f func()) {
	s.onCloseFn = f
}

func (s *sfu) join(rid proto.RID, mid proto.MID) (*iavp.WebRTCTransport, *nats.Subscription, error) {
	log.Infof("joining sfu session: %s", rid)

	t := iavp.NewWebRTCTransport(string(rid), s.config)

	var pendingDescriptions []webrtc.ICECandidateInit
	var hasRemoteDescription util.AtomicBool

	// handle sfu message
	rpcID := nid + "-" + string(rid)
	sub, err := nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("handle sfu message: %T, %+v", msg, msg)

		switch v := msg.(type) {
		case *proto.SfuTrickleMsg:
			log.Infof("got remote candidate: %v", v.Candidate)
			if hasRemoteDescription.Get() {
				if err := t.AddICECandidate(v.Candidate); err != nil {
					log.Errorf("add ice candidate error: %s", err)
					return nil, err
				}
			} else {
				log.Infof("pending remote candidate: %v", v.Candidate)
				pendingDescriptions = append(pendingDescriptions, v.Candidate)
			}
		case *proto.SfuOfferMsg:
			log.Infof("got remote description: %v", v.Jsep)
			if err := t.SetRemoteDescription(v.Jsep); err != nil {
				log.Errorf("set remote description error: ", err)
				return nil, err
			}

			answer, err := t.CreateAnswer()
			if err != nil {
				log.Errorf("create answer error: ", err)
				return nil, err
			}

			if err = t.SetLocalDescription(answer); err != nil {
				log.Errorf("set local description error: ", err)
				return nil, err
			}

			log.Infof("send description to [%s]: %v", s.client, answer)
			if err := nrpc.Publish(s.client, proto.SfuAnswerMsg{
				MID:     mid,
				RTCInfo: proto.RTCInfo{Jsep: answer},
			}); err != nil {
				log.Errorf("send description to [%s] error: %v", s.client, err)
				return nil, err
			}
		default:
			return nil, errors.New("unkonw message")
		}

		return nil, nil
	})
	if err != nil {
		log.Errorf("nrpc subscribe error: %v", err)
	}

	// join to sfu
	offer, err := t.CreateOffer()
	if err != nil {
		log.Errorf("creating offer error: %v", err)
		return nil, nil, err
	}
	if err = t.SetLocalDescription(offer); err != nil {
		log.Errorf("set local description error: %v", err)
		return nil, nil, err
	}
	req := proto.ToSfuJoinMsg{
		RPCID:   rpcID,
		MID:     mid,
		RID:     rid,
		RTCInfo: proto.RTCInfo{Jsep: offer},
	}
	log.Infof("join to [%s]: %v", s.client, req)
	resp, err := nrpc.Request(s.client, req)
	if err != nil {
		log.Errorf("join to [%s] failed: %s", s.client, err)
		return nil, nil, err
	}
	msg, ok := resp.(*proto.FromSfuJoinMsg)
	if !ok {
		log.Errorf("join reply msg parses failed")
		return nil, nil, errors.New("join reply msg parses failed")
	}
	log.Infof("join reply: %v", msg)
	if err := t.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("Error set remote description: %s", err)
		return nil, nil, err
	}

	hasRemoteDescription.Set(true)
	for _, c := range pendingDescriptions {
		if err := t.AddICECandidate(c); err != nil {
			log.Errorf("add ice candidate error: %s", err)
		}
	}

	// send candidates to sfu
	t.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			log.Infof("candidates gathering done")
			return
		}
		data := proto.SfuTrickleMsg{
			MID:       mid,
			Candidate: c.ToJSON(),
		}
		log.Infof("send trickle to [%s]: %v", s.client, data)
		if err := nrpc.Publish(s.client, data); err != nil {
			log.Errorf("send trickle to [%s] error: %v", s.client, err)
		}
	})

	if err != nil {
		log.Errorf("nrpc subscribe error: %v", err)
	}

	return t, sub, nil
}
