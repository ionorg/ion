package avp

import (
	"net"

	"github.com/sssgun/ion/ion-avp/pkg/log"
	pb "github.com/sssgun/ion/ion-avp/pkg/proto/avp"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedAVPServer
}

// InitLogLevel for avp
func InitLogLevel(level string) {
	log.Init(level)
}

// Init func
func Init(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAVPServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
}
