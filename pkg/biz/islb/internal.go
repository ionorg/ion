package biz

import (
	"fmt"
	"strings"
	"time"

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
						amqp.BroadCast(onStreamRemove)
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
			onStreamAdd := util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "info", info)
			log.Infof("streamAdd amqp.BroadCast %v", onStreamAdd)
			amqp.BroadCast(onStreamAdd)
		}

		go watchStream(rid, uid, mid, ssrcPt)
	}
}

func getPubs(rid, uid, from, corrID string) {
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
	for mid, ssrcPt := range midSsrcPt {
		ssrcs := "{"
		for ssrc, pt := range ssrcPt {
			ssrcs += fmt.Sprintf("\"%s\":%s, ", ssrc, pt)
		}
		ssrcs += "}"
		info := redis.HGetAll(proto.GetUserInfoPath(rid, uid))
		for i := range info {
			resp := util.Map("response", proto.IslbGetPubs, "rid", rid, "uid", uid, "mid", mid, "mediaInfo", ssrcs, "info", i)
			log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
			amqp.RpcCall(from, resp, corrID)
		}
	}
}

func clientJoin(rid, uid, info string) {
	onJoin := util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info)
	key := proto.GetUserInfoPath(rid, uid)
	log.Infof("redis.HSetTTL %v %v", key, info)
	redis.HSetTTL(key, info, "", redisLongKeyTTL)
	log.Infof("amqp.BroadCast onJoin=%v", onJoin)
	amqp.BroadCast(onJoin)
}

func clientLeave(rid, uid string) {
	key := proto.GetPubNodePath(rid, uid)
	redis.Del(key)
	onLeave := util.Map("rid", rid, "method", proto.IslbClientOnLeave, "uid", uid)
	log.Infof("amqp.BroadCast onLeave=%v", onLeave)
	//make broadcast leave msg after remove stream msg, for ion block bug
	time.Sleep(500 * time.Millisecond)
	amqp.BroadCast(onLeave)
}

func getMediaInfo(rid, mid, from, corrID string) {
	key := proto.GetPubMediaPath(rid, mid, 0)
	info := redis.HGetAll(key)
	infoStr := util.MarshalStrMap(info)
	resp := util.Map("response", proto.IslbGetMediaInfo, "mid", mid, "info", infoStr)
	log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
	amqp.RpcCall(from, resp, corrID)
}

func relay(rid, mid, from string) {
	uid := proto.GetUIDFromMID(mid)
	key := proto.GetPubNodePath(rid, uid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbRelay, "uid", uid, "sid", from, "mid", mid)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		amqp.RpcCall(ip, method, "")
	}
}

func unRelay(rid, mid, from, corrID string) {
	key := proto.GetPubNodePath(rid, mid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbUnrelay, "mid", mid, "sid", from)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		amqp.RpcCall(ip, method, "")
	}
	// time.Sleep(time.Millisecond * 10)
	resp := util.Map("response", proto.IslbUnrelay, "mid", mid, "sid", from)
	log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
	amqp.RpcCall(from, resp, corrID)
}

func broadcast(rid, uid, info string) {
	msg := util.Map("method", proto.IslbOnBroadcast, "rid", rid, "uid", uid, "info", info)
	log.Infof("amqp.BroadCast msg=%v", msg)
	amqp.BroadCast(msg)
}

func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		for m := range rpcMsgs {
			msg := util.Unmarshal(string(m.Body))

			from := m.ReplyTo
			corrID := m.CorrelationId
			if from == proto.IslbID {
				continue
			}
			method := util.Val(msg, "method")
			// log.Infof("rpc from=%v method=%v msg=%v", from, method, msg)
			if method == "" {
				continue
			}
			rid := util.Val(msg, "rid")
			uid := util.Val(msg, "uid")
			mid := util.Val(msg, "mid")
			switch method {
			case proto.IslbOnStreamAdd:
				ssrcPt := util.Unmarshal(util.Val(msg, "mediaInfo"))
				streamAdd(rid, uid, mid, from, ssrcPt)
			case proto.IslbGetPubs:
				getPubs(rid, uid, from, corrID)
			case proto.IslbClientOnJoin:
				info := util.Val(msg, "info")
				clientJoin(rid, uid, info)
			case proto.IslbClientOnLeave:
				clientLeave(rid, uid)
			case proto.IslbGetMediaInfo:
				getMediaInfo(rid, mid, from, corrID)
			case proto.IslbRelay:
				relay(rid, mid, from)
			case proto.IslbUnrelay:
				unRelay(rid, mid, from, corrID)
			case proto.IslbOnBroadcast:
				info := util.Val(msg, "info")
				broadcast(rid, uid, info)
			}
		}
	}()
}
