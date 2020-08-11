package biz

import (
	"encoding/json"
	"fmt"
	"net/http"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/dgrijalva/jwt-go"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

var (
	errorTokenRequired          = util.NewNpError(http.StatusUnauthorized, "Authorization token required for access")
	errorInvalidRoomToken       = util.NewNpError(http.StatusUnauthorized, "Invalid room token")
	errorUnauthorizedRoomAccess = util.NewNpError(http.StatusForbidden, "Permission not sufficient for room")
)

// ParseProtoo Unmarshals a protoo payload.
func ParseProtoo(msg json.RawMessage, connectionClaims *signal.Claims, msgType interface{}) *nprotoo.Error {
	if err := json.Unmarshal(msg, &msgType); err != nil {
		log.Errorf("Biz.Entry parse error %v", err.Error())
		return util.NewNpError(http.StatusBadRequest, fmt.Sprintf("Error parsing request object %v", err.Error()))
	}

	authenticatable, ok := msgType.(proto.Authenticatable)
	log.Debugf("msgType: %#v \nHasRoomInfo: %#v ok: %v", msgType, authenticatable, ok)
	if ok && roomAuth.Enabled {
		return authenticateRoom(msgType, connectionClaims, authenticatable)
	}

	return nil
}

// authenticateRoom checks both the connection token AND an optional message token for RID claims
// returns nil for success and returns an error if there are no valid claims for the RID
func authenticateRoom(msgType interface{}, connectionClaims *signal.Claims, authenticatable proto.Authenticatable) *nprotoo.Error {
	log.Debugf("authenticateRoom: checking claims on token %v", authenticatable.Token())
	// Connection token has valid claim on this room, succeed early
	if connectionClaims != nil && authenticatable.Room().RID == proto.RID(connectionClaims.RID) {
		log.Debugf("authenticateRoom: Valid RID in connectionClaims %v", authenticatable.Room().RID)
		return nil
	}

	// Check for a message level proto.RoomToken
	var msgClaims *signal.Claims = nil
	if t := authenticatable.Token(); t != "" {
		token, err := jwt.ParseWithClaims(t, &signal.Claims{}, roomAuth.KeyFunc)
		if err != nil {
			log.Debugf("authenticateRoom: Error parsing token: %v", err)
			return errorInvalidRoomToken
		}
		log.Debugf("authenticateRoom: Got Token %#v", token)
		msgClaims = token.Claims.(*signal.Claims)
	}

	// No tokens were passed in
	if connectionClaims == nil && msgClaims == nil {
		return errorTokenRequired
	}

	// Message token is valid, succeed
	if msgClaims != nil && authenticatable.Room().RID == proto.RID(msgClaims.RID) {
		log.Debugf("authenticateRoom: Valid RID in msgClaims %v", authenticatable.Room().RID)
		return nil
	}

	// If this is reached, a token was passed but it did not have a valid RID claim
	return errorUnauthorizedRoomAccess
}

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg json.RawMessage, claims *signal.Claims, accept signal.RespondFunc, reject signal.RejectFunc) {
	var result interface{}
	topErr := util.NewNpError(http.StatusBadRequest, fmt.Sprintf("Unkown method [%s]", method))

	//TODO DRY this up
	switch method {
	case proto.ClientJoin:
		var msgData proto.JoinMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = join(peer, msgData)
		}
	case proto.ClientLeave:
		var msgData proto.LeaveMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = leave(peer, msgData)
		}
	case proto.ClientPublish:
		var msgData proto.PublishMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = publish(peer, msgData)
		}
	case proto.ClientUnPublish:
		var msgData proto.UnpublishMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = unpublish(peer, msgData)
		}
	case proto.ClientSubscribe:
		var msgData proto.SubscribeMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = subscribe(peer, msgData)
		}
	case proto.ClientUnSubscribe:
		var msgData proto.UnsubscribeMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = unsubscribe(peer, msgData)
		}
	case proto.ClientBroadcast:
		var msgData proto.BroadcastMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
			result, topErr = broadcast(peer, msgData)
		}
	case proto.ClientTrickleICE:
		var msgData proto.TrickleMsg
		if topErr = ParseProtoo(msg, claims, &msgData); topErr == nil {
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

func handleSFUBroadCast(msg nprotoo.Notification, subj string) {
	go func(msg nprotoo.Notification) {
		var data proto.MediaInfo
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Errorf("handleSFUBroadCast Unmarshall error %v", err)
			return
		}

		log.Infof("handleSFUBroadCast: method=%s, data=%v", msg.Method, data)

		switch msg.Method {
		case proto.SFUTrickleICE:
			signal.NotifyAllWithoutID(data.RID, data.UID, proto.ClientOnStreamAdd, data)
		case proto.SFUStreamRemove:
			islb, found := getRPCForIslb()
			if found {
				islb.AsyncRequest(proto.IslbOnStreamRemove, data)
			}
		}
	}(msg)
}

func getRPCForSFU(mid proto.MID, rid proto.RID) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "mid", mid, "rid", rid))
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
