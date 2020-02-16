package rtpengine

import (
	"net"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine/udp"
	"github.com/pion/ion/pkg/rtc/transport"
)

const (
	maxRtpConnSize = 1024
)

var (
	listener *udp.Listener
	stop     bool
)

// Serve listen on a port and accept udp conn
// func Serve(port int) chan *udp.Conn {
func Serve(port int) chan *transport.RTPTransport {
	log.Infof("UDP listening:%d", port)
	if listener != nil {
		listener.Close()
	}
	ch := make(chan *transport.RTPTransport, maxRtpConnSize)
	var err error
	listener, err = udp.Listen("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Errorf("failed to listen %v", err)
		return nil
	}

	go func() {
		for {
			if stop {
				return
			}
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("failed to accept conn %v", err)
				continue
			}
			log.Infof("accept new rtp conn %s", conn.RemoteAddr().String())

			ch <- transport.NewRTPTransport(conn)
		}
	}()
	return ch
}

// Close close listener and break loop
func Close() {
	if !stop {
		return
	}
	stop = true
	listener.Close()
}
