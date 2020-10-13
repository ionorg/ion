package sfu

import (
	"sync"

	sfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/proto"
)

type server struct {
	sfu   *sfu.SFU
	peers map[proto.MID]*sfu.Peer
	mu    sync.RWMutex
}

func newServer(config *sfu.Config) *server {
	return &server{
		sfu:   sfu.NewSFU(*config),
		peers: make(map[proto.MID]*sfu.Peer),
	}
}

func (s *server) addPeer(mid proto.MID) *sfu.Peer {
	p := sfu.NewPeer(s.sfu)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[mid] = &p
	return &p
}

func (s *server) getPeer(mid proto.MID) *sfu.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peers[mid]
}

func (s *server) delPeer(mid proto.MID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, mid)
}
