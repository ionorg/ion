package rtc

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtp"
)

var (
	pipes    = make(map[string]*pipeline)
	pipeLock sync.RWMutex
)

// DelPub delete pub
func DelPub(mid string) {
	log.Infof("DelPub mid=%s", mid)
	p := getPipeline(mid)
	if p == nil {
		log.Infof("DelPub p=nil")
		return
	}
	p.Close()
	pipeLock.Lock()
	defer pipeLock.Unlock()
	delete(pipes, mid)
}

// Close close all pipeline
func Close() {
	pipeLock.Lock()
	for mid, pipeline := range pipes {
		if pipeline != nil {
			pipeline.Close()
			delete(pipes, mid)
		}
	}
	pipeLock.Unlock()
}

// GetPub get pub
func GetPub(mid string) Transport {
	p := getPipeline(mid)
	if p == nil {
		return nil
	}
	return p.getPub()
}

// GetWebRtcMIDByPID ..
func GetWebRtcMIDByPID(id string) []string {
	m := getPipelinesByPrefix(id)
	var mids []string

	//find webrtc pub mid
	for mid, p := range m {
		switch p.getPub().(type) {
		case *WebRTCTransport:
		default:
			mids = append(mids, mid)
		}
	}
	return mids
}

// GetSubs get sub by mid
func GetSubs(mid string) map[string]Transport {
	p := getPipeline(mid)
	if p == nil {
		return nil
	}
	return p.getSubs()
}

// DelSubFromAllPub del all sub by id
func DelSubFromAllPub(id string) map[string]string {
	log.Infof("DelSubFromAllPub id=%v", id)
	m := make(map[string]string)
	pipeLock.Lock()
	defer pipeLock.Unlock()
	for mid, p := range pipes {
		p.delSub(id)
		if p.noSub() && p.isRtpPub() {
			m[mid] = mid
		}
	}
	return m
}

// DelSub del sub
func DelSub(mid, id string) {
	p := getPipeline(mid)
	if p == nil {
		return
	}
	p.delSub(id)
}

// NewWebRTCTransport new a webrtc transport
func NewWebRTCTransport(mid, id string, isPub bool) *WebRTCTransport {
	log.Infof("rtc.NewWebRTCTransport mid=%v id=%v isPub=%v", mid, id, isPub)
	p := getPipeline(mid)
	if p == nil {
		p = newPipeline(mid)
	}
	wt := newWebRTCTransport(mid)
	if isPub {
		p.addPub(mid, wt)
	} else {
		p.addSub(id, wt)
	}
	return wt
}

// NewRTPTransportSub new a rtp transport suber
func NewRTPTransportSub(mid, sid, addr string) {
	log.Infof("rtc.NewRTPTransport mid=%v sid=%v addr=%v", mid, sid, addr)
	p := getPipeline(mid)
	if p == nil {
		p = newPipeline(mid)
	}
	if p.getSubByAddr(addr) == nil {
		p.addSub(sid, newPubRTPTransport(sid, mid, addr))
	}
}

// Stat show all pipelines' stat
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

// DelSubFromAllPubByPrefix del sub from all pipelines by prefix
func DelSubFromAllPubByPrefix(id string) map[string]string {
	log.Infof("DelSubFromAllPubByPrefix id=%s", id)
	m := make(map[string]string)
	pipeLock.Lock()
	defer pipeLock.Unlock()
	for mid, p := range pipes {
		p.delSub(id)
		if p.noSub() && p.isRtpPub() {
			m[mid] = mid
		}
	}
	return m
}

func getPipelinesByPrefix(id string) map[string]*pipeline {
	pipeLock.RLock()
	defer pipeLock.RUnlock()
	m := make(map[string]*pipeline)
	for mid := range pipes {
		if strings.Contains(mid, id) {
			m[mid] = pipes[mid]
		}
	}
	return m
}

func getPipeline(mid string) *pipeline {
	// log.Infof("getPipeline mid=%v", mid)
	pipeLock.RLock()
	defer pipeLock.RUnlock()
	return pipes[mid]
}

func newPipeline(id string) *pipeline {
	p := &pipeline{
		sub:   make(map[string]Transport),
		pubCh: make(chan *rtp.Packet, maxPipelineSize),
		subCh: make(chan *rtp.Packet, maxPipelineSize),
	}
	p.addMiddleware(jitterBuffer, newBuffer(jitterBuffer, p))
	p.start()
	pipeLock.Lock()
	defer pipeLock.Unlock()
	pipes[id] = p
	return p
}
