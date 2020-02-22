package biz

import (
	"github.com/pion/ion/pkg/discovery"
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

func lookupSFU(rid string) *discovery.Node {
	return &nodes[0]
}

// broadcast msg from islb
func handleBroadCastFromIslb(eventID string) {
	protoo.OnBroadcast(eventID, func(msg map[string]interface{}, subj string) {
		method := util.Val(msg, "method")
		log.Infof("handleBroadCastFromIslb: msg=%v", msg)
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
