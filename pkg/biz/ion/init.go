package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
)

var (
	protoo  *nprotoo.NatsProtoo
	islbRPC *nprotoo.Requestor
	nodes   []discovery.Node
	ionID   string
)

// Init func
func Init(rpcID, eventID string) {
	ionID = rpcID
	nodes = []discovery.Node{}
	protoo = nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	islbRPC = protoo.NewRequestor(rpcID)
	handleBroadCastFromIslb(eventID)
	checkRTC()
}

// WatchServiceNodes .
func WatchServiceNodes(service string, nodes []discovery.Node) {
	for _, node := range nodes {
		service := node.Info["service"]
		id := node.Info["id"]
		name := node.Info["name"]
		log.Infof("Service [%s] %s => %s", service, name, id)
	}
}

// Close func
func Close() {
	if protoo != nil {
		protoo.Close()
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
