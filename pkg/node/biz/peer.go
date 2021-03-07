package biz

import (
	"sync"

	"github.com/pion/ion/pkg/util"
)

// Peer represents a peer for client
type Peer struct {
	uid       string
	sid       string
	info      []byte
	leaveOnce sync.Once
	closed    util.AtomicBool
	send      func(msg interface{}) error
}

// NewPeer create peer instance for client
func NewPeer(sid string, uid string, info []byte) *Peer {
	p := &Peer{
		uid:  uid,
		sid:  sid,
		info: info,
	}
	return p
}

// Close peer
func (p *Peer) Close() {
	if p.closed.Get() {
		return
	}
	p.closed.Set(true)

	// leave all rooms
	//if err := p.Leave(&proto.FromClientLeaveMsg{SID: p.sid}); err != nil {
	//	log.Infof("peer(%s) leave error: %v", p.sid, err)
	//}
}

// UID return peer uid
func (p *Peer) UID() string {
	return p.uid
}

// SID return session id
func (p *Peer) SID() string {
	return p.sid
}

/*
func (p *Peer) sfu() (string, error) {
	return p.s.getNode(proto.ServiceSFU, p.uid, p.sid, p.mid)
}

func (p *Peer) avp() (string, error) {
	return p.s.getNode(proto.ServiceAVP, p.uid, p.sid, p.mid)
}
*/
