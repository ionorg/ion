package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	sig "os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	conf "github.com/pion/ion/pkg/conf/biz"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/ion/pkg/signal"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)

	signal.Init(signal.WebSocketServerConfig{
		Host:           conf.Signal.Host,
		Port:           conf.Signal.Port,
		CertFile:       conf.Signal.Cert,
		KeyFile:        conf.Signal.Key,
		WebSocketPath:  conf.Signal.WebSocketPath,
		AuthConnection: conf.Signal.AuthConnection,
	}, conf.Signal.AllowDisconnected, biz.Entry)
}

func main() {
	log.Infof("--- Starting BIZ Node ---")

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("Start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	if err := biz.Init(conf.Global.Dc, conf.Etcd.Addrs, conf.Nats.URL, conf.Signal.AuthRoom, conf.Avp.Elements); err != nil {
		log.Errorf("biz init error: %v", err)
	}
	defer biz.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal)
	sig.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ch:
		return
	}
}
