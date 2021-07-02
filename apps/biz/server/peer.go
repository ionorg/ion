package server

import (
	biz "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/ion/proto/ion"
)

// Peer represents a peer for client
type Peer struct {
	uid             string
	sid             string
	info            []byte
	lastStreamEvent *ion.StreamEvent
	closed          util.AtomicBool
	sndCh           chan *biz.SignalReply
}

func NewPeer(sid string, uid string, info []byte, senCh chan *biz.SignalReply) *Peer {
	p := &Peer{
		uid:   uid,
		sid:   sid,
		info:  info,
		sndCh: senCh,
	}
	return p
}

// Close peer
func (p *Peer) Close() {
	if !p.closed.Set(true) {
		return
	}
}

// UID return peer uid
func (p *Peer) UID() string {
	return p.uid
}

// SID return session id
func (p *Peer) SID() string {
	return p.sid
}

func (p *Peer) send(data *biz.SignalReply) error {
	select {
	case p.sndCh <- data:
	default:
		go func() {
			p.sndCh <- data
		}()
	}
	return nil
}

func (p *Peer) sendPeerEvent(event *ion.PeerEvent) error {
	data := &biz.SignalReply{
		Payload: &biz.SignalReply_PeerEvent{
			PeerEvent: event,
		},
	}
	return p.send(data)
}

func (p *Peer) sendStreamEvent(event *ion.StreamEvent) error {
	data := &biz.SignalReply{
		Payload: &biz.SignalReply_StreamEvent{
			StreamEvent: event,
		},
	}
	return p.send(data)
}

func (p *Peer) sendMessage(msg *ion.Message) error {
	data := &biz.SignalReply{
		Payload: &biz.SignalReply_Msg{
			Msg: msg,
		},
	}
	return p.send(data)
}
