package rtc

import (
	"crypto/sha1"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"net"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/udp"
	kcp "github.com/xtaci/kcp-go"
)

func serve(port int) error {
	log.Infof("UDP listening:%d", port)
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
				cnt := 0
				for pid == "" && cnt < 10 {
					pid = t.getPID()
					time.Sleep(time.Millisecond)
					cnt++
				}
				if pid == "" && cnt >= 10 {
					log.Infof("pid == \"\" && cnt >=10 return")
					return
				}
				log.Infof("accept new rtp pid=%s conn=%s", pid, conn.RemoteAddr().String())
				getOrNewPipeline(pid).addPub(pid, t)
			}()
		}
	}()
	return nil
}

func serveKcp(port int) error {
	log.Infof("[kcp] listening:%d", port)
	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	listener, err := kcp.ListenWithOptions("0.0.0.0:"+string(port), block, 10, 3)

	if err != nil {
		log.Errorf("[kcp] failed to listen %v", err)
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("[kcp] failed to accept conn %v", err)
				return
			}
			log.Infof("[kcp] accept new rtp conn %s", conn.RemoteAddr().String())
			go func() {
				t := newRTPTransport(conn)
				if t != nil {
					t.receiveRTP()
				}
				pid := t.getPID()
				cnt := 0
				for pid == "" && cnt < 10 {
					pid = t.getPID()
					time.Sleep(time.Millisecond)
					cnt++
				}
				if pid == "" && cnt >= 10 {
					log.Infof("[kcp] pid == \"\" && cnt >=10 return")
					return
				}
				log.Infof("[kcp] accept new rtp pid=%s conn=%s", pid, conn.RemoteAddr().String())
				getOrNewPipeline(pid).addPub(pid, t)
			}()
		}
	}()
	return nil

}

func addPipeline(pid string) *pipeline {
	pipeLock.Lock()
	defer pipeLock.Unlock()
	pipes[pid] = newPipeline(pid)
	return pipes[pid]
}

func getPipeline(pid string) *pipeline {
	pipeLock.RLock()
	defer pipeLock.RUnlock()
	return pipes[pid]
}

func getOrNewPipeline(pid string) *pipeline {
	p := getPipeline(pid)
	if p == nil {
		p = addPipeline(pid)
	}
	return p
}

func DelPub(pid string) {
	log.Infof("DelPub pid=%s", pid)
	p := getPipeline(pid)
	if p == nil {
		log.Infof("DelPub p=nil")
		return
	}
	p.Close()
	pipeLock.Lock()
	delete(pipes, pid)
	pipeLock.Unlock()
}

func Close() {
	pipeLock.Lock()
	for pid, pipeline := range pipes {
		if pipeline != nil {
			pipeline.Close()
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
	return p.getPub()
}

func IsWebRtcPub(pid string) bool {
	p := getPipeline(pid)
	if p != nil {
		switch p.getPub().(type) {
		case *WebRTCTransport:
			return true
		}
	}
	return false
}

func IsRtpPub(pid string) bool {
	p := getPipeline(pid)
	if p != nil {
		switch p.getPub().(type) {
		case *RTPTransport:
			return true
		}
	}
	return false
}

func GetSubs(pid string) map[string]Transport {
	p := getPipeline(pid)
	if p == nil {
		return nil
	}
	return p.getSubs()
}

func DelSubFromAllPub(sid string) map[string]string {
	m := make(map[string]string)
	pipeLock.Lock()
	defer pipeLock.Unlock()
	for pid, p := range pipes {
		p.delSub(sid)
		if p.noSub() && p.isRtpPub() {
			m[pid] = pid
		}
	}
	return m
}

func DelSub(pid, sid string) {
	p := getPipeline(pid)
	if p == nil {
		return
	}
	p.delSub(sid)
}

func AddNewRTPSub(pid, sid, addr string) {
	log.Infof("rtc.AddNewRTPSub pid=%v sid=%v addr=%v", pid, sid, addr)
	p := getOrNewPipeline(pid)
	if p.getSubByAddr(addr) == nil {
		p.addSub(sid, newPubRTPTransport(sid, pid, addr))
	}
}

func AddNewWebRTCPub(pid string) *WebRTCTransport {
	log.Infof("rtc.AddNewWebRTCPub pid=%v", pid)
	wt := newWebRTCTransport(pid)
	getOrNewPipeline(pid).addPub(pid, wt)
	return wt
}

func AddNewWebRTCSub(pid, sid string) *WebRTCTransport {
	log.Infof("rtc.AddNewWebRTCSub pid=%v sid=%v", pid, sid)
	wt := newWebRTCTransport(sid)
	getOrNewPipeline(pid).addSub(sid, wt)
	return wt
}

func Stat() {
	var info string
	info += "\n----------------pipeline-----------------\n"
	pipeLock.RLock()
	for id, pipeline := range pipes {
		info += "pub: " + id + "\n"
		subs := pipeline.getSubs()
		info += fmt.Sprintf("subs: %d\n\n", len(subs))
	}
	pipeLock.RUnlock()
	log.Infof(info)
}
