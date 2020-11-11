package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/islb"
	"github.com/pion/ion/pkg/db"
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
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	redisCfg := db.Config{
		Addrs: conf.Redis.Addrs,
		Pwd:   conf.Redis.Pwd,
		DB:    conf.Redis.DB,
	}

	if err := islb.Init(conf.Global.Dc, conf.Etcd.Addrs, conf.Nats.URL, redisCfg); err != nil {
		log.Errorf("islb init error: %v", err)
	}
	defer islb.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ch:
		return
	}
}
