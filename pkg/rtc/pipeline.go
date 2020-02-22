package rtc

import (
	"errors"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/plugins"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const (
	maxWriteErr = 100
	maxSize     = 1024
	jbPlugin    = "jitterBuffer"
	liveCycle   = 6 * time.Second
	checkCycle  = 3 * time.Second
)

var (
	errInvalidPlugin = errors.New("plugin is nil")
)

// pipeline is a rtp pipeline
// pipline has three loops, in handler and out
// |-----in-----|-----handler------|---------out---------|
//                                    +--->sub
//                                    |
// pub--->pubCh-->plugin...-->subCh---+--->sub
//                                    |
//                                    +--->sub
type pipeline struct {
	pub        Transport
	subs       map[string]Transport
	subLock    sync.RWMutex
	plugins    []plugin
	pluginLock sync.RWMutex
	pubCh      chan *rtp.Packet
	subCh      chan *rtp.Packet
	stop       bool
	pubLive    gcache.Cache
	live       bool
	rtcpCh     chan rtcp.Packet
}

func newPipeline(id string) *pipeline {
	jb := plugins.NewJitterBuffer(jbPlugin)
	p := &pipeline{
		subs:    make(map[string]Transport),
		pubCh:   make(chan *rtp.Packet, maxSize),
		subCh:   make(chan *rtp.Packet, maxSize),
		pubLive: gcache.New(maxSize).Simple().Build(),
		live:    true,
		rtcpCh:  jb.GetRTCPChan(),
	}
	p.addPlugin(jbPlugin, jb)
	p.start()
	return p
}

func (p *pipeline) check() {
	go func() {
		ticker := time.NewTicker(checkCycle)
		defer ticker.Stop()
		for range ticker.C {
			if p.stop {
				return
			}
			pub := p.getPub()
			if pub != nil {
				val, err := p.pubLive.Get(pub.ID())
				if err != nil || val == "" {
					log.Warnf("pub is not alive val=%v err=%v", val, err)
					p.live = false
				}
			}
		}
	}()
}

func (p *pipeline) in() {
	go func() {
		defer util.Recover("[pipeline.in]")
		count := uint64(0)
		for {
			if p.stop {
				return
			}
			pub := p.getPub()
			if pub == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			rtp, err := pub.ReadRTP()
			if err == nil {
				// log.Infof("rtp.Extension=%t rtp.ExtensionProfile=%x rtp.ExtensionPayload=%x", rtp.Extension, rtp.ExtensionProfile, rtp.ExtensionPayload)
				p.pubCh <- rtp
				if count%300 == 0 {
					p.pubLive.SetWithExpire(p.getPub().ID(), "live", liveCycle)
				}
				count++
			} else {
				log.Errorf("pipeline.in err=%v", err)
			}
		}
	}()
}

func (p *pipeline) handle() {
	go func() {
		defer util.Recover("[pipeline.handle]")
		count := uint64(0)
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
			if util.IsVideo(pkt.PayloadType) {
				if count%3000 == 0 {
					// Init args: (ssrc uint32, pt uint8, rembCycle int, pliCycle int)
					p.getPlugin(jbPlugin).Init(pkt.SSRC, pkt.PayloadType, 2, 1)
				}
				p.getPlugin(jbPlugin).PushRTP(pkt)
				count++
			}
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
				for _, t := range p.getSubs() {
					if t == nil {
						log.Errorf("Transport is nil")
						continue
					}

					// log.Infof("pipeline.out WriteRTP %v:%v to %v ", pkt.SSRC, pkt.SequenceNumber, t.ID())
					if err := t.WriteRTP(pkt); err != nil {
						log.Errorf("wt.WriteRTP err=%v", err)
						// del sub when err is increasing
						if t.writeErrTotal() > maxWriteErr {
							p.delSub(t.ID())
						}
					}
					t.writeErrReset()
				}
			}()
		}
	}()
}

func (p *pipeline) jitter() {
	go func() {
		defer util.Recover("[pipeline.out]")
		for {
			if p.stop {
				return
			}

			pkt := <-p.rtcpCh
			switch pkt.(type) {
			case *rtcp.TransportLayerNack, *rtcp.ReceiverEstimatedMaximumBitrate, *rtcp.PictureLossIndication:
				log.Infof("pipeline.jitter p.getPub().WriteRTCP %v", pkt)
				p.getPub().WriteRTCP(pkt)
			}
		}
	}()
}

func (p *pipeline) start() {
	p.in()
	p.out()
	p.handle()
	p.check()
	p.jitter()
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
	p.subs[id] = t
	log.Infof("pipeline.AddSub id=%s t=%p", id, t)
	return t
}

func (p *pipeline) getSub(id string) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	// log.Infof("pipeline.GetSub id=%s p.subs=%v", id, p.subs)
	return p.subs[id]
}

func (p *pipeline) getSubByAddr(addr string) Transport {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	for _, sub := range p.subs {
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
	return p.subs
}

func (p *pipeline) noSub() bool {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	isNoSub := len(p.subs) == 0
	log.Infof("pipeline.noSub %v", isNoSub)
	return isNoSub
}

func (p *pipeline) delSub(id string) {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	if p.subs[id] != nil {
		p.subs[id].Close()
	}
	delete(p.subs, id)
	log.Infof("pipeline.DelSub id=%s", id)
}

func (p *pipeline) delSubs() {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	for _, sub := range p.subs {
		if sub != nil {
			sub.Close()
		}
	}
}

func (p *pipeline) addPlugin(id string, i plugin) {
	p.pluginLock.Lock()
	defer p.pluginLock.Unlock()
	p.plugins = append(p.plugins, i)
}

func (p *pipeline) getPlugin(id string) plugin {
	p.pluginLock.RLock()
	defer p.pluginLock.RUnlock()
	for i := 0; i < len(p.plugins); i++ {
		if p.plugins[i].ID() == id {
			return p.plugins[i]
		}
	}
	return nil
}

func (p *pipeline) delPlugin(id string) {
	p.pluginLock.Lock()
	defer p.pluginLock.Unlock()
	for i := 0; i < len(p.plugins); i++ {
		if p.plugins[i].ID() == id {
			p.plugins[i].Stop()
			p.plugins = append(p.plugins[:i], p.plugins[i+1:]...)
		}
	}
}

func (p *pipeline) delPlugins() {
	p.pluginLock.Lock()
	defer p.pluginLock.Unlock()
	for _, plugin := range p.plugins {
		plugin.Stop()
	}
}

// Close release all
func (p *pipeline) Close() {
	if p.stop {
		return
	}
	p.delPub()
	p.stop = true
	p.delPlugins()
	p.delSubs()
}

func (p *pipeline) writeRTP(sid string, ssrc uint32, sn uint16) bool {
	if p.pub == nil {
		return false
	}
	hd := p.getPlugin(jbPlugin)
	if hd != nil {
		jb := hd.(*plugins.JitterBuffer)
		pkt := jb.GetPacket(ssrc, sn)
		if pkt == nil {
			// log.Infof("pipeline.writeRTP pkt not found sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
			return false
		}
		sub := p.getSub(sid)
		if sub != nil {
			sub.WriteRTP(pkt)
			// log.Infof("pipeline.writeRTP sid=%s ssrc=%d sn=%d", sid, ssrc, sn)
			return true
		}
	}
	return false
}

func (p *pipeline) IsLive() bool {
	return p.live
}

func (p *pipeline) PushRTCP(pkt rtcp.Packet) error {
	jbPlugin := p.getPlugin(jbPlugin)
	if jbPlugin == nil {
		return errInvalidPlugin
	}
	return jbPlugin.PushRTCP(pkt)
}
