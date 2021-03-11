package ion

import (
	"sync"
	"testing"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/stretchr/testify/assert"
)

var (
	natsURL  = "nats://127.0.0.1:4222"
	nc       *nats.Conn
	registry *discovery.Registry
	wg       *sync.WaitGroup
	nid      = "testnid001"
)

func init() {
	fixByFile := []string{"asm_amd64.s", "proc.go"}
	fixByFunc := []string{}
	log.Init("debug", fixByFile, fixByFunc)
	wg = new(sync.WaitGroup)
	nc, _ = util.NewNatsConn(natsURL)
	var err error
	registry, err = discovery.NewRegistry(nc)
	if err != nil {
		log.Errorf("%v", err)
	}
}

func TestWatch(t *testing.T) {
	n := NewNode(nid)

	registry.Listen(func(action string, node discovery.Node) {
		log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)
		assert.Equal(t, node.NID, nid)
		assert.Equal(t, node.Service, proto.ServiceBIZ)
		wg.Done()
	})

	wg.Add(1)
	err := n.Start(natsURL)
	if err != nil {
		t.Error(err)
	}

	node := discovery.Node{
		DC:      "dc",
		Service: proto.ServiceBIZ,
		NID:     nid,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     natsURL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go n.KeepAlive(node)
	wg.Wait()
	assert.NotEmpty(t, n.ServiceRegistrar())

	n.Watch(proto.ServiceBIZ)

	n.Close()
}
