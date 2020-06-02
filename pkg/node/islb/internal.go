package islb

import (
	"context"
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

			eventID := discovery.GetEventChannel(node)
			log.Infof("handleBroadCast: eventID => [%s]", eventID)
			protoo.OnBroadcast(eventID, handleBroadcast)
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
func findServiceNode(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	service := util.Val(data, "service")
	mid := ""
	if data["mid"] != nil {
		mid = util.Val(data, "mid")
	}
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
					resp := util.Map("name", name, "rpc-id", rpcID, "event-id", eventID, "service", service, "id", id)
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
			resp := util.Map("name", name, "rpc-id", rpcID, "event-id", eventID, "service", service, "id", id)
			log.Infof("findServiceNode: [%s] %s => %s", service, name, rpcID)
			return resp, nil
		}
	}

	return nil, util.NewNpError(404, fmt.Sprintf("Service node [%s] not found", service))
}

func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	nid := util.Val(data, "nid")
	mid := util.Val(data, "mid")

	ukey := proto.UserInfo{
		DC:  dc,
		RID: rid,
		UID: uid,
	}.BuildKey()
	mkey := proto.MediaInfo{
		DC:  dc,
		NID: nid,
		RID: rid,
		UID: uid,
		MID: mid,
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

	tracks := data["tracks"].(map[string]interface{})
	for msid, track := range tracks {
		var infos []proto.TrackInfo
		for _, tinfo := range track.([]interface{}) {
			tmp := tinfo.(map[string]interface{})
			infos = append(infos, proto.TrackInfo{
				ID:      tmp["id"].(string),
				Type:    tmp["type"].(string),
				Ssrc:    int(tmp["ssrc"].(float64)),
				Payload: int(tmp["pt"].(float64)),
				Codec:   tmp["codec"].(string),
				Fmtp:    tmp["fmtp"].(string),
			})
		}
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
	msg := util.Map("rid", rid, "uid", uid, "mid", mid, "info", fields["info"], "tracks", tracks)
	log.Infof("Broadcast: [stream-add] => %v", msg)
	broadcaster.Say(proto.IslbOnStreamAdd, msg)

	watchStream(mkey)
	return util.Map(), nil
}

func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")

	mkey := proto.MediaInfo{
		DC:  dc,
		RID: rid,
		UID: uid,
		MID: mid,
	}.BuildKey()

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

func getPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	key := proto.MediaInfo{
		DC:  dc,
		RID: rid,
	}.BuildKey()
	log.Infof("getPubs: root key=%s", key)

	var pubs []map[string]interface{}
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
		pub := util.Map("rid", rid, "uid", info.UID, "mid", info.MID, "info", fields["info"], "tracks", tracks)
		pubs = append(pubs, pub)
	}

	resp := util.Map("rid", rid, "uid", uid)
	resp["pubs"] = pubs
	log.Infof("getPubs: resp=%v", resp)
	return resp, nil
}

func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")

	ukey := proto.UserInfo{
		DC:  dc,
		RID: rid,
		UID: uid,
	}.BuildKey()
	log.Infof("clientJoin: set %s => %v", ukey, info)
	err := redis.HSetTTL(ukey, "info", info, redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}
	msg := util.Map("rid", rid, "uid", uid, "info", info)
	log.Infof("Broadcast: peer-join = %v", msg)
	broadcaster.Say(proto.IslbClientOnJoin, msg)
	return util.Map(), nil
}

func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	ukey := proto.UserInfo{
		DC:  dc,
		RID: rid,
		UID: uid,
	}.BuildKey()
	log.Infof("clientLeave: remove key => %s", ukey)
	err := redis.Del(ukey)
	if err != nil {
		log.Errorf("redis.Del err = %v", err)
	}
	msg := util.Map("rid", rid, "uid", uid)
	log.Infof("Broadcast peer-leave = %v", msg)
	//make broadcast leave msg after remove stream msg, for ion block bug
	time.Sleep(500 * time.Millisecond)
	broadcaster.Say(proto.IslbClientOnLeave, msg)
	return util.Map(), nil
}

func getMediaInfo(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")

	mkey := proto.MediaInfo{
		DC:  dc,
		RID: rid,
		MID: mid,
	}.BuildKey()
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

		resp := util.Map("mid", mid, "tracks", tracks)
		log.Infof("getMediaInfo: resp=%v", resp)
		return resp, nil
	}

	return nil, util.NewNpError(404, "MediaInfo Not found")
}

func relay(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	from := util.Val(data, "from")

	key := proto.GetPubNodePath(rid, mid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbRelay, "sid", from, "mid", mid)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		//amqp.RpcCall(ip, method, "")
	}
	return util.Map(), nil
}

func unRelay(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	from := util.Val(data, "from")

	key := proto.GetPubNodePath(rid, mid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbUnrelay, "mid", mid, "sid", from)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		//amqp.RpcCall(ip, method, "")
	}
	// time.Sleep(time.Millisecond * 10)
	resp := util.Map("mid", mid, "sid", from)
	log.Infof("unRelay: resp=%v", resp)
	return resp, nil
}

func broadcast(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")
	msg := util.Map("rid", rid, "uid", uid, "info", info)
	log.Infof("broadcaster.Say msg=%v", msg)
	broadcaster.Say(proto.IslbOnBroadcast, msg)
	return util.Map(), nil
}

func handle(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	method := msg["method"].(string)
	data := msg["data"].(map[string]interface{})
	log.Infof("method => %s, data => %v", method, data)

	switch method {
	case proto.IslbFindService:
		return findServiceNode(data)
	case proto.IslbOnStreamAdd:
		return streamAdd(data)
	case proto.IslbOnStreamRemove:
		return streamRemove(data)
	case proto.IslbGetPubs:
		return getPubs(data)
	case proto.IslbClientOnJoin:
		return clientJoin(data)
	case proto.IslbClientOnLeave:
		return clientLeave(data)
	case proto.IslbGetMediaInfo:
		return getMediaInfo(data)
	case proto.IslbRelay:
		return relay(data)
	case proto.IslbUnrelay:
		return unRelay(data)
	case proto.IslbOnBroadcast:
		return broadcast(data)
	}

	return nil, util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))
}

func handleBroadcast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}) {
		_, err := handle(msg)
		if err != nil {
			log.Debugf("handleBroadcast error => %s", err.Reason)
		}
	}(msg)
}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			result, err := handle(request)
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}
