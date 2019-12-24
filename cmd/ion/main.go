package main

import (
	"fmt"
	"net/http"

	_ "net/http/pprof"

	biz "github.com/pion/ion/pkg/biz/ion"
	conf "github.com/pion/ion/pkg/conf/ion"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc"
	"github.com/pion/ion/pkg/signal"
)

var (
	ionID = fmt.Sprintf("%s:%d", conf.Global.Addr, conf.Rtp.Port)
)

func init() {
	log.Init(conf.Log.Level)
	biz.Init(ionID, conf.Amqp.URL)
	rtc.Init(conf.Rtp.Port, conf.WebRTC.ICE)
	signal.Init(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key, biz.Entry)
	discovery.Init(conf.Etcd.Addrs)
	discovery.UpdateLoad(conf.Global.Addr, conf.Rtp.Port)
}

func close() {
	biz.Close()
}

func main() {
	defer close()
	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			http.ListenAndServe(conf.Global.Pprof, nil)
		}()
	}

	select {}
}
