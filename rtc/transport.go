package rtc

import "github.com/pion/rtp"

type Transport interface {
	ID() string
	ReadRTP() (*rtp.Packet, error)
	WriteRTP(*rtp.Packet) error
	sendPLI()
	Close()
}
