package server

import (
	"errors"

	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/util"
)

// Peer represents a peer for client
type Peer struct {
	info   *room.Peer
	sig    room.RoomSignal_SignalServer
	room   *Room
	closed util.AtomicBool
}

// NewPeer create a peer
// args: sid, uid, dest, name, role, protocol, direction, info, avatar, vendor
// at least sid and uid
func NewPeer() *Peer {
	p := &Peer{}
	return p
}

// NewPeer create a peer
// args: sid, uid, dest, name, role, protocol, direction, info, avatar, vendor
// at least sid and uid
// func NewPeer(args ...string) *Peer {
// 	if len(args) < 2 {
// 		return nil
// 	}

// 	sid, uid, dest, name, role, protocol, direction, info, avatar, vendor := util.GetArgs(args...)

// 	log.Infof("args= %v  %v  %v  %v  %v  %v %v  %v ", sid, uid, dest, name, role, protocol, direction, info)
// 	p := &Peer{
// 		sid:         sid,
// 		uid:         uid,
// 		dest:        dest,
// 		displayName: name,
// 		role:        role,
// 		protocol:    protocol,
// 		direction:   direction,
// 		info:        []byte(info),
// 		avatar:      avatar,
// 		vendor:      vendor,
// 	}
// 	return p
// }

// Close peer
func (p *Peer) Close() {
	if !p.closed.Set(true) {
		return
	}
	p.room.delPeer(p)
}

// UID return peer uid
func (p *Peer) UID() string {
	return p.info.Uid
}

// SID return session id
func (p *Peer) SID() string {
	return p.info.Sid
}

func (p *Peer) send(data *room.Reply) error {
	if p.sig == nil {
		return errors.New("p.sig == nil maybe not join")
	}
	return p.sig.Send(data)
}

func (p *Peer) sendPeerEvent(event *room.PeerEvent) error {
	data := &room.Reply{
		Payload: &room.Reply_Peer{
			Peer: event,
		},
	}
	return p.send(data)
}

func (p *Peer) sendMessage(msg *room.Message) error {
	data := &room.Reply{
		Payload: &room.Reply_Message{
			Message: msg,
		},
	}
	return p.send(data)
}
