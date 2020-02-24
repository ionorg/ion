package biz

import (
	"fmt"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg map[string]interface{}, accept signal.AcceptFunc, reject signal.RejectFunc) {
	log.Infof("method => %s, data => %v", method, msg)
	var result map[string]interface{}
	err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

	switch method {
	case proto.ClientClose:
		result, err = clientClose(peer, msg)
	case proto.ClientLogin:
		result, err = login(peer, msg)
	case proto.ClientJoin:
		result, err = join(peer, msg)
	case proto.ClientLeave:
		result, err = leave(peer, msg)
	case proto.ClientPublish:
		result, err = publish(peer, msg)
	case proto.ClientUnPublish:
		result, err = unpublish(peer, msg)
	case proto.ClientSubscribe:
		result, err = subscribe(peer, msg)
	case proto.ClientUnSubscribe:
		result, err = unsubscribe(peer, msg)
	case proto.ClientBroadcast:
		result, err = broadcast(peer, msg)
	}

	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}

func getRPCForIslb() (*nprotoo.Requestor, bool) {
	for _, item := range services {
		if item.Info["service"] == "islb" {
			id := item.Info["id"]
			rpc, found := rpcs[id]
			if !found {
				rpcID := node.GetRPCChannel(item)
				log.Infof("Create rpc [%s] for islb", rpcID)
				rpc = protoo.NewRequestor(rpcID)
				rpcs[id] = rpc
			}
			return rpc, true
		}
	}
	log.Warnf("No islb node was found.")
	return nil, false
}

func getRPCForSFU() (*nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu"))
	if err != nil {
		return nil, err
	}
	log.Infof("SFU result => %v", result)
	rpcID := result["rpc-id"].(string)
	sfu := protoo.NewRequestor(rpcID)
	return sfu, nil
}

func login(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.login peer.ID()=%s msg=%v", peer.ID(), msg)
	//TODO auth check, maybe jwt
	return emptyMap, nil
}

// join room
func join(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	if ok, err := verifyData(msg, "rid"); !ok {
		return nil, err
	}
	rid := util.Val(msg, "rid")
	//already joined this room
	if signal.HasPeer(rid, peer) {
		return emptyMap, nil
	}
	signal.AddPeer(rid, peer)

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	// Send join => islb
	info := util.Val(msg, "info")
	uid := peer.ID()
	islb.AsyncRequest(proto.IslbClientOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
	// Send getPubs => islb
	islb.AsyncRequest(proto.IslbGetPubs, util.Map("rid", rid, "uid", uid)).Then(
		func(result map[string]interface{}) {
			log.Infof("IslbGetPubs: result=%v", result)
			if result["pubs"] == nil {
				return
			}
			uid := result["uid"].(string)
			rid := result["rid"].(string)
			pubs := result["pubs"].([]interface{})
			for _, pub := range pubs {
				info := pub.(map[string]interface{})["info"].(interface{})
				mid := pub.(map[string]interface{})["mid"].(string)
				log.Infof("IslbGetPubs: mid=%v info=%v", mid, info)
				// peer <=  range pubs(mid)
				if mid != "" {
					peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "info", info))
				}
			}
		},
		func(err *nprotoo.Error) {

		})

	return emptyMap, nil
}

func leave(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")

	if ok, err := verifyData(msg, "rid"); !ok {
		return nil, err
	}

	rid := util.Val(msg, "rid")
	uid := peer.ID()

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	islb.AsyncRequest(proto.IslbClientOnLeave, util.Map("rid", rid, "uid", uid))
	signal.DelPeer(rid, peer.ID())
	return emptyMap, nil
}

func clientClose(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	return leave(peer, msg)
}

func publish(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.publish peer.ID()=%s", peer.ID())

	sfu, err := getRPCForSFU()
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	jsep := msg["jsep"].(map[string]interface{})
	options := msg["options"].(map[string]interface{})
	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		return nil, util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
	}

	uid := peer.ID()
	result, err := sfu.SyncRequest(proto.ClientPublish, util.Map("uid", uid, "jsep", jsep, "options", options))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("publish: result => %v", result)

	mid := util.Val(result, "mid")
	rid := room.ID()
	ssrcpts := result["ssrcpts"].([]interface{})
	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	ssrcPts := "{"
	var arr []string
	for _, item := range ssrcpts {
		ssrc := item.(map[string]interface{})["ssrc"].(string)
		pt := item.(map[string]interface{})["pt"].(string)
		arr = append(arr, fmt.Sprintf("\"%s\":\"%s\"", ssrc, pt))
	}
	ssrcPts += strings.Join(arr, ",")
	ssrcPts += "}"
	islb.AsyncRequest(proto.IslbOnStreamAdd, util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "mediaInfo", ssrcPts))
	return result, nil
}

// unpublish from app
func unpublish(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)

	sfu, err := getRPCForSFU()
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, err
	}

	mid := util.Val(msg, "mid")
	rid := util.Val(msg, "rid")
	uid := peer.ID()

	_, err = sfu.SyncRequest(proto.ClientUnPublish, util.Map("mid", mid))
	if err != nil {
		return nil, err
	}

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	// if this mid is a webrtc pub
	// tell islb stream-remove, `rtc.DelPub(mid)` will be done when islb broadcast stream-remove
	islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
	return emptyMap, nil
}

func subscribe(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.subscribe peer.ID()=%s ", peer.ID())

	sfu, err := getRPCForSFU()
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	if ok, err := verifyData(msg, "jsep", "mid"); !ok {
		return nil, err
	}

	room := signal.GetRoomByPeer(peer.ID())
	uid := peer.ID()
	rid := room.ID()
	mid := util.Val(msg, "mid")
	jsep := msg["jsep"].(map[string]interface{})

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	result, err := islb.SyncRequest(proto.IslbGetMediaInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	info := result["info"].(string)
	result, err = sfu.SyncRequest(proto.ClientSubscribe, util.Map("uid", uid, "mid", mid, "info", info, "jsep", jsep))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("subscribe: result => %v", result)
	return result, nil
}

func unsubscribe(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)

	sfu, err := getRPCForSFU()
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	if ok, err := verifyData(msg, "mid"); !ok {
		return nil, err
	}
	mid := util.Val(msg, "mid")

	result, err := sfu.SyncRequest(proto.ClientUnSubscribe, util.Map("mid", mid))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("publish: result => %v", result)
	return result, nil
}

func broadcast(peer *signal.Peer, msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)

	if ok, err := verifyData(msg, "rid", "uid", "info"); !ok {
		return nil, err
	}

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	rid, uid, info := util.Val(msg, "rid"), util.Val(msg, "uid"), util.Val(msg, "info")
	islb.AsyncRequest(proto.IslbOnBroadcast, util.Map("rid", rid, "uid", uid, "info", info))
	return emptyMap, nil
}
