package sfu

import (
	"fmt"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
<<<<<<< HEAD
	sfu "github.com/pion/ion-sfu/pkg"
	sfulog "github.com/pion/ion-sfu/pkg/log"
=======
	"github.com/google/uuid"
	sdptransform "github.com/notedit/sdp"
	sfu "github.com/pion/ion-sfu/pkg"
>>>>>>> Handle join with ion-sfu.
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

var emptyMap = map[string]interface{}{}

<<<<<<< HEAD
// TODO(kevmo314): Move to a config.toml.
var server = sfu.NewSFU(sfu.Config{
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

var peers = map[proto.UID]*sfu.WebRTCTransport{}
=======
var server = sfu.NewSFU(sfu.Config{})
>>>>>>> Handle join with ion-sfu.

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
		case proto.SfuClientJoin:
			var msgData proto.ToSfuJoinMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = join(msgData)
			}
		case proto.SfuClientOffer:
			var msgData proto.ToSfuOfferMsg
=======
		case proto.SfuClientOnJoin:
			var msgData proto.JoinMsg
			if err = data.Unmarshal(&msgData); err != nil {
				result, err = join(msgData)
			}
		case proto.ClientPublish:
			var msgData proto.PublishMsg
>>>>>>> Handle join with ion-sfu.
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = offer(msgData)
			}
		case proto.SfuClientLeave:
			var msgData proto.ToSfuLeaveMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = leave(msgData)
			}
		case proto.SfuClientAnswer:
			var msgData proto.ToSfuAnswerMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = answer(msgData)
			}
		case proto.SfuClientTrickle:
			var msgData proto.ToSfuTrickleMsg
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

<<<<<<< HEAD
func join(msg proto.ToSfuJoinMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("join msg=%v", msg)
=======
func handleTrickle(r *rtc.Router, t *transport.WebRTCTransport) {
	for {
		trickle := <-t.GetCandidateChan()
		if trickle != nil {
			broadcaster.Say(proto.SfuTrickleICE, util.Map("mid", t.ID(), "trickle", trickle.ToJSON()))
		} else {
			return
		}
	}
}

func join(msg proto.JoinMsg) (interface{}, *nprotoo.Error) {
	log.Infof("join msg=%v", msg)
	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}
	mid := uuid.New().String()
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: msg.Jsep.SDP}
	peer, err := sfu.NewPeer(offer)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	log.Infof("peer %s join room %s", peer.ID(), msg.RID)

	room := server.GetRoom(string(msg.RID))
	if room == nil {
		room = server.CreateRoom(string(msg.RID))
	}
	room.AddTransport(peer)

	err = peer.SetRemoteDescription(offer)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	answer, err := peer.CreateAnswer()
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	err = peer.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}

	peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			// Gathering done
			return
		}
		broadcaster.Say(proto.SfuTrickleICE, util.Map("mid", mid, "trickle", c.ToJSON()))
	})

	peer.OnNegotiationNeeded(func() {
		log.Debugf("on negotiation needed called")
		offer, err := peer.CreateOffer()
		if err != nil {
			log.Errorf("CreateOffer error: %v", err)
			return
		}

		err = peer.SetLocalDescription(offer)
		if err != nil {
			log.Errorf("SetLocalDescription error: %v", err)
			return
		}

		broadcaster.Say(proto.SfuClientOnOffer, util.Map("mid", mid, "type", offer.Type, "sdp", offer.SDP))
	})

	resp := proto.JoinResponseMsg{
		RTCInfo:   proto.RTCInfo{Jsep: answer},
		MediaInfo: proto.MediaInfo{MID: proto.MID(mid)},
	}
	return resp, nil
}

// publish .
func publish(msg proto.PublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("publish msg=%v", msg)
>>>>>>> Handle join with ion-sfu.
	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}
	peer, err := sfu.NewWebRTCTransport(msg.Jsep)
	if err != nil {
		log.Errorf("join error: %v", err)
		return nil, util.NewNpError(415, "join error")
	}
	peers[msg.UID] = peer

	log.Infof("peer %s join room %s", peer.ID(), msg.RID)

	room := server.GetSession(string(msg.RID))
	if room == nil {
		room = server.NewSession(string(msg.RID))
	}
	room.AddTransport(peer)

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
		broadcaster.Say(proto.SfuTrickleICE, proto.FromSfuTrickleMsg{
			RoomInfo:  msg.RoomInfo,
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

		broadcaster.Say(proto.SfuClientOffer, proto.FromSfuOfferMsg{
			RoomInfo: msg.RoomInfo,
			RTCInfo:  proto.RTCInfo{Jsep: offer},
		})
	})

	// TODO(kevmo314): Correctly handle transport closure.

	// peer.OnClose(func() {
	// 	broadcaster.Say(proto.SfuClientLeave, proto.FromSfuLeaveMsg{
	// 		MediaInfo: proto.MediaInfo{RID: msg.RID, UID: msg.UID, MID: proto.MID(peer.ID())},
	// 	})
	// })

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

		broadcaster.Say(proto.SfuClientOffer, proto.FromSfuOfferMsg{
			RoomInfo: msg.RoomInfo,
			RTCInfo:  proto.RTCInfo{Jsep: offer},
		})
	}()

	resp := proto.FromSfuJoinMsg{
		MediaInfo: proto.MediaInfo{RID: msg.RID, UID: msg.UID, MID: proto.MID(peer.ID())},
		RTCInfo:   proto.RTCInfo{Jsep: answer},
	}
	return resp, nil
}

func offer(msg proto.ToSfuOfferMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("offer msg=%v", msg)
	peer, ok := peers[msg.UID]
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

	resp := proto.FromSfuAnswerMsg{
		RoomInfo: proto.RoomInfo{UID: msg.UID, RID: msg.RID},
		RTCInfo:  proto.RTCInfo{Jsep: answer},
	}
	return resp, nil
}

func leave(msg proto.ToSfuLeaveMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("leave msg=%v", msg)
	peer, ok := peers[msg.UID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}
	delete(peers, msg.UID)

	if err := peer.Close(); err != nil {
		return nil, util.NewNpError(415, "failed to close peer")
	}

	return nil, nil
}

func answer(msg proto.ToSfuAnswerMsg) (interface{}, *nprotoo.Error) {
	log.Debugf("answer msg=%v", msg)
	peer, ok := peers[msg.UID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}
	return nil, nil
}

func trickle(msg proto.ToSfuTrickleMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Debugf("trickle msg=%v", msg)
	peer, ok := peers[msg.UID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	if err := peer.AddICECandidate(msg.Candidate); err != nil {
		return nil, util.NewNpError(415, "error adding ice candidate")
	}

	return nil, nil
}
