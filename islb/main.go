package main

import (
	"net/http"

	"github.com/pion/ion/islb/biz"
	"github.com/pion/ion/islb/conf"
	"github.com/pion/ion/islb/db"
	"github.com/pion/ion/log"
)

func main() {
	log.Init(conf.Log.Level)
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	config := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}
	biz.Init(conf.Amqp.Url, config)
	select {}
}
