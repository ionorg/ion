package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
)

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	switch method {
	case proto.ClientClose:
		clientClose(peer, msg, accept, reject)
	case proto.ClientLogin:
		login(peer, msg, accept, reject)
	case proto.ClientJoin:
		join(peer, msg, accept, reject)
	case proto.ClientLeave:
		leave(peer, msg, accept, reject)
	case proto.ClientPublish:
		publish(peer, msg, accept, reject)
	case proto.ClientUnPublish:
		unpublish(peer, msg, accept, reject)
	case proto.ClientSubscribe:
		subscribe(peer, msg, accept, reject)
	case proto.ClientUnSubscribe:
		unsubscribe(peer, msg, accept, reject)
	case proto.ClientBroadcast:
		broadcast(peer, msg, accept, reject)
	}
}

func login(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.login peer.ID()=%s msg=%v", peer.ID(), msg)
	//TODO auth check, maybe jwt
	accept(emptyMap)
}

// join room
func join(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) {
		return
	}

	rid := util.Val(msg, "rid")
	//already joined this room
	if signal.HasPeer(rid, peer) {
		accept(emptyMap)
		return
	}

	info := util.Val(msg, "info")
	signal.AddPeer(rid, peer)

	amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", peer.ID(), "info", info), "")

	respHandler := func(m map[string]interface{}) {
		info := m["info"]
		mid := m["mid"]
		uid := m["uid"]
		log.Infof("biz.join respHandler mid=%v info=%v", mid, info)
		if mid != "" {
			peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "info", info))
		}
	}
	// find pubs from islb ,skip this ion
	log.Infof("amqp.RpcCallWithResp")
	amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbGetPubs, "rid", rid, "uid", peer.ID()), respHandler)
	accept(emptyMap)
}

func leave(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")
	if invalid(msg, "rid", reject) {
		return
	}

	rid := util.Val(msg, "rid")
	// if this is a webrtc pub
	mids := rtc.GetWebRtcMIDByPID(peer.ID())
	for _, mid := range mids {
		// tell islb stream-remove
		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", peer.ID(), "mid", mid), "")

		rtc.DelPub(mid)
		quitChMapLock.Lock()
		for k := range quitChMap {
			if strings.Contains(k, mid) {
				close(quitChMap[k])
				delete(quitChMap, k)
			}
		}
		quitChMapLock.Unlock()

		// del sub and get the rtp's pub which has none sub
		noSubRtpPubMid := rtc.DelSubFromAllPub(mid)
		// del pub which has none sub when received resp
		log.Infof("biz.leave noSubRtpPubMid=%v", noSubRtpPubMid)
		for mid := range noSubRtpPubMid {
			respUnrelayHandler := func(m map[string]interface{}) {
				log.Infof("biz.leave respUnrelayHandler m=%v", m)
				mid := util.Val(m, "mid")
				rtc.DelPub(mid)
			}
			// tell islb stop relay
			amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbUnrelay, "rid", rid, "mid", mid), respUnrelayHandler)
		}
	}
	amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbClientOnLeave, "rid", rid, "uid", peer.ID()), "")

	accept(emptyMap)
	signal.DelPeer(rid, peer.ID())
}

func clientClose(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	leave(peer, msg, accept, reject)
}

func publish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.publish peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	j := msg["jsep"].(map[string]interface{})
	if invalid(j, "sdp", reject) {
		return
	}

	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		reject(codeRoomErr, codeStr(codeRoomErr))
		return
	}

	sdp := util.Val(j, "sdp")
	options := msg["options"].(map[string]interface{})
	mid := getMID(peer.ID())
	jsep := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	islbStoreSsrc := func(ssrc uint32, pt uint8) {
		ssrcPt := fmt.Sprintf("{\"%d\":%d}", ssrc, pt)
		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamAdd, "rid", room.ID(), "uid", peer.ID(), "mid", mid, "mediaInfo", ssrcPt), "")
		keepAlive := getKeepAliveID(mid, ssrc)
		quitChMapLock.Lock()
		quitChMap[keepAlive] = make(chan struct{})
		quitChMapLock.Unlock()
		go func() {
			t := time.NewTicker(500 * time.Millisecond)
			for {
				select {
				case <-t.C:
					amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbKeepAlive, "rid", room.ID(), "uid", peer.ID(), "mid", mid, "mediaInfo", ssrcPt), "")
				case <-quitChMap[keepAlive]:
					return
				}
			}
		}()
	}

	answer, err := rtc.AddNewWebRTCPub(mid).AnswerPublish(room.ID(), jsep, options, islbStoreSsrc)
	if err != nil {
		log.Errorf("biz.publish answer err=%s jsep=%v", err.Error(), jsep)
		reject(codePublishErr, codeStr(codePublishErr))
		return
	}

	accept(util.Map("jsep", answer, "mid", mid))
}

// unpublish from app
func unpublish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)
	//get rid from msg, because room may be already deleted from signal
	//so we can't get rid from signal's room
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	mid := util.Val(msg, "mid")
	// if this mid is a webrtc pub
	if rtc.IsWebRtcPub(mid) {
		// tell islb stream-remove, `rtc.DelPub(mid)` will be done when islb braodcast stream-remove
		quitChMapLock.Lock()
		for k := range quitChMap {
			if strings.Contains(k, mid) {
				close(quitChMap[k])
				delete(quitChMap, k)
			}
		}
		quitChMapLock.Unlock()
	}

	accept(emptyMap)
}

func subscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.subscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "jsep", reject) || invalid(msg, "mid", reject) {
		return
	}
	j := msg["jsep"].(map[string]interface{})
	if invalid(j, "sdp", reject) {
		return
	}

	mid, sdp := util.Val(msg, "mid"), util.Val(j, "sdp")
	room := signal.GetRoomByPeer(peer.ID())
	jsep := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	var answer webrtc.SessionDescription
	var err error
	webrtcSub := rtc.AddNewWebRTCSub(mid, peer.ID())
	pub := rtc.GetPub(mid)
	switch pub.(type) {
	case *rtc.WebRTCTransport:
		//pub is on this ion
		wt := pub.(*rtc.WebRTCTransport)
		ssrcPT := wt.SsrcPT()
		// waiting two payload type
		for i := 0; len(ssrcPT) < 2; ssrcPT = wt.SsrcPT() {
			if i > 20 {
				break
			}
			time.Sleep(5 * time.Millisecond)
			i++
		}
		answer, err = webrtcSub.AnswerSubscribe(jsep, ssrcPT, mid)
		if err != nil {
			log.Warnf("biz subscribe answer err=%v", err.Error())
			reject(codeUnknownErr, err.Error())
			return
		}
		accept(util.Map("jsep", answer))
	case *rtc.RTPTransport:
		// the pub is on other ion, rtp pub already exist
		rt := pub.(*rtc.RTPTransport)
		ssrcPT := rt.SsrcPT()
		for i := 0; len(ssrcPT) < 2; ssrcPT = rt.SsrcPT() {
			if i > 20 {
				break
			}
			time.Sleep(5 * time.Millisecond)
			i++
		}
		answer, err = webrtcSub.AnswerSubscribe(jsep, ssrcPT, mid)
		if err != nil {
			log.Errorf("biz.subscribe answer err=%v", err.Error())
			reject(codeUnknownErr, err.Error())
			return
		}
		accept(util.Map("jsep", answer))

	default:
		respHandler := func(m map[string]interface{}) {
			log.Infof("biz.subscribe respHandler m=%v", m)
			//m=map[info:map[3792725445:96 782016151:111] mid:64024e34-8cd6-427c-9960-04a49df5205f#IWDYJD response:getMediaInfo]
			ssrcPT := make(map[uint32]uint8)
			info := util.Val(m, "info")
			if info != "" {
				for ssrc, pt := range util.Unmarshal(info) {
					ssrcPT[util.StrToUint32(ssrc)] = util.StrToUint8(pt.(string))
				}
			}
			answer, err = webrtcSub.AnswerSubscribe(jsep, ssrcPT, mid)
			if err != nil {
				log.Errorf("biz.subscribe answer err=%v", err.Error())
				reject(codeUnknownErr, err.Error())
				return
			}
			relayRespHandler := func(m map[string]string) {
				log.Infof("biz.subscribe relayRespHandler m=%v", m)
			}
			amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbRelay, "rid", room.ID(), "mid", mid), relayRespHandler)
			accept(util.Map("jsep", answer))
		}
		// the pub is on other ion, rtp pub not exist
		amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbGetMediaInfo, "rid", room.ID(), "mid", mid), respHandler)
	}
}

func unsubscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "mid", reject) {
		return
	}
	mid := util.Val(msg, "mid")
	// if this is on this ion, ion delete the webrtctransport sub
	rtc.DelSub(mid, peer.ID())
	// if no sub, delete pub
	if len(rtc.GetSubs(mid)) == 0 {
		rtc.DelPub(mid)
	}
	// if this is relay from this ion, ion auto delete the rtptransport sub when next ion deleted pub
	accept(emptyMap)
}

func broadcast(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "uid", reject) || invalid(msg, "info", reject) {
		return
	}
	rid, uid, info := util.Val(msg, "rid"), util.Val(msg, "uid"), util.Val(msg, "info")
	amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "info", info), "")
	accept(emptyMap)
}
