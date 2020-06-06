package rtpengine

import (
	"crypto/sha1"
	"net"

	"fmt"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine/udp"
	"github.com/pion/ion/pkg/rtc/transport"
	kcp "github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

const (
	maxRtpConnSize = 1024
)

var (
	listener    net.Listener
	kcpListener *kcp.Listener
	stop        bool
)

// Serve listen on a port and accept udp conn
// func Serve(port int) chan *udp.Conn {
func Serve(port int) (chan *transport.RTPTransport, error) {
	log.Infof("rtpengine.Serve port=%d ", port)
	if listener != nil {
		listener.Close()
	}
	ch := make(chan *transport.RTPTransport, maxRtpConnSize)
	var err error
	listener, err = udp.Listen("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Errorf("failed to listen %v", err)
		return nil, err
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
	return ch, nil
}

// ServeWithKCP accept kcp conn
func ServeWithKCP(port int, kcpPwd, kcpSalt string) (chan *transport.RTPTransport, error) {
	log.Infof("kcp Serve port=%d", port)
	if kcpListener != nil {
		kcpListener.Close()
	}
	ch := make(chan *transport.RTPTransport, maxRtpConnSize)
	var err error
	key := pbkdf2.Key([]byte(kcpPwd), []byte(kcpSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	kcpListener, err = kcp.ListenWithOptions(fmt.Sprintf("0.0.0.0:%d", port), block, 10, 3)
	if err != nil {
		log.Errorf("kcp Listen err=%v", err)
		return nil, err
	}

	go func() {
		for {
			if stop {
				return
			}
			conn, err := kcpListener.AcceptKCP()
			if err != nil {
				log.Errorf("failed to accept conn %v", err)
				continue
			}
			log.Infof("accept new kcp conn %s", conn.RemoteAddr().String())

			ch <- transport.NewRTPTransport(conn)
		}
	}()
	return ch, nil
}

// Close close listener and break loop
func Close() {
	if !stop {
		return
	}
	stop = true
	listener.Close()
	kcpListener.Close()
}
