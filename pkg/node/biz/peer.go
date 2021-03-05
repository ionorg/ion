package biz

import (
	"sync"

	"github.com/google/uuid"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

// Peer represents a peer for client
type Peer struct {
	uid       proto.UID
	mid       proto.MID
	sid       proto.SID
	info      []byte
	leaveOnce sync.Once
	closed    util.AtomicBool
	s         *Server
	send      func(msg interface{}) error
}

// NewPeer create peer instance for client
func NewPeer(uid proto.UID, s *Server, send func(msg interface{}) error) *Peer {
	id := uuid.New().String()
	p := &Peer{
		uid:  uid,
		mid:  proto.MID(id),
		s:    s,
		send: send,
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
func (p *Peer) UID() proto.UID {
	return p.uid
}

/*
func (p *Peer) sfu() (string, error) {
	return p.s.getNode(proto.ServiceSFU, p.uid, p.sid, p.mid)
}

func (p *Peer) avp() (string, error) {
	return p.s.getNode(proto.ServiceAVP, p.uid, p.sid, p.mid)
}
*/
