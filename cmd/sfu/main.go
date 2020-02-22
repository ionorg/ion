package main

import (
	"net/http"
	_ "net/http/pprof"

	sfu "github.com/pion/ion/pkg/biz/sfu"
	conf "github.com/pion/ion/pkg/conf/ion"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/rtc"
)

func init() {
	log.Init(conf.Log.Level)
	rtc.Init(conf.Rtp.Port, conf.WebRTC.ICE, "", "")
}

func main() {
	log.Infof("--- Starting SFU Node ---")
	node.Init()
	serviceNode := node.NewServiceNode(conf.Etcd.Addrs)
	serviceNode.RegisterNode("sfu", "node-sfu", "sfu-channel-id")
	sfu.Init(serviceNode.GetRPCChannel(), serviceNode.GetEventChannel())

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	select {}
}
