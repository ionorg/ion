package avp

import (
	"sync"

	iavp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

// server represents an server instance
type server struct {
	config  iavp.Config
	clients map[string]*sfu
	mu      sync.RWMutex
}

// newServer creates a new avp server instance
func newServer(conf iavp.Config, elems map[string]iavp.ElementFun) *server {
	a := &server{
		config:  conf,
		clients: make(map[string]*sfu),
	}

	iavp.Init(elems)

	return a
}

// Process starts a process for a track.
func (a *server) Process(addr, pid, rid, tid, eid string, config []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		log.Infof("create a sfu client, addr=%s", addr)
		if c, err = newSFU(addr, a.config); err != nil {
			return err
		}
		c.onClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			log.Infof("sfu client close, addr=%s", addr)
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	} else {
		log.Infof("sfu client exist, addr=%s", addr)
	}

	t, err := c.getTransport(proto.RID(rid))
	if err != nil {
		return err
	}

	return t.Process(pid, tid, eid, config)
}
