package biz

import (
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

// strToMap make string value to map
func strToMap(msg map[string]interface{}, key string) {
	val := util.Val(msg, key)
	if val != "" {
		m := util.Unmarshal(val)
		msg[key] = m
	}
}

// request msg from islb
func handleRPCMsgMethod(from, method string, msg map[string]interface{}) {
	log.Infof("biz.handleRPCMsgMethod from=%s, method=%s msg=%v", from, method, msg)

	switch method {
	case proto.IslbOnStreamAdd:
		uid := util.Val(msg, "uid")
		rid := util.Val(msg, "rid")
		streamAdd := util.Map("rid", rid, "uid", uid)
		signal.NotifyAll(rid, proto.ClientOnStreamAdd, streamAdd)
	case proto.IslbRelay:
		sid := util.Val(msg, "sid")
		mid := util.Val(msg, "mid")
		rtc.NewRTPTransportSub(mid, sid, sid)
	case proto.IslbUnrelay:
		mid := util.Val(msg, "mid")
		sid := util.Val(msg, "sid")
		rtc.DelSub(mid, sid)
	}

}

// response msg from islb
func handleRPCMsgResp(corrID, from, resp string, msg map[string]interface{}) {
	log.Infof("biz.handleRPCMsgResp corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	strToMap(msg, "info")
	switch resp {
	case proto.IslbGetPubs, proto.IslbGetMediaInfo, proto.IslbUnrelay:
		amqp.Emit(corrID, msg)
	default:
		log.Warnf("biz.handleRPCMsgResp invalid protocol corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	}

}

// rpc msg from islb, two kinds: request response
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
			if from == ionID {
				continue
			}

			log.Infof("biz.handleRPCMsgs msg=%v", msg)
			method := util.Val(msg, "method")
			resp := util.Val(msg, "response")
			if method != "" {
				handleRPCMsgMethod(from, method, msg)
				continue
			}
			if resp != "" {
				corrID := m.CorrelationId
				handleRPCMsgResp(corrID, from, resp, msg)
			}
		}
	}()

}

// broadcast msg from islb
func handleBroadCastMsgs() {
	broadCastMsgs, err := amqp.ConsumeBroadcast()
	if err != nil {
		log.Errorf(err.Error())
	}

	go func() {
		defer util.Recover("biz.handleBroadCastMsgs")
		for m := range broadCastMsgs {
			msg := util.Unmarshal(string(m.Body))
			method := util.Val(msg, "method")
			if method == "" {
				continue
			}
			log.Infof("biz.handleBroadCastMsgs msg=%v", msg)

			rid := util.Val(msg, "rid")
			uid := util.Val(msg, "uid")
			//make signal.Notify send "info" as a json object, otherwise is a string (:
			strToMap(msg, "info")
			switch method {
			case proto.IslbOnStreamAdd:
				signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, msg)
			case proto.IslbOnStreamRemove:
				mid := util.Val(msg, "mid")
				signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamRemove, msg)
				rtc.DelPub(mid)
			case proto.IslbClientOnJoin:
				signal.NotifyAllWithoutID(rid, uid, proto.ClientOnJoin, msg)
			case proto.IslbClientOnLeave:
				signal.NotifyAllWithoutID(rid, uid, proto.ClientOnLeave, msg)
				rtc.DelSubFromAllPubByPrefix(uid)
			case proto.IslbOnBroadcast:
				signal.NotifyAllWithoutID(rid, uid, proto.ClientBroadcast, msg)
			}

		}
	}()
}
