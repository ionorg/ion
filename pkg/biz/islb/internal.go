package biz

import (
	"fmt"
	"strings"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"go.etcd.io/etcd/clientv3"
)

func Watch(ch clientv3.WatchChan) {
	go func() {
		for {
			msg := <-ch
			for _, ev := range msg.Events {
				fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				if ev.Type == clientv3.EventTypeDelete {
					//room1/media/pub/74baff6e-b8c9-4868-9055-b35d50b73ed6#LUMGUQ
					rid, mid, uid := proto.GetRIDMIDUIDFromMediaKey(string(ev.Kv.Key))
					if _, ok := streamDelCache[mid]; !ok {
						streamDelCacheLock.Lock()
						streamDelCache[mid] = true
						streamDelCacheLock.Unlock()
						time.AfterFunc(1*time.Second, func() {
							streamDelCacheLock.Lock()
							delete(streamDelCache, mid)
							streamDelCacheLock.Unlock()
						})
						onStreamRemove := util.Map("rid", rid, "method", proto.IslbOnStreamRemove, "uid", uid, "mid", mid)
						log.Infof("amqp.BroadCast onStreamRemove=%v", onStreamRemove)
						broadcaster.Say(proto.IslbOnStreamRemove, onStreamRemove)
					}
				}
			}
		}
	}()
}

func watchStream(rid, uid, mid string, ssrcPt map[string]interface{}) {
	key := proto.GetPubMediaPath(rid, mid, 0)
	discovery.Watch(key, Watch, true)
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

	key := proto.GetPubNodePath(rid, uid)
	redis.HSetTTL(key, from, "", redisKeyTTL)
	key = proto.GetPubMediaPath(rid, mid, 0)
	for ssrc, pt := range ssrcPt {
		redis.HSetTTL(key, ssrc, pt, redisKeyTTL)
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
		infoMap := redis.HGetAll(proto.GetUserInfoPath(rid, uid))
		for info := range infoMap {
			msg := util.Map("rid", rid, "uid", uid, "mid", mid, "info", info)
			log.Infof("Broadcast: [stream-add] => %v", msg)
			broadcaster.Say(proto.IslbOnStreamAdd, msg)
		}
		go watchStream(rid, uid, mid, ssrcPt)
	}
	return util.Map(), nil
}

func getPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	key := proto.GetPubMediaPathKey(rid)
	log.Infof("getPubs rid=%s uid=%s key=%s", rid, uid, key)
	midSsrcPt := make(map[string]map[string]string)
	for path, pt := range discovery.GetByPrefix(key) {
		log.Infof("key=%s path=%s pt=%s skip uid=%v", key, path, pt, uid)
		//key=room1/media/pub/ k=room1/media/pub/5514c31f-2375-427f-9517-db46e967f842#MGEGMG/3318957691 v=96 skip uid=6d3c3e56-93bf-4210-aa96-294964743beb
		if !strings.Contains(path, uid) {
			strs := strings.Split(path, "/")
			if midSsrcPt[strs[3]] == nil {
				midSsrcPt[strs[3]] = make(map[string]string)
			}
			midSsrcPt[strs[3]][strs[4]] = pt
		}
	}
	var pubs []map[string]interface{}
	resp := util.Map("rid", rid, "uid", uid)
	for mid, ssrcPt := range midSsrcPt {
		ssrcs := "{"
		for ssrc, pt := range ssrcPt {
			ssrcs += fmt.Sprintf("\"%s\":%s, ", ssrc, pt)
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

	msg := util.Map("rid", rid, "uid", uid, "info", info)
	key := proto.GetUserInfoPath(rid, uid)
	log.Infof("redis.HSetTTL %v %v", key, info)
	redis.HSetTTL(key, info, "", redisLongKeyTTL)
	log.Infof("Broadcast: peer-join = %v", msg)
	broadcaster.Say(proto.IslbClientOnJoin, msg)
	return util.Map(), nil
}

func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	key := proto.GetPubNodePath(rid, uid)
	redis.Del(key)
	msg := util.Map("uid", uid)
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
	info := redis.HGetAll(key)
	infoStr := util.MarshalStrMap(info)
	resp := util.Map("mid", mid, "info", infoStr)
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
