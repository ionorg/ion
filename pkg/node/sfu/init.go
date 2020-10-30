package sfu

import (
	"github.com/pion/ion/pkg/proto"
)

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid  = "sfu-unkown-node-id"
	nrpc *proto.NatsRPC
)

// Init func
func Init(dcID, nodeID, natsURL string) {
	dc = dcID
	nid = nodeID
	nrpc = proto.NewNatsRPC(natsURL)
	handleRequest(nodeID)
}
