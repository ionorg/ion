package rtc

import (
	"fmt"
	"testing"
	"time"

	"github.com/pion/ion/pkg/rtc/rtpengine"
	"github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/rtp"
)

func TestRTPEngine(t *testing.T) {
	connCh := rtpengine.Serve(6789)
	go func() {
		for {
			select {
			case conn := <-connCh:
				fmt.Println("accept new conn from connCh", conn.RemoteAddr())
				b := make([]byte, 4000)
				go func() {
					for {
						// must read otherwise can't get new conn
						n, err := conn.Read(b)
						fmt.Println("read from conn ", n, err)
					}
				}()
			}
		}
	}()

	for i := 0; i < 3; i++ {
		rawPkt := []byte{
			0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
			0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
		}

		rtp := &rtp.Packet{}
		rtpTransport := transport.NewOutRTPTransport("1", "1", "0.0.0.0:6789")
		if err := rtp.Unmarshal(rawPkt); err == nil {
			rtpTransport.WriteRTP(rtp)
		} else {
			fmt.Println("rtpTransport.WriteRTP ", err)
		}
		time.Sleep(time.Second)
	}
}
