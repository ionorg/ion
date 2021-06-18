package server

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/ion/proto/ion"
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
	log.Init("debug")

	wg = new(sync.WaitGroup)
	nc, _ = util.NewNatsConn(natsURL)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	log.Infof("--- Listening at %s ---", addr)

	s := grpc.NewServer()

	bn := NewBIZ(nid)

	err = bn.Node.Start(natsURL)
	if err != nil {
		log.Panicf("failed to start biz node: %v", err)
	}

	bs, err = newBizServer(bn, dc, nid, nc)
	if err != nil {
		log.Panicf("failed to start biz node: %v", err)
	}

	//Watch ISLB nodes.
	go func() {
		err := bn.Node.Watch(proto.ServiceISLB)
		if err != nil {
			log.Panicf("failed to Watch: %v", err)
		}
	}()

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

	err = stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Join{
			Join: &pb.Join{
				Peer: &ion.Peer{
					Sid: sid,
					Uid: uid,
				},
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

	reply, err := stream.Recv()
	if err != nil {
		t.Error(err)
	}

	log.Infof("join reply %v", reply)

	r := bs.getRoom(sid)
	assert.EqualValues(t, sid, r.sid)

	p := r.getPeer(uid)
	assert.EqualValues(t, uid, p.UID())

	log.Infof("TestJoin done")
}

func TestBizMessage(t *testing.T) {
	err := stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Msg{
			Msg: &ion.Message{
				From: uid,
				To:   uid,
				Data: []byte(msg),
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

	reply, err := stream.Recv()
	if err != nil {
		t.Error(err)
	}

	log.Infof("Message reply %v", reply)
	log.Infof("TestBizMessage done")
}

func TestBizLeave(t *testing.T) {
	err := stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Leave{
			Leave: &pb.Leave{
				Uid: uid,
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

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
