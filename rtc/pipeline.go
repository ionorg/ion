package rtc

import (
	"errors"
	"sync"
	"time"

	"github.com/pion/ion/log"
	"github.com/pion/ion/rtc/samplebuilder"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
)

const (
	maxPipelineSize = 1024

	vp8ClockRate  = 90000
	H264ClockRate = 90000
	// The amount of RTP packets it takes to hold one full video frame
	// The MTU of ~1400 meant that one video buffer had to be split across 7 packets
	sampleSize = 10
)

var (
	payloaders = make(map[uint8]rtp.Payloader)
	clockRates = make(map[uint8]uint32)
)

func init() {
	payloaders[webrtc.DefaultPayloadTypeVP8] = &codecs.VP8Payloader{}
	payloaders[webrtc.DefaultPayloadTypeH264] = &codecs.H264Payloader{}
	payloaders[webrtc.DefaultPayloadTypeOpus] = &codecs.OpusPayloader{}
	// TODO VP9

	clockRates[webrtc.DefaultPayloadTypeVP8] = 90000
	clockRates[webrtc.DefaultPayloadTypeH264] = 90000
	clockRates[webrtc.DefaultPayloadTypeOpus] = 8000
	// TODO VP9
}

type Handler interface {
	ID() string
	Push(*rtp.Packet) error
	Pop() map[uint8][]*rtp.Packet
	Stop()
}

type buffer struct {
	id      string
	sbs     map[uint8]*samplebuilder.SampleBuilder
	sbsLock sync.RWMutex
	gapSeq  chan uint16
	notify  chan struct{}
	// keyFrame     []*rtp.Packet
	// keyFrameLock sync.RWMutex
}

func newBuffer(id string) *buffer {
	r := &buffer{
		sbs:    make(map[uint8]*samplebuilder.SampleBuilder),
		gapSeq: make(chan uint16, 1000),
		notify: make(chan struct{}),
		id:     id,
	}
	r.readGapSeq()
	return r
}

func (b *buffer) ID() string {
	return b.id
}

func (b *buffer) Push(pkt *rtp.Packet) error {
	pt := pkt.PayloadType
	b.sbsLock.RLock()
	sb := b.sbs[pt]
	b.sbsLock.RUnlock()
	if sb == nil {
		b.sbsLock.Lock()
		switch pt {
		case webrtc.DefaultPayloadTypeVP8:
			b.sbs[pt] = samplebuilder.New(sampleSize, &codecs.VP8Packet{})
		case webrtc.DefaultPayloadTypeOpus:
			b.sbs[pt] = samplebuilder.New(sampleSize, &codecs.OpusPacket{})
		case webrtc.DefaultPayloadTypeH264:
			// TODO
		case webrtc.DefaultPayloadTypeVP9:
			// TODO
		default:
			log.Errorf("unknown PayloadType %d", pt)
		}
		sb = b.sbs[pt]
		b.sbsLock.Unlock()
	}
	if sb == nil {
		return errors.New("samplebuilder is nil")
	}

	//samplebuilder will send gap sequence to channel
	sb.Push(pkt, b.gapSeq)
	return nil
}

func (b *buffer) readGapSeq() {
	go func() {
		for {
			select {
			case gapSeq := <-b.gapSeq:
				log.Infof("GOP.ReadGapSeq %d", gapSeq)
			case <-b.notify:
				return
			}
		}
	}()
}

func (b *buffer) Pop() map[uint8][]*rtp.Packet {
	m := make(map[uint8][]*rtp.Packet)

	b.sbsLock.RLock()
	sbs := b.sbs
	b.sbsLock.RUnlock()

	for pt, sb := range sbs {
		for rtps := sb.PopPackets(); len(rtps) != 0; rtps = sb.PopPackets() {
			// log.Infof(" out pop rtps=%v", rtps)
			// for _, v := range rtps {
			// log.Infof("pop rtp=%v", v.SequenceNumber)
			// }

			// if pt == webrtc.DefaultPayloadTypeVP8 && len(rtps) > 0 {
			// vp8 := &codecs.VP8Packet{}
			// vp8.Unmarshal(rtps[0].Payload)
			// // start of a frame, there is a payload header  when S == 1
			// if vp8.S == 1 && vp8.Payload[0]&0x01 == 0 {
			// //key frame
			// log.Infof("vp8.Payload[0]=%b  keyframe=%d rtps=%v", vp8.Payload[0], vp8.Payload[0]&0x01, rtps)
			// b.keyFrameLock.Lock()
			// b.keyFrame = rtps
			// b.keyFrameLock.Unlock()
			// }
			// }

			for _, rtp := range rtps {
				m[pt] = append(m[pt], rtp)
			}
		}
	}
	return m
}

// func (b *buffer) GetKeyFrame() []*rtp.Packet {
// b.keyFrameLock.RLock()
// defer b.keyFrameLock.RUnlock()
// return b.keyFrame
// }

func (b *buffer) Stop() {
	close(b.gapSeq)
	close(b.notify)
}

type Pipeline struct {
	pub         Transport
	sub         map[string]Transport
	subLock     sync.RWMutex
	handler     []Handler
	handlerLock sync.RWMutex
	pubCh       chan *rtp.Packet
	subCh       chan *rtp.Packet
}

func newPipeline(id string) *Pipeline {
	p := &Pipeline{
		sub:   make(map[string]Transport),
		pubCh: make(chan *rtp.Packet, maxPipelineSize),
		subCh: make(chan *rtp.Packet, maxPipelineSize),
	}
	p.handlerLock.Lock()
	p.handler = append(p.handler, newBuffer(id))
	p.handlerLock.Unlock()
	p.start()
	return p
}

func (p *Pipeline) in() {
	go func() {
		for {
			if p.pub == nil {
				time.Sleep(time.Millisecond)
				continue
			}

			switch p.pub.(type) {
			case *WebRTCTransport:
				wt := p.pub.(*WebRTCTransport)
				if rtp, _ := wt.ReadRTP(); rtp != nil {
					p.pubCh <- rtp
				}
			case *RTPTransport:
				rt := p.pub.(*RTPTransport)
				if rtp, _ := rt.ReadRTP(); rtp != nil {
					// log.Infof("in rtp=%v", rtp)
					p.pubCh <- rtp
				}
			}
		}
	}()
}

func (p *Pipeline) isVideo(pkt *rtp.Packet) bool {
	if pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 ||
		pkt.PayloadType == webrtc.DefaultPayloadTypeVP9 ||
		pkt.PayloadType == webrtc.DefaultPayloadTypeH264 {
		return true
	}
	return false
}

func (p *Pipeline) handle() {

	//without buffer
	go func() {
		for {
			pkt := <-p.pubCh
			p.subCh <- pkt
		}
	}()

	//with buffer
	// go func() {
	// p.handlerLock.RLock()
	// gop := p.handler[0]
	// p.handlerLock.RUnlock()
	// for {
	// pkt := <-p.pubCh
	// gop.Push(pkt)
	// pkts := gop.Pop()
	// if len(pkts) == 0 {
	// continue
	// }
	// for _, rtps := range pkts {
	// for i := 0; i < len(rtps); i++ {
	// p.subCh <- rtps[i]
	// }
	// }
	// }
	// }()

	// TODO more handler

}

func (p *Pipeline) out() {
	go func() {
		for {
			pkt := <-p.subCh
			p.subLock.RLock()
			if len(p.sub) == 0 {
				p.subLock.RUnlock()
				time.Sleep(time.Millisecond * 10)
				continue
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
					}
				case *RTPTransport:
					rt := t.(*RTPTransport)
					if err := rt.WriteRTP(pkt); err != nil {
						log.Errorf("rt.WriteRTP err=%v", err)
						rt.ResetExtSent()
					}

					log.Debugf("send RTP: %v", pkt)
				}
			}
			p.subLock.RUnlock()
		}
	}()
}

func (p *Pipeline) start() {
	p.in()
	p.out()
	p.handle()
}

func (p *Pipeline) AddPub(id string, t Transport) Transport {
	p.pub = t
	return t
}

func (p *Pipeline) DelPub() {
	log.Infof("Pipeline.DelPub")
	// first close pub
	if p.pub != nil {
		p.pub.Close()
	}
	log.Infof("Pipeline.DelPub p.pub=%v", p.pub)
}

func (p *Pipeline) GetPub() Transport {
	log.Infof("Pipeline.GetPub %v", p.pub)
	return p.pub
}

func (p *Pipeline) AddSub(id string, t Transport) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	p.sub[id] = t
	log.Infof("Pipeline.AddSub id=%s t=%p", id, t)
	return t
}

func (p *Pipeline) GetSub(id string) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	log.Infof("Pipeline.GetSub id=%s p.sub[id]=%p", id, p.sub[id])
	return p.sub[id]
}

func (p *Pipeline) getSubByAddr(addr string) Transport {
	p.subLock.Lock()
	defer p.subLock.Unlock()
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
	// log.Infof("Pipeline.GetSub id=%s p.sub[id]=%p", id, p.sub[id])
	// return p.sub[id]
}

func (p *Pipeline) GetSubs() map[string]Transport {
	p.subLock.RLock()
	defer p.subLock.RUnlock()
	log.Infof("Pipeline.GetSubs p.sub=%v", p.sub)
	return p.sub
}

func (p *Pipeline) DelSub(id string) {
	p.subLock.Lock()
	defer p.subLock.Unlock()
	if p.sub[id] != nil {
		p.sub[id].Close()
	}
	delete(p.sub, id)
	log.Infof("Pipeline.DelSub id=%s", id)
}

func (p *Pipeline) AddHandler(id string, t Handler) {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	p.handler = append(p.handler, t)
}

func (p *Pipeline) GetHandler(id string) Handler {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	for i := 0; i < len(p.handler); i++ {
		if p.handler[i].ID() == id {
			return p.handler[i]
		}
	}
	return nil
}

func (p *Pipeline) DelHandler(id string) {
	p.handlerLock.Lock()
	defer p.handlerLock.Unlock()
	for i := 0; i < len(p.handler); i++ {
		if p.handler[i].ID() == id {
			p.handler[i].Stop()
			p.handler = append(p.handler[:i], p.handler[i+1:]...)
		}
	}
}

func (p *Pipeline) Stop() {
	if p.pub != nil {
		p.pub.Close()
	}
	p.subLock.Lock()
	for _, sub := range p.sub {
		if sub != nil {
			sub.Close()
		}
	}
	p.subLock.Unlock()
	p.handlerLock.Lock()
	for _, handler := range p.handler {
		if handler != nil {
			handler.Stop()
		}
	}
	p.handlerLock.Unlock()
}

func (p *Pipeline) SendPLI() {
	if p.pub != nil {
		p.pub.sendPLI()
	}
}

//TODO don't use this func, because timestamp and sequence need to be modify otherwise it's ts and sn is invalid for client
// func (p *Pipeline) SendKeyFrame(sid string) {
// p.handlerLock.RLock()
// buffer := p.handler[0].(*buffer)
// p.handlerLock.RUnlock()

// p.subLock.RLock()
// for id, t := range p.sub {
// if t == nil {
// continue
// }
// if id == sid {
// for _, rtp := range buffer.GetKeyFrame() {
// switch t.(type) {
// case *WebRTCTransport:
// wt := t.(*WebRTCTransport)
// if err := wt.WriteRTP(rtp); err != nil {
// log.Errorf("wt.WriteRTP err=%v", err)
// }
// case *RTPTransport:
// rt := t.(*RTPTransport)
// log.Infof("SendKeyFrame rtp=%v", rtp)
// if err := rt.WriteRTP(rtp); err != nil {
// log.Errorf(err.Error())
// }
// }
// }
// }
// }
// p.subLock.RUnlock()
// }
