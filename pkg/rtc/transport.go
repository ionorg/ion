package rtc

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type Transport interface {
	ID() string
	ReadRTP() (*rtp.Packet, error)
	WriteRTP(*rtp.Packet) error
	sendNack(*rtcp.TransportLayerNack)
	sendREMB(float64)
	Close()
}
