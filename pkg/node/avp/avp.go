package avp

import (
	"fmt"
	"os"
	"path"
	"sync"

	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

var s *avp

// initAVP create a avp server
func initAVP(conf *Config) {
	elems := make(map[string]iavp.ElementFun)

	if conf.Element.Webmsaver.On {
		if _, err := os.Stat(conf.Element.Webmsaver.Path); os.IsNotExist(err) {
			if err = os.MkdirAll(conf.Element.Webmsaver.Path, 0755); err != nil {
				log.Errorf("make dir error: %v", err)
			}
		}
		elems["webmsaver"] = func(rid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", rid, pid)))
			webm := elements.NewWebmSaver()
			webm.Attach(filewriter)
			return webm
		}
	}

	s = newAVP(conf.Config, elems)
}

// Config for avp
type Config struct {
	iavp.Config
	Element elementConf `mapstructure:"element"`
}

type webmsaver struct {
	On   bool   `mapstructure:"on"`
	Path string `mapstructure:"path"`
}
type elementConf struct {
	Webmsaver webmsaver `mapstructure:"webmsaver"`
}

// avp represents an avp instance
type avp struct {
	config  iavp.Config
	clients map[string]*sfu
	mu      sync.RWMutex
}

// newAVP creates a new avp instance
func newAVP(conf iavp.Config, elems map[string]iavp.ElementFun) *avp {
	a := &avp{
		config:  conf,
		clients: make(map[string]*sfu),
	}

	iavp.Init(elems)

	return a
}

// Process starts a process for a track.
func (a *avp) Process(addr, pid, rid, tid, eid string, config []byte) error {
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
