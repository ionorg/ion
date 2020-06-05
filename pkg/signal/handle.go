package signal

import (
	"encoding/json"
	"net/http"

	"github.com/cloudwebrtc/go-protoo/logger"
	pr "github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func in(transport *transport.WebSocketTransport, request *http.Request) {
	vars := request.URL.Query()
	peerID := vars["peer"]
	if peerID == nil || len(peerID) < 1 {
		return
	}

	id := peerID[0]
	log.Infof("signal.in, id => %s", id)
	peer := newPeer(id, transport)

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
		bizCall(method, peer, data, emptyAccept, reject)
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
				if code > 1000 {
					msg := proto.LeaveMsg{
						RoomInfo: proto.RoomInfo{RID: room.ID()},
					}
					msgStr, _ := json.Marshal(msg)
					bizCall(proto.ClientLeave, peer, msgStr, emptyAccept, reject)
				}
				room.RemovePeer(peer.ID())
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
