package rtc

import (
	"fmt"
	"testing"
	"time"

	"github.com/pion/ion/pkg/rtc/rtpengine"
)

func TestEngine(t *testing.T) {
	connCh := rtpengine.Serve(6789)
	go func() {
		for {
			select {
			case conn := <-connCh:
				fmt.Println("conn from connCh", conn.RemoteAddr())
				b := make([]byte, 4000)
				go func() {
					for {
						n, err := conn.Read(b)
						fmt.Println("conn read", n, err)
					}
				}()
			}
		}
	}()
	time.Sleep(time.Second)
	select {}
}
