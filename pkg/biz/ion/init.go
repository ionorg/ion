package biz

import (
	"sync"

	"github.com/pion/ion/pkg/mq"
)

var (
	amqp          *mq.Amqp
	ionID         string
	quitChMap     = make(map[string]chan struct{})
	quitChMapLock sync.RWMutex
)

// Init func
func Init(id, mqURL string) {
	ionID = id
	amqp = mq.New(id, mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
}

// Close func
func Close() {
	quitChMapLock.Lock()
	for _, v := range quitChMap {
		if v != nil {
			close(v)
		}
	}
	quitChMapLock.Unlock()
	if amqp != nil {
		amqp.Close()
	}
}
