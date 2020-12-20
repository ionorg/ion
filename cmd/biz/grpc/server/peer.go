package server

import (
	"errors"

	pb "github.com/pion/ion/cmd/biz/grpc/proto"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/ion/pkg/proto"
)

// Peer represents a peer for client
type Peer struct {
	*biz.Peer
	stream pb.BIZ_SignalServer
}

// newPeer create a peer instance
func newPeer(uid proto.UID, bs *biz.Server, stream pb.BIZ_SignalServer) *Peer {
	p := &Peer{
		stream: stream,
	}
	p.Peer = biz.NewPeer(uid, bs, p.send)
	return p
}

// send message to the peer
func (p *Peer) send(msg interface{}) error {
	switch v := msg.(type) {
	case *proto.ClientOfferMsg:
		payload := &pb.Server_Offer{
			Offer: &pb.Offer{
				Desc: []byte(v.Desc.SDP),
			},
		}
		srvMsg := &pb.Server{
			Payload: payload,
		}
		return p.stream.Send(srvMsg)
	case *proto.ClientTrickleMsg:
		payload := &pb.Server_TrickleEvent{}
		srvMsg := &pb.Server{
			Payload: payload,
		}
		return p.stream.Send(srvMsg)
	case *proto.ToClientPeersMsg:
		payload := &pb.Server_PeersEvent{}
		srvMsg := &pb.Server{
			Payload: payload,
		}
		return p.stream.Send(srvMsg)
	case *proto.FromIslbStreamAddMsg:
	case *proto.ToClientPeerJoinMsg:
	case *proto.IslbPeerLeaveMsg:
		payload := &pb.Server_PeersEvent{}
		srvMsg := &pb.Server{
			Payload: payload,
		}
		return p.stream.Send(srvMsg)
	case *proto.IslbBroadcastMsg:
		payload := &pb.Server_BroadcastEvent{}
		srvMsg := &pb.Server{
			Payload: payload,
		}
		return p.stream.Send(srvMsg)
	}
	return errors.New("unknown message")
}

func (p *Peer) Close() {

}
