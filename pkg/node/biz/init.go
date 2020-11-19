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
	dc          string
	nid         string
	subs        map[string]*nats.Subscription
	roomAuth    conf.AuthConfig
	avpElements []string
	nrpc        *proto.NatsRPC
	nodeLock    sync.RWMutex
	nodes       map[string]discovery.Node
	serv        *discovery.Service
)

// Init biz
func Init(dcID string, etcdAddrs []string, natsURLs string, authConf conf.AuthConfig, elements []string) error {
	var err error

	dc = dcID
	roomAuth = authConf
	avpElements = elements
	nodes = make(map[string]discovery.Node)
	subs = make(map[string]*nats.Subscription)

	if nrpc, err = proto.NewNatsRPC(natsURLs); err != nil {
		return err
	}

	if serv, err = discovery.NewService("biz", dcID, etcdAddrs); err != nil {
		return err
	}
	if err := serv.GetNodes("islb", nodes); err != nil {
		return err
	}
	log.Infof("nodes up: %+v", nodes)
	nid = serv.NID()
	serv.Watch("islb", watchIslbNodes)
	serv.KeepAlive()

	return nil
}

// Close all
func Close() {
	closeSubs()
	if nrpc != nil {
		nrpc.Close()
	}
	if serv != nil {
		serv.Close()
	}
}

// watchNodes watch islb nodes up/down
func watchIslbNodes(state discovery.State, id string, node *discovery.Node) {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	if state == discovery.NodeUp {
		if _, found := nodes[id]; !found {
			nodes[id] = *node
		}
		if _, found := subs[id]; !found {
			log.Infof("subscribe islb: %s", node.NID)
			if sub, err := nrpc.Subscribe(node.NID+"-event", handleIslbBroadcast); err == nil {
				subs[id] = sub
			} else {
				log.Errorf("subcribe error: %v", err)
			}
		}
	} else if state == discovery.NodeDown {
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
