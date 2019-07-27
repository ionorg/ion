package signal

import (
	"net/http"

	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/pion/ion/log"
	"github.com/pion/ion/util"
)

var (
	bizCall func(method string, peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc)
)

const (
	ErrInvalidMethod = "method not found"
	ErrInvalidData   = "data not found"
)

func SetBizCall(i interface{}) {
	bizCall = i.(func(method string, peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc))
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

	handleRequest := func(request map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
		// defer util.Recover("signal.in handleRequest")
		method := util.GetValue(request, "method")
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, ErrInvalidMethod)
			return
		}

		data := request["data"]
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		log.Infof("signal.in handleRequest id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, msg, accept, reject)
	}

	handleNotification := func(notification map[string]interface{}) {
		defer util.Recover("signal.in handleNotification")
		method := util.GetValue(notification, "method")
		if method == "" {
			log.Errorf("method => %v", method)
			reject(-1, ErrInvalidMethod)
			return
		}

		data := notification["data"]
		if data == nil {
			log.Errorf("data => %v", data)
			reject(-1, ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		log.Infof("signal.in handleNotification id=%s method => %s", peer.ID(), method)
		bizCall(method, peer, msg, accept, reject)
	}

	handleClose := func() {
		rooms := GetRoomsByPeer(peer.ID())
		for _, room := range rooms {
			if room != nil {
				room.RemovePeer(peer.ID())
			}
		}
		log.Infof("signal.in handleClose => peer (%s) ", peer.ID())
	}

	peer.On("request", handleRequest)
	peer.On("notification", handleNotification)
	peer.On("close", handleClose)
}
