package sfu

import (
	"fmt"
	"strconv"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	sdptransform "github.com/notedit/go-sdp-transform"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
	"github.com/sanity-io/litter"
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
	if err == nil {
		litter.Dump(sdpObj)
	}

	ssrcpts := []map[string]string{}
	for _, media := range sdpObj.Media {
		ssrcMap := make(map[string]string)
		for _, ssrc := range media.Ssrcs {
			if _, found := ssrcMap["ssrc"]; !found {
				ssrcMap["ssrc"] = strconv.FormatUint(uint64(ssrc.Id), 10)
			}
			ssrcMap["type"] = media.Type
			if media.Type == "audio" {
				ssrcMap["pt"] = "111"
			} else if media.Type == "video" {
				ssrcMap["pt"] = "96"
			}
		}

		ssrcpts = append(ssrcpts, ssrcMap)
	}

	log.Infof("publish ssrcpts %v, answer = %v", ssrcpts, answer)

	return util.Map("jsep", answer, "mid", mid, "ssrcpts", ssrcpts), nil
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

	ssrcPT := make(map[uint32]uint8)
	info := util.Val(msg, "info")
	log.Infof("subscribe info=%v", info)

	options := make(map[string]interface{})
	options["transport-cc"] = "false"
	options["subscribe"] = "true"
	options["codec"] = "vp8"

	smid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))
	if info != "" {
		for ssrc, pt := range util.Unmarshal(info) {
			ssrcPT[util.StrToUint32(ssrc)] = util.StrToUint8(pt.(string))
		}
	}

	options["ssrcpt"] = ssrcPT
	sub := transport.NewWebRTCTransport(smid, options)

	for ssrc, pt := range ssrcPT {
		log.Infof("AddTrack ssrc:%d,pt:%d", ssrc, pt)
		_, err := sub.AddTrack(ssrc, pt)
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
