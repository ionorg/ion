package sfu

import (
	"fmt"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/google/uuid"
	sdptransform "github.com/notedit/sdp"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
)

var emptyMap = map[string]interface{}{}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)
	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Debugf("handleRequest: method => %s, data => %v", method, data)

		var result map[string]interface{}
		err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

		switch method {
		case proto.ClientPublish:
			result, err = publish(data)
		case proto.ClientUnPublish:
			result, err = unpublish(data)
		case proto.ClientSubscribe:
			result, err = subscribe(data)
		case proto.ClientUnSubscribe:
			result, err = unsubscribe(data)
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
			broadcaster.Say(proto.SFUTrickleICE, util.Map("mid", t.ID(), "trickle", trickle.ToJSON()))
		} else {
			return
		}
	}
}

// publish .
func publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("publish msg=%v", msg)
	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return nil, util.NewNpError(415, "publish: jsep invaild.")
	}
	sdp := util.Val(jsep, "sdp")
	uid := util.Val(msg, "uid")
	mid := uuid.New().String()
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}

	rtcOptions := transport.RTCOptions{
		TransportCC: false,
		Publish:     true,
	}

	options := msg["options"]
	if options != nil {
		options, ok := msg["options"].(map[string]interface{})
		if ok {
			rtcOptions.Codec = options["codec"].(string)
			// TODO parse int
			rtcOptions.Bandwidth = int(options["bandwidth"].(float64))
		}
	}

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

	router := rtc.GetOrNewRouter(mid)

	go handleTrickle(router, pub)

	answer, err := pub.Answer(offer, rtcOptions)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "publish: pub.Answer failed.")
	}

	router.AddPub(uid, pub)

	log.Infof("publish tracks %v, answer = %v", tracks, answer)
	return util.Map("jsep", answer, "mid", mid, "tracks", tracks), nil
}

// unpublish .
func unpublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("unpublish msg=%v", msg)

	mid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(mid)
	if router != nil {
		router.Close()
		rtc.DelRouter(mid)
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, "unpublish: Router not found!")
}

// subscribe .
func subscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("subscribe msg=%v", msg)

	mid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(mid)
	if router == nil {
		return nil, util.NewNpError(404, "subscribe: Router not found!")
	}

	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return nil, util.NewNpError(415, "subscribe: Unsupported Media Type")
	}

	sdp := util.Val(jsep, "sdp")

	rtcOptions := transport.RTCOptions{
		TransportCC: false,
		Subscribe:   true,
	}

	options := msg["options"]
	if options != nil {
		options, ok := msg["options"].(map[string]interface{})
		if ok {
			rtcOptions.Codec = options["codec"].(string)
			rtcOptions.Bandwidth = int(options["bandwidth"].(float64)) // TODO parse
		}
	}

	subID := uuid.New().String()

	tracksMap := msg["tracks"].(map[string]interface{})
	log.Infof("subscribe tracks=%v", tracksMap)
	rtcOptions.Ssrcpt = make(map[uint32]uint8)

	tracks := make(map[string]proto.TrackInfo)
	for msid, track := range tracksMap {
		for _, item := range track.([]interface{}) {
			info := item.(map[string]interface{})
			trackInfo := proto.TrackInfo{
				ID:      info["id"].(string),
				Type:    info["type"].(string),
				Ssrc:    int(info["ssrc"].(float64)),
				Payload: int(info["pt"].(float64)),
				Codec:   info["codec"].(string),
				Fmtp:    info["fmtp"].(string),
			}
			rtcOptions.Ssrcpt[uint32(trackInfo.Ssrc)] = uint8(trackInfo.Payload)
			tracks[msid] = trackInfo
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
	sub := transport.NewWebRTCTransport(subID, rtcOptions)

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
func unsubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("unsubscribe msg=%v", msg)
	mid := util.Val(msg, "mid")
	found := false
	rtc.MapRouter(func(id string, r *rtc.Router) {
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
