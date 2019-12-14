package rtc

import (
	"strings"
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtp"
)

const (
	maxErrCnt = 100
)

type Handler interface {
	ID() string
	Push(*rtp.Packet) error
	Stop()
}

type pipeline struct {
	pub                               Transport
	sub                               map[string]Transport
	subLock                           sync.RWMutex
	handler                           []Handler
	handlerLock                       sync.RWMutex
	pubCh                             chan *rtp.Packet
	subCh                             chan *rtp.Packet
	stopInCh, stopHandleCh, stopOutCh chan struct{}
	wg                                sync.WaitGroup
}

func newPipeline(id string) *pipeline {
	p := &pipeline{
		sub:          make(map[string]Transport),
		pubCh:        make(chan *rtp.Packet, maxPipelineSize),
		subCh:        make(chan *rtp.Packet, maxPipelineSize),
		stopInCh:     make(chan struct{}),
		stopHandleCh: make(chan struct{}),
		stopOutCh:    make(chan struct{}),
	}
	p.addHandler(jitterBuffer, newBuffer(jitterBuffer, p))
	p.start()
	return p
}

func (p *pipeline) in() {
	go func() {
		defer util.Recover("[pipeline.in]")
		for {
			select {
			case <-p.stopInCh:
				log.Debugf("pipeline.in stop ok!")
				p.wg.Done()
				return
			default:
				if p.pub == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				if rtp, _ := p.pub.ReadRTP(); rtp != nil {
					p.pubCh <- rtp
				}
			}
		}
	}()
}

func (p *pipeline) handle() {
	go func() {
		defer util.Recover("[pipeline.handle]")
		for {
			select {
			case <-p.stopHandleCh:
				p.wg.Done()
				return
			default:
				pkt := <-p.pubCh
				log.Debugf("pkt := <-p.pubCh %v", pkt)
				p.subCh <- pkt
				log.Debugf("p.subCh <- pkt %v", pkt)
				if pkt == nil {
					continue
				}
				//only buffer video
				// if pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 ||
				// pkt.PayloadType == webrtc.DefaultPayloadTypeVP9 ||
				// pkt.PayloadType == webrtc.DefaultPayloadTypeH264 {
				// go p.getHandler(jitterBuffer).Push(pkt)
				// }
			}
		}
	}()
}

func (p *pipeline) out() {
	go func() {
		defer util.Recover("[pipeline.out]")
		for {
			select {
			case <-p.stopOutCh:
				p.wg.Done()
				return
			default:
				pkt := <-p.subCh
				log.Debugf("pkt := <-p.subCh %v", pkt)
				if pkt == nil {
					continue
				}
				go func() {
					p.subLock.RLock()
					if len(p.sub) == 0 {
						p.subLock.RUnlock()
						return
					}
					for _, t := range p.sub {
						if t == nil {
							log.Errorf("Transport is nil")
						}
						switch t.(type) {
						case *WebRTCTransport:
							wt := t.(*WebRTCTransport)
							if err := wt.WriteRTP(pkt); err != nil {
								log.Errorf("wt.WriteRTP err=%v", err)
								if wt.errCnt() > maxErrCnt {
									p.delSub(t.ID())
								}
								wt.addErrCnt()
							}
							wt.clearErrCnt()
						case *RTPTransport:
							rt := t.(*RTPTransport)
							if err := rt.WriteRTP(pkt); err != nil {
								log.Errorf("rt.WriteRTP err=%v", err)
								rt.ResetExtSent()
								if rt.errCnt() > maxErrCnt {
									p.delSub(t.ID())
								}
								rt.addErrCnt()
							}
							rt.clearErrCnt()
							// log.Debugf("send RTP: %v", pkt)
						}
					}
					p.subLock.RUnlock()
				}()
			}
		}
	}()
}

func (p *pipeline) start() {
	p.wg.Add(1)
	p.in()
	p.wg.Add(1)
	p.out()
	p.wg.Add(1)
	p.handle()
}

func (p *pipeline) addPub(id string, t Transport) Transport {
	p.pub = t
	return t
}

func (p *pipeline) isRtpPub() bool {
	if p.pub != nil {
		switch p.pub.(type) {
		case *RTPTransport:
			return true
		}
	}
	return false
}

func (p *pipeline) delPub() {
	// first close pub
	if p.pub != nil {
		p.pub.Close()
	}
}

func (p *pipeline) getPub() Transport {
	return p.pub
}

func (p *pipeline) addSub(id string, t Transport) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	p.sub[id] = t
	log.Infof("pipeline.AddSub id=%s t=%p", id, t)
	return t
}

func (p *pipeline) getSub(id string) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	// log.Infof("pipeline.GetSub id=%s p.sub[id]=%p", id, p.sub[id])
	return p.sub[id]
}

func (p *pipeline) getSubByAddr(addr string) Transport {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	for _, sub := range p.sub {
		switch sub.(type) {
		case *RTPTransport:
			rt := sub.(*RTPTransport)
			if rt.getAddr() == addr {
				return rt
			}
		}
	}
	return nil
}

func (p *pipeline) getSubs() map[string]Transport {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	log.Infof("pipeline.GetSubs p.sub=%v", p.sub)
	return p.sub
}

func (p *pipeline) noSub() bool {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	isNoSub := len(p.sub) == 0
	log.Infof("pipeline.noSub %v", isNoSub)
	return isNoSub
}

func (p *pipeline) delSub(id string) {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	if p.sub[id] != nil {
		p.sub[id].Close()
	}
	delete(p.sub, id)
	log.Infof("pipeline.delSub id=%s", id)
}

func (p *pipeline) delSubByPrefix(id string) {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	for k, sub := range p.sub {
		if strings.Contains(k, id) {
			if sub != nil {
				sub.Close()
				delete(p.sub, k)
				log.Infof("pipeline.delSubByPrefix id=%s", id)
			}
		}
	}
}

func (p *pipeline) delSubs() {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	for _, sub := range p.sub {
		if sub != nil {
			sub.Close()
		}
	}
	p.sub = make(map[string]Transport)
}

func (p *pipeline) addHandler(id string, t Handler) {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	p.handler = append(p.handler, t)
}

func (p *pipeline) getHandler(id string) Handler {
	p.handlerLock.RLock()
	defer p.handlerLock.RUnlock()
	// log.Infof("getHandler id=%s handler=%v", id, p.handler)
	for i := 0; i < len(p.handler); i++ {
		if p.handler[i].ID() == id {
			// log.Infof("==id return p ")
			return p.handler[i]
		}
	}
	return nil
}

func (p *pipeline) delHandler(id string) {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	for i := 0; i < len(p.handler); i++ {
		if p.handler[i].ID() == id {
			p.handler[i].Stop()
			p.handler = append(p.handler[:i], p.handler[i+1:]...)
		}
	}
}

func (p *pipeline) delHandlers() {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	for _, handler := range p.handler {
		if handler != nil {
			handler.Stop()
		}
	}
}

func (p *pipeline) Close() {
	// for ReadRTP not block
	p.delPub()
	close(p.stopInCh)
	close(p.stopHandleCh)
	close(p.stopOutCh)
	close(p.pubCh)
	p.wg.Wait()
	p.delSubs()
	p.delHandlers()
	close(p.subCh)
}

func (p *pipeline) writePacket(sid string, ssrc uint32, sn uint16) bool {
	if p.pub == nil {
		return false
	}
	hd := p.getHandler(jitterBuffer)
	if hd != nil {
		jb := hd.(*buffer)
		pkt := jb.GetPacket(ssrc, sn)
		if pkt == nil {
			log.Debugf("pipeline.writePacket pkt not found sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
			return false
		}
		p.getSub(sid).WriteRTP(pkt)
		log.Infof("pipeline.writePacket sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
		log.Debugf("pipeline.writePacket ok")
		return true
	}
	return false
}
