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
	switch msg.(type) {
	case *proto.ClientOfferMsg:
		reply := &pb.Server{}
		return p.stream.Send(reply)
		//return p.conn.Notify(p.ctx, proto.ClientOffer, v.Desc)
	case *proto.ClientTrickleMsg:
		//return p.conn.Notify(p.ctx, proto.ClientTrickle, v)
	case *proto.ToClientPeersMsg:
		//return p.conn.Notify(p.ctx, proto.ClientOnList, v)
	case *proto.FromIslbStreamAddMsg:
		//return p.conn.Notify(p.ctx, proto.ClientOnStreamAdd, v)
	case *proto.ToClientPeerJoinMsg:
		//return p.conn.Notify(p.ctx, proto.ClientOnJoin, v)
	case *proto.IslbPeerLeaveMsg:
		//return p.conn.Notify(p.ctx, proto.ClientOnLeave, v)
	case *proto.IslbBroadcastMsg:
		//return p.conn.Notify(p.ctx, proto.ClientBroadcast, v)
	}
	return errors.New("unknown message")
}
