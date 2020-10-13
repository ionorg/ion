package sfu

import (
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	sfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

var s *server

// InitSFU init sfu server
func InitSFU(config *sfu.Config) {
	s = newServer(config)
}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Infof("handleRequest: method => %s, data => %s", method, data)

		var result interface{}
		err := util.NewNpError(400, fmt.Sprintf("Unknown method [%s]", method))

		switch method {
		case proto.SfuClientJoin:
			var msgData proto.ToSfuJoinMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = join(msgData)
			}
		case proto.SfuClientOffer:
			var msgData proto.SfuNegotiationMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = offer(msgData)
			}
		case proto.SfuClientLeave:
			var msgData proto.ToSfuLeaveMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = leave(msgData)
			}
		case proto.SfuClientAnswer:
			var msgData proto.SfuNegotiationMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = answer(msgData)
			}
		case proto.SfuClientTrickle:
			var msgData proto.SfuTrickleMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = trickle(msgData)
			}
		}

		if err != nil {
			reject(err.Code, err.Reason)
		} else {
			accept(result)
		}
	})
}

func join(msg proto.ToSfuJoinMsg) (interface{}, *nprotoo.Error) {
	log.Infof("join msg=%v", msg)
	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}

	peer := s.addPeer(msg.MID)

	log.Infof("peer %s join room %s", msg.MID, msg.RID)

	answer, err := peer.Join(string(msg.SID), msg.Jsep)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	peer.OnOffer = func(offer *webrtc.SessionDescription) {
		broadcaster.Say(proto.SfuClientOffer, proto.SfuNegotiationMsg{
			UID:     msg.UID,
			RID:     msg.RID,
			MID:     msg.MID,
			RTCInfo: proto.RTCInfo{Jsep: *offer},
		})
	}

	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit) {
		broadcaster.Say(proto.SfuTrickleICE, proto.SfuTrickleMsg{
			UID:       msg.UID,
			RID:       msg.RID,
			MID:       msg.MID,
			Candidate: *candidate,
		})
	}

	// TODO: ion-sfu may need to provide an peer.OnClose to notify clients that a peer is closed.
	// peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
	// 	if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
	// 		broadcaster.Say(proto.SfuClientLeave, proto.FromSfuLeaveMsg{
	// 			RID: msg.RID, UID: msg.UID, MID: msg.MID,
	// 		})
	// 	}
	// })

	resp := proto.FromSfuJoinMsg{RTCInfo: proto.RTCInfo{Jsep: *answer}}
	return resp, nil
}

func offer(msg proto.SfuNegotiationMsg) (interface{}, *nprotoo.Error) {
	log.Infof("offer msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, util.NewNpError(415, "peer not found")
	}

	answer, err := peer.Answer(msg.Jsep)
	if err != nil {
		log.Errorf("peer.Answer: %v", err)
		return nil, util.NewNpError(415, "peer.Answer error")
	}

	resp := proto.SfuNegotiationMsg{
		UID: msg.UID, RID: msg.RID, MID: msg.MID,
		RTCInfo: proto.RTCInfo{Jsep: *answer},
	}
	return resp, nil
}

func leave(msg proto.ToSfuLeaveMsg) (interface{}, *nprotoo.Error) {
	log.Infof("leave msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, util.NewNpError(415, "peer not found")
	}
	s.delPeer(msg.MID)

	if err := peer.Close(); err != nil {
		return nil, util.NewNpError(415, "failed to close peer")
	}

	return nil, nil
}

func answer(msg proto.SfuNegotiationMsg) (interface{}, *nprotoo.Error) {
	log.Infof("answer msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}
	return nil, nil
}

func trickle(msg proto.SfuTrickleMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("trickle msg=%v", msg)
	peer := s.getPeer(msg.MID)
	if peer == nil {
		log.Warnf("peer not found, mid=%s", msg.MID)
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.Trickle(msg.Candidate); err != nil {
		return nil, util.NewNpError(415, "error adding ice candidate")
	}

	return nil, nil
}
