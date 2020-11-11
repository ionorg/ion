package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	conf "github.com/pion/ion/pkg/conf/sfu"
	"github.com/pion/ion/pkg/node/sfu"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go", "icegatherer.go"}
	fixByFunc := []string{}
	log.Init(conf.Log.Level, fixByFunc, fixByFile)

	sfu.InitSFU(&isfu.Config{
		WebRTC: *conf.WebRTC,
		Log:    *conf.Log,
		Router: *conf.Router,
	})
}

func main() {
	log.Infof("--- Starting SFU Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	if err := sfu.Init(conf.Global.Dc, conf.Etcd.Addrs, conf.Nats.URL); err != nil {
		log.Errorf("sfu init error: %v", err)
	}
	defer sfu.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ch:
		return
	}
}
