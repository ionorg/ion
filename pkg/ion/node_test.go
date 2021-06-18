package ion

import (
	"sync"
	"testing"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-discovery/pkg/registry"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/stretchr/testify/assert"
)

var (
	natsURL = "nats://127.0.0.1:4222"
	nc      *nats.Conn
	reg     *registry.Registry
	wg      *sync.WaitGroup
	nid     = "testnid001"
)

func init() {
	log.Init("debug")
	wg = new(sync.WaitGroup)
	nc, _ = util.NewNatsConn(natsURL)
	var err error
	reg, err = registry.NewRegistry(nc, discovery.DefaultExpire)
	if err != nil {
		log.Errorf("%v", err)
	}
}

func TestWatch(t *testing.T) {
	n := NewNode(nid)

	err := reg.Listen(func(action discovery.Action, node discovery.Node) (bool, error) {
		log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)
		assert.Equal(t, node.NID, nid)
		assert.Equal(t, node.Service, proto.ServiceBIZ)
		wg.Done()
		return true, nil
	}, func(service string, params map[string]interface{}) ([]discovery.Node, error) {
		return []discovery.Node{}, nil
	})
	if err != nil {
		t.Error(err)
	}

	wg.Add(1)
	err = n.Start(natsURL)
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

	go func() {
		err := n.KeepAlive(node)
		if err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
	assert.NotEmpty(t, n.ServiceRegistrar())

	err = n.Watch(proto.ServiceBIZ)
	if err != nil {
		t.Error(err)
	}

	n.Close()
}
