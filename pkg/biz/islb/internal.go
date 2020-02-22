package biz

import (
	"fmt"
	"strings"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
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

func findServiceNode(service string, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	accept(util.Map())
}

func streamAdd(rid, uid, mid, from string, ssrcPt map[string]interface{}) {
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
			data := util.Map("rid", rid, "uid", uid, "mid", mid, "info", info)
			log.Infof("Broadcast: [stream-add] => %v", data)
			broadcaster.Say(proto.IslbOnStreamAdd, data)
		}
		go watchStream(rid, uid, mid, ssrcPt)
	}
}

func getPubs(rid, uid string) map[string]interface{} {
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
	resp := util.Map("response", proto.IslbGetPubs, "rid", rid, "uid", uid)
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
	return resp
}

func clientJoin(rid, uid, info string) {
	data := util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info)
	key := proto.GetUserInfoPath(rid, uid)
	log.Infof("redis.HSetTTL %v %v", key, info)
	redis.HSetTTL(key, info, "", redisLongKeyTTL)
	log.Infof("Broadcast: peer-join = %v", data)
	broadcaster.Say(proto.IslbClientOnJoin, data)
}

func clientLeave(rid, uid string) {
	key := proto.GetPubNodePath(rid, uid)
	redis.Del(key)
	data := util.Map(proto.IslbClientOnLeave, "uid", uid)
	log.Infof("Broadcast peer-leave = %v", data)
	//make broadcast leave msg after remove stream msg, for ion block bug
	time.Sleep(500 * time.Millisecond)
	broadcaster.Say(proto.IslbClientOnLeave, data)
}

func getMediaInfo(rid, mid string) map[string]interface{} {
	key := proto.GetPubMediaPath(rid, mid, 0)
	info := redis.HGetAll(key)
	infoStr := util.MarshalStrMap(info)
	resp := util.Map("mid", mid, "info", infoStr)
	log.Infof("getMediaInfo: resp=%v", resp)
	return resp
}

func relay(rid, mid, from string) {
	uid := proto.GetUIDFromMID(mid)
	key := proto.GetPubNodePath(rid, uid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbRelay, "uid", uid, "sid", from, "mid", mid)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		//amqp.RpcCall(ip, method, "")
	}
}

func unRelay(rid, mid, from string) map[string]interface{} {
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
	return resp
}

func broadcast(rid, uid, info string) {
	msg := util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "info", info)
	log.Infof("broadcaster.Say msg=%v", msg)
	broadcaster.Say(proto.IslbOnBroadcast, msg)
}

func handleRequest(rpcID string) {
	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Infof("method => %s, data => %v", method, data)
		rid := util.Val(data, "rid")
		uid := util.Val(data, "uid")
		mid := util.Val(data, "mid")
		from := util.Val(data, "from")
		switch method {
		case proto.IslbFindService:
			/*Find service nodes by name, such as sfu|mcu|sip-gateway|rtmp-gateway */
			service := util.Val(data, "service")
			findServiceNode(service, accept, reject)
		case proto.IslbOnStreamAdd:
			ssrcPt := util.Unmarshal(util.Val(data, "mediaInfo"))
			streamAdd(rid, uid, mid, from, ssrcPt)
			accept(make(map[string]interface{}))
		case proto.IslbGetPubs:
			resp := getPubs(rid, uid)
			accept(resp)
		case proto.IslbClientOnJoin:
			info := util.Val(data, "info")
			clientJoin(rid, uid, info)
			accept(make(map[string]interface{}))
		case proto.IslbClientOnLeave:
			clientLeave(rid, uid)
			accept(make(map[string]interface{}))
		case proto.IslbGetMediaInfo:
			resp := getMediaInfo(rid, mid)
			accept(resp)
		case proto.IslbRelay:
			relay(rid, mid, from)
			accept(make(map[string]interface{}))
		case proto.IslbUnrelay:
			unRelay(rid, mid, from)
		case proto.IslbOnBroadcast:
			info := util.Val(data, "info")
			broadcast(rid, uid, info)
			accept(make(map[string]interface{}))
		}

		//accept(JsonEncode(`{"answer": "dummy-sdp2"}`))
		//reject(404, "Not found")
	})
}
