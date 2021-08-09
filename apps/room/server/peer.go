package server

import (
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/util"
)

/*
string sid = 1;
string uid = 2;
string displayName = 3;
bytes extraInfo = 4;
Role role = 5;
string avatar = 6;
string vendor = 7;
string token = 8;
*/

// Peer represents a peer for client
type Peer struct {
	uid         string
	sid         string
	info        []byte
	displayName string
	sig         room.RoomSignal_SignalServer
	closed      util.AtomicBool
}

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

func (p *Peer) send(data *room.Reply) error {
	return p.sig.Send(data)
}

func (p *Peer) sendMediaPresentation(event *room.MediaPresentation) error {
	data := &room.Reply{
		Payload: &room.Reply_Presentation{
			Presentation: event,
		},
	}
	return p.send(data)
}

func (p *Peer) sendPeerEvent(event *room.ParticipantEvent) error {
	data := &room.Reply{
		Payload: &room.Reply_Participant{
			Participant: event,
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
