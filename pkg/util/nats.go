package util

import (
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
)

// NewNatsConn .
func NewNatsConn(url string) (*nats.Conn, error) {
	// connect options
	opts := []nats.Option{nats.Name("nats conn")}
	opts = setupConnOptions(opts)

	// connect to nats server
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		if !nc.IsClosed() {
			log.Infof("Disconnected due to: %s, will attempt reconnects for %.0fm", err, totalWait.Minutes())
		}
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Infof("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		if !nc.IsClosed() {
			log.Errorf("Exiting: no servers available")
		} else {
			log.Errorf("Exiting")
		}
	}))
	return opts
}
