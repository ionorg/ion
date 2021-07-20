package main

import (
	"flag"
	"net/http"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
)

func main() {
	var confFile, addr, cert, key, loglevel string
	flag.StringVar(&confFile, "c", "", "sfu config file")
	flag.StringVar(&addr, "addr", ":9090", "grpc listening addr")
	flag.StringVar(&cert, "cert", "", "cert for tls")
	flag.StringVar(&key, "key", "", "key for tls")
	flag.StringVar(&loglevel, "l", "info", "log level")
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
	log.Infof("--- Starting ION SFU ---")

	grpcServer := grpc.NewServer()
	options := util.DefaultWrapperedServerOptions()
	options.Addr = addr
	options.Cert = cert
	options.Key = key

	sfu := sfu.NewSFUService(conf.Config)

	sfu.RegisterService(grpcServer)

	wrapperedSrv := util.NewWrapperedGRPCWebServer(options, grpcServer)
	if err := wrapperedSrv.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
