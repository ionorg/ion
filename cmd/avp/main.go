package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/avp"
	"github.com/pion/ion/pkg/rtc"
)

func init() {
	var icePortStart, icePortEnd uint16

	if len(conf.WebRTC.ICEPortRange) == 2 {
		icePortStart = conf.WebRTC.ICEPortRange[0]
		icePortEnd = conf.WebRTC.ICEPortRange[1]
	}

	log.Init(conf.Log.Level)
	if err := rtc.Init(conf.Rtp.Port, conf.WebRTC.ICE, icePortStart, icePortEnd, "", ""); err != nil {
		panic(err)
	}
}

func main() {
	log.Infof("--- Starting AVP Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs, conf.Global.Dc)
	serviceNode.RegisterNode("avp", "node-avp", "avp-channel-id")

	rpcID := serviceNode.GetRPCChannel()
	eventID := serviceNode.GetEventChannel()
	avp.Init(conf.Global.Dc, serviceNode.NodeInfo().ID, rpcID, eventID, conf.Nats.URL, conf.Avp.Processors)

	serviceWatcher := discovery.NewServiceWatcher(conf.Etcd.Addrs, conf.Global.Dc)
	serviceWatcher.WatchServiceNode("islb", avp.WatchServiceNodes)

	select {}
}
