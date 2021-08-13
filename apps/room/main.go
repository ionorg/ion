// Package cmd contains an entrypoint for running an ion-sfu instance.
package main

import (
	"flag"

	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/apps/room/proto"
	room "github.com/pion/ion/apps/room/server"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
)

func main() {
	var confFile, addr, cert, key, loglevel string
	flag.StringVar(&confFile, "c", "", "config file")
	flag.StringVar(&addr, "addr", ":5551", "grpc listening addr")
	flag.StringVar(&cert, "cert", "", "cert for tls")
	flag.StringVar(&key, "key", "", "key for tls")
	flag.StringVar(&loglevel, "l", "info", "log level")
	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}
	conf := room.Config{}
	err := conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return
	}

	log.Init(loglevel)
	log.Infof("--- Starting Room Service ---")

	grpcServer := grpc.NewServer()
	options := util.DefaultWrapperedServerOptions()
	options.Addr = addr
	options.Cert = cert
	options.Key = key

	roomService := room.NewRoomService(conf.Redis)
	pb.RegisterRoomServiceServer(grpcServer, roomService)

	roomSignalSerivce := room.NewRoomSignalService(roomService)
	pb.RegisterRoomSignalServer(grpcServer, roomSignalSerivce)

	wrapperedSrv := util.NewWrapperedGRPCWebServer(options, grpcServer)
	if err := wrapperedSrv.Serve(); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}
