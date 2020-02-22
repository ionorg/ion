package biz

import (
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
)

var emptyMap = map[string]interface{}{}

func handleRequest(rpcID string) {
	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Infof("handleRequest: method => %s, data => %v", method, data)
		switch method {
		case "publish":
			AddPublisher(data, accept, reject)
		case "unpublish":
			RemovePublisher(data, accept, reject)
		case "subscribe":
			AddSubscriber(data, accept, reject)
		case "unsubscribe":
			RemoveSubscriber(data, accept, reject)
		}
	})
}

// AddPublisher .
func AddPublisher(msg map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	log.Infof("AddPublisher msg=%v", msg)
	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		reject(415, "Unsupported Media Type")
		return
	}
	sdp := util.Val(jsep, "sdp")
	options := msg["options"].(map[string]interface{})
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}

	pub := transport.NewWebRTCTransport(mid, options)
	answer, err := pub.Answer(offer, options)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		reject(415, "Unsupported Media Type")
		return
	}
	router := rtc.GetOrNewRouter(mid)
	router.AddPub(uid, pub)
	accept(util.Map("jsep", answer, "mid", mid))
}

// RemovePublisher .
func RemovePublisher(msg map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	log.Infof("RemovePublisher msg=%v", msg)

	mid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(mid)
	if router != nil {
		router.DelPub()
		rtc.DelRouter(mid)
		accept(emptyMap)
		return
	}
	reject(404, "Router not found!")
}

// AddSubscriber .
func AddSubscriber(msg map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	log.Infof("AddSubscriber msg=%v", msg)

	pmid := util.Val(msg, "mid")
	router := rtc.GetOrNewRouter(pmid)
	if router == nil {
		reject(404, "Router not found!")
		return
	}

	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		reject(415, "Unsupported Media Type")
		return
	}

	sdp := util.Val(jsep, "sdp")
	options := msg["options"].(map[string]interface{})
	uid := util.Val(msg, "uid")
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	pub := router.GetPub().(*transport.WebRTCTransport)

	for ssrc, track := range pub.GetOutTracks() {
		options["ssrcpt"].(map[uint32]uint8)[ssrc] = track.PayloadType()
	}

	smid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))
	sub := transport.NewWebRTCTransport(smid, options)
	answer, err := sub.Answer(offer, options)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		reject(415, "Unsupported Media Type")
		return
	}
	router.AddSub(smid, sub)
	accept(util.Map("jsep", answer, "mid", smid))
}

// RemoveSubscriber .
func RemoveSubscriber(msg map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	log.Infof("RemoveSubscriber msg=%v", msg)
	pmid := util.Val(msg, "pmid")
	smid := util.Val(msg, "smid")
	router := rtc.GetOrNewRouter(pmid)
	if router != nil {
		router.DelSub(smid)
		accept(emptyMap)
		return
	}
	reject(404, "Router not found!")
}
