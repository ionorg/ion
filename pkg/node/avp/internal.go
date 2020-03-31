package avp

import (
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/pion/webrtc/v2/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v2/pkg/media/oggwriter"
)

// strToMap make string value to map
func strToMap(msg map[string]interface{}, key string) {
	val := util.Val(msg, key)
	if val != "" {
		m := util.Unmarshal(val)
		msg[key] = m
	}
}

func getRPCForIslb() (*nprotoo.Requestor, bool) {
	for _, item := range services {
		if item.Info["service"] == "islb" {
			id := item.Info["id"]
			rpc, found := rpcs[id]
			if !found {
				rpcID := discovery.GetRPCChannel(item)
				log.Infof("Create rpc [%s] for islb", rpcID)
				rpc = protoo.NewRequestor(rpcID)
				rpcs[id] = rpc
			}
			return rpc, true
		}
	}
	log.Warnf("No islb node was found.")
	return nil, false
}

func getRPCForSFU(mid string) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "mid", mid))
	if err != nil {
		return "", nil, err
	}

	log.Infof("SFU result => %v", result)
	rpcID := result["rpc-id"].(string)
	nodeID := result["id"].(string)
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		rpcs[rpcID] = rpc
	}
	return nodeID, rpc, nil
}

func saveToDisk(i media.Writer, track *webrtc.Track) {
	defer func() {
		if err := i.Close(); err != nil {
			panic(err)
		}
	}()

	for {
		rtpPacket, err := track.ReadRTP()
		if err != nil {
			panic(err)
		}
		if err := i.WriteRTP(rtpPacket); err != nil {
			panic(err)
		}
	}
}

// broadcast msg from islb
func handleIslbBroadCast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		log.Infof("OnIslbBroadcast: method=%s, data=%v", method, data)

		//make signal.Notify send "info" as a json object, otherwise is a string (:
		strToMap(data, "info")
		switch method {
		case proto.IslbOnStreamAdd:
			handleOnStreamAdd(data)
			// case proto.IslbOnStreamRemove:
			// 	TODO
		}
	}(msg)
}

func getAnswerForOffer(uid string, rid string, mid string, offer webrtc.SessionDescription) (webrtc.SessionDescription, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return webrtc.SessionDescription{}, util.NewNpError(500, "Not found any node for islb.")
	}

	mediaInfo, err := islb.SyncRequest(proto.IslbGetMediaInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		return webrtc.SessionDescription{}, util.NewNpError(err.Code, err.Reason)
	}

	_, sfu, err := getRPCForSFU(mid)

	if err != nil {
		log.Warnf("stream-add: sfu node not found, reject: %d => %s", err.Code, err.Reason)
		return webrtc.SessionDescription{}, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("Client Subscribe => uid: %s mid: %s tracks: %v", uid, mid, mediaInfo["tracks"])
	result, err := sfu.SyncRequest(proto.ClientSubscribe, util.Map("uid", uid, "mid", mid, "tracks", mediaInfo["tracks"], "jsep", offer))

	if err != nil {
		log.Warnf("stream-add: error subscribing to stream, reject: %d => %s", err.Code, err.Reason)
		return webrtc.SessionDescription{}, util.NewNpError(err.Code, err.Reason)
	}

	jsep := result["jsep"].(map[string]interface{})

	if jsep == nil {
		return webrtc.SessionDescription{}, util.NewNpError(415, "stream-add: jsep invaild.")
	}

	sdp := util.Val(jsep, "sdp")
	return webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}, nil
}

func handleOnStreamAdd(data map[string]interface{}) *nprotoo.Error {
	uid := util.Val(data, "uid")
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")

	log.Infof("IslbOnStreamAdd: uid=%s, mid=%s", uid, mid)

	rtcOptions := make(map[string]interface{})
	rtcOptions["subscribe"] = "true"

	sub := transport.NewWebRTCTransport(mid, rtcOptions)
	offer, err := sub.Offer()

	if err != nil {
		log.Warnf("Error creating offer, reject: %d => %s", 415, err)
		return util.NewNpError(415, "steam-add: error creating offer")
	}

	answer, nerr := getAnswerForOffer(uid, rid, mid, offer)

	if nerr != nil {
		log.Warnf("Error receiving answer, reject: %d => %s", 415, nerr)
		return util.NewNpError(415, "steam-add: error receiving answer")
	}

	sub.SetRemoteSDP(answer)

	oggFile, err := oggwriter.New("output.ogg", 48000, 2)
	if err != nil {
		panic(err)
	}
	ivfFile, err := ivfwriter.New("output.ivf")
	if err != nil {
		panic(err)
	}

	sub.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := sub.WriteRTCP(&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()})
				if errSend != nil {
					log.Warnf("Error sending RTCP")
				}
			}
		}()

		codec := track.Codec()
		log.Infof("Codec %s", codec)
		if codec.Name == webrtc.Opus {
			log.Infof("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
			saveToDisk(oggFile, track)
		} else if codec.Name == webrtc.VP8 {
			log.Infof("Got VP8 track, saving to disk as output.ivf")
			saveToDisk(ivfFile, track)
		}
	})

	return nil
}
