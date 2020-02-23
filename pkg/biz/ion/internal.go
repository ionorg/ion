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
func handleIslbBroadCast(eventID string) {
	log.Infof("handleIslbBroadCast: eventID => [%s]", eventID)

	protoo.OnBroadcast(eventID, func(msg map[string]interface{}, subj string) {
		method := util.Val(msg, "method")
		log.Infof("OnIslbBroadcast: msg=%v", msg)
		rid := util.Val(msg, "rid")
		uid := util.Val(msg, "uid")
		//make signal.Notify send "info" as a json object, otherwise is a string (:
		strToMap(msg, "info")
		switch method {
		case proto.IslbOnStreamAdd:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamAdd, msg)
		case proto.IslbOnStreamRemove:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnStreamRemove, msg)
		case proto.IslbClientOnJoin:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnJoin, msg)
		case proto.IslbClientOnLeave:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientOnLeave, msg)
		case proto.IslbOnBroadcast:
			signal.NotifyAllWithoutID(rid, uid, proto.ClientBroadcast, msg)
		}
	})
}
