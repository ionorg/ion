package sfu

import (
	"fmt"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/google/uuid"
	sdptransform "github.com/notedit/sdp"
	sfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

var emptyMap = map[string]interface{}{}

var server = sfu.NewSFU(sfu.Config{})

var peers = map[proto.UID]*sfu.Peer{}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Debugf("handleRequest: method => %s, data => %v", method, data)

		var result interface{}
		err := util.NewNpError(400, fmt.Sprintf("Unknown method [%s]", method))

		switch method {
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
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = publish(msgData)
			}
		case proto.ClientUnPublish:
			var msgData proto.UnpublishMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = unpublish(msgData)
			}
		case proto.ClientSubscribe:
			var msgData proto.SFUSubscribeMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = subscribe(msgData)
			}
		case proto.ClientUnSubscribe:
			var msgData proto.UnsubscribeMsg
			if err = data.Unmarshal(&msgData); err == nil {
				result, err = unsubscribe(msgData)
			}
		}

		if err != nil {
			reject(err.Code, err.Reason)
		} else {
			accept(result)
		}
	})
}

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
	peers[proto.UID(peer.ID())] = peer

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
		UID:       proto.UID(peer.ID()),
		RTCInfo:   proto.RTCInfo{Jsep: answer},
		MediaInfo: proto.MediaInfo{MID: proto.MID(mid)},
	}
	return resp, nil
}

func offer(msg proto.OfferMsg) (interface{}, *nprotoo.Error) {
	peer, ok := peers[msg.UID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	err := peer.SetRemoteDescription(msg.Jsep)
	if err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}

	answer, err := peer.CreateAnswer()
	if err != nil {
		log.Errorf("create answer error: %v", err)
		return nil, util.NewNpError(415, "create answer error")
	}

	err = peer.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("set local description error: %v", err)
		return nil, util.NewNpError(415, "set local description error")
	}

	resp := proto.AnswerMsg{
		RoomInfo: proto.RoomInfo{UID: msg.UID, RID: msg.RID},
		RTCInfo:  proto.RTCInfo{Jsep: answer},
	}
	return resp, nil
}

func answer(msg proto.AnswerMsg) (interface{}, *nprotoo.Error) {
	peer, ok := peers[msg.UID]
	if !ok {
		return nil, util.NewNpError(415, "peer not found")
	}

	err := peer.SetRemoteDescription(msg.Jsep)
	if err != nil {
		log.Errorf("set remote description error: %v", err)
		return nil, util.NewNpError(415, "set remote description error")
	}
	return nil, nil
}

// publish .
func publish(msg proto.PublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("publish msg=%v", msg)
	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}
	uid := msg.UID
	mid := uuid.New().String()
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: msg.Jsep.SDP}

	rtcOptions := transport.RTCOptions{
		Publish: true,
	}

	rtcOptions.Codec = msg.Options.Codec
	rtcOptions.Bandwidth = msg.Options.Bandwidth
	rtcOptions.TransportCC = msg.Options.TransportCC

	videoCodec := strings.ToUpper(rtcOptions.Codec)

	sdpObj, err := sdptransform.Parse(offer.SDP)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		return nil, util.NewNpError(415, "publish: sdp parse failed.")
	}

	allowedCodecs := make([]uint8, 0)
	tracks := make(map[string][]proto.TrackInfo)
	for _, stream := range sdpObj.GetStreams() {
		for id, track := range stream.GetTracks() {
			pt, codecType := getPubPTForTrack(videoCodec, track, sdpObj)

			var infos []proto.TrackInfo
			if len(track.GetSSRCS()) == 0 {
				return nil, util.NewNpError(415, "publish: ssrc not found.")
			}
			allowedCodecs = append(allowedCodecs, pt)
			infos = append(infos, proto.TrackInfo{Ssrc: int(track.GetSSRCS()[0]), Payload: int(pt), Type: track.GetMedia(), ID: id, Codec: codecType})
			tracks[stream.GetID()+" "+id] = infos
		}
	}

	rtcOptions.Codecs = allowedCodecs
	pub := transport.NewWebRTCTransport(mid, rtcOptions)
	if pub == nil {
		return nil, util.NewNpError(415, "publish: transport.NewWebRTCTransport failed.")
	}

	router := rtc.GetOrNewRouter(proto.MID(mid))

	go handleTrickle(router, pub)

	answer, err := pub.Answer(offer, rtcOptions)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "publish: pub.Answer failed.")
	}

	router.AddPub(uid, pub)

	log.Infof("publish tracks %v, answer = %v", tracks, answer)
	resp := proto.PublishResponseMsg{
		RTCInfo:   proto.RTCInfo{Jsep: answer},
		MediaInfo: proto.MediaInfo{MID: proto.MID(mid)},
		Tracks:    tracks,
	}
	return resp, nil
}

// unpublish .
func unpublish(msg proto.UnpublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("unpublish msg=%v", msg)

	mid := msg.MID
	router := rtc.GetOrNewRouter(mid)
	if router != nil {
		rtc.DelRouter(mid)
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, "unpublish: Router not found!")
}

// subscribe .
func subscribe(msg proto.SFUSubscribeMsg) (interface{}, *nprotoo.Error) {
	log.Infof("subscribe msg=%v", msg)
	router := rtc.GetOrNewRouter(msg.MID)
	if router == nil {
		return nil, util.NewNpError(404, "subscribe: Router not found!")
	}

	if msg.Jsep.SDP == "" {
		return nil, util.NewNpError(415, "subscribe: Unsupported Media Type")
	}

	sdp := msg.Jsep.SDP

	rtcOptions := transport.RTCOptions{
		Subscribe: true,
	}

	rtcOptions.Bandwidth = msg.Options.Bandwidth
	rtcOptions.TransportCC = msg.Options.TransportCC

	subID := proto.MID(uuid.New().String())

	log.Infof("subscribe tracks=%v", msg.Tracks)
	rtcOptions.Ssrcpt = make(map[uint32]uint8)

	tracks := make(map[string]proto.TrackInfo)
	for msid, track := range msg.Tracks {
		for _, item := range track {
			rtcOptions.Ssrcpt[uint32(item.Ssrc)] = uint8(item.Payload)
			tracks[msid] = item
		}
	}

	sdpObj, err := sdptransform.Parse(sdp)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		return nil, util.NewNpError(415, "subscribe: sdp parse failed.")
	}

	ssrcPTMap := make(map[int]uint8)
	allowedCodecs := make([]uint8, 0, len(tracks))

	for _, track := range tracks {
		// Find pt for track given track.Payload and sdp
		ssrcPTMap[track.Ssrc] = getSubPTForTrack(track, sdpObj)
		allowedCodecs = append(allowedCodecs, ssrcPTMap[track.Ssrc])
	}

	// Set media engine codecs based on found pts
	log.Infof("Allowed codecs %v", allowedCodecs)
	rtcOptions.Codecs = allowedCodecs

	// New api
	sub := transport.NewWebRTCTransport(string(subID), rtcOptions)

	if sub == nil {
		return nil, util.NewNpError(415, "subscribe: transport.NewWebRTCTransport failed.")
	}

	go handleTrickle(router, sub)

	for msid, track := range tracks {
		ssrc := uint32(track.Ssrc)
		// Get payload type from request track
		pt := uint8(track.Payload)
		if newPt, ok := ssrcPTMap[track.Ssrc]; ok {
			// Override with "negotiated" PT
			pt = newPt
		}

		// I2AacsRLsZZriGapnvPKiKBcLi8rTrO1jOpq c84ded42-d2b0-4351-88d2-b7d240c33435
		//                streamID                        trackID
		streamID := strings.Split(msid, " ")[0]
		trackID := track.ID
		log.Infof("AddTrack: codec:%s, ssrc:%d, pt:%d, streamID %s, trackID %s", track.Codec, ssrc, pt, streamID, trackID)
		_, err := sub.AddSendTrack(ssrc, pt, streamID, track.ID)
		if err != nil {
			log.Errorf("err=%v", err)
		}
	}

	// Build answer
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer, rtcOptions)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}

	router.AddSub(subID, sub)

	log.Infof("subscribe mid %s, answer = %v", subID, answer)
	return util.Map("jsep", answer, "mid", subID), nil
}

// unsubscribe .
func unsubscribe(msg proto.UnsubscribeMsg) (interface{}, *nprotoo.Error) {
	log.Infof("unsubscribe msg=%v", msg)
	mid := msg.MID
	found := false
	rtc.MapRouter(func(id proto.MID, r *rtc.Router) {
		subs := r.GetSubs()
		for sid := range subs {
			if sid == mid {
				r.DelSub(mid)
				found = true
				return
			}
		}
	})
	if found {
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, fmt.Sprintf("unsubscribe: Sub [%s] not found!", mid))
}

// func trickle(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
// 	log.Infof("trickle msg=%v", msg)
// 	router := util.Val(msg, "router")
// 	mid := util.Val(msg, "mid")
// 	//cand := msg["trickle"]
// 	r := rtc.GetOrNewRouter(router)
// 	t := r.GetSub(mid)
// 	if t != nil {
// 		//t.(*transport.WebRTCTransport).AddCandidate(cand)
// 	}

// 	return nil, util.NewNpError(404, "trickle: WebRTCTransport not found!")
// }
