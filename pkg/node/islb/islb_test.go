package islb

import (
	"sync"
	"testing"

	"github.com/pion/ion/pkg/db"
)

var (
	conf = Config{
		Nats: natsConf{
			URL: "nats://127.0.0.1:4222",
		},
		Redis: db.Config{
			DB:    0,
			Pwd:   "",
			Addrs: []string{":6379"},
		},
	}
	file string
)

func TestWatch(t *testing.T) {
	var wg sync.WaitGroup
	i := NewISLB()
	wg.Add(1)
	i.Start(conf)
	wg.Wait()
}
