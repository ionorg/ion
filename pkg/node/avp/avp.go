package avp

import (
	"sync"
	"time"

	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion/pkg/proto"
)

const (
	statCycle = 5 * time.Second
)

var s *avp

// InitAVP init avp server
func InitAVP(conf *Config, elems map[string]iavp.ElementFun) {
	s = newAVP(conf, elems)
}

// Config for avp
type Config struct {
	iavp.Config
}

// avp represents an avp instance
type avp struct {
	config  Config
	clients map[string]*sfu
	mu      sync.RWMutex
}

// newAVP creates a new avp instance
func newAVP(conf *Config, elems map[string]iavp.ElementFun) *avp {
	a := &avp{
		config:  *conf,
		clients: make(map[string]*sfu),
	}

	iavp.Init(elems)

	return a
}

// Process starts a process for a track.
func (a *avp) Process(addr, pid, rid, tid string, eid []string, config []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		if c, err = newSFU(addr, a.config.Config); err != nil {
			return err
		}
		c.onClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	t, err := c.getTransport(proto.RID(rid))
	if err != nil {
		return err
	}

	for _, e := range eid {
		if err := t.Process(pid, tid, e, config); err != nil {
			return err
		}
	}

	return nil
}
