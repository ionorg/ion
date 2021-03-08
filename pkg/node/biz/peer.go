package biz

import (
	"sync"

	"github.com/pion/ion/pkg/grpc/biz"
	"github.com/pion/ion/pkg/util"
)

// Peer represents a peer for client
type Peer struct {
	uid       string
	sid       string
	info      []byte
	leaveOnce sync.Once
	closed    util.AtomicBool
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

func (p *Peer) send(msg interface{}) error {
	return nil
}

func (p *Peer) handleRequest(req *biz.JoinRequest) (*biz.JoinReply, error) {
	return nil, nil
}
