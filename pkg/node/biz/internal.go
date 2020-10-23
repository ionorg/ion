package biz

import (
	"encoding/json"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

func handleIslbBroadcast(msg nprotoo.Notification, subj string) {
	var isblSignalTransformMap = map[string]string{
		proto.IslbStreamAdd: proto.ClientOnStreamAdd,
		proto.IslbPeerJoin:  proto.ClientOnJoin,
		proto.IslbPeerLeave: proto.ClientOnLeave,
		proto.IslbBroadcast: proto.ClientBroadcast,
	}
	go func(msg nprotoo.Notification) {
		var data struct {
			UID proto.UID `json:"uid"`
			RID proto.RID `json:"rid"`
		}
		if err := msg.Data.Unmarshal(&data); err != nil {
			log.Errorf("Error parsing message %v", err)
			return
		}

		log.Infof("OnIslbBroadcast: method=%s, data=%v", msg.Method, string(msg.Data))
		if newMethod, ok := isblSignalTransformMap[msg.Method]; ok {
			if r := signal.GetRoom(data.RID); r != nil {
				r.NotifyWithoutID(newMethod, msg.Data, data.UID)
			} else {
				log.Warnf("room not exits, rid=%s, uid=%, method=%s, msg=%s", data.RID, data.UID, newMethod, msg.Data)
			}
		}
	}(msg)
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

func handleSfuBroadcast(msg nprotoo.Notification, subj string) {
	go func(msg nprotoo.Notification) {
		log.Infof("handleSFUBroadCast: method=%s, data=%s", msg.Method, msg)

		switch msg.Method {
		case proto.SfuTrickleICE:
			var msgData proto.SfuTrickleMsg
			if err := json.Unmarshal(msg.Data, &msgData); err != nil {
				log.Errorf("handleSFUBroadCast failed to parse %v", err)
				return
			}
			signal.NotifyPeer(proto.ClientTrickleICE, msgData.RID, msgData.UID, proto.ClientTrickleMsg{
				RID:       msgData.RID,
				MID:       msgData.MID,
				Candidate: msgData.Candidate,
			})
		case proto.SfuClientOffer:
			var msgData proto.SfuNegotiationMsg
			if err := json.Unmarshal(msg.Data, &msgData); err != nil {
				log.Errorf("handleSFUBroadCast failed to parse %v", err)
				return
			}
			signal.NotifyPeer(proto.ClientOffer, msgData.RID, msgData.UID, proto.ClientNegotiationMsg{
				RID:     msgData.RID,
				MID:     msgData.MID,
				RTCInfo: msgData.RTCInfo,
			})
		}
	}(msg)
}

func getRPCForSFU(uid proto.UID, rid proto.RID, mid proto.MID) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindSfu, proto.ToIslbFindSfuMsg{
		UID: uid,
		RID: rid,
		MID: mid,
	})
	if err != nil {
		return "", nil, err
	}

	var answer proto.FromIslbFindSfuMsg
	if err := json.Unmarshal(result, &answer); err != nil {
		return "", nil, &nprotoo.Error{Code: 123, Reason: "Unmarshal error getRPCForSFU"}
	}

	log.Infof("SFU result => %v", answer)
	rpcID := answer.RPCID
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		protoo.OnBroadcast(answer.EventID, handleSfuBroadcast)
		rpcs[rpcID] = rpc
	}
	return answer.ID, rpc, nil
}
