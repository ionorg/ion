package biz

import (
	"github.com/pion/ion/pkg/mq"
)

var (
	amqp  *mq.Amqp
	ionID string
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
	if amqp != nil {
		amqp.Close()
	}
}
