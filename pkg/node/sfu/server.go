package sfu

import (
	"errors"
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
)

type server struct {
	sfu   *isfu.SFU
	peers map[proto.MID]*isfu.Peer
	mu    sync.RWMutex
	nid   string
	nrpc  *proto.NatsRPC
	sub   *nats.Subscription
}

func newServer(conf isfu.Config, nid string, nrpc *proto.NatsRPC) *server {
	return &server{
		sfu:   isfu.NewSFU(conf),
		peers: make(map[proto.MID]*isfu.Peer),
		nid:   nid,
		nrpc:  nrpc,
	}
}

func (s *server) getPeer(mid proto.MID) *isfu.Peer {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.peers[mid]
	if p == nil {
		p = isfu.NewPeer(s.sfu)
		s.peers[mid] = p
	}
	return p
}

func (s *server) delPeer(mid proto.MID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, mid)
}

func (s *server) start() error {
	var err error
	if s.sub, err = s.nrpc.Subscribe(s.nid, s.handle); err != nil {
		return err
	}
	return nil
}

func (s *server) close() {
	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", s.sub.Subject, err)
		}
	}
}

func (s *server) handle(msg interface{}) (interface{}, error) {
	log.Infof("handle: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.ToSfuJoinMsg:
		return s.join(v)
	case *proto.SfuOfferMsg:
		return s.offer(v)
	case *proto.SfuAnswerMsg:
		return s.answer(v)
	case *proto.SfuTrickleMsg:
		return s.trickle(v)
	case *proto.ToSfuLeaveMsg:
		return s.leave(v)
	default:
		return nil, errors.New("unkonw message")
	}
}

func (s *server) join(msg *proto.ToSfuJoinMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)

	answer, err := peer.Join(string(msg.SID), msg.Offer)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, err
	}

	// notify user of new ice candidate
	peer.OnOffer = func(offer *webrtc.SessionDescription) {
		data := proto.SfuOfferMsg{
			SID:  msg.SID,
			UID:  msg.UID,
			MID:  msg.MID,
			Desc: *offer,
		}
		log.Infof("send offer to [%s]: %v", msg.MID, data)
		if err := s.nrpc.Publish(msg.RPC, data); err != nil {
			log.Errorf("send offer: %v", err)
		}
	}

	// notify user of new offer
	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		data := proto.SfuTrickleMsg{
			SID:       msg.SID,
			UID:       msg.UID,
			MID:       msg.MID,
			Candidate: *candidate,
			Target:    target,
		}
		log.Infof("send candidate to [%s]: %v", msg.MID, data)
		if err := s.nrpc.Publish(msg.RPC, data); err != nil {
			log.Errorf("send candidate to [%s] error: %v", msg.MID, err)
		}
	}

	peer.OnICEConnectionStateChange = func(state webrtc.ICEConnectionState) {
		data := proto.SfuICEConnectionStateMsg{
			SID:   msg.SID,
			UID:   msg.UID,
			MID:   msg.MID,
			State: state,
		}
		log.Infof("send ice connection state to [%s]: %v", msg.MID, data)
		if err := s.nrpc.Publish(msg.RPC, data); err != nil {
			log.Errorf("send candidate to [%s] error: %v", msg.MID, err)
		}
	}

	// return answer
	resp := proto.FromSfuJoinMsg{Answer: *answer}
	log.Infof("reply join: %v", resp)
	return resp, nil
}

func (s *server) offer(msg *proto.SfuOfferMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	answer, err := peer.Answer(msg.Desc)
	if err != nil {
		log.Errorf("peer.Answer: %v", err)
		return nil, err
	}

	resp := proto.SfuAnswerMsg{
		MID:  msg.MID,
		Desc: *answer,
	}

	log.Infof("reply answer: %v", resp)

	return resp, nil
}

func (s *server) leave(msg *proto.ToSfuLeaveMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}
	s.delPeer(msg.MID)

	if err := peer.Close(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *server) answer(msg *proto.SfuAnswerMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Desc); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, err
	}
	return nil, nil
}

func (s *server) trickle(msg *proto.SfuTrickleMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	if err := peer.Trickle(msg.Candidate, msg.Target); err != nil {
		return nil, err
	}

	return nil, nil
}
