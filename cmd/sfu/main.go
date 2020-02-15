package main

import (
	biz "github.com/pion/ion/pkg/biz/ion"
	conf "github.com/pion/ion/pkg/conf/ion"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
)

func init() {
	log.Init(conf.Log.Level)
	rtc.Init(conf.Rtp.Port, conf.WebRTC.ICE)
	signal.Init(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key, biz.Entry)
	//discovery.Init(conf.Etcd.Addrs)
	//discovery.UpdateLoad(conf.Global.Addr, conf.Rtp.Port)
}

func main() {
	log.Infof("--- Starting SFU Node ---")
	node.Init()
	node := node.NewServiceNode(conf.Etcd.Addrs)
	node.RegisterNode("sfu", "node-sfu", "sfu-channel-id")

	select {}
}
