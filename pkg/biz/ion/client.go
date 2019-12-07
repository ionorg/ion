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
	webrtc "github.com/pion/webrtc/v2"
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
	case proto.ClientOnStreamAdd:
		streamAdd(peer, msg, accept, reject)
	}
}

func login(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.login peer.ID()=%s msg=%v", peer.ID(), msg)
	//TODO auth check, maybe jwt
	accept(util.Unmarshal(`{}`))
}

// join room
func join(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := util.Val(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	//aleady joined this room
	if signal.HasPeer(rid, peer) {
		accept(util.Unmarshal(`{}`))
		return
	}

	info := util.Val(msg, "info")

	// add peer to signal room
	signal.AddPeer(rid, peer)

	amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbClientOnJoin, "rid", rid, "id", peer.ID(), "info", info), "")

	respHandler := func(m map[string]interface{}) {
		info := m["info"]
		mid := m["mid"]
		pid := m["pid"]
		log.Infof("biz.join respHandler mid=%v info=%v", mid, info)
		if mid != "" {
			peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "pid", pid, "mid", mid, "info", info))
		}
	}
	// find pubs from islb ,skip this ion
	amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbGetPubs, "rid", rid, "pid", peer.ID()), respHandler)
	accept(util.Unmarshal(`{}`))
}

func leave(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := util.Val(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	// if this is a webrtc pub
	mids := rtc.GetWebRtcMIDByPID(peer.ID())
	for _, mid := range mids {
		// tell islb stream-remove
		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "pid", peer.ID(), "mid", mid), "")

		rtc.DelPub(mid)
		quitLock.Lock()
		for k := range quit {
			if strings.Contains(k, mid) {
				close(quit[k])
				delete(quit, k)
			}
		}
		quitLock.Unlock()

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

		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbClientOnLeave, "rid", rid, "id", peer.ID()), "")

	}

	accept(util.Unmarshal(`{}`))
	signal.DelPeer(rid, peer.ID())
}

func clientClose(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	leave(peer, msg, accept, reject)
	accept(util.Unmarshal(`{}`))
}

func publish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.publish peer.ID()=%s msg=%v", peer.ID(), msg)
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

	sdp := util.Val(j, "sdp")
	options := msg["options"].(map[string]interface{})
	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		reject(-1, errInvalidRoom)
		return
	}

	mid := fmt.Sprintf("%s#%s", peer.ID(), util.RandStr(6))
	jsep := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	islbStoreSsrc := func(ssrc uint32, pt uint8) {
		ssrcPt := fmt.Sprintf("{\"%d\":%d}", ssrc, pt)
		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamAdd, "rid", room.ID(), "pid", peer.ID(), "mid", mid, "mediaInfo", ssrcPt), "")
		keepAlive := fmt.Sprintf("%s#%d", mid, ssrc)
		quitLock.Lock()
		quit[keepAlive] = make(chan struct{})
		quitLock.Unlock()
		go func() {
			t := time.NewTicker(time.Second)
			for {
				select {
				case <-t.C:
					amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbKeepAlive, "rid", room.ID(), "pid", peer.ID(), "mid", mid, "mediaInfo", ssrcPt), "")
				case <-quit[keepAlive]:
					return
				}
			}
		}()
	}

	answer, err := rtc.AddNewWebRTCPub(mid).AnswerPublish(room.ID(), jsep, options, islbStoreSsrc)
	if err != nil {
		log.Errorf("biz.publish answer err=%s jsep=%v", err.Error(), jsep)
		reject(-1, err.Error())
		return
	}

	accept(util.Map("jsep", answer, "mid", mid))
}

// unpublish from app
func unpublish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)
	//broadcast onUnpublish
	// room := signal.GetRoomByPeer(peer.ID())
	// if room == nil {
	// reject(-1, errInvalidRoom)
	// return
	// }
	//get rid from msg, because room may be already deleted from signal
	//so we can't get rid from signal's room
	rid := util.Val(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}
	mid := util.Val(msg, "mid")
	if mid == "" {
		log.Errorf(errInvalidMID)
		reject(-1, errInvalidMID)
		return
	}
	// if this mid is a webrtc pub
	if rtc.IsWebRtcPub(mid) {
		// tell islb stream-remove
		amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "mid", mid), "")

		rtc.DelPub(mid)
		quitLock.Lock()
		for k := range quit {
			if strings.Contains(k, mid) {
				close(quit[k])
				delete(quit, k)
			}
		}
		quitLock.Unlock()
	}

	accept(util.Unmarshal(`{}`))
}

func subscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.subscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	j := msg["jsep"].(map[string]interface{})
	sdp := util.Val(j, "sdp")
	if sdp == "" {
		log.Errorf(errInvalidSDP)
		reject(-1, errInvalidSDP)
		return
	}

	mid := util.Val(msg, "mid")
	if mid == "" {
		log.Errorf(errInvalidMID)
		reject(-1, errInvalidMID)
		return
	}

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
			reject(-1, err.Error())
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
			reject(-1, err.Error())
			return
		}
		accept(util.Map("jsep", answer))

	default:
		respHandler := func(m map[string]interface{}) {
			log.Infof("biz.subscribe respHandler m=%v", m)
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
				reject(-1, err.Error())
				return
			}
			relayRespHandler := func(m map[string]string) {
				log.Infof("biz.subscribe relayRespHandler m=%v", m)
			}
			amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbRelay, "rid", room.ID(), "pid", pub.ID(), "mid", mid), relayRespHandler)
			accept(util.Map("jsep", answer))
		}
		// the pub is on other ion, rtp pub not exist
		amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbGetMediaInfo, "rid", room.ID(), "mid", mid), respHandler)
	}
}

func unsubscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)

	rid := util.Val(msg, "rid")
	if rid == "" {
		log.Errorf(errInvalidRoom)
		reject(-1, errInvalidRoom)
		return
	}

	mid := util.Val(msg, "mid")
	if mid == "" {
		log.Errorf(errInvalidMID)
		reject(-1, errInvalidMID)
		return
	}

	// if this is on this ion, ion delete the webrtctransport sub
	rtc.DelSub(mid, peer.ID())
	// if no sub, delete pub
	if len(rtc.GetSubs(mid)) == 0 {
		rtc.DelPub(mid)
	}
	// if this is relay from this ion, ion auto delete the rtptransport sub when next ion deleted pub
	accept(util.Unmarshal(`{}`))
}

// streamAdd from other ion
func streamAdd(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.streamAdd peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := util.Val(msg, "rid")
	if rid == "" {
		reject(-1, errInvalidRoom)
		return
	}

	mid := util.Val(msg, "mid")
	if mid == "" {
		reject(-1, errInvalidMID)
		return
	}

	// tell other subs onPublish
	signal.NotifyAllWithoutPeer(rid, peer, proto.ClientOnStreamAdd, msg)

	// upload the person number
	accept(util.Unmarshal(`{}`))
}
