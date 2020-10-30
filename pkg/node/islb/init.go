package islb

import (
	"time"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc = "default"
	//nolint:unused
	nid      = "islb-unkown-node-id"
	bid      string
	nrpc     *proto.NatsRPC
	redis    *db.Redis
	services map[string]discovery.Node
)

// Init func
func Init(dcID, nodeID string, redisCfg db.Config, etcd []string, natsURL string) {
	dc = dcID
	nid = nodeID
	bid = nodeID + "-event"
	redis = db.NewRedis(redisCfg)
	nrpc = proto.NewNatsRPC(natsURL)
	services = make(map[string]discovery.Node)
	handleRequest(nid)
}

// WatchServiceNodes .
func WatchServiceNodes(service string, state discovery.NodeStateType, node discovery.Node) {
	id := node.ID
	if state == discovery.UP {
		if _, found := services[id]; !found {
			services[id] = node
			service := node.Info["service"]
			name := node.Info["name"]
			log.Debugf("Service [%s] UP %s => %s", service, name, id)
		}
	} else if state == discovery.DOWN {
		if _, found := services[id]; found {
			service := node.Info["service"]
			name := node.Info["name"]
			log.Debugf("Service [%s] DOWN %s => %s", service, name, id)
			delete(services, id)
		}
	}
}
