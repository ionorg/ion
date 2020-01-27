package rtc

import (
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtp"
)

const (
	maxWriteErr     = 100
	maxPipelineSize = 1024
	jitterBuffer    = "JB"
)

// pipeline is a rtp pipeline
// pipline has three loops, in handler and out
// |-----in-----|-----handler------|---------out---------|
//                                        +--->sub
//                                        |
// pub--->pubCh-->middleware...-->subCh---+--->sub
//                                        |
//                                        +--->sub
type pipeline struct {
	pub            Transport
	sub            map[string]Transport
	subLock        sync.RWMutex
	middlewares    []middleware
	middlewareLock sync.RWMutex
	pubCh          chan *rtp.Packet
	subCh          chan *rtp.Packet
	stop           bool
}

func (p *pipeline) in() {
	go func() {
		defer util.Recover("[pipeline.in]")
		for {
			if p.stop {
				return
			}

			if p.pub == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			rtp, err := p.pub.ReadRTP()
			if err == nil {
				// log.Infof("rtp.Extension=%t rtp.ExtensionProfile=%x rtp.ExtensionPayload=%x", rtp.Extension, rtp.ExtensionProfile, rtp.ExtensionPayload)
				p.pubCh <- rtp
			} else {
				log.Errorf("pipeline.in err=%v", err)
			}
		}
	}()
}

func (p *pipeline) handle() {
	go func() {
		defer util.Recover("[pipeline.handle]")
		for {
			if p.stop {
				return
			}

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
			// go p.getMiddleware(jitterBuffer).Push(pkt)
			// }
		}
	}()
}

func (p *pipeline) out() {
	go func() {
		defer util.Recover("[pipeline.out]")
		for {
			if p.stop {
				return
			}

			pkt := <-p.subCh
			log.Debugf("pkt := <-p.subCh %v", pkt)
			if pkt == nil {
				continue
			}
			// nonblock sending
			go func() {
				p.subLock.RLock()
				for _, t := range p.sub {
					if t == nil {
						log.Errorf("Transport is nil")
						continue
					}

					if err := t.WriteRTP(pkt); err != nil {
						log.Errorf("wt.WriteRTP err=%v", err)
						// del sub when err is increasing
						if t.writeErrTotal() > maxWriteErr {
							p.delSub(t.ID())
						}
					}
					t.writeErrReset()
				}
				p.subLock.RUnlock()
			}()
		}
	}()
}

func (p *pipeline) start() {
	p.in()
	p.out()
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
	log.Infof("pipeline.DelSub id=%s", id)
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

func (p *pipeline) addMiddleware(id string, m middleware) {
	p.middlewareLock.Lock()
	defer p.middlewareLock.Unlock()
	p.middlewares = append(p.middlewares, m)
}

func (p *pipeline) getMiddleware(id string) middleware {
	p.middlewareLock.RLock()
	defer p.middlewareLock.RUnlock()
	// log.Infof("getMiddleware id=%s handler=%v", id, p.middlewares)
	for i := 0; i < len(p.middlewares); i++ {
		if p.middlewares[i].ID() == id {
			// log.Infof("==id return p ")
			return p.middlewares[i]
		}
	}
	return nil
}

func (p *pipeline) delMiddleware(id string) {
	p.middlewareLock.Lock()
	defer p.middlewareLock.Unlock()
	for i := 0; i < len(p.middlewares); i++ {
		if p.middlewares[i].ID() == id {
			p.middlewares[i].Stop()
			p.middlewares = append(p.middlewares[:i], p.middlewares[i+1:]...)
		}
	}
}

func (p *pipeline) delMiddlewares() {
	p.middlewareLock.Lock()
	defer p.middlewareLock.Unlock()
	for _, handler := range p.middlewares {
		if handler != nil {
			handler.Stop()
		}
	}
}

// Close release all
func (p *pipeline) Close() {
	if p.stop {
		return
	}
	p.delPub()
	p.stop = true
	p.delSubs()
	p.delMiddlewares()
}

func (p *pipeline) writePacket(sid string, ssrc uint32, sn uint16) bool {
	if p.pub == nil {
		return false
	}
	hd := p.getMiddleware(jitterBuffer)
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
