package main

import (
	"net/http"

	_ "net/http/pprof"

	biz "github.com/pion/ion/pkg/biz/ion"
	conf "github.com/pion/ion/pkg/conf/ion"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
	"github.com/pion/ion/pkg/signal"
)

func init() {
	log.Init(conf.Log.Level)
	signal.Init(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key, biz.Entry)
}

func close() {
	biz.Close()
}

func main() {
	node.Init()
	log.Infof("--- Starting Biz Node ---")

	serviceNode := node.NewServiceNode(conf.Etcd.Addrs)
	serviceNode.RegisterNode("biz", "node-biz", "biz-channel-id")
	biz.Init(serviceNode.GetRPCChannel(), serviceNode.GetEventChannel())

	serviceWatcher := node.NewServiceWatcher(conf.Etcd.Addrs)
	serviceWatcher.WatchServiceNode("islb", biz.WatchServiceNodes)

	defer close()
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	select {}
}
