package service

import (
	"sync"
	"time"

	"github.com/cloudwebrtc/go-protoo/room"
	"github.com/pion/ion/log"

	"github.com/pion/ion/media"
	"github.com/pion/webrtc/v2"
)

func getRoom(id string) *Room {
	roomLock.RLock()
	defer roomLock.RUnlock()
	return rooms[id]
}

func createRoom(id string) *Room {
	roomLock.Lock()
	defer roomLock.Unlock()
	rooms[id] = newRoom(id)
	return rooms[id]
}

func deleteRoom(id string) {
	roomLock.Lock()
	defer roomLock.Unlock()
	delete(rooms, id)
}

type Room struct {
	room.Room
	ID string

	pubPeers    map[string]*media.WebRTCPeer
	subPeers    map[string]*media.WebRTCPeer
	pubPeerLock sync.RWMutex
	subPeerLock sync.RWMutex
}

func newRoom(id string) *Room {
	r := &Room{
		pubPeers: make(map[string]*media.WebRTCPeer),
		subPeers: make(map[string]*media.WebRTCPeer),
		ID:       id,
	}
	r.Room = *room.NewRoom(id)

	log.Infof("NewRoom r=%+v", r)
	return r
}

func (r *Room) getWebRTCPeer(id string, sender bool) *media.WebRTCPeer {
	if sender {
		r.pubPeerLock.RLock()
		defer r.pubPeerLock.RUnlock()
		return r.pubPeers[id]
	} else {
		r.subPeerLock.RLock()
		defer r.subPeerLock.RUnlock()
		return r.subPeers[id]
	}
	return nil
}

func (r *Room) delWebRTCPeer(id string, sender bool) {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		if r.pubPeers[id] != nil {
			if r.pubPeers[id].PC != nil {
				r.pubPeers[id].PC.Close()
			}
			r.pubPeers[id].Stop()
		}
		delete(r.pubPeers, id)

	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
		if r.subPeers[id] != nil {
			if r.subPeers[id].PC != nil {
				r.subPeers[id].PC.Close()
			}
			r.subPeers[id].Stop()
		}
		delete(r.subPeers, id)
	}
}

func (r *Room) addWebRTCPeer(id string, sender bool) {
	if sender {
		r.pubPeerLock.Lock()
		defer r.pubPeerLock.Unlock()
		if r.pubPeers[id] != nil {
			r.pubPeers[id].Stop()
		}
		r.pubPeers[id] = media.NewWebRTCPeer(id)
	} else {
		r.subPeerLock.Lock()
		defer r.subPeerLock.Unlock()
		if r.subPeers[id] != nil {
			r.subPeers[id].Stop()
		}
		r.subPeers[id] = media.NewWebRTCPeer(id)
	}
}

func (r *Room) answer(id string, pubid string, offer webrtc.SessionDescription, sender bool) (webrtc.SessionDescription, error) {
	log.Infof("Room.answer id=%s, pubid=%s, offer=%v", id, pubid, offer)

	p := r.getWebRTCPeer(id, sender)

	var err error
	var answer webrtc.SessionDescription
	if sender {
		answer, err = p.AnswerSender(offer)
	} else {
		r.pubPeerLock.RLock()
		pub := r.pubPeers[pubid]
		r.pubPeerLock.RUnlock()
		ticker := time.NewTicker(time.Millisecond * 2000)
		for {
			select {
			case <-ticker.C:
				goto ENDWAIT
			default:
				if pub.VideoTrack == nil || pub.AudioTrack == nil {
					time.Sleep(time.Millisecond * 100)
				} else {
					goto ENDWAIT
				}
			}
		}
	ENDWAIT:
		answer, err = p.AnswerReceiver(offer, &pub.VideoTrack, &pub.AudioTrack)
	}
	return answer, err
}

func (r *Room) Close() {
	log.Infof("Room.Close")
	r.pubPeerLock.Lock()
	defer r.pubPeerLock.Unlock()
	for _, v := range r.pubPeers {
		if v != nil {
			v.Stop()
			if v.PC != nil {
				v.PC.Close()
			}
		}
	}
}

func (r *Room) sendPLI(skipID string) {
	log.Infof("Room.sendPLI")
	r.pubPeerLock.RLock()
	defer r.pubPeerLock.RUnlock()
	for k, v := range r.pubPeers {
		if k != skipID {
			v.SendPLI()
		}
	}
}
