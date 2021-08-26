package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/server"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/pkg/util"

	"github.com/pion/ion/pkg/runner"
)

func main() {
	var roomConfFile, sfuConfFile, addr, logLevel, certFile, keyFile string
	flag.StringVar(&roomConfFile, "bc", "", "room config file")
	flag.StringVar(&sfuConfFile, "sc", "", "sfu config file")
	flag.StringVar(&addr, "addr", ":5551", "grpc listening addr")
	flag.StringVar(&certFile, "cert", "", "cert file")
	flag.StringVar(&keyFile, "key", "", "key file")
	flag.StringVar(&logLevel, "l", "info", "log level")
	flag.Parse()
	if roomConfFile == "" && sfuConfFile == "" {
		flag.PrintDefaults()
		return
	}

	log.Init(logLevel)
	log.Infof("--- Starting Conference ---")

	options := util.DefaultWrapperedServerOptions()
	options.Addr = addr
	options.Cert = certFile
	options.Key = keyFile

	r := runner.New(options)
	err := r.AddService(
		runner.ServiceUnit{
			Service:    room.New(),
			ConfigFile: roomConfFile,
		},
		runner.ServiceUnit{
			Service:    sfu.New(),
			ConfigFile: sfuConfFile,
		})
	if err != nil {
		log.Errorf("runner AddService error: %v", err)
		return
	}
	defer r.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
