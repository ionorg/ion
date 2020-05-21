package plugins

import (
	"github.com/pion/rtp"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/transport"
)

// RTPForwarderConfig .
type RTPForwarderConfig struct {
	ID      string
	MID     string
	On      bool
	Addr    string
	KcpKey  string
	KcpSalt string
}

// RTPForwarder core
type RTPForwarder struct {
	id         string
	stop       bool
	Transport  *transport.RTPTransport
	outRTPChan chan *rtp.Packet
}

// NewRTPForwarder Create new RTP Forwarder
func NewRTPForwarder(config RTPForwarderConfig) *RTPForwarder {
	log.Infof("New RTPForwarder Plugin with id %s address %s for mid %s", config.ID, config.Addr, config.MID)
	var rtpTransport *transport.RTPTransport

	if config.KcpKey != "" && config.KcpSalt != "" {
		rtpTransport = transport.NewOutRTPTransportWithKCP(config.MID, config.Addr, config.KcpKey, config.KcpSalt)
	} else {
		rtpTransport = transport.NewOutRTPTransport(config.MID, config.Addr)
	}

	return &RTPForwarder{
		id:         config.ID,
		Transport:  rtpTransport,
		outRTPChan: make(chan *rtp.Packet, maxSize),
	}
}

// ID Return RTPForwarder ID
func (r *RTPForwarder) ID() string {
	return r.id
}

// WriteRTP Forward rtp packet which from pub
func (r *RTPForwarder) WriteRTP(pkt *rtp.Packet) error {
	if r.stop {
		return nil
	}

	r.outRTPChan <- pkt
	go func() {
		err := r.Transport.WriteRTP(pkt)
		if err != nil {
			log.Errorf("r.Transport.WriteRTP => %s", err)
		}
	}()
	return nil
}

// ReadRTP Forward rtp packet which from pub
func (r *RTPForwarder) ReadRTP() <-chan *rtp.Packet {
	return r.outRTPChan
}

// Stop Stop plugin
func (r *RTPForwarder) Stop() {
	r.stop = true
	r.Transport.Close()
}
