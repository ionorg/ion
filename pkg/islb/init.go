package islb

import (
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/db"
)

const (
	redisKeyTTL     = 1500 * time.Millisecond
	redisLongKeyTTL = 24 * time.Hour
)

var (
	np    *nprotoo.NatsProtoo
	redis *db.Redis
)

// Init func
func Init(config db.Config) {
	np = nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	redis = db.NewRedis(config)
}
