package biz

import (
	"encoding/json"
	"fmt"
	"net/http"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/dgrijalva/jwt-go"
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
	if connectionClaims != nil && authenticatable.Room() == proto.RID(connectionClaims.RID) {
		log.Debugf("authenticateRoom: Valid RID in connectionClaims %v", authenticatable.Room())
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
	if msgClaims != nil && authenticatable.Room() == proto.RID(msgClaims.RID) {
		log.Debugf("authenticateRoom: Valid RID in msgClaims %v", authenticatable.Room())
		return nil
	}

	// If this is reached, a token was passed but it did not have a valid RID claim
	return errorUnauthorizedRoomAccess
}

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg json.RawMessage, accept signal.RespondFunc, reject signal.RejectFunc) {
	var result interface{}
	topErr := util.NewNpError(http.StatusBadRequest, fmt.Sprintf("Unkown method [%s]", method))

	//TODO DRY this up
	switch method {
	case proto.ClientJoin:
		var msgData proto.FromClientJoinMsg
		if topErr = ParseProtoo(msg, peer.Claims(), &msgData); topErr == nil {
			result, topErr = join(peer, msgData)
		}
	case proto.ClientOffer:
		var msgData proto.ClientNegotiationMsg
		if topErr = ParseProtoo(msg, peer.Claims(), &msgData); topErr == nil {
			result, topErr = offer(peer, msgData)
		}
	case proto.ClientTrickleICE:
		var msgData proto.ClientTrickleMsg
		if topErr = ParseProtoo(msg, peer.Claims(), &msgData); topErr == nil {
			result, topErr = trickle(peer, msgData)
		}
	case proto.ClientBroadcast:
		var msgData proto.FromClientBroadcastMsg
		if topErr = ParseProtoo(msg, peer.Claims(), &msgData); topErr == nil {
			result, topErr = broadcast(peer, msgData)
		}
	case proto.SignalClose:
		var msgData proto.SignalCloseMsg
		if topErr = ParseProtoo(msg, peer.Claims(), &msgData); topErr == nil {
			result, topErr = close(peer, msgData)
		}
	}

	if topErr != nil {
		reject(topErr.Code, topErr.Reason)
	} else {
		accept(result)
	}
}
