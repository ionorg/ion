package rtc

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// Transport is a interface
type Transport interface {
	ID() string
	ReadRTP() (*rtp.Packet, error)
	WriteRTP(*rtp.Packet) error
	Close()

	sendNack(*rtcp.TransportLayerNack)
	sendREMB(float64)

	writeErrTotal() int
	writeErrReset()
}
