package service

import (
	"sync"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/pion/sfu/log"

	"github.com/pion/sfu/media"
	"github.com/pion/sfu/signal"
	"github.com/pion/webrtc/v2"
)

type Room struct {
	ID string

	pubPeers    map[string]*media.WebRTCPeer
	subPeers    map[string]*media.WebRTCPeer
	pubPeerLock sync.RWMutex
	subPeerLock sync.RWMutex

	signal        *signal.Client
	signalSubs    map[string]*centrifuge.Subscription
	signalSubLock sync.RWMutex

	eventLock sync.RWMutex
	reqQueue  chan ReqMsg
	respQueue chan RespMsg
	quit      chan bool
}

func NewRoom(id string) *Room {
	r := &Room{
		pubPeers:   make(map[string]*media.WebRTCPeer),
		subPeers:   make(map[string]*media.WebRTCPeer),
		signal:     signal.NewClient(),
		signalSubs: make(map[string]*centrifuge.Subscription),
		ID:         id,
		reqQueue:   make(chan ReqMsg, 1000),
		respQueue:  make(chan RespMsg, 1000),
		quit:       make(chan bool),
	}

	r.signal.SetEventCallback(r.EventCall)
	if err := r.signal.Connect(); err != nil {
		panic(err)
	}
	r.signal.Subscribe(id)
	log.Infof("NewRoom id=%s", id)
	return r
}

func (r *Room) AddSignal(id string) {
	r.signalSubLock.Lock()
	defer r.signalSubLock.Unlock()
	if r.signalSubs[id] == nil {
		r.signalSubs[id] = r.signal.Subscribe(id)
	}
}

func (r *Room) DelSignal(id string) {
	r.signalSubLock.Lock()
	defer r.signalSubLock.Unlock()
	if r.signalSubs[id] != nil {
		r.signalSubs[id].Unsubscribe()
		delete(r.signalSubs, id)
	}
}

func (r *Room) SignalBroadcast(data string, skipID string) {
	log.Infof("Room.SignalBroadcast %s skipID=%s", data, skipID)
	r.signalSubLock.RLock()
	defer r.signalSubLock.RUnlock()
	for k, v := range r.signalSubs {
		if k != skipID {
			go v.Publish([]byte(data))
		}
	}
}

func (r *Room) SignalPublish(data string, id string) {
	log.Infof("Room.SignalPublish to %s", id)
	r.signalSubLock.RLock()
	defer r.signalSubLock.RUnlock()
	if sub, ok := r.signalSubs[id]; ok {
		sub.Publish([]byte(data))
	}
}

func (r *Room) GetWebRTCPeer(id string, sender bool) *media.WebRTCPeer {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		return r.pubPeers[id]
	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
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
	log.Infof("Room.Answer id=%s", id)

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

func (r *Room) EventCall(event, channel, data string) {
	r.eventLock.Lock()
	defer r.eventLock.Unlock()
	switch event {
	case signal.EventOnPublish:
		msg := ReqUnmarshal(data)
		log.Infof("EventCall msg.Req=%v channel=%v", msg.Req, channel)
		switch msg.Req {
		case ReqJoin, ReqLeave:
			msg.client = msg.Msg["client"].(string)
			r.reqQueue <- msg
		case ReqPublish, ReqSubscribe:
			msg.client = channel
			r.reqQueue <- msg
		default:
			log.Warnf("unknown msg.Req = %s", msg.Req)
		}
	}
}

func (r *Room) Run() {
	for {
		select {
		case req := <-r.reqQueue:
			switch req.Req {
			case ReqJoin:
				r.processJoin(req)
			case ReqLeave:
				r.processLeave(req)
			case ReqPublish:
				r.processPublish(req)
			case ReqSubscribe:
				r.processSubscribe(req)
			case ReqOnPublish:
				r.processOnPublish(req)
			case ReqOnUnpublish:
				r.processOnUnpublish(req)
			}
		case resp := <-r.respQueue:
			log.Infof("r.SignalPublish resp.client=%s resp.resp=%s", resp.client, resp.Resp)
			r.SignalPublish(RespMarshal(resp), resp.client)
		case <-r.quit:
			return
		}
	}
}

func (r *Room) Close() {
	close(r.quit)
	log.Infof("Room.Close")
}

func (r *Room) processJoin(req ReqMsg) {
	//a new one joined room, send onPublish to him
	r.pubPeerLock.RLock()
	for client, _ := range r.pubPeers {
		if client != req.Msg["client"] {
			onPublishMsg := ReqMsg{
				Req: ReqOnPublish,
				ID:  0,
				Msg: make(map[string]interface{}),
			}
			onPublishMsg.Msg["type"] = "sender"
			onPublishMsg.Msg["pubid"] = client
			onPublishMsg.client = req.client
			r.reqQueue <- onPublishMsg
		}
	}
	r.pubPeerLock.RUnlock()
	if req.Msg["type"].(string) == "sender" {
		r.AddSignal(req.client)
		r.AddWebRTCPeer(req.client, true)
	}
}

func (r *Room) processLeave(req ReqMsg) {
	r.pubPeerLock.RLock()
	for client, _ := range r.subPeers {
		if client != req.Msg["client"] {
			onUnpublishMsg := ReqMsg{
				Req: ReqOnUnpublish,
				ID:  0,
				Msg: make(map[string]interface{}),
			}
			onUnpublishMsg.Msg["pubid"] = client
			onUnpublishMsg.client = r.ID
			r.reqQueue <- onUnpublishMsg
		}
	}
	r.pubPeerLock.RUnlock()
	r.DelWebRTCPeer(req.client, true)
	r.DelWebRTCPeer(req.client, false)
	r.DelSignal(req.client)
}

func (r *Room) processPublish(req ReqMsg) {
	if req.Msg["jsep"] == nil {
		log.Errorf("jsep not found in map")
		return
	}
	j := req.Msg["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found in jsep")
		return
	}
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	pubResp := RespMsg{Resp: "success", ID: req.ID, Msg: make(map[string]interface{}), client: req.client}
	answer, err := r.Answer(req.client, "", jsep, true)
	if err != nil {
		log.Errorf("answer err=%v\n jsep=%v", err.Error(), jsep)
		return
	}
	pubResp.Msg["type"] = req.Msg["type"]
	pubResp.Msg["jsep"] = answer

	r.respQueue <- pubResp
	if req.Msg["type"] == "sender" {
		onPublishMsg := ReqMsg{
			Req: ReqOnPublish,
			ID:  0,
			Msg: make(map[string]interface{}),
		}
		onPublishMsg.Msg["type"] = "sender"
		onPublishMsg.Msg["pubid"] = req.client
		r.signalSubLock.RLock()
		for k, _ := range r.signalSubs {
			if k != req.client {
				onPublishMsg.client = k
				r.reqQueue <- onPublishMsg
			}
		}
		r.signalSubLock.RUnlock()
	}
}

func (r *Room) processSubscribe(req ReqMsg) {
	if req.Msg["jsep"] == nil {
		log.Errorf("jsep not found in map")
		return
	}
	j := req.Msg["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf("sdp not found in jsep")
		return
	}
	jsep := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  j["sdp"].(string),
	}
	r.AddSignal(req.client)
	r.AddWebRTCPeer(req.client, false)
	answer, err := r.Answer(req.client, req.Msg["pubid"].(string), jsep, false)
	if err != nil {
		log.Errorf("answer err=%v", err.Error())
		return
	}
	resp := RespMsg{Resp: "success", ID: req.ID, Msg: make(map[string]interface{}), client: req.client}

	resp.Msg["jsep"] = answer
	resp.Msg["pubid"] = req.Msg["pubid"].(string)
	r.sendPLI(req.client)
	r.respQueue <- resp
}

func (r *Room) processOnPublish(req ReqMsg) {
	// r.SignalBroadcast(ReqMarshal(req), req.Msg["pubid"].(string))
	r.SignalPublish(ReqMarshal(req), req.client)
}

func (r *Room) processOnUnpublish(req ReqMsg) {
	r.SignalBroadcast(ReqMarshal(req), req.Msg["pubid"].(string))
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
