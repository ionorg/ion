package islb

import (
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

const (
	redisKeyTTL     = 1500 * time.Millisecond
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc          = "default"
	protoo      *nprotoo.NatsProtoo
	redis       *db.Redis
	services    []discovery.Node
	broadcaster *nprotoo.Broadcaster
)

// Init func
func Init(dcID, rpcID, eventID string, redisCfg db.Config, etcd []string, natsURL string) {
	dc = dcID
	redis = db.NewRedis(redisCfg)
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(eventID)
	handleRequest(rpcID)
	// handleBroadCastMsgs()
}

// WatchServiceNodes .
func WatchServiceNodes(service string, nodes []discovery.Node) {
	for _, node := range nodes {
		service := node.Info["service"]
		id := node.Info["id"]
		name := node.Info["name"]
		log.Debugf("Service [%s] %s => %s", service, name, id)
		//TODO: Watch `proto.IslbOnStreamRemove` from sfu nodes.
	}
	services = nodes
}
