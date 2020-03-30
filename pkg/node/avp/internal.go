package avp

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	baseprocessor "github.com/pion/ion/pkg/node/avp/processors"
	"github.com/pion/ion/pkg/proto"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
)

func getRPCForIslb() (*nprotoo.Requestor, *nprotoo.Error) {
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
			return rpc, nil
		}
	}
	return nil, util.NewNpError(500, "islb node not found.")
}

func getRPCForSFU(mid string) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, err := getRPCForIslb()
	if err != nil {
		return "", nil, err
	}

	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "mid", mid))
	if err != nil {
		return "", nil, err
	}

	rpcID := result["rpc-id"].(string)
	nodeID := result["id"].(string)
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		rpcs[rpcID] = rpc
	}
	return nodeID, rpc, nil
}

func getMediaInfo(rid string, mid string) (map[string]interface{}, *nprotoo.Error) {
	islb, err := getRPCForIslb()
	if err != nil {
		return nil, err
	}
	return islb.SyncRequest(proto.IslbGetMediaInfo, util.Map("rid", rid, "mid", mid))
}

func subscribe(uid string, mid string, mediaInfo map[string]interface{}, offer webrtc.SessionDescription) (map[string]interface{}, *nprotoo.Error) {
	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("sfu node not found, reject: %d => %s", err.Code, err.Reason)
		return nil, err
	}

	log.Infof("subscribe => uid: %s mid: %s tracks: %v", uid, mid, mediaInfo["tracks"])
	return sfu.SyncRequest(proto.ClientSubscribe, util.Map("uid", uid, "mid", mid, "tracks", mediaInfo["tracks"], "jsep", offer))
}

// broadcast msg from islb
func handleIslbBroadCast(msg map[string]interface{}, subj string) {
	method := util.Val(msg, "method")
	data := msg["data"].(map[string]interface{})
	log.Infof("OnIslbBroadcast: method=%s, data=%v", method, data)

	util.StrToMap(data, "info")
	switch method {
	case proto.IslbOnStreamAdd:
		handleOnStreamAdd(data)
	case proto.IslbOnStreamRemove:
		handleStreamRemove(data)
	}
}

func handleStreamRemove(data map[string]interface{}) {
	mid := util.Val(data, "mid")

	log.Infof("IslbOnStreamRemove: mid=%s", mid)

	midprocessors := processors[mid]
	if midprocessors != nil {
		for name, processor := range midprocessors {
			if processor.AudioWriter != nil {
				processor.AudioWriter.Close()
			}
			if processor.VideoWriter != nil {
				processor.VideoWriter.Close()
			}
			midprocessors[name] = nil
		}
		processors[mid] = nil
	}
}

func handleOnStreamAdd(data map[string]interface{}) *nprotoo.Error {
	uid := util.Val(data, "uid")
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")

	log.Infof("IslbOnStreamAdd: uid=%s, mid=%s", uid, mid)

	rtcOptions := make(map[string]interface{})
	rtcOptions["subscribe"] = "true"

	sub := transport.NewWebRTCTransport(mid, rtcOptions)
	sub.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		log.Infof("OnTrack called with processor factories: %v", factories)
		for name, Factory := range factories {
			processor := Factory(mid)
			processors[mid] = make(map[string]*baseprocessor.Processor)
			processors[mid][name] = processor

			codec := track.Codec()
			log.Infof("Got track with codec: %s", codec)
			if codec.Name == webrtc.Opus {
				if processor.AudioWriter != nil {
					for {
						// Read RTP packets being sent to Pion
						rtp, err := track.ReadRTP()
						if err != nil {
							log.Warnf("Error writing audio for processor %s. Closing writer.", name)
							processor.AudioWriter.Close()
							break
						}
						processor.AudioWriter.WriteRTP(rtp)
					}
				}
			} else if codec.Name == webrtc.VP8 {
				if processor.VideoWriter != nil {
					for {
						// Read RTP packets being sent to Pion
						rtp, err := track.ReadRTP()
						if err != nil {
							log.Warnf("Error writing video for processor %s. Closing writer.", name)
							processor.VideoWriter.Close()
							break
						}
						processor.VideoWriter.WriteRTP(rtp)
					}
				}
			}
		}
	})

	offer, err := sub.Offer()

	if err != nil {
		log.Warnf("stream-add: error creating offer, reject: %d => %s", 415, err)
		return util.NewNpError(415, "steam-add: error creating offer")
	}

	mediaInfo, nerr := getMediaInfo(rid, mid)
	if nerr != nil {
		return nerr
	}

	result, nerr := subscribe(uid, mid, mediaInfo, offer)

	if nerr != nil {
		log.Warnf("stream-add: error subscribing to stream, reject: %d => %s", nerr.Code, nerr.Reason)
		return nerr
	}

	jsep := result["jsep"].(map[string]interface{})

	if jsep == nil {
		log.Warnf("stream-add: error jsep invalid")
		return util.NewNpError(415, "stream-add: jsep invaild.")
	}

	sdp := util.Val(jsep, "sdp")
	sub.SetRemoteSDP(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp})

	return nil
}
