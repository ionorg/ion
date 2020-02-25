package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

var (
	protoo   *nprotoo.NatsProtoo
	rpcs     map[string]*nprotoo.Requestor
	services []discovery.Node
)

// Init func
func Init(rpcID, eventID string, natsURL string) {
	services = []discovery.Node{}
	rpcs = make(map[string]*nprotoo.Requestor)
	protoo = nprotoo.NewNatsProtoo(natsURL)
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
			rpcID := discovery.GetRPCChannel(item)
			eventID := discovery.GetEventChannel(item)
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
