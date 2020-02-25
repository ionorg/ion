package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/sfu"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/pkg/rtc"
)

func init() {
	log.Init(conf.Log.Level)
	rtc.Init(conf.Rtp.Port, conf.WebRTC.ICE, "", "")
}

func main() {
	log.Infof("--- Starting SFU Node ---")
	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs)
	serviceNode.RegisterNode("sfu", "node-sfu", "sfu-channel-id")
	sfu.Init(serviceNode.GetRPCChannel(), serviceNode.GetEventChannel(), conf.Nats.URL)

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	select {}
}
