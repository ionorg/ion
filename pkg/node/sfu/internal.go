package sfu

import (
	"errors"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
)

var s *server

// InitSFU init sfu server
func InitSFU(config *isfu.Config) {
	s = newServer(config)
}

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
	log.Infof("join msg=%v", msg)

	peer := s.addPeer(msg.MID)

	answer, err := peer.Join(string(msg.RID), msg.Jsep)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, err
	}

	// Notify user of new ice candidate
	peer.OnOffer = func(offer *webrtc.SessionDescription) {
		data := proto.SfuOfferMsg{
			MID:     msg.MID,
			RTCInfo: proto.RTCInfo{Jsep: *offer},
		}
		log.Infof("send offer to [%s]: %v", msg.RPCID, data)
		if err := nrpc.Publish(msg.RPCID, data); err != nil {
			log.Errorf("send offer: %v", err)
		}
	}

	// Notify user of new offer
	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit) {
		data := proto.SfuTrickleMsg{
			MID:       msg.MID,
			Candidate: *candidate,
		}
		log.Infof("send candidate to [%s]: %v", msg.RPCID, data)
		if err := nrpc.Publish(msg.RPCID, data); err != nil {
			log.Errorf("send candidate to [%s] error: %v", msg.RPCID, err)
		}
	}

	// return answer
	resp := proto.FromSfuJoinMsg{RTCInfo: proto.RTCInfo{Jsep: *answer}}
	log.Infof("reply join: %v", resp)
	return resp, nil
}

func offer(msg *proto.SfuOfferMsg) (interface{}, error) {
	log.Infof("offer msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	answer, err := peer.Answer(msg.Jsep)
	if err != nil {
		log.Errorf("peer.Answer: %v", err)
		return nil, errors.New("peer.Answer error")
	}

	resp := proto.SfuAnswerMsg{
		MID:     msg.MID,
		RTCInfo: proto.RTCInfo{Jsep: *answer},
	}

	log.Infof("reply answer: %v", resp)

	return resp, nil
}

func leave(msg *proto.ToSfuLeaveMsg) (interface{}, error) {
	log.Infof("leave msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}
	s.delPeer(msg.MID)

	if err := peer.Close(); err != nil {
		return nil, errors.New("failed to close peer")
	}

	return nil, nil
}

func answer(msg *proto.SfuAnswerMsg) (interface{}, error) {
	log.Infof("answer msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, errors.New("set remote description error")
	}
	return nil, nil
}

func trickle(msg *proto.SfuTrickleMsg) (map[string]interface{}, error) {
	log.Infof("trickle msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, errors.New("peer not found")
	}

	if err := peer.Trickle(msg.Candidate); err != nil {
		return nil, errors.New("error adding ice candidate")
	}

	return nil, nil
}
