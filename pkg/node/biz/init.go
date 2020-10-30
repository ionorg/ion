package biz

import (
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/biz"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid         = "biz-unkown-node-id"
	subs        map[string]*nats.Subscription
	nodeLock    sync.RWMutex
	nodes       map[string]discovery.Node
	roomAuth    conf.AuthConfig
	avpElements []string
	nrpc        *proto.NatsRPC
)

// Init biz
func Init(dcID, nodeID, natsURL string, authConf conf.AuthConfig, elements []string) {
	dc = dcID
	nid = nodeID
	nodes = make(map[string]discovery.Node)
	subs = make(map[string]*nats.Subscription)
	nrpc = proto.NewNatsRPC(natsURL)
	roomAuth = authConf
	avpElements = elements
}

// Close nats rpc
func Close() {
	closeSubs()
	nrpc.Close()
}

// WatchIslbNodes watch islb nodes up/down
func WatchIslbNodes(service string, state discovery.NodeStateType, node discovery.Node) {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	id := node.ID

	if state == discovery.UP {
		if _, found := nodes[id]; !found {
			nodes[id] = node
		}

		service := node.Info["service"]
		name := node.Info["name"]
		log.Debugf("service [%s] %s => %s", service, name, id)

		_, found := subs[id]
		if !found {
			islb := node.Info["id"]
			log.Infof("subscribe islb: %s", islb)
			if sub, err := nrpc.Subscribe(islb+"-event", handleIslbBroadcast); err == nil {
				subs[id] = sub
			} else {
				log.Errorf("subcribe error: %v", err)
			}
		}
	} else if state == discovery.DOWN {
		delete(subs, id)
		delete(nodes, id)
	}
}

func getNodes() map[string]discovery.Node {
	nodeLock.RLock()
	defer nodeLock.RUnlock()
	return nodes
}

func closeSubs() {
	nodeLock.Lock()
	defer nodeLock.Unlock()
	for _, s := range subs {
		s.Unsubscribe()
	}
}
