package sfu

import (
	"errors"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
)

func handleRequest(rpcID string) (*nats.Subscription, error) {
	log.Infof("handleRequest: rpcID => [%s]", rpcID)
	return nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("handleRequest: %T, %+v", msg, msg)

		switch v := msg.(type) {
		case *proto.ToSfuJoinMsg:
			return join(v)
		case *proto.SfuOfferMsg:
			return offer(v)
		case *proto.SfuAnswerMsg:
			return answer(v)
		case *proto.SfuTrickleMsg:
			return trickle(v)
		case *proto.ToSfuLeaveMsg:
			return leave(v)
		default:
			return nil, errors.New("unkonw message")
		}
	})
}

func join(msg *proto.ToSfuJoinMsg) (interface{}, error) {
	peer := s.getPeer(msg.MID)

	answer, err := peer.Join(string(msg.RID), msg.Offer)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, err
	}

	// Notify user of new ice candidate
	peer.OnOffer = func(offer *webrtc.SessionDescription) {
		data := proto.SfuOfferMsg{
			MID:  msg.MID,
			Desc: *offer,
		}
		log.Infof("send offer to [%s]: %v", msg.MID, data)
		if err := nrpc.Publish(string(msg.MID), data); err != nil {
			log.Errorf("send offer: %v", err)
		}
	}

	// Notify user of new offer
	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		data := proto.SfuTrickleMsg{
			MID:       msg.MID,
			Candidate: *candidate,
			Target:    target,
		}
		log.Infof("send candidate to [%s]: %v", msg.MID, data)
		if err := nrpc.Publish(string(msg.MID), data); err != nil {
			log.Errorf("send candidate to [%s] error: %v", msg.MID, err)
		}
	}

	peer.OnICEConnectionStateChange = func(state webrtc.ICEConnectionState) {
		data := proto.SfuICEConnectionStateMsg{
			MID:   msg.MID,
			State: state,
		}
		log.Infof("send ice connection state to [%s]: %v", msg.MID, data)
		if err := nrpc.Publish(string(msg.MID), data); err != nil {
			log.Errorf("send candidate to [%s] error: %v", msg.MID, err)
		}
	}

	// return answer
	resp := proto.FromSfuJoinMsg{Answer: *answer}
	log.Infof("reply join: %v", resp)
	return resp, nil
}

func offer(msg *proto.SfuOfferMsg) (interface{}, error) {
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

func leave(msg *proto.ToSfuLeaveMsg) (interface{}, error) {
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

func answer(msg *proto.SfuAnswerMsg) (interface{}, error) {
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

func trickle(msg *proto.SfuTrickleMsg) (interface{}, error) {
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
