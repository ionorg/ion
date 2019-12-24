package rtc

import "github.com/pion/rtp"

type middleware interface {
	ID() string
	Push(*rtp.Packet) error
	Stop()
}
