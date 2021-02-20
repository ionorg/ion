package ion

import (
	"testing"

	"github.com/tj/assert"
)

var (
	natURL = "nats://127.0.0.1:4222"
)

func TestStart(t *testing.T) {
	n := &Node{
		NID: "node-01",
	}

	err := n.Start(natURL)
	if err != nil {
		t.Error(err)
	}

	assert.NotEmpty(t, n.ServiceRegistrar())
	n.Close()
}
