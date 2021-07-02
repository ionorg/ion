// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"flag"
	"fmt"
	"os"

	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	nproxy "github.com/cloudwebrtc/nats-grpc/pkg/rpc/proxy"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/cmd/signal/server"
	"github.com/pion/ion/pkg/node/signal"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	conf = signal.Config{}
	file string
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
	addr := fmt.Sprintf("%s:%d", conf.Signal.GRPC.Host, conf.Signal.GRPC.Port)
	log.Infof("--- Starting Signal (gRPC + gRPC-Web) Server ---")
	log.Infof("--- Bind to %s, NID = %v ---", addr, conf.Node.NID)

	options := server.DefaultWrapperedServerOptions()
	options.Addr = addr
	options.Cert = conf.Signal.GRPC.Cert
	options.Key = conf.Signal.GRPC.Key

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
		grpc.UnknownServiceHandler(nproxy.TransparentHandler(sig.Director)))

	s := server.NewWrapperedGRPCWebServer(options, srv)
	if err := s.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
