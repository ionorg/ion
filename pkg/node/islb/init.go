package islb

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc       string
	nid      string
	bid      string
	nrpc     *proto.NatsRPC
	sub      *nats.Subscription
	redis    *db.Redis
	nodes    map[string]discovery.Node
	nodeLock sync.RWMutex
	serv     *discovery.Service
)

// Init islb
func Init(dcID string, etcdAddrs []string, natsURLs string, redisConf db.Config) error {
	var err error

	dc = dcID
	nodes = make(map[string]discovery.Node)

	if nrpc, err = proto.NewNatsRPC(natsURLs); err != nil {
		return err
	}

	redis = db.NewRedis(redisConf)

	if serv, err = discovery.NewService("islb", dcID, etcdAddrs); err != nil {
		return err
	}
	nid = serv.NID()
	bid = nid + "-event"
	serv.Watch("", watchNodes)
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		return err
	}

	return nil
}

// watchNodes watch nodes
func watchNodes(state discovery.State, node discovery.Node) {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	id := node.ID()
	if state == discovery.NodeUp {
		if _, found := nodes[id]; !found {
			nodes[id] = node
		}
	} else if state == discovery.NodeDown {
		if _, found := nodes[id]; found {
			delete(nodes, id)
		}
	}
}

func getNodes() map[string]discovery.Node {
	nodeLock.RLock()
	defer nodeLock.RUnlock()

	return nodes
}

// Close all
func Close() {
	if sub != nil {
		sub.Unsubscribe()
	}
	if nrpc != nil {
		nrpc.Close()
	}
	if redis != nil {
		redis.Close()
	}
	if serv != nil {
		serv.Close()
	}
}
