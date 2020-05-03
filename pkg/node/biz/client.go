package biz

import (
	"encoding/json"
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v2"
)

// Entry is the biz entry
func Entry(method string, peer *signal.Peer, msg json.RawMessage, accept signal.RespondFunc, reject signal.RejectFunc) {
	log.Infof("method => %s, data => %v", method, msg)
	var result map[string]interface{}
	topErr := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

	parseErr := util.NewNpError(400, fmt.Sprintf("Error parsing request object"))
	//TODO DRY this up
	switch method {
	case proto.ClientClose:
		var msgData CloseMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = clientClose(peer, msgData)
	case proto.ClientLogin:
		var msgData LoginMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = login(peer, msgData)
	case proto.ClientJoin:
		var msgData JoinMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = join(peer, msgData)
	case proto.ClientLeave:
		var msgData LeaveMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = leave(peer, msgData)
	case proto.ClientPublish:
		var msgData PublishMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = publish(peer, msgData)
	case proto.ClientUnPublish:
		var msgData UnpublishMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = unpublish(peer, msgData)
	case proto.ClientSubscribe:
		var msgData SubscribeMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = subscribe(peer, msgData)
	case proto.ClientUnSubscribe:
		var msgData UnsubscribeMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = unsubscribe(peer, msgData)
	case proto.ClientBroadcast:
		var msgData BroadcastMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = broadcast(peer, msgData)
	case proto.ClientTrickleICE:
		var msgData TrickleMsg
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Infof("Marshal error")
			topErr = parseErr
			break
		}
		result, topErr = trickle(peer, msgData)
	}

	if topErr != nil {
		reject(topErr.Code, topErr.Reason)
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
				rpcID := discovery.GetRPCChannel(item)
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

func handleSFUBroadCast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		log.Infof("handleSFUBroadCast: method=%s, data=%v", method, data)
		rid := util.Val(data, "rid")
		uid := util.Val(data, "uid")
		switch method {
		case proto.SFUTrickleICE:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, data)
		case proto.SFUStreamRemove:
			mid := util.Val(data, "mid")
			islb, found := getRPCForIslb()
			if found {
				islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("mid", mid))
			}
		}
	}(msg)
}

func getRPCForSFU(mid string) (string, *nprotoo.Requestor, *nprotoo.Error) {
	islb, found := getRPCForIslb()
	if !found {
		return "", nil, util.NewNpError(500, "Not found any node for islb.")
	}
	result, err := islb.SyncRequest(proto.IslbFindService, util.Map("service", "sfu", "mid", mid))
	if err != nil {
		return "", nil, err
	}

	log.Infof("SFU result => %v", result)
	rpcID := result["rpc-id"].(string)
	eventID := result["event-id"].(string)
	nodeID := result["id"].(string)
	rpc, found := rpcs[rpcID]
	if !found {
		rpc = protoo.NewRequestor(rpcID)
		protoo.OnBroadcast(eventID, handleSFUBroadCast)
		rpcs[rpcID] = rpc
	}
	return nodeID, rpc, nil
}

type LoginMsg struct {
}

func login(peer *signal.Peer, msg LoginMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.login peer.ID()=%s msg=%v", peer.ID(), msg)
	//TODO auth check, maybe jwt
	return emptyMap, nil
}

type UserInfo struct {
	Name string `json:"name"`
}

type JoinMsg struct {
	RoomInfo
	Info UserInfo `json:"info"`
}

// join room
func join(peer *signal.Peer, msg JoinMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := msg.Rid
	// TODO Verify room ID
	// if ok, err := verifyData(msg, "rid"); !ok {
	// 	return nil, err
	// }

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
	islb.AsyncRequest(proto.IslbGetPubs, util.Map("rid", rid, "uid", uid)).Then(
		func(result map[string]interface{}) {
			log.Infof("IslbGetPubs: result=%v", result)
			if result["pubs"] == nil {
				return
			}
			pubs := result["pubs"].([]interface{})
			for _, pub := range pubs {
				info := pub.(map[string]interface{})["info"].(string)
				mid := pub.(map[string]interface{})["mid"].(string)
				uid := pub.(map[string]interface{})["uid"].(string)
				rid := result["rid"].(string)
				tracks := pub.(map[string]interface{})["tracks"].(map[string]interface{})

				var infoObj map[string]interface{}
				err := json.Unmarshal([]byte(info), &infoObj)
				if err != nil {
					log.Errorf("json.Unmarshal: err=%v", err)
				}
				log.Infof("IslbGetPubs: mid=%v info=%v", mid, infoObj)
				// peer <=  range pubs(mid)
				if mid != "" {
					peer.Notify(proto.ClientOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "info", infoObj, "tracks", tracks))
				}
			}
		},
		func(err *nprotoo.Error) {

		})

	return emptyMap, nil
}

type LeaveMsg struct {
	RoomInfo
	Info UserInfo `json:"info"`
}

func leave(peer *signal.Peer, msg LeaveMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")

	rid := msg.Rid
	// TODO Verify room ID
	// if ok, err := verifyData(msg, "rid"); !ok {
	// 	return nil, err
	// }

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

type CloseMsg struct {
	LeaveMsg
}

func clientClose(peer *signal.Peer, msg CloseMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.close peer.ID()=%s msg=%v", peer.ID(), msg)
	return leave(peer, msg.LeaveMsg)
}

type RoomInfo struct {
	Rid string `json:"rid"`
	Uid string `json:"uid"`
}

type PublishOptions struct {
	Codec      string `json:"codec"`
	Resolution string `json:"resolution"`
	Bandwidth  int    `json:"bandwidth"`
	Audio      bool   `json:"audio"`
	Video      bool   `json:"video"`
	Screen     bool   `json:"screen"`
}

type PublishMsg struct {
	RoomInfo
	Jsep    webrtc.SessionDescription `json:"jsep"`
	Options PublishOptions            `json:"options"`
}

func publish(peer *signal.Peer, msg PublishMsg) (map[string]interface{}, *nprotoo.Error) {
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
	result, err := sfu.SyncRequest(proto.ClientPublish, util.Map("uid", uid, "rid", rid, "jsep", jsep, "options", options))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
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

type UnpublishMsg struct {
	RoomInfo
	Options PublishOptions `json:"options"`
	Mid     string         `json:"mid"`
}

// unpublish from app
func unpublish(peer *signal.Peer, msg UnpublishMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("signal.unpublish peer.ID()=%s msg=%v", peer.ID(), msg)

	mid := msg.Mid
	rid := msg.Rid
	uid := peer.ID()

	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, err
	}

	_, err = sfu.SyncRequest(proto.ClientUnPublish, util.Map("mid", mid, "rid", rid))
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

type SubscribeMsg struct {
	Mid  string                    `json:"mid"`
	Jsep webrtc.SessionDescription `json:"jsep"`
}

func subscribe(peer *signal.Peer, msg SubscribeMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.subscribe peer.ID()=%s ", peer.ID())

	// TODO veriy jsep, mid
	// if ok, err := verifyData(msg, "jsep", "mid"); !ok {
	// 	return nil, err
	// }

	mid := msg.Mid
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

	result, err := islb.SyncRequest(proto.IslbGetMediaInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}
	result, err = sfu.SyncRequest(proto.ClientSubscribe, util.Map("uid", uid, "rid", rid, "mid", mid, "tracks", result["tracks"], "jsep", jsep))
	if err != nil {
		log.Warnf("reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	log.Infof("subscribe: result => %v", result)
	return result, nil
}

type UnsubscribeMsg struct {
	Mid string `json:"mid"`
}

func unsubscribe(peer *signal.Peer, msg UnsubscribeMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.unsubscribe peer.ID()=%s msg=%v", peer.ID(), msg)

	//TODO verify mid
	// if ok, err := verifyData(msg, "mid"); !ok {
	// 	return nil, err
	// }

	mid := msg.Mid

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

type BroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

func broadcast(peer *signal.Peer, msg BroadcastMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.broadcast peer.ID()=%s msg=%v", peer.ID(), msg)

	//TODO verify rid, uid, info
	// if ok, err := verifyData(msg, "rid", "uid", "info"); !ok {
	// 	return nil, err
	// }

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	rid, uid, info := msg.RoomInfo.Rid, msg.RoomInfo.Uid, msg.Info
	islb.AsyncRequest(proto.IslbOnBroadcast, util.Map("rid", rid, "uid", uid, "info", info))
	return emptyMap, nil
}

type TrickleMsg struct {
	RoomInfo
	Mid     string          `json:"mid"`
	Info    json.RawMessage `json:"info"`
	Trickle json.RawMessage `json:"trickle"`
}

func trickle(peer *signal.Peer, msg TrickleMsg) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("biz.trickle peer.ID()=%s msg=%v", peer.ID(), msg)

	mid := msg.Mid

	//TODO verify rid, uid, info
	// if ok, err := verifyData(msg, "rid", "uid", "info"); !ok {
	// 	return nil, err
	// }

	_, sfu, err := getRPCForSFU(mid)
	if err != nil {
		log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
		return nil, util.NewNpError(err.Code, err.Reason)
	}

	trickle := msg.Trickle

	sfu.AsyncRequest(proto.ClientTrickleICE, util.Map("mid", mid, "trickle", trickle))
	return emptyMap, nil
}
