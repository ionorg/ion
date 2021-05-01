package islb

import (
	"context"
	"testing"

	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	proto "github.com/pion/ion/pkg/grpc/islb"
)

var (
	conf = Config{
		Nats: natsConf{
			URL: "nats://127.0.0.1:4222",
		},
		Redis: db.Config{
			DB:    0,
			Pwd:   "",
			Addrs: []string{":6379"},
		},
	}

	nid = "islb-01"
)

func init() {
	log.Init(conf.Log.Level)

}

func TestStart(t *testing.T) {
	i := NewISLB(nid)

	err := i.Start(conf)
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
	cli := proto.NewISLBClient(ncli)

	reply, err := cli.FindNode(context.Background(), &proto.FindNodeRequest{
		Nid: "sfu-001",
	})

	if err != nil {
		t.Error(err)
	}

	log.Debugf("reply => %v", reply)

	i.Close()
}
