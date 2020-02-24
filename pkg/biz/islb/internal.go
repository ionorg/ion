package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func watchStream(key string) {
	log.Infof("watchStream: key = %s", key)
	go func() {
		for cmd := range redis.Watch(context.TODO(), key) {
			log.Infof("watchStream: key %s cmd %s modified", key, cmd)
			if cmd == "del" || cmd == "expired" {
				//room1/media/pub/74baff6e-b8c9-4868-9055-b35d50b73ed6#LUMGUQ
				rid, mid, uid := proto.GetRIDMIDUIDFromMediaKey(key)
				if _, ok := streamDelCache[mid]; !ok {
					streamDelCacheLock.Lock()
					streamDelCache[mid] = true
					streamDelCacheLock.Unlock()
					time.AfterFunc(1*time.Second, func() {
						streamDelCacheLock.Lock()
						delete(streamDelCache, mid)
						streamDelCacheLock.Unlock()
					})
					msg := util.Map("rid", rid, "uid", uid, "mid", mid)
					log.Infof("watchStream.BroadCast: onStreamRemove=%v", msg)
					broadcaster.Say(proto.IslbOnStreamRemove, msg)
				}
			}
		}
	}()
}

/*Find service nodes by name, such as sfu|mcu|sip-gateway|rtmp-gateway */
func findServiceNode(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	service := util.Val(data, "service")
	for _, item := range services {
		if service == item.Info["service"] {
			rpcID := node.GetRPCChannel(item)
			name := item.Info["name"]
			resp := util.Map("name", name, "rpc-id", rpcID, "service", service)
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
	from := util.Val(data, "from")
	ssrcPt := util.Unmarshal(util.Val(data, "mediaInfo"))

	//room1/media/pub/
	key := proto.GetPubNodePath(rid, uid)
	redis.HSetTTL(key, from, "", redisKeyTTL)

	// room1/media/pub/${mid}
	streamID := proto.GetPubMediaPath(rid, mid, 0)
	redis.HSetTTL(streamID, fmt.Sprintf("%d", len(ssrcPt)), "", redisLongKeyTTL)

	for ssrc, pt := range ssrcPt {
		// room1/media/pub/${mid}/${ssrc}
		key := proto.GetPubMediaPath(rid, mid, util.StrToUint32(ssrc))
		log.Infof("Set MediaInfo %s => %s", key, pt)
		// room1/media/pub/${mid}/${ssrc} ${pt}
		redis.HSetTTL(key, "pt", pt.(string), redisLongKeyTTL)
	}

	//receive more than one streamAdd in 1s, only send once
	if _, ok := streamAddCache[mid]; !ok {
		streamAddCacheLock.Lock()
		streamAddCache[mid] = true
		streamAddCacheLock.Unlock()
		time.AfterFunc(1*time.Second, func() {
			streamAddCacheLock.Lock()
			delete(streamAddCache, mid)
			streamAddCacheLock.Unlock()
		})
		// room1/user/info/${uid}
		infoMap := redis.HGetAll(proto.GetUserInfoPath(rid, uid))
		for info := range infoMap {
			msg := util.Map("rid", rid, "uid", uid, "mid", mid, "info", info)
			log.Infof("Broadcast: [stream-add] => %v", msg)
			broadcaster.Say(proto.IslbOnStreamAdd, msg)
		}
	}

	watchStream(streamID)
	return util.Map(), nil
}

func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	if mid == "" {
		mid = util.Val(data, "uid")
	}
	key := proto.GetPubMediaPath(rid, mid, 0)
	//log.Infof("streamRemove key=%s", key)
	for _, path := range redis.Keys(key + "*") {
		log.Infof("streamRemove path=%s", path)
		redis.Del(path)
	}
	return util.Map(), nil
}

func getPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	key := proto.GetPubMediaPathKey(rid)
	log.Infof("getPubs rid=%s uid=%s key=%s", rid, uid, key)
	midSsrcPt := make(map[string]map[string]string)
	for _, path := range redis.Keys(key + "*/*") {
		pts := redis.HGetAll(path)
		log.Infof("key=%s path=%s pt=%s skip uid=%v", key, path, pts["pt"], uid)
		//key=room1/media/pub/ k=room1/media/pub/5514c31f-2375-427f-9517-db46e967f842#MGEGMG/3318957691 v=96 skip uid=6d3c3e56-93bf-4210-aa96-294964743beb
		if !strings.Contains(path, uid) {
			strs := strings.Split(path, "/")
			if midSsrcPt[strs[3]] == nil {
				midSsrcPt[strs[3]] = make(map[string]string)
			}
			midSsrcPt[strs[3]][strs[4]] = pts["pt"]
		}
	}
	var pubs []map[string]interface{}
	resp := util.Map("rid", rid, "uid", uid)
	for mid, ssrcPt := range midSsrcPt {
		ssrcs := "{"
		for ssrc, pt := range ssrcPt {
			ssrcs += fmt.Sprintf("\"%s\":\"%s\",", ssrc, pt)
		}
		ssrcs += "}"
		info := redis.HGetAll(proto.GetUserInfoPath(rid, uid))
		for i := range info {
			pubs = append(pubs, util.Map("mid", mid, "mediaInfo", ssrcs, "info", i))
		}
	}
	resp["pubs"] = pubs
	log.Infof("getPubs: resp=%v", resp)
	return resp, nil
}

func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	info := util.Val(data, "info")

	key := proto.GetUserInfoPath(rid, uid)
	log.Infof("clientJoin: set %s => %v", key, info)
	redis.HSetTTL(key, info, "", redisLongKeyTTL)

	msg := util.Map("rid", rid, "uid", uid, "info", info)
	log.Infof("Broadcast: peer-join = %v", msg)
	broadcaster.Say(proto.IslbClientOnJoin, msg)
	return util.Map(), nil
}

func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	key := proto.GetUserInfoPath(rid, uid)
	log.Infof("clientLeave: remove key => %s", key)
	redis.Del(key)
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
	key := proto.GetPubMediaPath(rid, mid, 0)
	log.Infof("getMediaInfo key=%s", key)

	ssrcPts := "{"
	var arr []string
	for _, path := range redis.Keys(key + "*/*") {
		log.Infof("Streams path=%s", path)
		ssrc := ""
		if strings.Contains(path, mid) {
			strs := strings.Split(path, "/")
			ssrc = strs[4]
		}
		pts := redis.HGetAll(path)
		pt := pts["pt"]
		arr = append(arr, fmt.Sprintf("\"%s\":\"%s\"", ssrc, pt))
	}
	ssrcPts += strings.Join(arr, ",")
	ssrcPts += "}"
	resp := util.Map("mid", mid, "info", ssrcPts)
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
	msg := util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "info", info)
	log.Infof("broadcaster.Say msg=%v", msg)
	broadcaster.Say(proto.IslbOnBroadcast, msg)
	return util.Map(), nil
}

func handleRequest(rpcID string) {
	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
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
	})
}
