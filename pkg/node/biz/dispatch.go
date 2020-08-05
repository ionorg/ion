package biz

import (
	"encoding/json"
	"fmt"
	"net/http"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

// ParseProtoo Unmarshals a protoo payload.
func ParseProtoo(msg json.RawMessage, msgType interface{}) *nprotoo.Error {
	if err := json.Unmarshal(msg, &msgType); err != nil {
		log.Errorf("Biz.Entry parse error %v", err.Error())
		return util.NewNpError(http.StatusBadRequest, fmt.Sprintf("Error parsing request object %v", err.Error()))
	}
	return nil
}

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg json.RawMessage, accept signal.RespondFunc, reject signal.RejectFunc) {
	var result interface{}
	topErr := util.NewNpError(http.StatusBadRequest, fmt.Sprintf("Unkown method [%s]", method))

	//TODO DRY this up
	switch method {
	case proto.ClientJoin:
		var msgData proto.FromClientJoinMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = join(peer, msgData)
		}
	case proto.ClientLeave:
		var msgData proto.FromSignalLeaveMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			log.Infof("LEAVE FROM SIGNAL")
			result, topErr = leave(msgData)
		}
	case proto.ClientOffer:
		var msgData proto.FromClientOfferMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = offer(peer, msgData)
		}
	case proto.ClientTrickleICE:
		var msgData proto.FromClientTrickleMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = trickle(peer, msgData)
		}
	case proto.ClientBroadcast:
		var msgData proto.FromClientBroadcastMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = broadcast(peer, msgData)
		}
	}

	if topErr != nil {
		reject(topErr.Code, topErr.Reason)
	} else {
		accept(result)
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

func handleSFUBroadCast(msg nprotoo.Notification, subj string) {
	go func(msg nprotoo.Notification) {
		log.Infof("handleSFUBroadCast: method=%s, data=%v", msg.Method, msg)

		switch msg.Method {
		case proto.SfuTrickleICE:
			var msgData proto.FromSfuTrickleMsg
			if err := json.Unmarshal(msg.Data, &msgData); err != nil {
				log.Errorf("handleSFUBroadCast failed to parse %v", err)
				return
			}
			signal.GetRoom(msgData.RID).GetPeer(string(msgData.UID)).Notify(proto.ClientTrickleICE, proto.ToClientTrickleMsg{
				RID:       msgData.RID,
				Candidate: msgData.Candidate,
			})
		case proto.SfuClientOffer:
			var msgData proto.FromSfuOfferMsg
			if err := json.Unmarshal(msg.Data, &msgData); err != nil {
				log.Errorf("handleSFUBroadCast failed to parse %v", err)
				return
			}
			signal.GetRoom(msgData.RID).GetPeer(string(msgData.UID)).Request(proto.ClientOffer, proto.ToClientOfferMsg{
				RID:     msgData.RID,
				RTCInfo: msgData.RTCInfo,
			}, func(answer json.RawMessage) {
				var answerData proto.FromClientAnswerMsg
				if err := ParseProtoo(answer, &answerData); err != nil {
					log.Warnf("Failed to parse client answer %s", answer)
					return
				}

				_, sfu, err := getRPCForSFU(msgData.RID)
				if err != nil {
					log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
					return
				}
				if _, err := sfu.SyncRequest(proto.SfuClientAnswer, proto.ToSfuAnswerMsg{
					RoomInfo: msgData.RoomInfo,
					RTCInfo:  answerData.RTCInfo,
				}); err != nil {
					log.Errorf("SfuClientOnAnswer failed %v", err.Error())
				}
			}, func(errorCode int, errorReason string) {
				log.Warnf("ClientOffer failed [%d] %s", errorCode, errorReason)
			})
		case proto.SfuClientLeave:
			var msgData proto.FromSfuLeaveMsg
			if err := json.Unmarshal(msg.Data, &msgData); err != nil {
				log.Errorf("handleSFUBroadCast failed to parse %v", err)
				return
			}
			log.Infof("LEAVE FROM SFU")
			leave(proto.FromSignalLeaveMsg{RoomInfo: proto.RoomInfo{RID: msgData.RID, UID: msgData.UID}})
		}
	}(msg)
}

func getRPCForSFU(rid proto.RID) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "rid", rid))
	if err != nil {
		return "", nil, err
	}

	var answer proto.GetSFURPCParams
	if err := json.Unmarshal(result, &answer); err != nil {
		return "", nil, &nprotoo.Error{Code: 123, Reason: "Unmarshal error getRPCForSFU"}
	}

	log.Infof("SFU result => %v", result)
	rpcID := answer.RPCID
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		protoo.OnBroadcast(answer.EventID, handleSFUBroadCast)
		rpcs[rpcID] = rpc
	}
	return answer.ID, rpc, nil
}
