package signal

import (
	"sync"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/pion/ion/pkg/proto"
)

func newPeer(id proto.UID, t *transport.WebSocketTransport, claims *Claims) *Peer {
	return &Peer{
		Peer:   *peer.NewPeer(string(id), t),
		claims: claims,
	}
}

// Peer represents a peer
type Peer struct {
	sync.Mutex
	peer.Peer
	claims *Claims
	closed bool
}

// ID user/peer id
func (p *Peer) ID() proto.UID {
	return proto.UID(p.Peer.ID())
}

// Claims return the connection claims
func (p *Peer) Claims() *Claims {
	return p.claims
}

// Request to peer
func (p *Peer) Request(method string, data interface{}, accept peer.AcceptFunc, reject peer.RejectFunc) {
	p.Lock()
	defer p.Unlock()
	p.Peer.Request(method, data, accept, reject)
}

// Notify a message to the peer
func (p *Peer) Notify(method string, data interface{}) {
	p.Lock()
	defer p.Unlock()
	p.Peer.Notify(method, data)
}

// Close peer
func (p *Peer) Close() {
	p.Lock()
	defer p.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	p.Peer.Close()
}
