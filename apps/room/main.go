// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/server"
)

// run as distributed node
func main() {
	var confFile, addr, cert, key, logLevel string
	flag.StringVar(&confFile, "c", "", "config file")
	flag.StringVar(&addr, "addr", ":5551", "grpc listening addr")
	flag.StringVar(&cert, "cert", "", "cert for tls")
	flag.StringVar(&key, "key", "", "key for tls")
	flag.StringVar(&logLevel, "l", "info", "log level")
	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}

	log.Init(logLevel)
	log.Infof("--- Starting Room Service ---")

	node := room.New()
	err := node.Load(confFile)
	if err != nil {
		log.Errorf("node load error: %v", err)
		os.Exit(-1)
	}

	err = node.Start()
	if err != nil {
		log.Errorf("node init start: %v", err)
		os.Exit(-1)
	}

	defer node.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
