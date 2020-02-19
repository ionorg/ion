package plugins

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// Plugin some interfaces
type Plugin interface {
	ID() string
	Init(...interface{})
	PushRTP(*rtp.Packet) error
	PushRTCP(rtcp.Packet) error
	Stop()
}
