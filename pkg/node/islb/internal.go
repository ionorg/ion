package islb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
			if service == "sfu" {
				removeStreamsByNode(node.Info["id"])
			}
			delete(services, id)
		}
	}
}

func removeStreamsByNode(nodeID string) {
	log.Infof("removeStreamsByNode: node => %s", nodeID)
	mkey := proto.MediaInfo{
		DC:  dc,
		NID: nodeID,
	}.BuildKey()
	for _, key := range redis.Keys(mkey + "*") {
		log.Infof("streamRemove: key => %s", key)
		minfo, err := proto.ParseMediaInfo(key)
		if err == nil {
			log.Warnf("TODO: Internal Server Error (500) to %s", minfo.UID)
		}
		err = redis.Del(key)
		if err != nil {
			log.Errorf("redis.Del err = %v", err)
		}
	}
}

// WatchAllStreams .
func WatchAllStreams() {
	mkey := proto.MediaInfo{
		DC: dc,
	}.BuildKey()
	log.Infof("Watch all streams, mkey = %s", mkey)
	for _, key := range redis.Keys(mkey) {
		log.Infof("Watch stream, key = %s", key)
		watchStream(key)
	}
}

func watchStream(key string) {
	log.Infof("Start watch stream: key = %s", key)
	go func() {
		for cmd := range redis.Watch(context.TODO(), key) {
			log.Infof("watchStream: key %s cmd %s modified", key, cmd)
			if cmd == "del" || cmd == "expired" {
				// dc1/room1/media/pub/74baff6e-b8c9-4868-9055-b35d50b73ed6#LUMGUQ
				info, err := proto.ParseMediaInfo(key)
				if err == nil {
					msg := util.Map("dc", info.DC, "rid", info.RID, "uid", info.UID, "mid", info.MID)
					log.Infof("watchStream.BroadCast: onStreamRemove=%v", msg)
					broadcaster.Say(proto.IslbOnStreamRemove, msg)
				}
				//Stop watch loop after key removed.
				break
			}
		}
		log.Infof("Stop watch stream: key = %s", key)
	}()
}

/*Find service nodes by name, such as sfu|mcu|sip-gateway|rtmp-gateway */
func findServiceNode(data proto.FindServiceParams) (interface{}, *nprotoo.Error) {
	service := data.Service
	mid := data.MID
	if mid != "" {
		mkey := proto.MediaInfo{
			DC:  dc,
			MID: mid,
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
					resp := proto.GetSFURPCParams{Name: name, RPCID: rpcID, EventID: eventID, Service: service, ID: id}
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
			resp := proto.GetSFURPCParams{Name: name, RPCID: rpcID, EventID: eventID, Service: service, ID: id}
			log.Infof("findServiceNode: [%s] %s => %s", service, name, rpcID)
			return resp, nil
		}
	}

	return nil, util.NewNpError(404, fmt.Sprintf("Service node [%s] not found", service))
}

func streamAdd(data proto.StreamAddMsg) (interface{}, *nprotoo.Error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()

	mInfo := data.MediaInfo
	mInfo.DC = dc
	mkey := mInfo.BuildKey()

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

	for msid, track := range data.Tracks {
		var infos []proto.TrackInfo
		infos = append(infos, track...)

		field, value, err := proto.MarshalTrackField(msid, infos)
		if err != nil {
			log.Errorf("MarshalTrackField: %v ", err)
			continue
		}
		log.Infof("SetTrackField: mkey, field, value = %s, %s, %s", mkey, field, value)
		err = redis.HSetTTL(mkey, field, value, redisLongKeyTTL)
		if err != nil {
			log.Errorf("redis.HSetTTL err = %v", err)
		}
	}

	// dc1/room1/user/info/${uid} info {"name": "Guest"}
	fields := redis.HGetAll(ukey)

	var extraInfo proto.ClientUserInfo = proto.ClientUserInfo{}
	if infoStr, ok := fields["info"]; ok {
		if err := json.Unmarshal([]byte(infoStr), &extraInfo); err != nil {
			log.Errorf("Unmarshal pub extra info %v", err)
			extraInfo = data.Info
		}
		data.Info = extraInfo
	}

	log.Infof("Broadcast: [stream-add] => %v", data)
	broadcaster.Say(proto.IslbOnStreamAdd, data)

	watchStream(mkey)
	return struct{}{}, nil
}

func streamRemove(data proto.StreamRemoveMsg) (map[string]interface{}, *nprotoo.Error) {
	data.DC = dc
	mkey := data.BuildKey()

	log.Infof("streamRemove: key => %s", mkey)
	for _, key := range redis.Keys(mkey + "*") {
		log.Infof("streamRemove: key => %s", key)
		err := redis.Del(key)
		if err != nil {
			log.Errorf("redis.Del err = %v", err)
		}
	}
	return util.Map(), nil
}

func getPubs(data proto.RoomInfo) (proto.GetPubResp, *nprotoo.Error) {
	rid := data.RID //util.Val(data, "rid")

	key := proto.MediaInfo{
		DC:  dc,
		RID: rid,
	}.BuildKey()
	log.Infof("getPubs: root key=%s", key)

	var pubs []proto.PubInfo
	for _, path := range redis.Keys(key + "*") {
		log.Infof("getPubs media info path = %s", path)
		info, err := proto.ParseMediaInfo(path)
		if err != nil {
			log.Errorf("%v", err)
		}
		fields := redis.HGetAll(proto.UserInfo{
			DC:  info.DC,
			RID: info.RID,
			UID: info.UID,
		}.BuildKey())
		trackFields := redis.HGetAll(path)

		tracks := make(map[string][]proto.TrackInfo)
		for key, value := range trackFields {
			if strings.HasPrefix(key, "track/") {
				msid, infos, err := proto.UnmarshalTrackField(key, value)
				if err != nil {
					log.Errorf("%v", err)
				}
				log.Debugf("msid => %s, tracks => %v\n", msid, infos)
				tracks[msid] = *infos
			}
		}

		log.Infof("Fields %v", fields)

		var extraInfo proto.ClientUserInfo = proto.ClientUserInfo{}
		if infoStr, ok := fields["info"]; ok {
			if err := json.Unmarshal([]byte(infoStr), &extraInfo); err != nil {
				log.Errorf("Unmarshal pub extra info %v", err)
				extraInfo = proto.ClientUserInfo{} // Needed?
			}
		}
		pub := proto.PubInfo{
			MediaInfo: *info,
			Info:      extraInfo,
			Tracks:    tracks,
		}
		pubs = append(pubs, pub)
	}

	resp := proto.GetPubResp{
		RoomInfo: data,
		Pubs:     pubs,
	}
	log.Infof("getPubs: resp=%v", resp)
	return resp, nil
}

func clientJoin(data proto.JoinMsg) (interface{}, *nprotoo.Error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()
	log.Infof("clientJoin: set %s => %v", ukey, &data.Info)
	err := redis.HSetTTL(ukey, "info", &data.Info, redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}
	log.Infof("Broadcast: peer-join = %v", data)
	broadcaster.Say(proto.IslbClientOnJoin, data)
	return struct{}{}, nil
}

func clientLeave(data proto.RoomInfo) (interface{}, *nprotoo.Error) {
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
	log.Infof("Broadcast peer-leave = %v", data)
	//make broadcast leave msg after remove stream msg, for ion block bug
	time.Sleep(500 * time.Millisecond)
	broadcaster.Say(proto.IslbClientOnLeave, data)
	return struct{}{}, nil
}

func getMediaInfo(data proto.MediaInfo) (interface{}, *nprotoo.Error) {
	// Ensure DC
	data.DC = dc

	mkey := data.BuildKey()
	log.Infof("getMediaInfo key=%s", mkey)

	if keys := redis.Keys(mkey + "*"); len(keys) > 0 {
		key := keys[0]
		log.Infof("Got: key => %s", key)
		fields := redis.HGetAll(key)
		tracks := make(map[string][]proto.TrackInfo)
		for key, value := range fields {
			if strings.HasPrefix(key, "track/") {
				msid, infos, err := proto.UnmarshalTrackField(key, value)
				if err != nil {
					log.Errorf("%v", err)
				}
				log.Debugf("msid => %s, tracks => %v\n", msid, infos)
				tracks[msid] = *infos
			}
		}

		resp := util.Map("mid", data.MID, "tracks", tracks)
		log.Infof("getMediaInfo: resp=%v", resp)
		return resp, nil
	}

	return nil, util.NewNpError(404, "MediaInfo Not found")
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

func broadcast(data proto.BroadcastMsg) (interface{}, *nprotoo.Error) {
	log.Infof("broadcaster.Say msg=%v", data)
	broadcaster.Say(proto.IslbOnBroadcast, data)
	return struct{}{}, nil
}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		go func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
			method := request.Method
			msg := request.Data
			log.Infof("method => %s", method)

			var result interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			case proto.IslbFindService:
				var msgData proto.FindServiceParams
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = findServiceNode(msgData)
				}
			case proto.IslbOnStreamAdd:
				var msgData proto.StreamAddMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = streamAdd(msgData)
				}
			case proto.IslbOnStreamRemove:
				var msgData proto.StreamRemoveMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = streamRemove(msgData)
				}
			case proto.IslbGetPubs:
				var msgData proto.RoomInfo
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = getPubs(msgData)
				}
			case proto.IslbClientOnJoin:
				var msgData proto.JoinMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = clientJoin(msgData)
				}
			case proto.IslbClientOnLeave:
				var msgData proto.RoomInfo
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = clientLeave(msgData)
				}
			case proto.IslbGetMediaInfo:
				var msgData proto.MediaInfo
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = getMediaInfo(msgData)
				}
			// case proto.IslbRelay:
			// 	result, err = relay(data)
			// case proto.IslbUnrelay:
			// 	result, err = unRelay(data)
			case proto.IslbOnBroadcast:
				var msgData proto.BroadcastMsg
				if err = msg.Unmarshal(&msgData); err == nil {
					result, err = broadcast(msgData)
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
