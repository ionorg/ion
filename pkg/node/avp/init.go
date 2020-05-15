package avp

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

var (
	dc          = "default"
	nid         = "avp-unkown-node-id"
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
)

// Init func
func Init(dcID, nodeID, rpcID, eventID, natsURL string) {
	dc = dcID
	nid = nodeID
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(eventID)
}
