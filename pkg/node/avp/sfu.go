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
	"github.com/pion/webrtc/v3"
)

// sfu client
type sfu struct {
	ctx    context.Context
	cancel context.CancelFunc
	config iavp.Config
	mu     sync.RWMutex

	client string
	mid    proto.MID

	onCloseFn  func()
	transports map[string]*iavp.WebRTCTransport
}

// newSFU intializes a new SFU client
func newSFU(addr string, config iavp.Config) (*sfu, error) {
	log.Infof("Connecting to sfu: %s", addr)

	ctx, cancel := context.WithCancel(context.Background())
	return &sfu{
		ctx:    ctx,
		cancel: cancel,
		client: addr,
		config: config,

		mid: proto.MID(uuid.New().String()),

		transports: make(map[string]*iavp.WebRTCTransport),
	}, nil
}

// getTransport returns a webrtc transport for a session
func (s *sfu) getTransport(sid string) (*iavp.WebRTCTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.transports[sid]

	// no transport yet, create one
	if t == nil {
		var err error
		var sub *nats.Subscription
		if t, sub, err = s.join(sid); err != nil {
			return nil, err
		}
		t.OnClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.transports, sid)
			sub.Unsubscribe()
			if len(s.transports) == 0 && s.onCloseFn != nil {
				s.cancel()
				s.onCloseFn()
			}
		})
		s.transports[sid] = t
	}

	return t, nil
}

// onClose handler called when sfu client is closed
func (s *sfu) onClose(f func()) {
	s.onCloseFn = f
}

func (s *sfu) join(sid string) (*iavp.WebRTCTransport, *nats.Subscription, error) {
	log.Infof("Joining sfu session: %s", sid)

	t := iavp.NewWebRTCTransport(sid, s.config)

	// handle sfu message
	rpcID := nid + "-" + sid
	sub, err := nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("handle sfu message: %+v", msg)

		switch v := msg.(type) {
		case *proto.SfuTrickleMsg:
			log.Infof("got remote candidate: %v", v.Candidate)
			if err := t.AddICECandidate(v.Candidate); err != nil {
				log.Errorf("add ice candidate error: %s", err)
				return nil, err
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
				MID:     v.MID,
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
		log.Errorf("Error creating offer: %v", err)
		return nil, nil, err
	}
	if err = t.SetLocalDescription(offer); err != nil {
		log.Errorf("Error setting local description: %v", err)
		return nil, nil, err
	}
	req := proto.ToSfuJoinMsg{
		RPCID:   rpcID,
		MID:     s.mid,
		SID:     proto.SID(sid),
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

	// send candidates to sfu
	t.OnICECandidate(func(c *webrtc.ICECandidate) {
		log.Errorf("OnICECandidate: %v", c)
		if c == nil {
			// Gathering done
			return
		}
		data := proto.SfuTrickleMsg{
			MID:       s.mid,
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
