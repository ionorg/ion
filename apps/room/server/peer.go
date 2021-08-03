package server

import (
	room "github.com/pion/ion/apps/room/proto"
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
	/*
		select {
		case p.sndCh <- data:
		default:
			go func() {
				p.sndCh <- data
			}()
		}
	*/
	return nil
}

func (p *Peer) sendPeerEvent(event *room.ParticipantEvent) error {
	data := &room.Reply{
		Payload: &room.Reply_Notification{
			Notification: &room.Notification{
				Payload: &room.Notification_Participant{
					Participant: event,
				},
			},
		},
	}
	return p.send(data)
}

func (p *Peer) sendMessage(msg *room.Message) error {
	data := &room.Reply{
		Payload: &room.Reply_Notification{
			Notification: &room.Notification{
				Payload: &room.Notification_Message{
					Message: msg,
				},
			},
		},
	}
	return p.send(data)
}
