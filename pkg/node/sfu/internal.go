package sfu

import (
	"fmt"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
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

// publish .
func publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("publish msg=%v", msg)
	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}
	sdp := util.Val(jsep, "sdp")
	//options := msg["options"].(map[string]interface{})
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}

	options := make(map[string]interface{})
	options["transport-cc"] = "false"
	options["publish"] = "true"
	options["codec"] = "vp8"
	pub := transport.NewWebRTCTransport(mid, options)
	answer, err := pub.Answer(offer, options)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}

	router := rtc.GetOrNewRouter(mid)
	router.AddPub(uid, pub)

	sdpObj, err := sdptransform.Parse(offer.SDP)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}

	tracks := make(map[string][]proto.TrackInfo)
	for _, stream := range sdpObj.GetStreams() {

		for id, track := range stream.GetTracks() {
			pt := int(0)
			if track.GetMedia() == "audio" {
				pt = webrtc.DefaultPayloadTypeOpus
			} else if track.GetMedia() == "video" {
				pt = webrtc.DefaultPayloadTypeVP8
			}
			var infos []proto.TrackInfo
			infos = append(infos, proto.TrackInfo{Ssrc: int(track.GetSSRCS()[0]), Payload: pt, Type: track.GetMedia(), ID: id})
			tracks[stream.GetID()+" "+id] = infos
		}
	}
	log.Infof("publish tracks %v, answer = %v", tracks, answer)
	return util.Map("jsep", answer, "mid", mid, "tracks", tracks), nil
}

// unpublish .
func unpublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("unpublish msg=%v", msg)

	mid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(mid)
	if router != nil {
		router.DelPub()
		router.Close()
		rtc.DelRouter(mid)
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, "Router not found!")
}

// subscribe .
func subscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("subscribe msg=%v", msg)

	pmid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(pmid)
	if router == nil {
		return nil, util.NewNpError(404, "Router not found!")
	}

	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}

	sdp := util.Val(jsep, "sdp")
	uid := util.Val(msg, "uid")

	options := make(map[string]interface{})
	options["transport-cc"] = "false"
	options["subscribe"] = "true"
	options["codec"] = "vp8"

	smid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	tracksMap := msg["tracks"].(map[string]interface{})
	log.Infof("subscribe tracks=%v", tracksMap)
	ssrcPT := make(map[uint32]uint8)
	options["ssrcpt"] = ssrcPT
	sub := transport.NewWebRTCTransport(smid, options)

	tracks := make(map[string]proto.TrackInfo)
	for msid, track := range tracksMap {
		for _, item := range track.([]interface{}) {
			info := item.(map[string]interface{})
			trackInfo := proto.TrackInfo{
				ID:      info["id"].(string),
				Type:    info["type"].(string),
				Ssrc:    int(info["ssrc"].(float64)),
				Payload: int(info["pt"].(float64)),
			}
			ssrcPT[uint32(trackInfo.Ssrc)] = uint8(trackInfo.Payload)
			tracks[msid] = trackInfo
		}
	}

	for msid, track := range tracks {
		ssrc := uint32(track.Ssrc)
		pt := uint8(track.Payload)
		// I2AacsRLsZZriGapnvPKiKBcLi8rTrO1jOpq c84ded42-d2b0-4351-88d2-b7d240c33435
		//                streamID                        trackID
		streamID := strings.Split(msid, " ")[0]
		trackID := track.ID
		log.Infof("AddTrack: ssrc:%d, pt:%d, streamID %s, trackID %s", ssrc, pt, streamID, trackID)
		_, err := sub.AddTrack(ssrc, pt, streamID, track.ID)
		if err != nil {
			log.Errorf("err=%v", err)
		}
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer, options)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}
	router.AddSub(smid, sub)

	log.Infof("subscribe mid %s, answer = %v", smid, answer)
	return util.Map("jsep", answer, "mid", smid), nil
}

// unsubscribe .
func unsubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("unsubscribe msg=%v", msg)
	pmid := util.Val(msg, "pmid")
	smid := util.Val(msg, "smid")
	router := rtc.GetOrNewRouter(pmid)
	if router != nil {
		router.DelSub(smid)
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, "Router not found!")
}
