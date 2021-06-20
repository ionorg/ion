package sfu

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion-sfu/cmd/signal/grpc/proto"
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

	nid = "sfu-01"
)

func init() {
	log.Init("info")

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

	ncli := rpc.NewClient(nc, nid, "unkown")
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
		err = stream.Send(&pb.SignalRequest{
			Payload: &pb.SignalRequest_Trickle{
				Trickle: &pb.Trickle{
					Target: pb.Trickle_PUBLISHER,
					Init:   string(bytes),
				},
			},
		})
		if err != nil {
			t.Error(err)
		}
	})

	_, err = pub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		t.Error(err)
	}
	offer, err := pub.CreateOffer(nil)
	if err != nil {
		t.Error(err)
	}
	log.Infof("offer => %v", offer)

	marshalled, err := json.Marshal(offer)
	if err != nil {
		t.Error(err)
	}

	err = stream.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Join{
			Join: &pb.JoinRequest{
				Sid:         "room1",
				Description: marshalled,
			},
		},
	})

	if err != nil {
		t.Error(err)
	}

	err = pub.SetLocalDescription(offer)

	if err != nil {
		t.Error(err)
	}

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
			err = pub.SetRemoteDescription(sdp)
			if err != nil {
				t.Error(err)
			}
		case *pb.SignalReply_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				t.Error(err)
			}
			err = pub.AddICECandidate(candidate)
			if err != nil {
				t.Error(err)
			}
			//return
		}
	}

	s.Close()
}
