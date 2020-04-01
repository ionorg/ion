package processor

import "github.com/pion/rtp"

// RTPWriter rtp writer interface
type RTPWriter interface {
	WriteRTP(packet *rtp.Packet) error
	Close() error
}

// Processor interface
type Processor struct {
	ID          string
	AudioWriter RTPWriter
	VideoWriter RTPWriter
}
