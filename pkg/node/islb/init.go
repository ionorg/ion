package islb

import (
	"sync"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
)

const (
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc = "default"
	//nolint:unused
	nid         = "islb-unkown-node-id"
	protoo      *nprotoo.NatsProtoo
	redis       *db.Redis
	services    map[string]discovery.Node
	broadcaster *nprotoo.Broadcaster
	roomMutex   sync.Mutex
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
