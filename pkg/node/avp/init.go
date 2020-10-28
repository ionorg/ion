package avp

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid    = "avp-unkown-node-id"
	protoo *nprotoo.NatsProtoo
)

// Init func
func Init(dcID, nodeID, rpcID, eventID, natsURL string) {
	dc = dcID
	nid = nodeID
	protoo = nprotoo.NewNatsProtoo(natsURL)
	handleRequest(rpcID)
}
