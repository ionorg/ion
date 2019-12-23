package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

func watchStream(rid, uid, mid string, ssrcPt map[string]interface{}) {
	key := getPubMediaPath(rid, mid)
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			info := redis.HGetAll(key)
			if len(info) == 0 {
				onStreamRemove := util.Map("rid", rid, "method", proto.IslbOnStreamRemove, "uid", uid, "mid", mid)
				log.Infof("amqp.BroadCast onStreamRemove=%v", onStreamRemove)
				amqp.BroadCast(onStreamRemove)
				return
			}
		}
	}
}

func streamAdd(rid, uid, mid, from string, ssrcPt map[string]interface{}) {
	key := getPubNodePath(rid, uid)
	redis.HSetTTL(key, from, "", redisKeyTTL)
	key = getPubMediaPath(rid, mid)
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
		infoMap := redis.HGetAll(getUserInfoPath(rid, uid))
		for info := range infoMap {
			onStreamAdd := util.Map("method", proto.IslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "info", info)
			amqp.BroadCast(onStreamAdd)
		}
		// keep watching mid
		go watchStream(rid, uid, mid, ssrcPt)
	}
}

func keepAlive(rid, uid, mid, from string, ssrcPt map[string]interface{}) {
	key := getPubNodePath(rid, uid)
	redis.HSetTTL(key, from, "", redisKeyTTL)
	key = getPubMediaPath(rid, mid)
	for ssrc, pt := range ssrcPt {
		redis.HSetTTL(key, ssrc, pt, redisKeyTTL)
	}
}

func getPubs(rid, uid, mid, from, corrID string) {
	key := getPubMediaPathKey(rid)
	for _, k := range redis.Keys(key) {
		log.Infof("key=%s k=%s skip uid=%v", key, k, uid)
		if !strings.Contains(k, uid) {
			mid := strings.Split(k, "/")[3]
			ssrcs := "{"
			for ssrc, pt := range redis.HGetAll(getPubMediaPath(rid, mid)) {
				ssrcs += fmt.Sprintf("%s:%s, ", ssrc, pt)
			}
			ssrcs += "}"
			info := redis.HGetAll(getUserInfoPath(rid, uid))
			for i := range info {
				resp := util.Map("response", proto.IslbGetPubs, "rid", rid, "uid", uid, "mid", mid, "mediaInfo", ssrcs, "info", i)
				log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
				amqp.RpcCall(from, resp, corrID)
			}
		}
	}
}

func clientJoin(rid, uid, info string) {
	onJoin := util.Map("method", proto.IslbClientOnJoin, "rid", rid, "uid", uid, "info", info)
	key := getUserInfoPath(rid, uid)
	log.Infof("redis.HSetTTL %v %v", key, info)
	redis.HSetTTL(key, info, "", redisLongKeyTTL)
	log.Infof("amqp.BroadCast onJoin=%v", onJoin)
	amqp.BroadCast(onJoin)
}

func clientLeave(rid, uid string) {
	key := getPubNodePath(rid, uid)
	redis.Del(key)
	onLeave := util.Map("rid", rid, "method", proto.IslbClientOnLeave, "uid", uid)
	log.Infof("amqp.BroadCast onLeave=%v", onLeave)
	amqp.BroadCast(onLeave)
}

func getMediaInfo(rid, mid, from, corrID string) {
	key := getPubMediaPath(rid, mid)
	info := redis.HGetAll(key)
	infoStr := util.MarshalStrMap(info)
	resp := util.Map("response", proto.IslbGetMediaInfo, "mid", mid, "info", infoStr)
	log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
	amqp.RpcCall(from, resp, corrID)
}

func relay(rid, mid, from string) {
	uid := getUIDFromMID(mid)
	key := getPubNodePath(rid, uid)
	info := redis.HGetAll(key)
	for ip := range info {
		method := util.Map("method", proto.IslbRelay, "uid", uid, "sid", from, "mid", mid)
		log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
		amqp.RpcCall(ip, method, "")
	}
}

func unRelay(rid, mid, from, corrID string) {
	key := getPubNodePath(rid, mid)
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
			case proto.IslbKeepAlive:
				ssrcPt := util.Unmarshal(util.Val(msg, "mediaInfo"))
				keepAlive(rid, uid, mid, from, ssrcPt)
			case proto.IslbGetPubs:
				getPubs(rid, uid, mid, from, corrID)
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
