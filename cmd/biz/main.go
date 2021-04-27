// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/pion/ion-log"
	pb "github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/ion/cmd/biz/proxy"
	"github.com/pion/ion/cmd/biz/server"
	bizpb "github.com/pion/ion/pkg/grpc/biz"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	flag.StringVar(&file, "c", "configs/biz.toml", "config file")
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

	director := func(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
		// Make sure we never forward internal services.
		if strings.HasPrefix(fullMethodName, "/com.example.internal.") {
			return ctx, nil, status.Errorf(codes.Unimplemented, "Unknown method")
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			// Decide on which backend to dial
			if val, exists := md[":authority"]; exists && val[0] == "staging.api.example.com" {
				// Make sure we use DialContext so the dialing can be cancelled/time out together with the context.
				conn, err := grpc.DialContext(ctx, "api-service.staging.svc.local", grpc.WithCodec(proxy.Codec()))
				return ctx, conn, err
			} else if val, exists := md[":authority"]; exists && val[0] == "api.example.com" {
				conn, err := grpc.DialContext(ctx, "api-service.prod.svc.local", grpc.WithCodec(proxy.Codec()))
				return ctx, conn, err
			}
		}
		return ctx, nil, status.Errorf(codes.Unimplemented, "Unknown method")
	}

	srv := grpc.NewServer(
		grpc.CustomCodec(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)))

	s := server.NewWrapperedGRPCWebServer(options, srv)

	node := biz.NewBIZ("biz01")
	if err := node.Start(conf); err != nil {
		log.Errorf("biz init start: %v", err)
		os.Exit(-1)
	}
	defer node.Close()
	s.GRPCServer.RegisterService(&bizpb.Biz_ServiceDesc, node.Service())

	//Register SFU sig bridge.
	sfusig := &biz.SFUSignalBridge{
		BizServer: node.Service(),
	}

	s.GRPCServer.RegisterService(&pb.SFU_ServiceDesc, sfusig)

	if err := s.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
