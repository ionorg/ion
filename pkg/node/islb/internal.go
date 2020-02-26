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
	for _, item := range services {
		if service == item.Info["service"] {
			rpcID := discovery.GetRPCChannel(item)
			name := item.Info["name"]
			resp := util.Map("name", name, "rpc-id", rpcID, "service", service, "id", item.Name)
			log.Infof("findServiceNode: [%s] %s => %s", service, name, rpcID)
			return resp, nil
		}
	}
	return nil, util.NewNpError(404, fmt.Sprintf("Service node [%s] not found", service))
}

func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")

	mkey := proto.BuildMediaInfoKey(dc, rid, mid)
	//TODO: Add identity fo sfu
	field, value, err := proto.MarshalNodeField(proto.NodeInfo{
		Name: "sfu-node-name1",
		ID:   "sfu-node-id",
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
	fields := redis.HGetAll(proto.BuildUserInfoKey(dc, rid, uid))
	msg := util.Map("rid", rid, "uid", uid, "mid", mid, "info", fields["info"])
	log.Infof("Broadcast: [stream-add] => %v", msg)
	broadcaster.Say(proto.IslbOnStreamAdd, msg)

	watchStream(mkey)
	return util.Map(), nil
}

func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")

	if mid == "" {
		uid := util.Val(data, "uid")
		mkey := proto.BuildMediaInfoKey(dc, rid, uid)
		for _, key := range redis.Keys(mkey + "*") {
			log.Infof("streamRemove: key => %s", key)
			err := redis.Del(mkey)
			if err != nil {
				log.Errorf("redis.Del err = %v", err)
			}
		}
		return util.Map(), nil
	}

	mkey := proto.BuildMediaInfoKey(dc, rid, mid)
	log.Infof("streamRemove: key => %s", mkey)
	err := redis.Del(mkey)
	if err != nil {
		log.Errorf("redis.Del err = %v", err)
	}
	return util.Map(), nil
}

func getPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	key := proto.BuildMediaInfoKey(dc, rid, "")
	log.Infof("getPubs: root key=%s", key)

	var pubs []map[string]interface{}
	for _, path := range redis.Keys(key + "*") {
		log.Infof("getPubs media info path = %s", path)
		info, err := proto.ParseMediaInfo(path)
		if err != nil {
			log.Errorf("%v", err)
		}
		fields := redis.HGetAll(proto.BuildUserInfoKey(info.DC, info.RID, info.UID))
		pub := util.Map("rid", rid, "uid", uid, "mid", info.MID, "info", fields["info"])
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

	ukey := proto.BuildUserInfoKey(dc, rid, uid)
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
	ukey := proto.BuildUserInfoKey(dc, rid, uid)
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
	mkey := proto.BuildMediaInfoKey(dc, rid, mid)
	log.Infof("getMediaInfo key=%s", mkey)
	fields := redis.HGetAll(mkey)

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

func relay(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	from := util.Val(data, "from")

	uid := proto.GetUIDFromMID(mid)
	key := proto.GetPubNodePath(rid, uid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbRelay, "uid", uid, "sid", from, "mid", mid)
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

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			switch method {
			case proto.IslbFindService:
				result, err = findServiceNode(data)
			case proto.IslbOnStreamAdd:
				result, err = streamAdd(data)
			case proto.IslbOnStreamRemove:
				result, err = streamRemove(data)
			case proto.IslbGetPubs:
				result, err = getPubs(data)
			case proto.IslbClientOnJoin:
				result, err = clientJoin(data)
			case proto.IslbClientOnLeave:
				result, err = clientLeave(data)
			case proto.IslbGetMediaInfo:
				result, err = getMediaInfo(data)
			case proto.IslbRelay:
				result, err = relay(data)
			case proto.IslbUnrelay:
				result, err = unRelay(data)
			case proto.IslbOnBroadcast:
				result, err = broadcast(data)
			}

			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}
