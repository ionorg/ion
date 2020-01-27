package rtc

import (
	"errors"
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine/packer"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	pktSize = 100
)

type buffer struct {
	pipeline   *pipeline
	id         string
	packers    map[uint32]*packer.Packer
	packerLock sync.RWMutex
	nackCh     chan *rtcp.TransportLayerNack
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func newBuffer(id string, p *pipeline) *buffer {
	r := &buffer{
		packers:  make(map[uint32]*packer.Packer),
		stopCh:   make(chan struct{}),
		id:       id,
		pipeline: p,
		nackCh:   make(chan *rtcp.TransportLayerNack, 100),
	}
	r.wg.Add(1)
	r.jitter()
	return r
}

func (b *buffer) ID() string {
	return b.id
}

func (b *buffer) Push(pkt *rtp.Packet) error {
	b.packerLock.RLock()
	ssrc := pkt.SSRC
	pt := pkt.PayloadType
	p := b.packers[ssrc]
	b.packerLock.RUnlock()
	if p == nil {
		switch pt {
		case webrtc.DefaultPayloadTypeVP8:
			b.packerLock.Lock()
			b.packers[ssrc] = packer.New(pktSize, b.nackCh)
			p = b.packers[ssrc]
			b.packerLock.Unlock()
		case webrtc.DefaultPayloadTypeOpus:
		case webrtc.DefaultPayloadTypeH264:
		case webrtc.DefaultPayloadTypeVP9:
			// TODO
		default:
			log.Errorf("unknown PayloadType %d", pt)
		}
	}
	if p == nil {
		return errors.New("packer is nil")
	}

	p.Push(pkt)
	return nil
}

func (b *buffer) jitter() {
	go func() {
		rembTicker := time.NewTicker(rembDuration)
		defer rembTicker.Stop()
		for {
			select {
			case nack := <-b.nackCh:
				log.Infof("buffer.jitter sendNack nack=%v", nack)
				b.pipeline.getPub().sendNack(nack)
			case <-rembTicker.C:
				b.packerLock.RLock()
				packers := b.packers
				b.packerLock.RUnlock()
				for _, packer := range packers {
					if util.IsVideo(packer.GetPayloadType()) {
						lost := packer.CalcLostRate()
						if b.pipeline != nil && b.pipeline.getPub() != nil {
							b.pipeline.getPub().sendREMB(lost)
						}
					}
				}

			case <-b.stopCh:
				b.wg.Done()
				return
			}
		}
	}()
}

func (b *buffer) GetPacket(ssrc uint32, sn uint16) *rtp.Packet {
	b.packerLock.RLock()
	packer := b.packers[ssrc]
	b.packerLock.RUnlock()
	if packer == nil {
		return nil
	}
	return packer.FindPacket(sn)
}

func (b *buffer) Stop() {
	close(b.stopCh)
	b.wg.Wait()
	close(b.nackCh)
}
