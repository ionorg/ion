// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	nproxy "github.com/cloudwebrtc/nats-grpc/pkg/rpc/proxy"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/signal"
	"github.com/pion/ion/pkg/util"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	conf        = signal.Config{}
	file, paddr string
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func unmarshal(rawVal interface{}) bool {
	if err := viper.Unmarshal(rawVal); err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", file, err)
		return false
	}
	return true
}

func load() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}

	if !unmarshal(&conf) || !unmarshal(&conf.Signal) {
		return false
	}
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", file, err)
		return false
	}

	fmt.Printf("config %s load ok!\n", file)

	return true
}

func parse() bool {
	flag.StringVar(&file, "c", "configs/sig.toml", "config file")
	flag.StringVar(&paddr, "paddr", ":6060", "pprof listening addr")

	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	log.Init(conf.Log.Level)

	if paddr != "" {
		go func() {
			log.Infof("start pprof on %s", paddr)
			err := http.ListenAndServe(paddr, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	addr := fmt.Sprintf("%s:%d", conf.Signal.GRPC.Host, conf.Signal.GRPC.Port)
	log.Infof("--- Starting Signal (gRPC + gRPC-Web) Server ---")
	log.Infof("--- Bind to %s  ---", addr)

	sig, err := signal.NewSignal(conf)
	if err != nil {
		log.Errorf("new signal: %v", err)
		os.Exit(-1)
	}
	err = sig.Start()
	if err != nil {
		log.Errorf("signal.Start: %v", err)
		os.Exit(-1)
	}
	defer sig.Close()

	srv := grpc.NewServer(
		grpc.CustomCodec(nrpc.Codec()), // nolint:staticcheck
		grpc.UnknownServiceHandler(nproxy.TransparentLongConnectionHandler(sig.Director)))

	s := util.NewWrapperedGRPCWebServer(util.NewWrapperedServerOptions(
		addr, conf.Signal.GRPC.Cert, conf.Signal.GRPC.Key, true), srv)

	if err := s.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
