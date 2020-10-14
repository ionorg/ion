package signal

import (
	"encoding/json"
	"net/http"

	"github.com/cloudwebrtc/go-protoo/logger"
	pr "github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

var (
	errorTokenClaimsInvalid = errors.Errorf("Token claims invalid: must have RID or UID")
)

// Claims supported in JWT
type Claims struct {
	UID string `json:"uid"`
	RID string `json:"rid"`
	*jwt.StandardClaims
}

func (c *Claims) Valid() error {
	if c.RID == "" && c.UID == "" {
		return errorTokenClaimsInvalid
	}

	if c.StandardClaims != nil {
		return c.StandardClaims.Valid()
	}
	return nil
}

func in(transport *transport.WebSocketTransport, request *http.Request) {
	vars := request.URL.Query()
	peerID := vars["peer"]
	if peerID == nil || len(peerID) < 1 {
		return
	}

	id := peerID[0]
	log.Infof("signal.in, id => %s", id)
	peer := newPeer(id, transport)
	connectionClaims := ForContext(request.Context())

	handleRequest := func(request pr.Request, accept func(interface{}), reject func(errorCode int, errorReason string)) {
		defer util.Recover("signal.in handleRequest")
		method := request.Method
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, errInvalidMethod)
			return
		}

		data := request.Data
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, errInvalidData)
			return
		}

		log.Infof("signal.in handleRequest id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, data, connectionClaims, accept, reject)
	}

	handleNotification := func(notification pr.Notification) {
		defer util.Recover("signal.in handleNotification")
		method := notification.Method
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, errInvalidMethod)
			return
		}

		data := notification.Data
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, errInvalidData)
			return
		}

		// msg := data.(map[string]interface{})
		log.Infof("signal.in handleNotification id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, data, connectionClaims, emptyAccept, reject)
	}

	handleClose := func(code int, err string) {
		if allowClientDisconnect {
			log.Infof("signal.in handleClose.AllowDisconnected => peer (%s)", peer.ID())
			return
		}

		roomLock.RLock()
		defer roomLock.RUnlock()
		rooms := GetRoomsByPeer(peer.ID())
		log.Infof("signal.in handleClose [%d] %s rooms=%v", code, err, rooms)
		for _, room := range rooms {
			if room != nil {
				oldPeer := room.GetPeer(peer.ID())
				// only remove if its the same peer. If newer peer joined before the cleanup, leave it.
				if oldPeer == &peer.Peer {
					if code > 1000 {
						msg := proto.SignalCloseMsg{
							RoomInfo: proto.RoomInfo{UID: proto.UID(peer.ID()), RID: room.ID()},
						}
						msgStr, _ := json.Marshal(msg)
						bizCall(proto.SignalClose, peer, msgStr, connectionClaims, emptyAccept, reject)
					}
					log.Infof("signal.in handleClose removing peer (%s) from room (%s)", peer.ID(), room.ID())
					room.RemovePeer(peer.ID())
				}
			}
		}
		log.Infof("signal.in handleClose => peer (%s) ", peer.ID())
	}

	for {
		select {
		case msg := <-peer.OnNotification:
			logger.Debugf("Handle Notification")
			handleNotification(msg)
		case msg := <-peer.OnRequest:
			handleRequest(msg.Request, msg.Accept, msg.Reject)
			logger.Debugf("Handle request")
		case msg := <-peer.OnClose:
			logger.Debugf("Handle Peer closing")
			handleClose(msg.Code, msg.Text)
		}
	}
}
