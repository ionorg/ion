package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/avp"
	"github.com/pion/ion/pkg/node/avp"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go"}
	fixByFunc := []string{}
	log.Init(conf.Avp.Log.Level, fixByFile, fixByFunc)
}

func main() {
	log.Infof("--- Starting AVP Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	if err := avp.Init(conf.Global.Dc, conf.Etcd.Addrs, conf.Nats.URL, conf.Avp); err != nil {
		log.Errorf("avp init error: %v", err)
	}
	defer avp.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ch:
		return
	}
}
