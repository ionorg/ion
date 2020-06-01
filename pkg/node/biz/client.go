package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

var (
	ridError  = util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
	jsepError = util.NewNpError(codeJsepErr, codeStr(codeJsepErr))
	// sdpError  = util.NewNpError(codeSDPErr, codeStr(codeSDPErr))
	midError = util.NewNpError(codeMIDErr, codeStr(codeMIDErr))
)

func login(peer *signal.Peer, msg proto.LoginMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.login peer.ID()=%s msg=%v", peer.ID(), msg)
	//TODO auth check, maybe jwt
	return emptyMap, nil
}

// join room
func join(peer *signal.Peer, msg proto.JoinMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := msg.RID

	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

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
	info := msg.Info
	uid := peer.ID()
	islb.SyncRequest(proto.IslbClientOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
	// Send getPubs => islb
	islb.AsyncRequest(proto.IslbGetPubs, msg.RoomInfo).Then(
		func(result nprotoo.RawMessage) {
			var resMsg proto.GetPubResp
			if err := result.Unmarshal(&resMsg); err != nil {
				log.Errorf("Unmarshal pub response %v", err)
				return
			}
			log.Infof("IslbGetPubs: result=%v", result)
			for _, pub := range resMsg.Pubs {
				if pub.MID == "" {
					continue
				}
				notif := proto.StreamAddMsg{
					MediaInfo: pub.MediaInfo,
					Info:      pub.Info,
					Tracks:    pub.Tracks,
				}
				peer.Notify(proto.ClientOnStreamAdd, notif)
			}
		},
		func(err *nprotoo.Error) {})

	return emptyMap, nil
}

func leave(peer *signal.Peer, msg proto.LeaveMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")

	rid := msg.RID

	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

	uid := peer.ID()

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	islb.SyncRequest(proto.IslbClientOnLeave, util.Map("rid", rid, "uid", uid))
	signal.DelPeer(rid, peer.ID())
	return emptyMap, nil
}

func clientClose(peer *signal.Peer, msg proto.CloseMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	return leave(peer, msg.LeaveMsg)
}

func publish(peer *signal.Peer, msg proto.PublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.publish peer.ID()=%s", peer.ID())

	nid, sfu, err := getRPCForSFU("")
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	jsep := msg.Jsep
	options := msg.Options
	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		return nil, util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
	}

	rid := room.ID()
	uid := peer.ID()
	resMsg, err := sfu.SyncRequest(proto.ClientPublish, util.Map("uid", uid, "rid", rid, "jsep", jsep, "options", options))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	var result map[string]interface{}
	if err := resMsg.Unmarshal(&result); err != nil {
		log.Errorf("Unmarshal pub response %v", err)
		return nil, err
	}

	log.Infof("publish: result => %v", result)
	mid := util.Val(result, "mid")
	tracks := result["tracks"]
	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	islb.AsyncRequest(proto.IslbOnStreamAdd, util.Map("rid", rid, "nid", nid, "uid", uid, "mid", mid, "tracks", tracks))
	return result, nil
}

// unpublish from app
func unpublish(peer *signal.Peer, msg proto.UnpublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)

	mid := msg.MID
	rid := msg.RID
	uid := peer.ID()

	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, err
	}

	_, err = sfu.SyncRequest(proto.ClientUnPublish, util.Map("mid", mid, "uid", uid, "rid", rid))
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

func subscribe(peer *signal.Peer, msg proto.SubscribeMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.subscribe peer.ID()=%s ", peer.ID())
	mid := msg.MID

	// Validate
	if mid == "" {
		return nil, midError
	} else if msg.Jsep.SDP == "" {
		return nil, jsepError
	}

	nodeID, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	// TODO:
	if nodeID != "node for mid" {
		log.Warnf("Not the same node, need to enable sfu-sfu relay!")
	}

	room := signal.GetRoomByPeer(peer.ID())
	uid := peer.ID()
	rid := room.ID()

	jsep := msg.Jsep

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	result, err := islb.SyncRequest(proto.IslbGetMediaInfo, proto.MediaInfo{RID: rid, MID: mid})
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	var some map[string]interface{}
	if err := result.Unmarshal(&some); err != nil {
		return nil, err
	}
	// subMsg := proto.SFUSubscribeMsg{
	// 	MediaInfo: proto.MediaInfo{
	// 		UID: uid, RID: rid, MID: mid,
	// 	},
	// }
	result, err = sfu.SyncRequest(proto.ClientSubscribe, util.Map("uid", uid, "rid", rid, "mid", mid, "tracks", some["tracks"], "jsep", jsep))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("subscribe: result => %v", result)
	return result, nil
}

func unsubscribe(peer *signal.Peer, msg proto.UnsubscribeMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)
	mid := msg.MID

	// Validate
	if mid == "" {
		return nil, midError
	}

	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	result, err := sfu.SyncRequest(proto.ClientUnSubscribe, util.Map("mid", mid))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("publish: result => %v", result)
	return result, nil
}

func broadcast(peer *signal.Peer, msg proto.BroadcastMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.broadcast peer.ID()=%s msg=%v", peer.ID(), msg)

	// Validate
	if msg.RID == "" || msg.UID == "" {
		return nil, ridError
	}

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	rid, uid, info := msg.RID, msg.UID, msg.Info
	islb.AsyncRequest(proto.IslbOnBroadcast, util.Map("rid", rid, "uid", uid, "info", info))
	return emptyMap, nil
}

func trickle(peer *signal.Peer, msg proto.TrickleMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.trickle peer.ID()=%s msg=%v", peer.ID(), msg)
	mid := msg.MID

	// Validate
	if msg.RID == "" || msg.UID == "" {
		return nil, ridError
	}

	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	trickle := msg.Trickle

	sfu.AsyncRequest(proto.ClientTrickleICE, util.Map("mid", mid, "trickle", trickle))
	return emptyMap, nil
}
