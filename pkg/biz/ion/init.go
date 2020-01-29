package biz

import (
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/mq"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
)

var (
	amqp  *mq.Amqp
	ionID string
)

// Init func
func Init(id, mqURL string) {
	ionID = id
	amqp = mq.New(id, mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
	checkRTC()
}

// Close func
func Close() {
	if amqp != nil {
		amqp.Close()
	}
}

// checkRTC send `stream-remove` msg to islb when some pub has been cleaned
func checkRTC() {
	log.Infof("biz.checkRTC start")
	go func() {
		for mid := range rtc.CleanChannel {
			uid := proto.GetUIDFromMID(mid)
			room := signal.GetRoomByPeer(uid)
			if room != nil {
				key := proto.GetPubMediaPath(room.ID(), mid, 0)
				discovery.Del(key, true)
				// amqp.RpcCall(proto.IslbID, util.Map("method", proto.IslbOnStreamRemove, "rid", room.ID(), "uid", uid, "mid", mid), "")
				// log.Infof("biz.checkRTC amqp.RpcCall mid=%v rid=%v uid=%v", mid, room.ID(), uid)
			}
		}
	}()
}
