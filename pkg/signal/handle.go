package signal

import (
	"encoding/json"
	"net/http"

	"github.com/cloudwebrtc/go-protoo/logger"
	pr "github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	log "github.com/pion/ion-log"
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

// Valid validates clains
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

	uid := proto.UID(peerID[0])
	log.Infof("signal.in, uid => %s", uid)
	peer := newPeer(uid, transport, ForContext(request.Context()))

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
		bizCall(method, peer, data, accept, reject)
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
		bizCall(method, peer, data, accept, reject)
	}

	handleClose := func(code int, err string) {
		if allowClientDisconnect {
			log.Infof("signal.in handleClose.AllowDisconnected => peer (%s)", peer.ID())
			return
		}
		rooms := GetRoomsByPeer(peer.ID())
		log.Infof("signal.in handleClose [%d] %s rooms=%v", code, err, rooms)
		for _, r := range rooms {
			// only remove if its the same peer. If newer peer joined before the cleanup, leave it.
			if r.GetPeer(peer.ID()) == peer {
				//if code > 1000 {
				msgStr, _ := json.Marshal(proto.FromClientLeaveMsg{
					UID: proto.UID(peer.ID()),
					RID: r.ID(),
				})
				bizCall(proto.ClientLeave, peer, msgStr, accept, reject)
				//}
				log.Infof("signal.in handleClose delete peer (%s) from room (%s)", peer.ID(), r.ID())
				r.DelPeer(peer.ID())
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

func accept(data interface{}) {
	log.Infof("peer accept data=%v", data)
}

func reject(errorCode int, errorReason string) {
	log.Infof("reject errorCode=%v errorReason=%v", errorCode, errorReason)
}
