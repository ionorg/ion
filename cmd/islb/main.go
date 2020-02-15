package main

import (
	"net/http"

	biz "github.com/pion/ion/pkg/biz/islb"
	conf "github.com/pion/ion/pkg/conf/islb"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
)

func main() {
	log.Init(conf.Log.Level)
	log.Infof("--- Starting ISLB Node ---")
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	discovery.Init(conf.Etcd.Addrs)
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	biz.Init(conf.Amqp.Url, config)
	node.Init()
	sw := node.NewServiceWatcher(conf.Etcd.Addrs)
	//protoo := nprotoo.NewNatsProtoo(conf.Nats.Addrs)
	go sw.WatchServiceNode("sfu", func(service string, nodes []discovery.Node) {
		log.Infof("Service [%s] => %v", service, nodes)
		/*
			for _, item := range nodes {
				req := protoo.NewRequestor(node.GetRPCChannel(item))
				req.Request("offer", JsonEncode(`{ "sdp": "dummy-sdp"}`),
					func(result map[string]interface{}) {
						log.Infof("offer success: =>  %s", result)
					},
					func(code int, err string) {
						log.Warnf("offer reject: %d => %s", code, err)
					})
			}*/
	})

	select {}
}
