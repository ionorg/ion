package server

import (
	"io"

	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/cmd/biz/grpc/proto"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/ion/pkg/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type signalConf struct {
	GRPC grpcConf `mapstructure:"grpc"`
}

// Config for server
type Config struct {
	biz.Config
	Signal signalConf `mapstructure:"signal"`
}

// signalConf represents signal server configuration
type grpcConf struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Cert            string `mapstructure:"cert"`
	Key             string `mapstructure:"key"`
	AllowAllOrigins bool   `mapstructure:"allow_all_origins"`
}

type BizServer struct {
	pb.UnimplementedBIZServer
	bs *biz.Server
}

func NewBizServerr(bs *biz.Server) *BizServer {
	return &BizServer{bs: bs}
}

func (bs *BizServer) Signal(stream pb.BIZ_SignalServer) error {
	//peer := sfu.NewPeer(s.SFU)
	var peer *Peer
	peer = nil
	for {
		in, err := stream.Recv()
		if err != nil {

			if peer != nil {
				peer.Close()
			}

			if err == io.EOF {
				return nil
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}

			log.Errorf("signal error %v %v", errStatus.Message(), errStatus.Code())
			return err
		}

		switch payload := in.Payload.(type) {
		case *pb.Client_Join:
			pid := proto.UID(payload.Join.Uid)
			sid := proto.SID(payload.Join.Sid)
			log.Infof("Client Join: sid => %v, pid => %v", sid, proto.IslbBroadcast)
			peer = newPeer(pid, bs.bs, stream)
		case *pb.Client_Leave:
		case *pb.Client_Offer:
		case *pb.Client_Answer:
		case *pb.Client_Broadcast:
		case *pb.Client_Trickle:
		}
	}
}
