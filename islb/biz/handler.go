package biz

import (
	"fmt"
	"strings"
	"time"

	"github.com/pion/ion/islb/db"
	"github.com/pion/ion/log"
	"github.com/pion/ion/mq"
	"github.com/pion/ion/proto"
	"github.com/pion/ion/util"
)

const (
	redisKeyTTL = 1500 * time.Millisecond
)

var (
	amqp  *mq.Amqp
	redis *db.Redis
)

func Init(mqUrl string, config db.Config) {
	amqp = mq.New(proto.IslbID, mqUrl)
	redis = db.NewRedis(config)
	handleRpcMsgs()
	handleBroadCastMsgs()
}

func handleRpcMsgs() {
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
			case proto.IslbPublish:
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
					onPublish := util.Map("rid", rid, "method", proto.IslbOnPublish, "pid", pid)
					amqp.BroadCast(onPublish)
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
			case proto.IslbUnpublish:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				key := rid + "/pub/media/" + pid
				redis.Del(key)
				key = rid + "/pub/node/" + pid
				redis.Del(key)
				onUnpublish := util.Map("rid", rid, "method", proto.IslbOnUnpublish, "pid", pid)
				log.Infof("amqp.BroadCast onUnpublish=%v", onUnpublish)
				amqp.BroadCast(onUnpublish)
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
				for ip, _ := range info {
					method := util.Map("method", proto.IslbRelay, "pid", pid, "sid", from)
					log.Infof("amqp.RpcCall ip=%s, method=%v", ip, method)
					amqp.RpcCall(ip, method, "")
				}
			case proto.IslbUnrelay:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				info := redis.HGetAll(rid + "/pub/node/" + pid)
				for ip, _ := range info {
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
