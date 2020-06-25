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
	"google.golang.org/grpc"

	"github.com/pion/ion-sfu/pkg/proto/sfu"
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
	case proto.ClientSubscribe:
		var msgData proto.SubscribeMsg
		if topErr = ParseProtoo(msg, &msgData); topErr == nil {
			result, topErr = subscribe(peer, msgData)
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

func getRPCForSFU(mid string) (string, sfu.SFUClient, error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "mid", mid))
	if err != nil {
		return "", nil, err
	}

	var answer proto.GetSFURPCParams
	if err := json.Unmarshal(result, &answer); err != nil {
		return "", nil, &nprotoo.Error{Code: 123, Reason: "Unmarshal error getRPCForSFU"}
	}

	log.Infof("SFU result => %v", result)
	grpcAddr := answer.GRPCAddress
	sfuGrpc, found := sfus[grpcAddr]
	if !found {
		conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Panicf("did not connect: %v", err)
			return "", nil, fmt.Errorf("sfu node not found")
		}

		sfuGrpc = sfu.NewSFUClient(conn)
		sfus[grpcAddr] = sfuGrpc
	}
	return answer.ID, sfuGrpc, nil
}
