package avp

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	iavp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
)

// sfu client
type sfu struct {
	ctx        context.Context
	cancel     context.CancelFunc
	config     iavp.Config
	mu         sync.RWMutex
	addr       string
	onCloseFn  func()
	transports map[proto.SID]*iavp.WebRTCTransport
	nid        string
	nrpc       *proto.NatsRPC
}

// newSFU intializes a new SFU client
func newSFU(addr string, config iavp.Config, nid string, nrpc *proto.NatsRPC) (*sfu, error) {
	log.Infof("connecting to sfu: %s", addr)

	ctx, cancel := context.WithCancel(context.Background())
	return &sfu{
		ctx:        ctx,
		cancel:     cancel,
		addr:       addr,
		config:     config,
		transports: make(map[proto.SID]*iavp.WebRTCTransport),
		nid:        nid,
		nrpc:       nrpc,
	}, nil
}

// getTransport returns a webrtc transport for a session
func (s *sfu) getTransport(sid proto.SID) (*iavp.WebRTCTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.transports[sid]

	// no transport yet, create one
	if t == nil {
		var err error
		mid := proto.MID(uuid.New().String())
		if t, err = s.join(sid, mid); err != nil {
			return nil, err
		}
		t.OnClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			log.Infof("transport close, sid=%s", sid)
			if err := s.nrpc.Publish(s.addr, proto.ToSfuLeaveMsg{
				MID: mid,
			}); err != nil {
				log.Errorf("leave to %s error: %v", s.addr, err.Error())
			}
			delete(s.transports, sid)
			if len(s.transports) == 0 && s.onCloseFn != nil {
				s.cancel()
				s.onCloseFn()
			}
		})
		s.transports[sid] = t
	} else {
		log.Infof("transport exist, sid=%s", sid)
	}

	return t, nil
}

// onClose handler called when sfu client is closed
func (s *sfu) onClose(f func()) {
	s.onCloseFn = f
}

func (s *sfu) join(sid proto.SID, mid proto.MID) (*iavp.WebRTCTransport, error) {
	log.Infof("joining sfu session: %s", sid)

	t := iavp.NewWebRTCTransport(string(sid), s.config)

	// join to sfu
	offer, err := t.CreateOffer()
	if err != nil {
		log.Errorf("creating offer error: %v", err)
		return nil, err
	}
	req := proto.ToSfuJoinMsg{
		RPC:   s.nid,
		SID:   sid,
		UID:   proto.UID(s.addr),
		MID:   mid,
		Offer: offer,
	}
	log.Infof("join to [%s]: %v", s.addr, req)
	resp, err := s.nrpc.Request(s.addr, req)
	if err != nil {
		log.Errorf("join to [%s] failed: %s", s.addr, err)
		return nil, err
	}
	msg, ok := resp.(*proto.FromSfuJoinMsg)
	if !ok {
		log.Errorf("join reply msg parses failed")
		return nil, errors.New("join reply msg parses failed")
	}
	log.Infof("join reply: %v", msg)
	if err := t.SetRemoteDescription(msg.Answer); err != nil {
		log.Errorf("Error set remote description: %s", err)
		return nil, err
	}

	// send candidates to sfu
	t.OnICECandidate(func(c *webrtc.ICECandidate, target int) {
		if c == nil {
			log.Infof("candidates gathering done")
			return
		}
		data := proto.SfuTrickleMsg{
			MID:       mid,
			Candidate: c.ToJSON(),
			Target:    target,
		}
		log.Infof("send trickle to [%s]: %v", s.addr, data)
		if err := s.nrpc.Publish(s.addr, data); err != nil {
			log.Errorf("send trickle to [%s] error: %v", s.addr, err)
		}
	})

	if err != nil {
		log.Errorf("nrpc subscribe error: %v", err)
	}

	return t, nil
}

func (s *sfu) handleSFUMessage(msg interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch v := msg.(type) {
	case *proto.SfuTrickleMsg:
		log.Infof("got remote candidate: %v", v.Candidate)
		t := s.transports[v.SID]
		if t == nil {
			log.Warnf("not found transport: %s", v.SID)
			break
		}
		if err := t.AddICECandidate(v.Candidate, v.Target); err != nil {
			log.Errorf("add ice candidate error: %s", err)
		}
	case *proto.SfuOfferMsg:
		log.Infof("got remote description: %v", v.Desc)
		t := s.transports[v.SID]
		if t == nil {
			log.Warnf("not found transport: %s", v.SID)
			break
		}
		answer, err := t.Answer(v.Desc)
		if err != nil {
			log.Errorf("create answer error: %v", err)
		}
		log.Infof("send description to [%s]: %v", s.addr, answer)
		if err := s.nrpc.Publish(s.addr, proto.SfuAnswerMsg{
			MID:  v.MID,
			Desc: answer,
		}); err != nil {
			log.Errorf("send description to [%s] error: %v", s.addr, err)
		}
	}
}
