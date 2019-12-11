package biz

import (
	"sync"

	"github.com/pion/ion/pkg/mq"
)

const (
	errInvalidJsep  = "jsep not found"
	errInvalidSDP   = "sdp not found"
	errInvalidRoom  = "room not found"
	errInvalidPubID = "pub id not found"
	errInvalidMID   = "media id not found"
	errInvalidAddr  = "addr not found"
	errInvalidUID   = "uid not found"
)

var (
	amqp     *mq.Amqp
	ionID    string
	quit     map[string]chan struct{}
	quitLock sync.RWMutex
)

func init() {
	quit = make(map[string]chan struct{})
}

// Init func
func Init(id, mqURL string) {
	ionID = id
	amqp = mq.New(id, mqURL)
	handleRPCMsgs()
	handleBroadCastMsgs()
}

// Close func
func Close() {
	quitLock.Lock()
	for _, v := range quit {
		if v != nil {
			close(v)
		}
	}
	quitLock.Unlock()
	if amqp != nil {
		amqp.Close()
	}
}
