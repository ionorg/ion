package biz

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
)

var (
	errorTokenRequired          = newError(http.StatusUnauthorized, "Authorization token required for access")
	errorInvalidRoomToken       = newError(http.StatusUnauthorized, "Invalid room token")
	errorUnauthorizedRoomAccess = newError(http.StatusForbidden, "Permission not sufficient for room")
)

// parseProtoo Unmarshals a protoo payload.
func parseProtoo(msg json.RawMessage, connectionClaims *signal.Claims, msgType interface{}) *httpError {
	if err := json.Unmarshal(msg, &msgType); err != nil {
		log.Errorf("Biz.Entry parse error %v", err.Error())
		return newError(http.StatusBadRequest, fmt.Sprintf("Error parsing request object %v", err.Error()))
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
func authenticateRoom(msgType interface{}, connectionClaims *signal.Claims, authenticatable proto.Authenticatable) *httpError {
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
	err := newError(http.StatusBadRequest, fmt.Sprintf("Unkown method [%s]", method))

	switch method {
	case proto.ClientJoin:
		var msgData proto.FromClientJoinMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = join(peer, msgData)
		}
	case proto.ClientOffer:
		var msgData proto.ClientOfferMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = offer(peer, msgData)
		}
	case proto.ClientAnswer:
		var msgData proto.ClientAnswerMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = answer(peer, msgData)
		}
	case proto.ClientTrickleICE:
		var msgData proto.ClientTrickleMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = trickle(peer, msgData)
		}
	case proto.ClientBroadcast:
		var msgData proto.FromClientBroadcastMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = broadcast(peer, msgData)
		}
	case proto.ClientLeave:
		var msgData proto.FromClientLeaveMsg
		if err = parseProtoo(msg, peer.Claims(), &msgData); err == nil {
			result, err = leave(peer, msgData)
		}
	}

	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}
