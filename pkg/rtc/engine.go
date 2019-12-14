package rtc

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/udp"
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

func getPipelinesByPrefix(pid string) map[string]*pipeline {
	pipeLock.RLock()
	defer pipeLock.RUnlock()
	m := make(map[string]*pipeline)
	for mid := range pipes {
		if strings.Contains(mid, pid) {
			m[mid] = pipes[mid]
		}
	}
	return m
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
	defer pipeLock.Unlock()
	delete(pipes, pid)
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

func GetWebRtcMIDByPID(pid string) []string {
	m := getPipelinesByPrefix(pid)
	var mids []string

	//find webrtc pub mid
	for mid, p := range m {
		switch p.getPub().(type) {
		case *WebRTCTransport:
			mids = append(mids, mid)
		default:
		}
	}
	return mids
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

func DelSubFromAllPubByPrefix(sid string) map[string]string {
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
