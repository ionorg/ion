package biz

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

var (
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
)

// Init func
func Init(rpcID string, eventID string) {
	protoo = nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	broadcaster = protoo.NewBroadcaster(eventID)
	handleRequest(rpcID)
}
