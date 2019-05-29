package main

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/pion/ion/gslb"

	"github.com/pion/ion/conf"
	"github.com/pion/ion/log"
	"github.com/pion/ion/service"
)

func main() {
	if conf.SFU.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.SFU.Pprof)
			http.ListenAndServe(conf.SFU.Pprof, nil)
		}()
	}

	if !conf.SFU.Single {
		g, err := gslb.New()
		if err != nil {
			log.Errorf("gslb err => %v", err)
			return
		}
		g.KeepAlive()
	}
	service.Start()
	select {}
}
