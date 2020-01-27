package biz

import (
	"sync"
	"time"

	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/mq"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisKeyTTL     = 1500 * time.Millisecond
	redisLongKeyTTL = 24 * time.Hour
)

var (
	amqp               *mq.Amqp
	redis              *db.Redis
	streamAddCache     = make(map[string]bool)
	streamAddCacheLock sync.RWMutex
	streamDelCache     = make(map[string]bool)
	streamDelCacheLock sync.RWMutex
)

// Init func
func Init(mqURL string, config db.Config) {
	amqp = mq.New(proto.IslbID, mqURL)
	redis = db.NewRedis(config)
	handleRPCMsgs()
	// handleBroadCastMsgs()
}
