package service

import (
	"sync"

	"github.com/cloudwebrtc/go-protoo/peer"
)

type peerMap map[*peer.Peer]*Room

var (
	peers    peerMap
	peerLock sync.RWMutex
)

func addPeerRoom(signalPeer *peer.Peer, room *Room) {
	peerLock.Lock()
	defer peerLock.Unlock()
	peers[signalPeer] = room
}

func getPeerRoom(signalPeer *peer.Peer) *Room {
	peerLock.Lock()
	defer peerLock.Unlock()
	return peers[signalPeer]
}

func deletePeerRoom(signalPeer *peer.Peer) {
	peerLock.Lock()
	defer peerLock.Unlock()
	delete(peers, signalPeer)
}
