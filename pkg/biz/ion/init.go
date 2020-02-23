package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

var (
	protoo   *nprotoo.NatsProtoo
	rpcs     map[string]*nprotoo.Requestor
	services []discovery.Node
)

// Init func
func Init(rpcID, eventID string) {
	services = []discovery.Node{}
	rpcs = make(map[string]*nprotoo.Requestor)
	protoo = nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	checkRTC()
}

// WatchServiceNodes .
func WatchServiceNodes(service string, nodes []discovery.Node) {
	for _, item := range nodes {
		service := item.Info["service"]
		id := item.Info["id"]
		name := item.Info["name"]
		log.Debugf("Service [%s] %s => %s", service, name, id)
		_, found := rpcs[id]
		if !found {
			rpcID := node.GetRPCChannel(item)
			eventID := node.GetEventChannel(item)
			log.Infof("Create islb requestor: rpcID => [%s]", rpcID)
			rpcs[id] = protoo.NewRequestor(rpcID)
			handleIslbBroadCast(eventID)
		}
	}
	services = nodes
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
				if rpc, ok := getRPCForIslb(); ok {
					data := util.Map("rid", room.ID(), "uid", uid, "mid", mid)
					rpc.AsyncRequest(proto.IslbOnStreamRemove, data)
					log.Infof("biz.checkRTC islb.RpcCall mid=%v rid=%v uid=%v", mid, room.ID(), uid)
				}
			}
		}
	}()
}
