package rtc

import (
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine"
	"github.com/pion/ion/pkg/rtc/transport"
)

const (
	statCycle = 3 * time.Second
)

var (

	//CleanChannel return the dead pub's mid
	CleanChannel = make(chan string)
)

// Init port and ice urls
func Init(port int, ices []string) {

	//init ice
	transport.InitICE(ices)

	// show stat about all pipelines
	go Check()

	// accept relay conn
	connCh := rtpengine.Serve(port)
	go func() {
		for {
			select {
			case conn := <-connCh:
				t := transport.NewRTPTransport(conn)
				mid := t.GetMID()
				cnt := 0
				for mid == "" && cnt < 10 {
					mid = t.GetMID()
					time.Sleep(time.Millisecond)
					cnt++
				}
				if mid == "" && cnt >= 10 {
					log.Infof("mid == \"\" && cnt >=10 return")
					return
				}
				log.Infof("accept new rtp mid=%s conn=%s", mid, conn.RemoteAddr().String())
				if router := AddRouter(mid); router != nil {
					router.AddPub(mid, t)
				}
			}
		}
	}()
}
