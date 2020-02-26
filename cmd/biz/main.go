package main

import (
	"net/http"

	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/biz"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/biz"
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
	log.Infof("--- Starting Biz Node ---")

	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs)
	serviceNode.RegisterNode("biz", "node-biz", "biz-channel-id")

	rpcID := serviceNode.GetRPCChannel()
	eventID := serviceNode.GetEventChannel()
	biz.Init(conf.Global.Dc, rpcID, eventID, conf.Nats.URL)

	serviceWatcher := discovery.NewServiceWatcher(conf.Etcd.Addrs)
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
