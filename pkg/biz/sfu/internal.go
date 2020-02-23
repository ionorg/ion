package biz

import (
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
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
			result, err = AddPublisher(data)
		case proto.ClientUnPublish:
			result, err = RemovePublisher(data)
		case proto.ClientSubscribe:
			result, err = AddSubscriber(data)
		case proto.ClientUnSubscribe:
			result, err = RemoveSubscriber(data)
		}

		if err != nil {
			reject(err.Code, err.Reason)
		} else {
			accept(result)
		}
	})
}

// AddPublisher .
func AddPublisher(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("AddPublisher msg=%v", msg)
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
	options["codec"] = "h264"
	options["transport-cc"] = ""
	options["publish"] = ""
	pub := transport.NewWebRTCTransport(mid, options)
	answer, err := pub.Answer(offer, options)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}
	router := rtc.GetOrNewRouter(mid)
	router.AddPub(uid, pub)
	return util.Map("jsep", answer, "mid", mid), nil
}

// RemovePublisher .
func RemovePublisher(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("RemovePublisher msg=%v", msg)

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

// AddSubscriber .
func AddSubscriber(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("AddSubscriber msg=%v", msg)

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
		return nil, util.NewNpError(415, "Unsupported Media Type")
	}
	router.AddSub(smid, sub)
	return util.Map("jsep", answer, "mid", smid), nil
}

// RemoveSubscriber .
func RemoveSubscriber(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("RemoveSubscriber msg=%v", msg)
	pmid := util.Val(msg, "pmid")
	smid := util.Val(msg, "smid")
	router := rtc.GetOrNewRouter(pmid)
	if router != nil {
		router.DelSub(smid)
		return emptyMap, nil
	}
	return nil, util.NewNpError(404, "Router not found!")
}
