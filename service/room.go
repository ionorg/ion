package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/room"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/log"

	"github.com/pion/sfu/media"
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

func jsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

// type roomMap map[string]*room.Room
type roomMap map[string]*Room

var (
	wsServer *server.WebSocketServer
	rooms    roomMap
	roomLock sync.RWMutex
)

func init() {
	rooms = make(map[string]*Room)
	wsServer = server.NewWebSocketServer(handleNewWebSocket)
	config := server.DefaultConfig()
	config.Host = conf.Cfg.Signal.Host
	config.Port, _ = strconv.Atoi(conf.Cfg.Signal.Port)
	config.CertFile = conf.Cfg.Signal.CertPem
	config.KeyFile = conf.Cfg.Signal.KeyPem
	go wsServer.Bind(config)
}

// func getRoom(id string) *room.Room {
func getRoom(id string) *Room {
	roomLock.RLock()
	defer roomLock.RUnlock()
	return rooms[id]
}

func createRoom(id string) *Room {
	roomLock.Lock()
	defer roomLock.Unlock()
	rooms[id] = NewRoom(id)
	return rooms[id]
}

func handleNewWebSocket(transport *transport.WebSocketTransport, request *http.Request) {
	vars := request.URL.Query()
	roomId, _ := vars["room"]
	if roomId == nil || len(roomId) < 1 {
		return
	}
	peerId, _ := vars["peer"]
	if peerId == nil || len(peerId) < 1 {
		return
	}

	log.Infof("handleNewWebSocket roomId => %s, peerId => %s", roomId, peerId)

	room := getRoom(roomId[0])
	if room == nil {
		room = createRoom(roomId[0])
	}

	signalPeer := room.GetPeer(peerId[0])
	if signalPeer == nil {
		signalPeer = room.CreatePeer(peerId[0], transport)
	}

	handleRequest := func(request map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
		method := request["method"]
		data := request["data"]
		if method == nil || method == "" || data == nil || data == "" {
			log.Errorf("method => %v, data => %v", method, data)
			reject(-1, "invalid method or data")
		}
		msg := data.(map[string]interface{})
		log.Infof("handleRequest method => %s, request => %v", method, request)
		switch method {
		case MethodLogin:
			room.processLogin(peerId[0], msg, accept, reject)
		case MethodJoin:
			room.processJoin(peerId[0], msg, accept, reject)
		case MethodLeave:
			room.processLeave(peerId[0], msg, accept, reject)
		case MethodPublish:
			room.processPublish(peerId[0], msg, accept, reject)
		case MethodSubscribe:
			room.processSubscribe(peerId[0], msg, accept, reject)
		}
	}

	handleNotification := func(notification map[string]interface{}) {
		logger.Infof("handleNotification => %s", notification["method"])
		method := notification["method"].(string)
		data := notification["data"].(map[string]interface{})
		//Forward notification to the room.
		room.Notify(signalPeer, method, data)
	}

	handleClose := func() {
		logger.Infof("handleClose => signalPeer (%s) ", signalPeer.ID())
	}

	signalPeer.On("request", handleRequest)
	signalPeer.On("notification", handleNotification)
	signalPeer.On("close", handleClose)
}

type Room struct {
	room.Room
	ID string

	pubPeers    map[string]*media.WebRTCPeer
	subPeers    map[string]*media.WebRTCPeer
	pubPeerLock sync.RWMutex
	subPeerLock sync.RWMutex
}

func NewRoom(id string) *Room {
	r := &Room{
		pubPeers: make(map[string]*media.WebRTCPeer),
		subPeers: make(map[string]*media.WebRTCPeer),
		ID:       id,
	}
	r.Room = *room.NewRoom(id)

	log.Infof("NewRoom r=%+v", r)
	return r
}

func (r *Room) GetWebRTCPeer(id string, sender bool) *media.WebRTCPeer {
	if sender {
		r.pubPeerLock.RLock()
		defer r.pubPeerLock.RUnlock()
		return r.pubPeers[id]
	} else {
		r.subPeerLock.RLock()
		defer r.subPeerLock.RUnlock()
		return r.subPeers[id]
	}
	return nil
}

func (r *Room) DelWebRTCPeer(id string, sender bool) {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		if r.pubPeers[id] != nil {
			if r.pubPeers[id].PC != nil {
				r.pubPeers[id].PC.Close()
			}
			r.pubPeers[id].Stop()
		}
		delete(r.pubPeers, id)

	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
		if r.subPeers[id] != nil {
			if r.subPeers[id].PC != nil {
				r.subPeers[id].PC.Close()
			}
			r.subPeers[id].Stop()
		}
		delete(r.subPeers, id)
	}
}

func (r *Room) AddWebRTCPeer(id string, sender bool) {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		if r.pubPeers[id] != nil {
			r.pubPeers[id].Stop()
		}
		r.pubPeers[id] = media.NewWebRTCPeer(id)
	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
		if r.subPeers[id] != nil {
			r.subPeers[id].Stop()
		}
		r.subPeers[id] = media.NewWebRTCPeer(id)
	}
}

func (r *Room) Answer(id string, pubid string, offer webrtc.SessionDescription, sender bool) (webrtc.SessionDescription, error) {
	log.Infof("Room.Answer id=%s, pubid=%s, offer=%v", id, pubid, offer)

	p := r.GetWebRTCPeer(id, sender)

	var err error
	var answer webrtc.SessionDescription
	if sender {
		answer, err = p.AnswerSender(offer)
	} else {
		r.pubPeerLock.RLock()
		pub := r.pubPeers[pubid]
		r.pubPeerLock.RUnlock()
		ticker := time.NewTicker(time.Millisecond * 2000)
		for {
			select {
			case <-ticker.C:
				goto ENDWAIT
			default:
				if pub.VideoTrack == nil || pub.AudioTrack == nil {
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	ENDWAIT:
		answer, err = p.AnswerReceiver(offer, &pub.VideoTrack, &pub.AudioTrack)
	}
	return answer, err
}

func (r *Room) Close() {
	log.Infof("Room.Close")
	r.pubPeerLock.Lock()
	defer r.pubPeerLock.Unlock()
	for _, v := range r.pubPeers {
		if v != nil {
			v.Stop()
			if v.PC != nil {
				v.PC.Close()
			}
		}
	}
}

func (r *Room) processLogin(client string, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	accept(jsonEncode(`{}`))
}

func (r *Room) processJoin(id string, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	accept(jsonEncode(`{}`))
}

func (r *Room) processLeave(id string, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	r.DelWebRTCPeer(id, true)
	r.DelWebRTCPeer(id, false)
	//broadcast onUnpublish
	onUnpublish := make(map[string]interface{})
	onUnpublish["pubid"] = id
	r.Notify(r.GetPeer(id), MethodOnUnpublish, onUnpublish)
	accept(jsonEncode(`{}`))
}

func (r *Room) processPublish(id string, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
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
	r.AddWebRTCPeer(id, true)
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.Answer(id, "", jsep, true)
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
		onPublish["type"] = "sender"
		onPublish["pubid"] = id
		r.Notify(r.GetPeer(id), MethodOnPublish, onPublish)
		peers := r.GetPeers()
		for peerId, item := range peers {
			if peerId != id {
				onPublish["pubid"] = item.ID()
				r.GetPeer(id).Notify(MethodOnPublish, onPublish)
			}
		}
		return
	}
	reject(-1, "unknown error")
}

func (r *Room) processSubscribe(id string, req map[string]interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
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

	r.AddWebRTCPeer(id, false)
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.Answer(id, req["pubid"].(string), jsep, false)
	if err != nil {
		log.Errorf("answer err=%v", err.Error())
		reject(-1, err.Error())
		return
	}
	jsepByte, err := json.Marshal(answer)
	if err != nil {
		log.Errorf(err.Error())
		reject(-1, err.Error())
		return
	}
	r.sendPLI(req["pubid"].(string))
	jsepStr := string(jsepByte)
	if jsepStr != "" {
		accept(jsonEncode(jsepStr))
		return
	}
	reject(-1, "unknown error")
}

func (r *Room) sendPLI(skipID string) {
	log.Infof("Room.sendPLI")
	r.pubPeerLock.RLock()
	defer r.pubPeerLock.RUnlock()
	for k, v := range r.pubPeers {
		if k != skipID {
			v.SendPLI()
		}
	}
}
