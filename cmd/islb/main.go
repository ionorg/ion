package main

import (
	"net/http"

	biz "github.com/pion/ion/pkg/biz/islb"
	conf "github.com/pion/ion/pkg/conf/islb"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

func main() {
	log.Init(conf.Log.Level)
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
	select {}
}
