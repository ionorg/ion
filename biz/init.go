package biz

import (
	"sync"

	"github.com/pion/ion/mq"
)

const (
	signalLogin       = "login"
	signalJoin        = "join"
	signalLeave       = "leave"
	signalPublish     = "publish"
	signalUnPublish   = "unpublish"
	signalSubscribe   = "subscribe"
	signalUnSubscribe = "unsubscribe"
	signalOnPublish   = "onPublish"
	signalOnUnpublish = "onUnpublish"

	errInvalidJsep  = "jsep not found"
	errInvalidSDP   = "sdp not found"
	errInvalidRoom  = "room not found"
	errInvalidPubID = "pubid not found"
	errInvalidAddr  = "addr not found"
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

func Init(id, mqURL string) {
	ionID = id
	amqp = mq.New(id, mqURL)
	handleRpcMsgs()
	handleBroadCastMsgs()
}

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
