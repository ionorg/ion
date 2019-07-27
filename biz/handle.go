package biz

import (
	"fmt"
	"time"

	"github.com/pion/ion/conf"
	"github.com/pion/ion/gslb"
	"github.com/pion/ion/log"
	"github.com/pion/ion/rtc"
	"github.com/pion/ion/signal"
	"github.com/pion/ion/util"
	"github.com/pion/webrtc/v2"
	"go.etcd.io/etcd/clientv3"
)

const (
	MethodLogin       = "login"
	MethodJoin        = "join"
	MethodLeave       = "leave"
	MethodPublish     = "publish"
	MethodUnPublish   = "unpublish"
	MethodSubscribe   = "subscribe"
	MethodUnSubscribe = "unsubscribe"
	MethodOnPublish   = "onPublish"
	MethodOnUnpublish = "onUnpublish"

	errInvalidJsep  = "jsep not found"
	errInvalidSDP   = "sdp not found"
	errInvalidRoom  = "room not found"
	errInvalidPubID = "pubid not found"
	errInvalidAddr  = "addr not found"
)

func BizEntry(method string, peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	switch method {
	case MethodLogin:
		login(peer, msg, accept, reject)
	case MethodJoin:
		join(peer, msg, accept, reject)
	case MethodLeave:
		leave(peer, msg, accept, reject)
	case MethodPublish:
		publish(peer, msg, accept, reject)
	case MethodUnPublish:
		unpublish(peer, msg, accept, reject)
	case MethodSubscribe:
		subscribe(peer, msg, accept, reject)
	case MethodUnSubscribe:
		unsubscribe(peer, msg, accept, reject)
	case MethodOnPublish:
		onpublish(peer, msg, accept, reject)
	}
}

func login(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	//TODO auth check, maybe jwt
	accept(util.JsonEncode(`{}`))
}

// join room
func join(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	rid := util.GetValue(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	// add peer to signal room
	signal.AddPeerToRoom(rid, peer)

	// watching for some new pub
	gslb.SubWatch(rid, peer.ID(), watch)

	// when some body join this room, tell him the old pubs
	pubs := gslb.GetPubs(rid)
	for pid := range pubs {
		peer.Notify(MethodOnPublish, util.GetMap("pubid", pid, "rid", rid))
	}

	//return ok to signal peer
	accept(util.JsonEncode(`{}`))
}

func leave(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	rid := util.GetValue(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	// if pub is publishing
	if gslb.IsPub(rid, peer.ID()) {
		//onunpublish
		onUnpublish := util.GetMap("cmd", MethodOnUnpublish, "rid", rid, "pid", peer.ID())
		gslb.NotifySubs(rid, onUnpublish)
	}
	rtc.DelPub(peer.ID())

	accept(util.JsonEncode(`{}`))
	signal.DeletePeerFromRoom(rid, peer.ID())
	gslb.DelWatch(rid, peer.ID())
}

func publish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("publish peer=%v", peer)
	if msg["jsep"] == nil {
		log.Errorf(errInvalidJsep)
		reject(-1, errInvalidJsep)
		return
	}

	j := msg["jsep"].(map[string]interface{})
	if j["sdp"] == nil {
		log.Errorf(errInvalidSDP)
		reject(-1, errInvalidSDP)
		return
	}

	sdp := util.GetValue(j, "sdp")

	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		reject(-1, errInvalidRoom)
		return
	}

	jsep := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := rtc.AddNewWebRTCPub(peer.ID()).AnswerPublish(room.ID(), jsep)
	if err != nil {
		log.Errorf("answer err=%v jsep=%v", err.Error(), jsep)
		reject(-1, err.Error())
		return
	}

	accept(util.GetMap("jsep", answer))
	gslb.PubWatch(room.ID(), peer.ID(), watch)
	gslb.NotifySubs(room.ID(), util.GetMap("pid", peer.ID(), "rid", room.ID(), "cmd", MethodPublish))
}

func watch(rch clientv3.WatchChan) {
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if len(ev.Kv.Value) == 0 {
				log.Warnf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				continue
			}
			log.Infof("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			m := util.JsonEncode(string(ev.Kv.Value))
			cmd := util.GetValue(m, "cmd")
			switch cmd {
			case MethodSubscribe:
				sid := util.GetValue(m, "sid")
				pid := gslb.GetPubID(ev.Kv.Key)
				addr := util.GetValue(m, "addr")
				rtc.AddNewRTPSub(pid, sid, addr)
			case MethodPublish:
				//PUT "ion://room/room1/sub/" : "{\"cmd\":\"publish\",\"id\":\"fa78f2cc-035d-4e52-b904-25fe6bff0294\",\"rid\":\"room1\"}"
				pid := util.GetValue(m, "pid")
				id := util.GetValue(m, "id")
				rid := util.GetValue(m, "rid")
				onpublish := util.GetMap("pubid", pid)
				signal.NotifyByID(rid, id, pid, MethodOnPublish, onpublish)
			case MethodOnUnpublish:
				pid := util.GetValue(m, "pid")
				rid := util.GetValue(m, "rid")
				onUnpublish := util.GetMap("pubid", pid)
				signal.NotifyAllWithoutID(rid, pid, MethodOnUnpublish, onUnpublish)
			}
		}
	}
}

// unpublish from app
func unpublish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {

	//broadcast onUnpublish
	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		reject(-1, errInvalidRoom)
		return
	}

	onUnpublish := util.GetMap("rid", room.ID(), "pubid", peer.ID())
	signal.NotifyAllWithoutPeer(room.ID(), peer, MethodOnUnpublish, onUnpublish)
	rtc.DelPub(peer.ID())
	accept(util.JsonEncode(`{}`))
}

func subscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	j := msg["jsep"].(map[string]interface{})
	sdp := util.GetValue(j, "sdp")
	if sdp == "" {
		log.Errorf(errInvalidSDP)
		reject(-1, errInvalidSDP)
		return
	}

	pid := util.GetValue(msg, "pubid")
	if pid == "" {
		log.Errorf(errInvalidPubID)
		reject(-1, errInvalidPubID)
		return
	}

	room := signal.GetRoomByPeer(peer.ID())
	jsep := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	var answer webrtc.SessionDescription
	var err error
	webrtcSub := rtc.AddNewWebRTCSub(pid, peer.ID())
	pub := rtc.GetPub(pid)
	log.Infof("pub=%v", pub)
	switch pub.(type) {
	case *rtc.WebRTCTransport:
		//pub is on this ion
		wt := pub.(*rtc.WebRTCTransport)
		// payloadSSRC := gslb.GetMediaInfo(room.ID(), pid)
		answer, err = webrtcSub.AnswerSubscribe(jsep, wt.PayloadSSRC(), pid)
	case *rtc.RTPTransport:
		// the pub is on other ion, rtp pub already exist
		rt := pub.(*rtc.RTPTransport)
		answer, err = webrtcSub.AnswerSubscribe(jsep, rt.PayloadSSRC(), pid)
	default:
		// the pub is on other ion, rtp pub not exist
		payloadSSRC := gslb.GetMediaInfo(room.ID(), pid)
		for i := 0; len(payloadSSRC) < 2; payloadSSRC = gslb.GetMediaInfo(room.ID(), pid) {
			if i > 20 {
				break
			}
			time.Sleep(10 * time.Millisecond)
			i++
		}
		addr := fmt.Sprintf("%s:%d", conf.Global.AdveritiseIP, conf.Rtp.Port)
		// tell pub's ion to send rtp stream
		gslb.NotifyPub(room.ID(), pid, util.GetMap("sid", peer.ID(), "cmd", MethodSubscribe, "addr", addr))
		answer, err = webrtcSub.AnswerSubscribe(jsep, payloadSSRC, pid)
	}

	if err != nil {
		log.Errorf("answer err=%v", err.Error())
		reject(-1, err.Error())
		return
	}
	accept(util.GetMap("jsep", answer))
}

func unsubscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	pid := util.GetValue(msg, "pubid")
	if pid == "" {
		log.Errorf(errInvalidPubID)
		reject(-1, errInvalidPubID)
		return
	}

	// if this is from app, ion delete the webrtctransport sub
	// if this is from ion, ion delete the rtprtctransport sub
	// if the ion's pub is rtptransport, ion should check if subs is 0; if 0 ion should send unsubscribe and delete the pub, then delete the room from the pipeline
	//
	rtc.DelSub(pid, peer.ID())
	accept(util.JsonEncode(`{}`))
}

// onpublish from other ion
func onpublish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	rid := util.GetValue(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	pid := util.GetValue(msg, "pid")
	if pid == "" {
		reject(-1, errInvalidPubID)
		return
	}

	// tell other subs onPublish
	log.Infof("signal.NotifyAllWithoutPeer rid=%s peer=%s, onpublish, msg=%v", rid, peer.ID(), msg)
	signal.NotifyAllWithoutPeer(rid, peer, MethodOnPublish, msg)

	// upload the person number
	accept(util.JsonEncode(`{}`))
}
