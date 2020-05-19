package avp

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid = "avp-unkown-node-id"
)

// Init func
func Init(dcID, nodeID, rpcID, eventID, natsURL string) {
	dc = dcID
	nid = nodeID
}
