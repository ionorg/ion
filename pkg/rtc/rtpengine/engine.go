package rtpengine

import (
	"net"
	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine/udp"
)

var (
	listener *udp.Listener
	stopCh   = make(chan struct{})
	wg       sync.WaitGroup
)

// Serve listen on a port and accept udp conn
func Serve(port int) chan *udp.Conn {
	log.Infof("UDP listening:%d", port)
	if listener != nil {
		listener.Close()
	}
	ch := make(chan *udp.Conn)
	var err error
	listener, err = udp.Listen("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Errorf("failed to listen %v", err)
		return nil
	}

	wg.Add(1)
	go func() {
		for {
			select {
			case <-stopCh:
				wg.Done()
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					log.Errorf("failed to accept conn %v", err)
					continue
				}
				log.Infof("accept new rtp conn %s", conn.RemoteAddr().String())
				ch <- conn
			}
		}
	}()
	return nil
}

// Close close listening loop
func Close() {
	close(stopCh)
	wg.Wait()
	listener.Close()
}
