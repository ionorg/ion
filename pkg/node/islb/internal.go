package islb

import (
	"encoding/json"
	"fmt"
	"hash/adler32"
	"math"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

// WatchServiceNodes .
func WatchServiceNodes(service string, state discovery.NodeStateType, node discovery.Node) {
	id := node.ID
	if state == discovery.UP {
		if _, found := services[id]; !found {
			services[id] = node
			service := node.Info["service"]
			name := node.Info["name"]
			log.Debugf("Service [%s] UP %s => %s", service, name, id)
		}
	} else if state == discovery.DOWN {
		if _, found := services[id]; found {
			service := node.Info["service"]
			name := node.Info["name"]
			log.Debugf("Service [%s] DOWN %s => %s", service, name, id)
			delete(services, id)
		}
	}
}

/*Find service nodes by name, such as sfu|mcu|sip-gateway|rtmp-gateway */
func findSfu(data proto.ToIslbFindSfuMsg) (interface{}, *nprotoo.Error) {
	service := "sfu"
	if data.RID != "" && data.UID != "" && data.MID != "" {
		mkey := proto.MediaInfo{
			DC:  dc,
			RID: data.RID,
			UID: data.UID,
			MID: data.MID,
		}.BuildKey()
		log.Infof("Find mids by mkey %s", mkey)
		for _, key := range redis.Keys(mkey + "*") {
			log.Infof("Got: key => %s", key)
			minfo, err := proto.ParseMediaInfo(key)
			if err != nil {
				break
			}
			for _, node := range services {
				name := node.Info["name"]
				id := node.Info["id"]
				if service == node.Info["service"] && minfo.NID == id {
					rpcID := discovery.GetRPCChannel(node)
					eventID := discovery.GetEventChannel(node)
					resp := proto.FromIslbFindSfuMsg{Name: name, RPCID: rpcID, EventID: eventID, Service: service, ID: id}
					log.Infof("findServiceNode: by node ID %s, [%s] %s => %s", minfo.NID, service, name, rpcID)
					return resp, nil
				}
			}
		}
	}

	// TODO: Add a load balancing algorithm.
	for _, node := range services {
		if service == node.Info["service"] {
			rpcID := discovery.GetRPCChannel(node)
			eventID := discovery.GetEventChannel(node)
			name := node.Info["name"]
			id := node.Info["id"]
			resp := proto.FromIslbFindSfuMsg{Name: name, RPCID: rpcID, EventID: eventID, Service: service, ID: id}
			log.Infof("findServiceNode: [%s] %s => %s", service, name, rpcID)
			return resp, nil
		}
	}

	// MID/RID Doesn't exist in Redis
	// Find least packed SFU to return
	sfuID := ""
	minStreamCount := math.MaxInt32
	for _, sfu := range services {
		if service == sfu.Info["service"] {
			// get stream count
			sfuKey := proto.MediaInfo{
				DC:  dc,
				NID: sfu.Info["id"],
			}.BuildKey()
			streamCount := len(redis.Keys(sfuKey))

			log.Infof("findServiceNode looking up sfu stream count [%s] = %v", sfuKey, streamCount)
			if streamCount <= minStreamCount {
				sfuID = sfu.ID
				minStreamCount = streamCount
			}
		}
	}
	log.Infof("findServiceNode: selecting SFU [%s] = %v", sfuID, minStreamCount)

	if node, ok := services[sfuID]; ok {
		log.Infof("findServiceNode: found best candidate SFU [%s]", node)
		rpcID := discovery.GetRPCChannel(node)
		eventID := discovery.GetEventChannel(node)
		name := node.Info["name"]
		id := node.Info["id"]
		resp := proto.GetSFURPCParams{Name: name, RPCID: rpcID, EventID: eventID, Service: service, ID: id}
		log.Infof("findServiceNode: [%s] %s => %s", service, name, rpcID)
		return resp, nil
	}

	return nil, util.NewNpError(404, fmt.Sprintf("Service node [%s] not found", service))
}

func streamAdd(data proto.ToIslbStreamAddMsg) (interface{}, *nprotoo.Error) {
	mkey := proto.MediaInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
		MID: data.MID,
	}.BuildKey()

	field, value, err := proto.MarshalNodeField(proto.NodeInfo{
		Name: nid,
		ID:   nid,
		Type: "origin",
	})
	if err != nil {
		log.Errorf("Set: %v ", err)
	}
	err = redis.HSetTTL(mkey, field, value, redisLongKeyTTL)
	if err != nil {
		log.Errorf("Set: %v ", err)
	}

	field = "track/" + string(data.StreamID)
	// The value here actually doesn't matter, so just store the associated MID in case it's useful in the future.
	log.Infof("SetTrackField: mkey, field, value = %s, %s, %s", mkey, field, data.MID)
	err = redis.HSetTTL(mkey, field, string(data.MID), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	log.Infof("Broadcast: [stream-add] => %v", data)
	broadcaster.Say(proto.IslbStreamAdd, proto.FromIslbStreamAddMsg{
		RID:    data.RID,
		UID:    data.UID,
		Stream: proto.Stream{UID: data.UID, StreamID: data.StreamID},
	})
	return struct{}{}, nil
}

func listMids(data proto.ToIslbListMids) (interface{}, *nprotoo.Error) {
	mkey := proto.MediaInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()

	mids := make([]proto.MID, 0)
	for _, key := range redis.Keys(mkey) {
		mediaInfo, err := proto.ParseMediaInfo(key)
		if err != nil {
			log.Errorf("Failed to parse media info %v", err)
			continue
		}
		mids = append(mids, mediaInfo.MID)
	}
	return proto.FromIslbListMids{MIDs: mids}, nil
}

func peerJoin(msg proto.ToIslbPeerJoinMsg) (interface{}, *nprotoo.Error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: msg.RID,
		UID: msg.UID,
	}.BuildKey()
	log.Infof("clientJoin: set %s => %v", ukey, string(msg.Info))

	// Tell everyone about the new peer.
	broadcaster.Say(proto.IslbPeerJoin, proto.ToClientPeerJoinMsg{
		UID: msg.UID, RID: msg.RID, Info: msg.Info,
	})

	// Tell the new peer about everyone currently in the room.
	searchKey := proto.UserInfo{
		DC:  dc,
		RID: msg.RID,
	}.BuildKey()
	keys := redis.Keys(searchKey)

	peers := make([]proto.Peer, 0)
	streams := make([]proto.Stream, 0)
	for _, key := range keys {
		fields := redis.HGetAll(key)
		parsedUserKey, err := proto.ParseUserInfo(key)
		if err != nil {
			log.Errorf("redis.HGetAll err = %v", err)
			continue
		}
		if info, ok := fields["info"]; ok {
			peers = append(peers, proto.Peer{
				UID:  parsedUserKey.UID,
				Info: json.RawMessage(info),
			})
		} else {
			log.Warnf("No info found for %v", key)
		}

		mkey := proto.MediaInfo{
			DC:  dc,
			RID: msg.RID,
			UID: parsedUserKey.UID,
		}.BuildKey()
		mediaKeys := redis.Keys(mkey)
		for _, mediaKey := range mediaKeys {
			mediaFields := redis.HGetAll(mediaKey)
			for mediaField := range mediaFields {
				log.Warnf("Received media field %s for key %s", mediaField, mediaKey)
				if len(mediaField) > 6 && mediaField[:6] == "track/" {
					streams = append(streams, proto.Stream{
						UID:      parsedUserKey.UID,
						StreamID: proto.StreamID(mediaField[6:]),
					})
				}
			}
		}
	}

	// Write the user info to redis.
	err := redis.HSetTTL(ukey, "info", string(msg.Info), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	// Get the SID for the room.
	mkey := proto.MediaInfo{
		DC:  dc,
		RID: msg.RID,
		UID: msg.UID,
		MID: msg.MID,
	}.BuildKey()
	fields := redis.HGetAll(mkey)
	var sid proto.SID
	val, ok := fields["sid"]
	if !ok {
		// TODO: Generate SID based off some load balancing strategy.
		adler := adler32.New()
		_, err = adler.Write([]byte(msg.RID))
		if err != nil {
			log.Errorf("adler.Write err = %v", err)
		}
		sid = proto.SID(fmt.Sprintf("%d", adler.Sum32()))
		err := redis.HSetTTL(mkey, "sid", sid, redisLongKeyTTL)
		if err != nil {
			log.Errorf("redis.HSetTTL err = %v", err)
		}
	} else {
		sid = proto.SID(val)
	}

	return proto.FromIslbPeerJoinMsg{
		Peers:   peers,
		Streams: streams,
		SID:     sid,
	}, nil
}

func peerLeave(data proto.IslbPeerLeaveMsg) (interface{}, *nprotoo.Error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()
	log.Infof("clientLeave: remove key => %s", ukey)
	err := redis.Del(ukey)
	if err != nil {
		log.Errorf("redis.Del err = %v", err)
	}
	broadcaster.Say(proto.IslbPeerLeave, proto.IslbPeerLeaveMsg(data))
	return struct{}{}, nil
}

// func relay(data map[string]interface{}) (interface{}, *nprotoo.Error) {
// 	rid := util.Val(data, "rid")
// 	mid := util.Val(data, "mid")
// 	from := util.Val(data, "from")

// 	key := proto.GetPubNodePath(rid, mid)
// 	info := redis.HGetAll(key)
// 	for ip := range info {
// 		method := util.Map("method", proto.IslbRelay, "sid", from, "mid", mid)
// 		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
// 		//amqp.RpcCall(ip, method, "")
// 	}
// 	return struct{}{}, nil
// }

// func unRelay(data map[string]interface{}) (interface{}, *nprotoo.Error) {
// 	rid := util.Val(data, "rid")
// 	mid := util.Val(data, "mid")
// 	from := util.Val(data, "from")

// 	key := proto.GetPubNodePath(rid, mid)
// 	info := redis.HGetAll(key)
// 	for ip := range info {
// 		method := util.Map("method", proto.IslbUnrelay, "mid", mid, "sid", from)
// 		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
// 		//amqp.RpcCall(ip, method, "")
// 	}
// 	// time.Sleep(time.Millisecond * 10)
// 	resp := util.Map("mid", mid, "sid", from)
// 	log.Infof("unRelay: resp=%v", resp)
// 	return resp, nil
// }

func broadcast(data proto.IslbBroadcastMsg) (interface{}, *nprotoo.Error) {
	broadcaster.Say(proto.IslbBroadcast, proto.IslbBroadcastMsg(data))
	return struct{}{}, nil
}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		go func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
			method := request.Method
			msg := request.Data
			log.Infof("handleRequest: method => %s, data => %s", method, msg)

			var result interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			case proto.IslbFindSfu:
				var msgData proto.ToIslbFindSfuMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = findSfu(msgData)
				}
			case proto.IslbPeerJoin:
				var msgData proto.ToIslbPeerJoinMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = peerJoin(msgData)
				}
			case proto.IslbPeerLeave:
				var msgData proto.IslbPeerLeaveMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = peerLeave(msgData)
				}
			case proto.IslbStreamAdd:
				var msgData proto.ToIslbStreamAddMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = streamAdd(msgData)
				}
			// case proto.IslbRelay:
			// 	result, err = relay(data)
			// case proto.IslbUnrelay:
			// 	result, err = unRelay(data)
			case proto.IslbBroadcast:
				var msgData proto.IslbBroadcastMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = broadcast(msgData)
				}
			case proto.IslbListMids:
				var msgData proto.ToIslbListMids
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = listMids(msgData)
				}
			}

			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}
