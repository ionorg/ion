package sfu

import (
	"fmt"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	sfu "github.com/pion/ion-sfu/pkg"
	sfulog "github.com/pion/ion-sfu/pkg/log"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

var (
	server *sfu.SFU
	peers  map[proto.MID]*sfu.WebRTCTransport
)

// InitSFU init sfu server
func InitSFU(webrtc *sfu.WebRTCConfig, receiver *sfu.ReceiverConfig, log *sfulog.Config) {
	server = sfu.NewSFU(sfu.Config{
		WebRTC:   *webrtc,
		Receiver: *receiver,
		Log:      *log,
	})
	peers = map[proto.MID]*sfu.WebRTCTransport{}
}

func handleRequest(rpcID string) {
	log.Debugf("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Debugf("handleRequest: method => %s, data => %v", method, data)

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
	log.Debugf("join msg=%v", msg)
	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}

	me := sfu.MediaEngine{}
	if err := me.PopulateFromSDP(msg.Jsep); err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	peer, err := server.NewWebRTCTransport(string(msg.SID), me)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}
	peers[msg.MID] = peer

	log.Infof("peer %s join room %s", peer.ID(), msg.RID)

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	answer, err := peer.CreateAnswer()
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	if err := peer.SetLocalDescription(answer); err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			// Gathering done
			return
		}
		broadcaster.Say(proto.SfuTrickleICE, proto.SfuTrickleMsg{
			UID:       msg.UID,
			RID:       msg.RID,
			MID:       msg.MID,
			Candidate: c.ToJSON(),
		})
	})

	peer.OnNegotiationNeeded(func() {
		log.Infof("on negotiation needed called")
		offer, err := peer.CreateOffer()
		if err != nil {
			log.Errorf("CreateOffer error: %v", err)
			return
		}

		if err := peer.SetLocalDescription(offer); err != nil {
			log.Errorf("SetLocalDescription error: %v", err)
			return
		}

		broadcaster.Say(proto.SfuClientOffer, proto.SfuNegotiationMsg{
			UID:     msg.UID,
			RID:     msg.RID,
			MID:     msg.MID,
			RTCInfo: proto.RTCInfo{Jsep: offer},
		})
	})

	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			broadcaster.Say(proto.SfuClientLeave, proto.FromSfuLeaveMsg{
				RID: msg.RID, UID: msg.UID, MID: msg.MID,
			})
		}
	})

	// TODO: Remove once OnNegotiationNeeded is supported.
	go func() {
		time.Sleep(1000 * time.Millisecond)

		log.Infof("on negotiation needed called")
		offer, err := peer.CreateOffer()
		if err != nil {
			log.Errorf("CreateOffer error: %v", err)
			return
		}

		if err := peer.SetLocalDescription(offer); err != nil {
			log.Errorf("SetLocalDescription error: %v", err)
			return
		}

		broadcaster.Say(proto.SfuClientOffer, proto.SfuNegotiationMsg{
			UID:     msg.UID,
			RID:     msg.RID,
			MID:     msg.MID,
			RTCInfo: proto.RTCInfo{Jsep: offer},
		})
	}()

	resp := proto.FromSfuJoinMsg{RTCInfo: proto.RTCInfo{Jsep: answer}}
	return resp, nil
}

func offer(msg proto.SfuNegotiationMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("offer msg=%v", msg)
	peer, ok := peers[msg.MID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}

	answer, err := peer.CreateAnswer()
	if err != nil {
		log.Errorf("create answer error: %v", err)
		return nil, util.NewNpError(415, "create answer error")
	}

	if err := peer.SetLocalDescription(answer); err != nil {
		log.Errorf("set local description error: %v", err)
		return nil, util.NewNpError(415, "set local description error")
	}

	resp := proto.SfuNegotiationMsg{
		UID: msg.UID, RID: msg.RID, MID: msg.MID,
		RTCInfo: proto.RTCInfo{Jsep: answer},
	}
	return resp, nil
}

func leave(msg proto.ToSfuLeaveMsg) (interface{}, *nprotoo.Error) {
	peer, ok := peers[msg.MID]
	if !ok {
		log.Warnf("peers %v", peers)
		return nil, util.NewNpError(415, "peer not found")
	}
	delete(peers, msg.MID)

	if err := peer.Close(); err != nil {
		return nil, util.NewNpError(415, "failed to close peer")
	}

	return nil, nil
}

func answer(msg proto.SfuNegotiationMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("answer msg=%v", msg)
	peer, ok := peers[msg.MID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}
	return nil, nil
}

func trickle(msg proto.SfuTrickleMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Debugf("trickle msg=%v", msg)
	peer, ok := peers[msg.MID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.AddICECandidate(msg.Candidate); err != nil {
		return nil, util.NewNpError(415, "error adding ice candidate")
	}

	return nil, nil
}
