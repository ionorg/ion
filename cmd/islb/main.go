package main

import (
	"net/http"
	_ "net/http/pprof"

	conf "github.com/pion/ion/pkg/conf/islb"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node/islb"
)

func main() {
	log.Init(conf.Log.Level)
	log.Infof("--- Starting ISLB Node ---")
	discovery.Init(conf.Etcd.Addrs)
	redisCfg := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}

	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs)
	serviceNode.RegisterNode("islb", "node-islb", "islb-channel-id")

	eventID := serviceNode.GetEventChannel()
	rpcID := serviceNode.GetRPCChannel()
	islb.Init(rpcID, eventID, redisCfg, conf.Etcd.Addrs, conf.Nats.URL)

	serviceWatcher := discovery.NewServiceWatcher(conf.Etcd.Addrs)
	go serviceWatcher.WatchServiceNode("", islb.WatchServiceNodes)

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	select {}
}
