package islb

import (
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisKeyTTL     = 1500 * time.Millisecond
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc          = "default"
	nid         = "islb-unkown-node-id"
	protoo      *nprotoo.NatsProtoo
	redis       *db.Redis
	services    map[string]discovery.Node
	broadcaster *nprotoo.Broadcaster
)

// Init func
func Init(dcID, nodeID, rpcID, eventID string, redisCfg db.Config, etcd []string, natsURL string) {
	dc = dcID
	nid = nodeID
	redis = db.NewRedis(redisCfg)
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(eventID)
	services = make(map[string]discovery.Node)
	handleRequest(rpcID)
	WatchAllStreams()
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
		log.Debugf("Service [%s] UP %s => %s", service, name, id)
	} else if state == discovery.DOWN {
		if _, found := services[id]; found {

			service := node.Info["service"]
			name := node.Info["name"]
			log.Debugf("Service [%s] DOWN %s => %s", service, name, id)

			delete(services, id)
		}
	}
}

// WatchAllStreams .
func WatchAllStreams() {
	mkey := proto.BuildMediaInfoKey(dc, "*", "*", "*")
	log.Infof("Watch all streams, mkey = %s", mkey)
	for _, key := range redis.Keys(mkey) {
		log.Infof("Watch stream, key = %s", key)
		watchStream(key)
	}
}
