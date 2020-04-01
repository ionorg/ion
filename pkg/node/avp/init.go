package avp

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	processor "github.com/pion/ion/pkg/node/avp/processors"
	"github.com/pion/ion/pkg/node/avp/processors/recorder"
)

var (
	dc          = "default"
	nid         = "avp-unkown-node-id"
	protoo      *nprotoo.NatsProtoo
	rpcs        map[string]*nprotoo.Requestor
	services    map[string]discovery.Node
	broadcaster *nprotoo.Broadcaster
	processors  map[string](func(id string) *processor.Processor)
)

// Init func
func Init(dcID, nodeID, rpcID, eventID, natsURL string, processorsCfg map[string]interface{}) {
	dc = dcID
	nid = nodeID
	protoo = nprotoo.NewNatsProtoo(natsURL)
	rpcs = make(map[string]*nprotoo.Requestor)
	services = make(map[string]discovery.Node)
	broadcaster = protoo.NewBroadcaster(eventID)
	processors = buildProcessors(processorsCfg)
}

func buildProcessors(conf map[string]interface{}) map[string]func(id string) *processor.Processor {
	processors := make(map[string]func(id string) *processor.Processor)
	for k, v := range conf {
		switch k {
		case "recorder":
			recorder.Init(v.(map[string]interface{}))
			processors[k] = (func(id string) *processor.Processor)(recorder.NewRecorder)
		}
	}
	return processors
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

			log.Infof("Register islb hander: handleIslbBroadCast: eventID => [%s]", eventID)
			protoo.OnBroadcast(eventID, handleIslbBroadCast)
		}

	} else if state == discovery.DOWN {
		if _, found := services[id]; found {
			delete(services, id)
		}
	}
}
