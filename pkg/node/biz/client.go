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

// join room
func join(peer *signal.Peer, msg proto.FromClientJoinMsg) (interface{}, *nprotoo.Error) {
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

	_, sfu, err := getRPCForSFU(rid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	info := msg.Info
	uid := peer.ID()
	// Send join => islb
	_, err = islb.SyncRequest(proto.IslbClientOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
	if err != nil {
		log.Errorf("IslbClientOnJoin failed %v", err.Error())
	}
	// Send join => sfu
	resp, err := sfu.SyncRequest(proto.SfuClientJoin, proto.ToSfuJoinMsg{
		RoomInfo: proto.RoomInfo{RID: rid, UID: proto.UID(uid)},
		RTCInfo:  msg.RTCInfo,
	})
	if err != nil {
		log.Errorf("SfuClientOnJoin failed %v", err.Error())
	}

	return resp, nil
}

func leave(msg proto.FromSignalLeaveMsg) (interface{}, *nprotoo.Error) {
	signal.DelPeer(msg.RID, string(msg.UID))

	islb, found := getRPCForIslb()
	if !found {
		log.Errorf("islb node not found")
	}
	if _, err := islb.SyncRequest(proto.IslbClientOnLeave, util.Map("rid", msg.RID, "uid", msg.UID)); err != nil {
		log.Errorf("IslbClientOnLeave failed %v", err.Error())
	}

	_, sfu, err := getRPCForSFU(mid, msg.RID)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	_, err = sfu.SyncRequest(proto.SfuClientLeave, proto.ToSfuLeaveMsg{
		RoomInfo: msg.RoomInfo,
	})
	if err != nil {
		log.Errorf("SfuClientLeave failed %v", err.Error())
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	return nil, nil
}

func offer(peer *signal.Peer, msg proto.FromClientOfferMsg) (interface{}, *nprotoo.Error) {
	_, sfu, err := getRPCForSFU(msg.RID)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	_, err = sfu.SyncRequest(proto.SfuClientOffer, util.Map("rid", msg.RID, "uid", peer.ID(), "jsep", msg.Jsep))
	if err != nil {
		log.Errorf("SfuClientOnOffer failed %v", err.Error())
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	return nil, nil
}

func broadcast(peer *signal.Peer, msg proto.FromClientBroadcastMsg) (interface{}, *nprotoo.Error) {
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

func trickle(peer *signal.Peer, msg proto.FromClientTrickleMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.trickle peer.ID()=%s msg=%v", peer.ID(), msg)
	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

	_, sfu, err := getRPCForSFU(mid, msg.RID)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	sfu.AsyncRequest(proto.ClientTrickleICE, proto.FromSfuTrickleMsg{
		RoomInfo: proto.RoomInfo{
			RID: msg.RID,
			UID: proto.UID(peer.ID()),
		},
		Candidate: msg.Candidate,
	})
	return emptyMap, nil
}
