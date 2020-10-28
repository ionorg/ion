package main

import (
	"net/http"
	_ "net/http/pprof"

	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/islb"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/node/islb"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)
}

func main() {
	log.Infof("--- Starting ISLB Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				panic(err)
			}
		}()
	}

	serviceNode := discovery.NewServiceNode(conf.Etcd.Addrs, conf.Global.Dc)
	serviceNode.RegisterNode("islb", "node-islb", "islb-channel-id")

	redisCfg := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	rpcID := serviceNode.GetRPCChannel()
	eventID := serviceNode.GetEventChannel()
	islb.Init(conf.Global.Dc, serviceNode.NodeInfo().ID, rpcID, eventID, redisCfg, conf.Etcd.Addrs, conf.Nats.URL)

	serviceWatcher := discovery.NewServiceWatcher(conf.Etcd.Addrs, conf.Global.Dc)
	go serviceWatcher.WatchServiceNode("", islb.WatchServiceNodes)

	select {}
}
