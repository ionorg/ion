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

func newServer(config sfu.Config) *server {
	return &server{
		sfu:   sfu.NewSFU(config),
		peers: make(map[proto.MID]*sfu.Peer),
	}
}

func (s *server) getPeer(mid proto.MID) *sfu.Peer {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.peers[mid]
	if p == nil {
		p = sfu.NewPeer(s.sfu)
		s.peers[mid] = p
	}
	return p
}

func (s *server) delPeer(mid proto.MID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, mid)
}
