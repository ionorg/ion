// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/cmd/biz/grpc/server"
	bizpb "github.com/pion/ion/pkg/grpc/biz"
	sfupb "github.com/pion/ion/pkg/grpc/sfu"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/spf13/viper"
)

var (
	conf = biz.Config{}
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
	flag.StringVar(&file, "c", "conf/conf.toml", "config file")
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

	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)
	addr := fmt.Sprintf("%s:%d", conf.Signal.GRPC.Host, conf.Signal.GRPC.Port)
	log.Infof("--- Starting Biz(gRPC + gRPC-Web) Node ---\n %s", addr)
	options := server.DefaultWrapperedServerOptions()

	options.Addr = addr
	options.AllowAllOrigins = true
	options.UseWebSocket = true
	options.Cert = conf.Signal.GRPC.Cert
	options.Key = conf.Signal.GRPC.Key
	options.EnableTLS = (len(options.Cert) > 0 && len(options.Key) > 0)

	if options.EnableTLS {
		options.TLSAddr = addr
	}

	s := server.NewWrapperedGRPCWebServer(options)

	node := biz.NewBIZ("biz1")
	if _, err := node.Start(conf); err != nil {
		log.Errorf("biz init start: %v", err)
		os.Exit(-1)
	}
	defer node.Close()

	s.GRPCServer.RegisterService(&sfupb.SFU_ServiceDesc, node.GRPCServer())
	s.GRPCServer.RegisterService(&bizpb.Biz_ServiceDesc, node.GRPCServer())

	if err := s.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
