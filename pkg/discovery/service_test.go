package discovery

import (
	"sync"
	"testing"

	log "github.com/pion/ion-log"
	"github.com/stretchr/testify/assert"
)

const (
	etcdAddr = "http://127.0.0.1:2379"
)

func init() {
	log.Init("info", []string{"asm_amd64.s", "proc.go"}, []string{})

}
func TestWatch(t *testing.T) {
	var wg sync.WaitGroup

	s, err := NewService("sfu", "dc1", []string{etcdAddr})
	assert.NoError(t, err)

	s.Watch("sfu", func(state State, id string, node *Node) {
		if state == NodeUp {
			assert.Equal(t, s.node, *node)
			wg.Done()
		} else if state == NodeDown {
			assert.Equal(t, s.node.ID(), id)
			wg.Done()
		}
	})

	wg.Add(1)
	s.KeepAlive()
	wg.Wait()

	wg.Add(1)
	s.Close()
	wg.Wait()
}

func TestGetNodes(t *testing.T) {
	var wg sync.WaitGroup

	islb, err := NewService("islb", "dc1", []string{etcdAddr})
	assert.NoError(t, err)

	biz, err := NewService("biz", "dc1", []string{etcdAddr})
	assert.NoError(t, err)

	islb.Watch("", func(state State, id string, node *Node) {
		if state == NodeUp {
			wg.Done()
		} else if state == NodeDown {
			wg.Done()
		}
	})

	wg.Add(2)
	biz.KeepAlive()
	islb.KeepAlive()
	wg.Wait()

	nodes := make(map[string]Node)
	err = islb.GetNodes("", nodes)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, biz.node, nodes[biz.node.ID()])
	assert.Equal(t, islb.node, nodes[islb.node.ID()])

	wg.Add(2)
	biz.Close()
	islb.Close()
	wg.Wait()
}
