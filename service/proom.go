package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwebrtc/go-protoo/logger"
	ppeer "github.com/cloudwebrtc/go-protoo/peer"
	proom "github.com/cloudwebrtc/go-protoo/room"
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

// type roomMap map[string]*proom.Room
type roomMap map[string]*PRoom

var (
	wsServer *server.WebSocketServer
	rooms    roomMap
	roomLock sync.RWMutex
)

func init() {
	rooms = make(map[string]*PRoom)
	wsServer = server.NewWebSocketServer(handleNewWebSocket)
	config := server.DefaultConfig()
	config.Host = conf.Cfg.Protoo.Host
	config.Port, _ = strconv.Atoi(conf.Cfg.Protoo.Port)
	config.CertFile = conf.Cfg.Protoo.CertPem
	config.KeyFile = conf.Cfg.Protoo.KeyPem
	go wsServer.Bind(config)
}

// func getRoom(id string) *proom.Room {
func getRoom(id string) *PRoom {
	roomLock.RLock()
	defer roomLock.RUnlock()
	return rooms[id]
}

func createRoom(id string) *PRoom {
	roomLock.Lock()
	defer roomLock.Unlock()
	rooms[id] = NewPRoom(id)
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

	room := getRoom(roomId[0])
	if room == nil {
		room = createRoom(roomId[0])
	}

	// TODO room.GetPeer
	peer := room.GetPeer(peerId[0])
	if peer == nil {
		peer = room.CreatePeer(peerId[0], transport)
	}

	handleRequest := func(request map[string]interface{}, accept ppeer.AcceptFunc, reject ppeer.RejectFunc) {

		method := request["method"]
		log.Infof("method==========%v, request==========%v", method, request)

		/*handle login and offer request*/
		switch method {
		case MethodLogin:
			accept(jsonEncode(`{"name":"xxxx","status":"login"}`))
		case MethodJoin:
			accept(jsonEncode(`{}`))
		case MethodLeave:
			accept(jsonEncode(`{}`))
			//broadcast onUnpublish
			onUnpublish := make(map[string]interface{})
			onUnpublish["pubid"] = peerId[0]
			room.Notify(room.GetPeer(peerId[0]), MethodOnUnpublish, onUnpublish)

		case MethodPublish:
			resp := room.processPublish(peerId[0], request["data"].(map[string]interface{}))
			if resp != "" {
				accept(jsonEncode(resp))
				// broad onPublish
				onPublish := make(map[string]interface{})
				onPublish["type"] = "sender"
				onPublish["pubid"] = peerId[0]
				room.Notify(room.GetPeer(peerId[0]), MethodOnPublish, onPublish)
			} else {
				reject(-1, "")
			}
		case MethodSubscribe:
			resp := room.processSubscribe(peerId[0], request)
			if resp != "" {
				accept(jsonEncode(resp))
			} else {
				reject(-1, "")
			}
		}
	}

	handleNotification := func(notification map[string]interface{}) {
		logger.Infof("handleNotification => %s", notification["method"])

		method := notification["method"].(string)
		data := notification["data"].(map[string]interface{})

		//Forward notification to testRoom.
		room.Notify(peer, method, data)
	}

	handleClose := func() {
		logger.Infof("handleClose => peer (%s) ", peer.ID())
	}

	peer.On("request", handleRequest)
	peer.On("notification", handleNotification)
	peer.On("close", handleClose)
}

type PRoom struct {
	proom.Room
	ID       string
	roomLock sync.RWMutex

	pubPeers    map[string]*media.WebRTCPeer
	subPeers    map[string]*media.WebRTCPeer
	pubPeerLock sync.RWMutex
	subPeerLock sync.RWMutex

	eventLock sync.RWMutex
	reqQueue  chan ReqMsg
	respQueue chan RespMsg
	quit      chan bool
}

func NewPRoom(id string) *PRoom {
	r := &PRoom{
		pubPeers:  make(map[string]*media.WebRTCPeer),
		subPeers:  make(map[string]*media.WebRTCPeer),
		ID:        id,
		reqQueue:  make(chan ReqMsg, 1000),
		respQueue: make(chan RespMsg, 1000),
		quit:      make(chan bool),
	}
	r.Room = *proom.NewRoom(id)

	log.Infof("NewPRoom r=%+v", r)
	return r
}

func (r *PRoom) GetWebRTCPeer(id string, sender bool) *media.WebRTCPeer {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		if r.pubPeers[id] == nil {
			r.AddWebRTCPeer(id, sender)
		}
		return r.pubPeers[id]
	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
		if r.subPeers[id] == nil {
			r.AddWebRTCPeer(id, sender)
		}
		return r.subPeers[id]
	}
	return nil
}

func (r *PRoom) DelWebRTCPeer(id string, sender bool) {
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

func (r *PRoom) AddWebRTCPeer(id string, sender bool) {
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

func (r *PRoom) Answer(id string, pubid string, offer webrtc.SessionDescription, sender bool) (webrtc.SessionDescription, error) {
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

func (r *PRoom) Run() {
	for {
		time.Sleep(time.Second)
	}
}

func (r *PRoom) Close() {
	close(r.quit)
	log.Infof("Room.Close")
}

func (r *PRoom) processJoin(client string, req map[string]interface{}) {
	//a new one joined room, send onPublish to him
	if req["type"].(string) == "sender" {
		r.AddWebRTCPeer(client, true)
	} else if req["type"].(string) == "recver" {
		r.AddWebRTCPeer(client, false)
	}

}

func (r *PRoom) processLeave(req ReqMsg) {
	r.DelWebRTCPeer(req.client, true)
	r.DelWebRTCPeer(req.client, false)
}

func (r *PRoom) processPublish(id string, req map[string]interface{}) string {
	if req == nil {
		log.Errorf("req == nil")
		return ""
	}
	log.Infof("req==========%+v", req)
	if req["jsep"] == nil {
		log.Errorf("jsep not found in map")
		return ""
	}
	j := req["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found in jsep")
		return ""
	}
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.Answer(id, "", jsep, true)
	if err != nil {
		log.Errorf("answer err=%v\n jsep=%v", err.Error(), jsep)
		return ""
	}
	log.Infof("answer===========%v", answer)
	jsepByte, err := json.Marshal(answer)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}
	return string(jsepByte)
}

func (r *PRoom) processSubscribe(id string, req map[string]interface{}) string {
	if req["jsep"] == nil {
		log.Errorf("jsep not found in map")
		return ""
	}
	j := req["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found in jsep")
		return ""
	}
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	answer, err := r.Answer(id, req["pubid"].(string), jsep, false)
	if err != nil {
		log.Errorf("answer err=%v", err.Error())
		return ""
	}
	jsepByte, err := json.Marshal(answer)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}
	return string(jsepByte)
}

func (r *PRoom) sendPLI(skipID string) {
	log.Infof("Room.sendPLI")
	r.pubPeerLock.RLock()
	defer r.pubPeerLock.RUnlock()
	for k, v := range r.pubPeers {
		if k != skipID {
			v.SendPLI()
		}
	}
}
