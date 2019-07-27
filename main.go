package main

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/pion/ion/biz"
	"github.com/pion/ion/conf"
	"github.com/pion/ion/log"
	"github.com/pion/ion/signal"
)

func main() {
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}
	signal.SetBizCall(biz.BizEntry)
	signal.Start()

	select {}
}
