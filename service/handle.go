package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/log"
	"github.com/pion/webrtc/v2"
)

const (
	MethodLogin       = "login"
	MethodJoin        = "join"
	MethodLeave       = "leave"
	MethodPublish     = "publish"
	MethodSubscribe   = "subscribe"
	MethodOnPublish   = "onPublish"
	MethodOnUnpublish = "onUnpublish"
)

type roomMap map[string]*Room

var (
	wsServer *server.WebSocketServer
	rooms    roomMap
	roomLock sync.RWMutex
)

func init() {
	rooms = make(map[string]*Room)
	peers = make(map[*peer.Peer]*Room)
}

func Start() {
	wsServer = server.NewWebSocketServer(handleNewWebSocket)
	config := server.DefaultConfig()
	config.Host = conf.Signal.Host
	config.Port, _ = strconv.Atoi(conf.Signal.Port)
	config.CertFile = conf.Signal.CertPem
	config.KeyFile = conf.Signal.KeyPem
	go wsServer.Bind(config)
}

func jsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

func handleNewWebSocket(transport *transport.WebSocketTransport, request *http.Request) {
	vars := request.URL.Query()
	peerId, _ := vars["peer"]
	if peerId == nil || len(peerId) < 1 {
		return
	}

	log.Infof("handleNewWebSocket , peerId => %s", peerId)

	signalPeer := peer.NewPeer(peerId[0], transport)

	handleRequest := func(request map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("handleRequest recover err => %v", err)
			}
		}()
		method := request["method"]
		data := request["data"]
		if method == nil || method == "" || data == nil || data == "" {
			log.Errorf("method => %v, data => %v", method, data)
			reject(-1, "invalid method or data")
			return
		}
		msg := data.(map[string]interface{})
		log.Infof("handleRequest method => %s, request => %v", method, request)
		switch method {
		case MethodLogin:
			processLogin(signalPeer, msg, accept, reject)
		case MethodJoin:
			processJoin(signalPeer, msg, accept, reject)
		case MethodLeave:
			processLeave(signalPeer, msg, accept, reject)
		case MethodPublish:
			processPublish(signalPeer, msg, accept, reject)
		case MethodSubscribe:
			processSubscribe(signalPeer, msg, accept, reject)
		}
	}

	handleNotification := func(notification map[string]interface{}) {
		logger.Infof("handleNotification => %s", notification["method"])
		method := notification["method"].(string)
		data := notification["data"].(map[string]interface{})
		//Forward notification to the room.
		r := getPeerRoom(signalPeer)
		if r != nil {
			r.Notify(signalPeer, method, data)
		}
	}

	handleClose := func() {
		logger.Infof("handleClose => signalPeer (%s) ", signalPeer.ID())
		deletePeerRoom(signalPeer)
	}

	signalPeer.On("request", handleRequest)
	signalPeer.On("notification", handleNotification)
	signalPeer.On("close", handleClose)
}

func processLogin(signalPeer *peer.Peer, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	accept(jsonEncode(`{}`))
}

func processJoin(signalPeer *peer.Peer, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	rid := req["rid"]
	if rid == nil {
		reject(-1, "rid not found")
		return
	}

	r := getRoom(rid.(string))
	if r == nil {
		r = createRoom(rid.(string))
	}
	r.AddPeer(signalPeer)
	addPeerRoom(signalPeer, r)
	onPublish := make(map[string]interface{})
	r.pubPeerLock.RLock()
	defer r.pubPeerLock.RUnlock()
	for peerId, _ := range r.pubPeers {
		if peerId != signalPeer.ID() {
			onPublish["pubid"] = peerId
			r.GetPeer(signalPeer.ID()).Notify(MethodOnPublish, onPublish)
		}
	}
	accept(jsonEncode(`{}`))
}

func processLeave(signalPeer *peer.Peer, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	rid := req["rid"]
	if rid == nil {
		reject(-1, "rid not found")
		return
	}

	//broadcast onUnpublish
	onUnpublish := make(map[string]interface{})
	id := signalPeer.ID()
	r := getRoom(rid.(string))
	if r == nil {
		reject(-1, "room not found")
		return
	}
	onUnpublish["pubid"] = id
	r.Notify(signalPeer, MethodOnUnpublish, onUnpublish)
	r.delWebRTCPeer(id, true)
	r.delWebRTCPeer(id, false)
	accept(jsonEncode(`{}`))
	deleteRoom(id)
}

func processPublish(signalPeer *peer.Peer, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	if req["jsep"] == nil {
		log.Errorf("jsep not found")
		reject(-1, "jsep not found")
		return
	}
	j := req["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found")
		reject(-1, "sdp not found")
		return
	}
	r := getPeerRoom(signalPeer)
	if r == nil {
		reject(-1, "room not found")
		return
	}
	r.addWebRTCPeer(signalPeer.ID(), true)
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.answer(signalPeer.ID(), "", jsep, true)
	if err != nil {
		log.Errorf("answer err=%v\n jsep=%v", err.Error(), jsep)
		reject(-1, err.Error())
		return
	}
	resp := make(map[string]interface{})
	resp["jsep"] = answer
	respByte, err := json.Marshal(resp)
	if err != nil {
		log.Errorf(err.Error())
		reject(-1, err.Error())
		return
	}
	respStr := string(respByte)
	if respStr != "" {
		accept(jsonEncode(respStr))
		// broad onPublish
		onPublish := make(map[string]interface{})
		onPublish["pubid"] = signalPeer.ID()
		r.Notify(signalPeer, MethodOnPublish, onPublish)
		return
	}
	reject(-1, "unknown error")
}

func processSubscribe(signalPeer *peer.Peer, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	if req["jsep"] == nil {
		log.Errorf("jsep not found")
		reject(-1, "jsep not found")
		return
	}
	j := req["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found in jsep")
		reject(-1, "sdp not found")
		return
	}
	r := getPeerRoom(signalPeer)
	r.addWebRTCPeer(signalPeer.ID(), false)
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.answer(signalPeer.ID(), req["pubid"].(string), jsep, false)
	if err != nil {
		log.Errorf("answer err=%v", err.Error())
		reject(-1, err.Error())
		return
	}
	resp := make(map[string]interface{})
	resp["jsep"] = answer
	jsepByte, err := json.Marshal(resp)
	if err != nil {
		log.Errorf(err.Error())
		reject(-1, err.Error())
		return
	}
	r.sendPLI(signalPeer.ID())
	jsepStr := string(jsepByte)
	if jsepStr != "" {
		accept(jsonEncode(jsepStr))
		return
	}
	reject(-1, "unknown error")
}
