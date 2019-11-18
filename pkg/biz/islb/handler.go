package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/mq"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

const (
	redisKeyTTL = 1500 * time.Millisecond
)

var (
	amqp  *mq.Amqp
	redis *db.Redis
)

// Init func
func Init(mqURL string, config db.Config) {
	amqp = mq.New(proto.IslbID, mqURL)
	redis = db.NewRedis(config)
	handleRPCMsgs()
	handleBroadCastMsgs()
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
			log.Infof("rpc from=%v method=%v msg=%v", from, method, msg)
			if method == "" {
				continue
			}
			switch method {
			case proto.IslbOnStreamAdd:
				pid := util.Val(msg, "pid")
				rid := util.Val(msg, "rid")
				ssrcPt := util.Unmarshal(util.Val(msg, "info"))
				key := rid + "/pub/node/" + pid
				redis.HSetTTL(key, from, "", redisKeyTTL)
				key = rid + "/pub/media/" + pid
				for ssrc, pt := range ssrcPt {
					redis.HSetTTL(key, ssrc, pt, redisKeyTTL)
				}
				if m := redis.HGetAll(rid + "/pub/media/" + pid); len(m) > 1 {
					onStreamAdd := util.Map("rid", rid, "method", proto.IslbOnStreamAdd, "pid", pid)
					amqp.BroadCast(onStreamAdd)
				}
			case proto.IslbKeepAlive:
				pid := util.Val(msg, "pid")
				rid := util.Val(msg, "rid")
				ssrcPt := util.Unmarshal(util.Val(msg, "info"))
				key := rid + "/pub/node/" + pid
				redis.HSetTTL(key, from, "", redisKeyTTL)
				key = rid + "/pub/media/" + pid
				for ssrc, pt := range ssrcPt {
					redis.HSetTTL(key, ssrc, pt, redisKeyTTL)
				}
			case proto.IslbGetPubs:
				rid := util.Val(msg, "rid")
				skipPid := util.Val(msg, "pid")
				key := rid + "/pub/media/*"
				for _, k := range redis.Keys(key) {
					if k != skipPid {
						pid := strings.Split(k, "/")[3]
						ssrcs := "{"
						for ssrc, pt := range redis.HGetAll(rid + "/pub/media/" + pid) {
							ssrcs += fmt.Sprintf("%s:%s, ", ssrc, pt)
						}
						ssrcs += "}"
						resp := util.Map("response", proto.IslbGetPubs, "rid", rid, "pid", pid, "info", ssrcs)
						log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
						amqp.RpcCall(from, resp, corrID)
					}
				}
			case proto.IslbOnStreamRemove:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				key := rid + "/pub/media/" + pid
				redis.Del(key)
				key = rid + "/pub/node/" + pid
				redis.Del(key)
				onStreamRemove := util.Map("rid", rid, "method", proto.IslbOnStreamRemove, "pid", pid)
				log.Infof("amqp.BroadCast onStreamRemove=%v", onStreamRemove)
				amqp.BroadCast(onStreamRemove)
			case proto.IslbClientOnJoin:
				rid := util.Val(msg, "rid")
				id := util.Val(msg, "id")
				onJoin := util.Map("rid", rid, "method", proto.IslbClientOnJoin, "id", id)
				log.Infof("amqp.BroadCast onJoin=%v", onJoin)
				amqp.BroadCast(onJoin)
			case proto.IslbClientOnLeave:
				rid := util.Val(msg, "rid")
				id := util.Val(msg, "id")
				key := rid + "/pub/media/" + id
				redis.Del(key)
				key = rid + "/pub/node/" + id
				redis.Del(key)
				onLeave := util.Map("rid", rid, "method", proto.IslbClientOnLeave, "id", id)
				log.Infof("amqp.BroadCast onLeave=%v", onLeave)
				amqp.BroadCast(onLeave)
			case proto.IslbGetMediaInfo:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				info := redis.HGetAll(rid + "/pub/media/" + pid)
				infoStr := util.MarshalStrMap(info)
				resp := util.Map("response", proto.IslbGetMediaInfo, "pid", pid, "info", infoStr)
				log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
				amqp.RpcCall(from, resp, corrID)
			case proto.IslbRelay:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				info := redis.HGetAll(rid + "/pub/node/" + pid)
				for ip := range info {
					method := util.Map("method", proto.IslbRelay, "pid", pid, "sid", from)
					log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
					amqp.RpcCall(ip, method, "")
				}
			case proto.IslbUnrelay:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				info := redis.HGetAll(rid + "/pub/node/" + pid)
				for ip := range info {
					method := util.Map("method", proto.IslbUnrelay, "pid", pid, "sid", from)
					log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
					amqp.RpcCall(ip, method, "")
				}
				// time.Sleep(time.Millisecond * 10)
				resp := util.Map("response", proto.IslbUnrelay, "pid", pid, "sid", from)
				log.Infof("amqp.RpcCall from=%s resp=%v corrID=%s", from, resp, corrID)
				amqp.RpcCall(from, resp, corrID)
			}
		}
	}()

}

func handleBroadCastMsgs() {
	broadCastMsgs, err := amqp.ConsumeBroadcast()
	if err != nil {
		log.Errorf(err.Error())
	}

	go func() {
		for m := range broadCastMsgs {
			msg := util.Unmarshal(string(m.Body))
			log.Infof("broadcast msg=%v", msg)
			from := util.Val(msg, "_from")
			id := util.Val(msg, "id")
			method := util.Val(msg, "method")
			if from == proto.IslbID || id == "" || method == "" {
				continue
			}
			switch method {
			}
		}
	}()
}
