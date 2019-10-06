package biz

import (
	"github.com/pion/ion/log"
	"github.com/pion/ion/proto"
	"github.com/pion/ion/rtc"
	"github.com/pion/ion/signal"
	"github.com/pion/ion/util"
)

func handleRpcMsgMethod(from, method string, msg map[string]interface{}) {
	log.Infof("biz.handleRpcMsgMethod from=%s, method=%s msg=%v", from, method, msg)
	switch method {
	case proto.IslbPublish:
		id := util.Val(msg, "id")
		rid := util.Val(msg, "rid")
		onpublish := util.Map("rid", rid, "pubid", id)
		signal.NotifyAll(rid, signalOnPublish, onpublish)
	case proto.IslbUnpublish:
		id := util.Val(msg, "id")
		rid := util.Val(msg, "rid")
		onUnpublish := util.Map("rid", rid, "pubid", id)
		signal.NotifyAll(rid, signalOnUnpublish, onUnpublish)
	case proto.IslbRelay:
		pid := util.Val(msg, "pid")
		sid := util.Val(msg, "sid")
		rtc.AddNewRTPSub(pid, sid, sid)
	case proto.IslbUnrelay:
		pid := util.Val(msg, "pid")
		sid := util.Val(msg, "sid")
		rtc.DelSub(pid, sid)
	}

}

func handleRpcMsgResp(corrID, from, resp string, msg map[string]interface{}) {
	log.Infof("biz.handleRpcMsgResp corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
	switch resp {
	case proto.IslbGetPubs:
		amqp.Emit(corrID, msg)
	case proto.IslbGetMediaInfo:
		amqp.Emit(corrID, msg)
	case proto.IslbUnrelay:
		amqp.Emit(corrID, msg)

	}

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
			if from == ionID {
				continue
			}
			log.Infof("biz.handleRpcMsgs msg=%v", msg)
			method := util.Val(msg, "method")
			resp := util.Val(msg, "response")
			if method != "" {
				handleRpcMsgMethod(from, method, msg)
			}
			if resp != "" {
				corrID := m.CorrelationId
				handleRpcMsgResp(corrID, from, resp, msg)
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
			method := util.Val(msg, "method")
			if method == "" {
				continue
			}
			log.Infof("biz.handleBroadCastMsgs msg=%v", msg)
			switch method {
			case proto.IslbOnPublish:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				msg["pubid"] = pid
				signal.NotifyAllWithoutID(rid, pid, signalOnPublish, msg)
			case proto.IslbOnUnpublish:
				rid := util.Val(msg, "rid")
				pid := util.Val(msg, "pid")
				msg["pubid"] = pid
				signal.NotifyAllWithoutID(rid, pid, signalOnUnpublish, msg)
			}
		}
	}()
}
