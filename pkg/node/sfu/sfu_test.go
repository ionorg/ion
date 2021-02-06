package sfu

import (
	"context"
	"testing"

	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	proto "github.com/pion/ion/pkg/grpc/rtc"
)

var (
	conf = Config{
		Nats: natsConf{
			URL: "nats://127.0.0.1:4222",
		},
	}
	file string

	nid = "sfu-01"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init(conf.Log.Level, fixByFile, fixByFunc)

}

func TestStart(t *testing.T) {
	s := NewSFU(nid)

	err := s.Start(conf)
	if err != nil {
		t.Error(err)
	}

	opts := []nats.Option{nats.Name("nats-grpc echo client")}
	// Connect to the NATS server.
	nc, err := nats.Connect(conf.Nats.URL, opts...)
	if err != nil {
		t.Error(err)
	}
	defer nc.Close()

	ncli := rpc.NewClient(nc, nid)
	cli := proto.NewRTCClient(ncli)

	stream, err := cli.Signal(context.Background())
	if err != nil {
		t.Error(err)
	}

	stream.Send(&proto.Signalling{
		Payload: &proto.Signalling_Join{
			Join: &proto.Join{
				Payload: &proto.Join_Req{
					Req: &proto.JoinRequest{
						Sid: "room1",
						Uid: "user1",
					},
				},
			},
		},
	})

	for {
		reply, err := stream.Recv()
		if err != nil {
			t.Fatalf("Signal: err %s", err)
			break
		}
		log.Debugf("Reply: reply %v", reply)
	}

	s.Close()
}
