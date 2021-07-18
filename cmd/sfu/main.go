package main

import (
	"flag"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/sfu"
)

func main() {
	var file string
	flag.StringVar(&file, "c", "configs/sfu.toml", "config file")
	flag.Parse()
	conf := sfu.Config{}
	err := conf.Load(file)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	node := sfu.New(conf)
	if err := node.Start(); err != nil {
		log.Errorf("sfu init start: %v", err)
		os.Exit(-1)
	}
	defer node.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
