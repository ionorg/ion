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
	case proto.ClientClose:
		var msgData proto.CloseMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = clientClose(peer, msgData)
		}
	case proto.ClientLogin:
		var msgData proto.LoginMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = login(peer, msgData)
		}
	case proto.ClientJoin:
		var msgData proto.JoinMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = join(peer, msgData)
		}
	case proto.ClientLeave:
		var msgData proto.LeaveMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = leave(peer, msgData)
		}
	case proto.ClientPublish:
		var msgData proto.PublishMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = publish(peer, msgData)
		}
	case proto.ClientUnPublish:
		var msgData proto.UnpublishMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = unpublish(peer, msgData)
		}
	case proto.ClientSubscribe:
		var msgData proto.SubscribeMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = subscribe(peer, msgData)
		}
	case proto.ClientUnSubscribe:
		var msgData proto.UnsubscribeMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = unsubscribe(peer, msgData)
		}
	case proto.ClientBroadcast:
		var msgData proto.BroadcastMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = broadcast(peer, msgData)
		}
	case proto.ClientTrickleICE:
		var msgData proto.TrickleMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = trickle(peer, msgData)
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

func handleSFUBroadCast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		log.Infof("handleSFUBroadCast: method=%s, data=%v", method, data)
		rid := util.Val(data, "rid")
		uid := util.Val(data, "uid")
		switch method {
		case proto.SFUTrickleICE:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, data)
		case proto.SFUStreamRemove:
			mid := util.Val(data, "mid")
			islb, found := getRPCForIslb()
			if found {
				islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("mid", mid))
			}
		}
	}(msg)
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
	eventID := result["event-id"].(string)
	nodeID := result["id"].(string)
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		protoo.OnBroadcast(eventID, handleSFUBroadCast)
		rpcs[rpcID] = rpc
	}
	return nodeID, rpc, nil
}
