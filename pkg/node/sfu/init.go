package sfu

import (
	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/sssgun/ion/pkg/log"
	"github.com/sssgun/ion/pkg/proto"
	"github.com/sssgun/ion/pkg/rtc"
	"github.com/sssgun/ion/pkg/util"
)

var (
	//nolint:unused
	dc = "default"
	//nolint:unused
	nid         = "sfu-unkown-node-id"
	protoo      *nprotoo.NatsProtoo
	broadcaster *nprotoo.Broadcaster
)

// Init func
func Init(dcID, nodeID, rpcID, eventID, natsURL string) {
	dc = dcID
	nid = nodeID
	protoo = nprotoo.NewNatsProtoo(natsURL)
	broadcaster = protoo.NewBroadcaster(eventID)
	handleRequest(rpcID)
	checkRTC()
}

// checkRTC send `stream-remove` msg to islb when some pub has been cleaned
func checkRTC() {
	log.Infof("SFU.checkRTC start")
	go func() {
		for mid := range rtc.CleanChannel {
			broadcaster.Say(proto.SFUStreamRemove, util.Map("mid", mid))
		}
	}()
}
