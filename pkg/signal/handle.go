package signal

import (
	"net/http"

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

	handleRequest := func(request map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
		defer util.Recover("signal.in handleRequest")
		method := util.Val(request, "method")
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, errInvalidMethod)
			return
		}

		data := request["data"]
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, errInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		log.Infof("signal.in handleRequest id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, msg, accept, reject)
	}

	handleNotification := func(notification map[string]interface{}) {
		defer util.Recover("signal.in handleNotification")
		method := util.Val(notification, "method")
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, errInvalidMethod)
			return
		}

		data := notification["data"]
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, errInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		log.Infof("signal.in handleNotification id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, msg, accept, reject)
	}

	handleClose := func(code int, err string) {
		rooms := GetRoomsByPeer(peer.ID())
		log.Infof("signal.in handleClose [%d] %s rooms=%v", code, err, rooms)
		for _, room := range rooms {
			if room != nil {
				if code > 1000 {
					msg := make(map[string]interface{})
					msg["rid"] = room.ID()
					bizCall(proto.ClientClose, peer, msg, accept, reject)
				}
				room.RemovePeer(peer.ID())
			}
		}
		log.Infof("signal.in handleClose => peer (%s) ", peer.ID())
	}

	peer.On("request", handleRequest)
	peer.On("notification", handleNotification)
	peer.On("close", handleClose)
	peer.On("error", handleClose)
}
