package biz

import (
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
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

// broadcast msg from islb
func handleIslbBroadCast(msg map[string]interface{}, subj string) {
	go func(msg map[string]interface{}){
		method := util.Val(msg, "method")
		data := msg["data"].(map[string]interface{})
		log.Infof("OnIslbBroadcast: method=%s, data=%v", method, data)
		rid := util.Val(data, "rid")
		uid := util.Val(data, "uid")
		//make signal.Notify send "info" as a json object, otherwise is a string (:
		strToMap(data, "info")
		switch method {
		case proto.IslbOnStreamAdd:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, data)
		case proto.IslbOnStreamRemove:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamRemove, data)
		case proto.IslbClientOnJoin:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnJoin, data)
		case proto.IslbClientOnLeave:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnLeave, data)
		case proto.IslbOnBroadcast:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientBroadcast, data)
		}
	}(msg)
}
