package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
)

// broadcast msg from islb
func handleIslbBroadCast(msg nprotoo.Notification, subj string) {
	var isblSignalTransformMap = map[string]string{
		proto.IslbOnStreamAdd:    proto.ClientOnStreamAdd,
		proto.IslbOnStreamRemove: proto.ClientOnStreamRemove,
		proto.IslbClientOnJoin:   proto.ClientOnJoin,
		proto.IslbClientOnLeave:  proto.ClientOnLeave,
		proto.IslbOnBroadcast:    proto.ClientBroadcast,
	}
	go func(msg nprotoo.Notification) {
		var data proto.BroadcastMsg
		if err := msg.Data.Unmarshal(&data); err != nil {
			log.Errorf("Error parsing message %v", err)
			return
		}
		var data2 map[string]interface{}
		if err := msg.Data.Unmarshal(&data2); err != nil {
			log.Errorf("Error parsing message %v", err)
			return
		}

		log.Infof("OnIslbBroadcast: method=%s, data=%v", msg.Method, data2)
		if newMethod, ok := isblSignalTransformMap[msg.Method]; ok {
			signal.NotifyAllWithoutID(data.RID, data.UID, newMethod, data2)
		}
	}(msg)
}
