package sfu

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/pkg/grpc/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/tj/assert"
)

var (
	conf = Config{
		Global: global{
			Dc: "dc1",
		},
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
	cli := pb.NewSFUClient(ncli)

	stream, err := cli.Signal(context.Background())
	if err != nil {
		t.Error(err)
	}

	me := webrtc.MediaEngine{}
	assert.NoError(t, err)
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pub, err := api.NewPeerConnection(webrtc.Configuration{})
	assert.NoError(t, err)

	pub.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Infof("ICEConnectionState %v", state.String())
	})

	pub.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		log.Infof("OnICECandidate %v", candidate)
		bytes, err := json.Marshal(candidate)
		if err != nil {
			log.Errorf("OnIceCandidate error %s", err)
		}
		stream.Send(&pb.SignalRequest{
			Payload: &pb.SignalRequest_Trickle{
				Trickle: &pb.Trickle{
					Target: pb.Trickle_PUBLISHER,
					Init:   string(bytes),
				},
			},
		})
	})

	_, err = pub.CreateDataChannel("ion-sfu", nil)
	offer, err := pub.CreateOffer(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("offer => %v", offer)

	marshalled, err := json.Marshal(offer)

	stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Join{
			Join: &pb.JoinRequest{
				Sid:         "room1",
				Description: marshalled,
			},
		},
	})

	pub.SetLocalDescription(offer)

	for {
		reply, err := stream.Recv()
		if err != nil {
			t.Fatalf("Signal: err %s", err)
			break
		}
		log.Debugf("\nReply: reply %v\n", reply)

		switch payload := reply.Payload.(type) {
		case *pb.SignalReply_Description:
			var sdp webrtc.SessionDescription
			err := json.Unmarshal(payload.Description, &offer)
			if err != nil {
				t.Error(err)
			}
			pub.SetRemoteDescription(sdp)
		case *pb.SignalReply_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				t.Error(err)
			}
			pub.AddICECandidate(candidate)
			//return
		}
	}

	s.Close()
}
