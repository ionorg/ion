package main

import (
	"flag"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
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

	log.Init(conf.Log.Level)
	log.Infof("--- Starting ION SFU ---")

	grpcServer := grpc.NewServer()

	sfu := sfu.NewSFUService(conf.Config)

	sfu.RegisterService(grpcServer)
	options := util.NewWrapperedServerOptions(addr, cert, key, true)
	wrapperedSrv := util.NewWrapperedGRPCWebServer(options, grpcServer)
	if err := wrapperedSrv.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
