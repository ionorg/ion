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
	wg := new(sync.WaitGroup)

	wg.Add(1)

	s, err := NewService("sfu", "dc1", []string{etcdAddr})
	assert.NoError(t, err)

	s.Watch("sfu", func(state State, node Node) {
		if state == NodeUp {
			assert.Equal(t, s.node, node)
			wg.Done()
		} else if state == NodeDown {
			assert.Equal(t, s.node, node)
			wg.Done()
		}
	})

	s.KeepAlive()
	wg.Wait()

	wg.Add(1)
	s.Close()
	wg.Wait()
}
