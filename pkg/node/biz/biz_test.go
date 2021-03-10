package biz

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/pkg/grpc/biz"
	"github.com/pion/ion/pkg/grpc/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

const (
	natsURL = "nats://127.0.0.1:4222"
	dc      = "dc1"
	nid     = "biztest01"
	sid     = "testsid01"
	uid     = "testuid01"
	addr    = ":5558"
	msg     = "testmsg"
)

var (
	nc       *nats.Conn
	bs       *BizServer
	wg       *sync.WaitGroup
	stream   pb.Biz_SignalClient
	grpcConn *grpc.ClientConn
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init("debug", fixByFile, fixByFunc)

	wg = new(sync.WaitGroup)
	nc, _ = util.NewNatsConn(natsURL)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	log.Infof("--- Listening at %s ---", addr)

	s := grpc.NewServer()

	bs = newBizServer(dc, nid, []string{}, nc)

	bs.nodes["islb00"] = &discovery.Node{
		Service: proto.ServiceISLB,
		NID:     "islb00",
		DC:      "dc1",
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     natsURL,
		},
	}

	pb.RegisterBizServer(s, bs)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Panicf("failed to serve: %v", err)
		}
	}()

	// Set up a connection to the avp server.
	grpcConn, err = grpc.Dial("127.0.0.1"+addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("did not connect: %v", err)
		return
	}

	log.Infof("init done")
}

func TestJBizJoin(t *testing.T) {

	c := pb.NewBizClient(grpcConn)

	var err error
	stream, err = c.Signal(context.Background())
	if err != nil {
		t.Error(err)
	}

	stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Join{
			Join: &pb.Join{
				Sid: sid,
				Uid: uid,
			},
		},
	})

	reply, err := stream.Recv()
	if err != nil {
		t.Error(err)
	}

	r := bs.getRoom(sid)
	assert.EqualValues(t, sid, r.sid)

	p := r.getPeer(uid)
	assert.EqualValues(t, uid, p.UID())

	log.Infof("join reply %v", reply)
	log.Infof("TestJoin done")
}

func TestBizMessage(t *testing.T) {
	stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Msg{
			Msg: &ion.Message{
				From: uid,
				To:   uid,
				Data: []byte(msg),
			},
		},
	})

	reply, err := stream.Recv()
	if err != nil {
		t.Error(err)
	}

	log.Infof("Message reply %v", reply)
	log.Infof("TestBizMessage done")
}

func TestBizLeave(t *testing.T) {
	stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Leave{
			Leave: &pb.Leave{
				Uid: uid,
			},
		},
	})

	reply, err := stream.Recv()
	if err != nil {
		t.Error(err)
	}

	log.Infof("Leave reply %v", reply)
	assert.Empty(t, bs.getRoom(sid))

	err = stream.CloseSend()
	if err != nil {
		t.Error(err)
	}
	log.Infof("TestLeave done")
	time.Sleep(time.Second * 1)
	grpcConn.Close()
}
