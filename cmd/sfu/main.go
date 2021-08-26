package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/sfu"
)

func main() {
	var confFile, addr, cert, key, logLevel string
	flag.StringVar(&confFile, "c", "", "sfu config file")
	flag.StringVar(&addr, "addr", ":5551", "grpc listening addr")
	flag.StringVar(&cert, "cert", "", "cert for tls")
	flag.StringVar(&key, "key", "", "key for tls")
	flag.StringVar(&logLevel, "l", "info", "log level")
	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}

	conf := sfu.Config{}
	err := conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	node := sfu.NewSFU(conf.Node.NID)
	if err := node.Start(conf); err != nil {
		log.Errorf("sfu init start: %v", err)
		os.Exit(-1)
	}
	defer node.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
