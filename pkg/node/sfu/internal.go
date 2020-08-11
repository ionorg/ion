package sfu

import (
	"fmt"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
<<<<<<< HEAD
<<<<<<< HEAD
	sfu "github.com/pion/ion-sfu/pkg"
	sfulog "github.com/pion/ion-sfu/pkg/log"
=======
	"github.com/google/uuid"
	sdptransform "github.com/notedit/sdp"
	sfu "github.com/pion/ion-sfu/pkg"
>>>>>>> Handle join with ion-sfu.
=======
	sfu "github.com/pion/ion-sfu/pkg"
	sfulog "github.com/pion/ion-sfu/pkg/log"
>>>>>>> Update SFU node to use ion-sfu.
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

var emptyMap = map[string]interface{}{}

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> Update SFU node to use ion-sfu.
// TODO(kevmo314): Move to a config.toml.
var server = sfu.NewSFU(sfu.Config{
	WebRTC: sfu.WebRTCConfig{
		ICEServers: []sfu.ICEServerConfig{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun.stunprotocol.org:3478"}},
		},
	},
	Receiver: sfu.ReceiverConfig{
		Video: sfu.WebRTCVideoReceiverConfig{
			REMBCycle:     2,
			PLICycle:      1,
			TCCCycle:      1,
			MaxBandwidth:  1000,
			MaxBufferTime: 100,
		},
	},
	Log: sfulog.Config{
		Level: "debug",
	},
})

<<<<<<< HEAD
var peers = map[proto.UID]*sfu.WebRTCTransport{}
<<<<<<< HEAD
=======
var server = sfu.NewSFU(sfu.Config{})
>>>>>>> Handle join with ion-sfu.

var peers = map[proto.UID]*sfu.Peer{}
=======
>>>>>>> Update SFU node to use ion-sfu.
=======
var peers = map[proto.MID]*sfu.WebRTCTransport{}
>>>>>>> Latest changes.

func handleRequest(rpcID string) {
	log.Debugf("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Debugf("handleRequest: method => %s, data => %v", method, data)

		var result interface{}
		err := util.NewNpError(400, fmt.Sprintf("Unknown method [%s]", method))

		switch method {
<<<<<<< HEAD
<<<<<<< HEAD
		case proto.SfuClientJoin:
			var msgData proto.ToSfuJoinMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = join(msgData)
			}
		case proto.SfuClientOffer:
<<<<<<< HEAD
			var msgData proto.ToSfuOfferMsg
=======
		case proto.SfuClientOnJoin:
			var msgData proto.JoinMsg
			if err = data.Unmarshal(&msgData); err != nil {
				result, err = join(msgData)
			}
		case proto.SfuClientOnOffer:
			var msgData proto.OfferMsg
			if err = data.Unmarshal(&msgData); err != nil {
				result, err = offer(msgData)
			}
		case proto.SfuClientOnAnswer:
			var msgData proto.AnswerMsg
			if err = data.Unmarshal(&msgData); err != nil {
				result, err = answer(msgData)
			}
		case proto.ClientPublish:
			var msgData proto.PublishMsg
>>>>>>> Handle join with ion-sfu.
=======
		case proto.SfuClientJoin:
			var msgData proto.ToSfuJoinMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = join(msgData)
			}
		case proto.SfuClientOffer:
			var msgData proto.ToSfuOfferMsg
>>>>>>> Update SFU node to use ion-sfu.
=======
			var msgData proto.SfuNegotiationMsg
>>>>>>> Latest changes.
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
	peer, err := server.NewWebRTCTransport(uint32(msg.SID), msg.Jsep)
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

<<<<<<< HEAD
	// TODO(kevmo314): Correctly handle transport closure.
<<<<<<< HEAD
<<<<<<< HEAD

=======
	
>>>>>>> Update SFU node to use ion-sfu.
=======

>>>>>>> Add TODO.
	// peer.OnClose(func() {
	// 	broadcaster.Say(proto.SfuClientLeave, proto.FromSfuLeaveMsg{
	// 		MediaInfo: proto.MediaInfo{RID: msg.RID, UID: msg.UID, MID: proto.MID(peer.ID())},
	// 	})
	// })
=======
	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			broadcaster.Say(proto.SfuClientLeave, proto.FromSfuLeaveMsg{
				RID: msg.RID, UID: msg.UID, MID: msg.MID,
			})
		}
	})
>>>>>>> Latest changes.

<<<<<<< HEAD
<<<<<<< HEAD
	// TODO: Remove once OnNegotiationNeeded is supported.
=======
>>>>>>> Update SFU node to use ion-sfu.
=======
	// TODO: Remove once OnNegotiationNeeded is supported.
>>>>>>> Add TODO.
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
