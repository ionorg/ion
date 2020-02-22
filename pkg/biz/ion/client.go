package biz

import (
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
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
	islbRPC.Request(proto.IslbClientOnJoin, util.Map("rid", rid, "uid", peer.ID(), "info", info),
		func(result map[string]interface{}) { //accept
			log.Infof("success: =>  %s", result)
			// TODO: foreach infos => mid, info
			info := result["info"]
			mid := result["mid"]
			uid := result["uid"]
			log.Infof("biz.join respHandler mid=%v info=%v", mid, info)
			if mid != "" {
				peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "info", info))
			}
		},
		func(code int, err string) { // reject
			log.Warnf("reject: %d => %s", code, err)
		})
	accept(emptyMap)
}

/*
islbRPC.Request(proto.IslbClientOnJoin, util.Map(),
		func(result map[string]interface{}) { //accept
			log.Infof("success: =>  %s", result)
		},
		func(code int, err string) { // reject
			log.Warnf("reject: %d => %s", code, err)
		})
*/

func leave(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")
	if invalid(msg, "rid", reject) {
		return
	}
	rid := util.Val(msg, "rid")
	/*

		// if this is a webrtc pub
		mids := rtc.GetWebRtcMIDByPID(peer.ID())
		for _, mid := range mids {
			// tell islb stream-remove
			amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", rid, "uid", peer.ID(), "mid", mid), "")

			rtc.DelPipeline(mid)

			// del sub and get the rtp's pub which has none sub
			noSubRtpPubMid := rtc.DelSubFromAllPub(mid)
			// del pub which has none sub when received resp
			log.Infof("biz.leave noSubRtpPubMid=%v", noSubRtpPubMid)
			for mid := range noSubRtpPubMid {
				respUnrelayHandler := func(m map[string]interface{}) {
					log.Infof("biz.leave respUnrelayHandler m=%v", m)
					mid := util.Val(m, "mid")
					rtc.DelPipeline(mid)
				}
				// tell islb stop relay
				amqp.RpcCallWithResp(proto.IslbID, util.Map("method", proto.IslbUnrelay, "rid", rid, "mid", mid), respUnrelayHandler)
			}
		}

		islbRPC.Request(proto.IslbClientOnLeave, util.Map("rid", rid, "uid", peer.ID()),
			func(result map[string]interface{}) { //accept
				log.Infof("success: =>  %s", result)
			},
			func(code int, err string) { // reject
				log.Warnf("reject: %d => %s", code, err)
			})
		accept(emptyMap)
	*/
	signal.DelPeer(rid, peer.ID())
}

func clientClose(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	leave(peer, msg, accept, reject)
}

func publish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.publish peer.ID()=%s", peer.ID())
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}
	rid := util.Val(msg, "rid")

	sfu := lookupSFU(rid)
	if sfu == nil {
		reject(500, "No SFU nodes available.")
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
	/*
		sdp := util.Val(j, "sdp")
		options := msg["options"].(map[string]interface{})
		mid := getMID(peer.ID())

		accept(util.Map("jsep", answer, "mid", mid))
	*/
}

// unpublish from app
func unpublish(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)
	//get rid from msg, because room may be already deleted from signal
	//so we can't get rid from signal's room
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	// if this mid is a webrtc pub
	// tell islb stream-remove, `rtc.DelPub(mid)` will be done when islb broadcast stream-remove
	key := proto.GetPubMediaPath(rid, mid, 0)
	discovery.Del(key, true)

	accept(emptyMap)
}

func subscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.subscribe peer.ID()=%s ", peer.ID())
	if invalid(msg, "jsep", reject) || invalid(msg, "mid", reject) {
		return
	}
	j := msg["jsep"].(map[string]interface{})
	if invalid(j, "sdp", reject) {
		return
	}

	room := signal.GetRoomByPeer(peer.ID())
	sfu := lookupSFU(room.ID())
	if sfu == nil {
		reject(500, "No SFU nodes available.")
		return
	}

	//mid, sdp := util.Val(msg, "mid"), util.Val(j, "sdp")
}

func unsubscribe(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "mid", reject) {
		return
	}
	//mid := util.Val(msg, "mid")

	// if this is relay from this ion, ion auto delete the rtptransport sub when next ion deleted pub
	accept(emptyMap)
}

func broadcast(peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	if invalid(msg, "rid", reject) || invalid(msg, "uid", reject) || invalid(msg, "info", reject) {
		return
	}
	rid, uid, info := util.Val(msg, "rid"), util.Val(msg, "uid"), util.Val(msg, "info")
	islbRPC.Request(proto.IslbOnBroadcast, util.Map("rid", rid, "uid", uid, "info", info),
		func(result map[string]interface{}) { //accept
			log.Infof("success: =>  %s", result)
		},
		func(code int, err string) { // reject
			log.Warnf("reject: %d => %s", code, err)
		})
	accept(emptyMap)
}
