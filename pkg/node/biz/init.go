package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	conf "github.com/pion/ion/pkg/conf/biz"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid      = "biz-unkown-node-id"
	protoo   *nprotoo.NatsProtoo
	rpcs     map[string]*nprotoo.Requestor
	services map[string]discovery.Node
	roomAuth conf.AuthConfig
)

// Init func
func Init(dcID, nodeID, rpcID, eventID string, natsURL string, authConf conf.AuthConfig) {
	dc = dcID
	nid = nodeID
	services = make(map[string]discovery.Node)
	rpcs = make(map[string]*nprotoo.Requestor)
	protoo = nprotoo.NewNatsProtoo(natsURL)
	roomAuth = authConf
}

// WatchServiceNodes .
func WatchServiceNodes(service string, state discovery.NodeStateType, node discovery.Node) {
	id := node.ID

	if state == discovery.UP {

		if _, found := services[id]; !found {
			services[id] = node
		}

		service := node.Info["service"]
		name := node.Info["name"]
		log.Debugf("Service [%s] %s => %s", service, name, id)

		_, found := rpcs[id]
		if !found {
			rpcID := discovery.GetRPCChannel(node)
			eventID := discovery.GetEventChannel(node)

			log.Infof("Create islb requestor: rpcID => [%s]", rpcID)
			rpcs[id] = protoo.NewRequestor(rpcID)

			log.Infof("handleIslbBroadCast: eventID => [%s]", eventID)
			protoo.OnBroadcast(eventID, handleIslbBroadcast)
		}

	} else if state == discovery.DOWN {
		delete(rpcs, id)
		delete(services, id)
	}
}

// Close func
func Close() {
	if protoo != nil {
		protoo.Close()
	}
}
