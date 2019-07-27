package rtc

import (
	"net"
	"sync"

	"github.com/pion/ion/conf"
	"github.com/pion/ion/log"
	"github.com/pion/ion/rtc/udp"
)

var (
	listener *udp.Listener
	pipes    map[string]*Pipeline
	pipeLock sync.RWMutex
)

func init() {
	pipes = make(map[string]*Pipeline)
	serve(conf.Rtp.Port)
}

func serve(port int) error {
	log.Infof("udp listening:%d", port)
	if listener != nil {
		listener.Close()
	}
	var err error
	listener, err = udp.Listen("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		log.Errorf("failed to listen %v", err)
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("failed to accept conn %v", err)
				return
			}
			log.Infof("accept new rtp conn %s", conn.RemoteAddr().String())
			go func() {
				t := newRTPTransport(conn)
				if t != nil {
					t.receiveRTP()
				}
				pid := t.getPID()
				for ; pid == ""; pid = t.getPID() {

				}
				log.Infof("accept new rtp %s len=%d", pid, len(pid))
				getOrNewPipeline(pid).AddPub(pid, t)
			}()
		}
	}()
	return nil
}

func addPipeline(pid string) *Pipeline {
	pipeLock.Lock()
	defer pipeLock.Unlock()
	pipes[pid] = newPipeline(pid)
	return pipes[pid]
}

func getPipeline(pid string) *Pipeline {
	log.Infof("getPipeline pid=%s len=%d pipes=%v", pid, len(pid), pipes)
	pipeLock.RLock()
	defer pipeLock.RUnlock()
	return pipes[pid]
}

func getOrNewPipeline(pid string) *Pipeline {
	p := getPipeline(pid)
	if p == nil {
		p = addPipeline(pid)
	}
	return p
}

func closePipeline(pid string) {
	pipeLock.RLock()
	p := pipes[pid]
	pipeLock.RUnlock()
	if p == nil {
		return
	}
	p.Stop()
	pipeLock.Lock()
	delete(pipes, pid)
	pipeLock.Unlock()
}

func Close() {
	pipeLock.Lock()
	for pid, pipeline := range pipes {
		if pipeline != nil {
			pipeline.Stop()
			delete(pipes, pid)
		}
	}
	pipeLock.Unlock()
	listener.Close()
}

func GetPub(pid string) Transport {
	p := getPipeline(pid)
	if p == nil {
		return nil
	}
	return p.GetPub()
}

func GetSubs(pid string) map[string]Transport {
	p := getPipeline(pid)
	if p == nil {
		return nil
	}
	return p.GetSubs()
}

func DelPub(pid string) {
	p := getPipeline(pid)
	if p == nil {
		return
	}
	p.DelPub()
}

func DelSub(pid, sid string) {
	p := getPipeline(pid)
	if p == nil {
		return
	}
	p.DelSub(sid)
}

func AddNewRTPSub(pid, sid, addr string) {
	log.Infof("AddNewRTPSub pid=%v sid=%v addr=%v", pid, sid, addr)
	p := getOrNewPipeline(pid)
	if p.getSubByAddr(addr) == nil {
		p.AddSub(sid, newPubRTPTransport(sid, pid, addr))
	}
}

func AddNewWebRTCPub(pid string) *WebRTCTransport {
	log.Infof("AddNewWebRTCPub pid=%v", pid)
	wt := newWebRTCTransport(pid)
	getOrNewPipeline(pid).AddPub(pid, wt).sendPLI()
	return wt
}

func AddNewWebRTCSub(pid, sid string) *WebRTCTransport {
	log.Infof("AddNewWebRTCSub pid=%v sid=%v", pid, sid)
	wt := newWebRTCTransport(sid)
	getOrNewPipeline(pid).AddSub(sid, wt)
	return wt
}
