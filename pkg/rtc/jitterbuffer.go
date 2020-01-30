package rtc

import (
	"errors"
	"fmt"
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
	pktSize = 200
)

type jitterBuffer struct {
	pipeline   *pipeline
	id         string
	packers    map[uint32]*packer.Packer
	packerLock sync.RWMutex
	nackCh     chan *rtcp.TransportLayerNack
	stop       bool
	rembBW     uint64
}

func newJitterBuffer(id string, p *pipeline) *jitterBuffer {
	r := &jitterBuffer{
		packers:  make(map[uint32]*packer.Packer),
		id:       id,
		pipeline: p,
		nackCh:   make(chan *rtcp.TransportLayerNack, 100),
	}
	r.jitter()
	return r
}

func (j *jitterBuffer) ID() string {
	return j.id
}

func (j *jitterBuffer) Push(pkt *rtp.Packet) error {
	j.packerLock.RLock()
	ssrc := pkt.SSRC
	pt := pkt.PayloadType
	p := j.packers[ssrc]
	j.packerLock.RUnlock()
	if p == nil {
		switch pt {
		case webrtc.DefaultPayloadTypeVP8:
			j.packerLock.Lock()
			j.packers[ssrc] = packer.New(pktSize, j.nackCh)
			p = j.packers[ssrc]
			j.packerLock.Unlock()
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

func (j *jitterBuffer) jitter() {
	go func() {
		rembTicker := time.NewTicker(rembDuration)
		defer rembTicker.Stop()
		for {
			select {
			case nack := <-j.nackCh:
				log.Debugf("jitterBuffer.jitter sendNack nack=%v", nack)
				if j.stop {
					return
				}
				j.pipeline.getPub().sendNack(nack)
			case <-rembTicker.C:
				if j.stop {
					return
				}
				j.packerLock.RLock()
				packers := j.packers
				j.packerLock.RUnlock()
				for _, packer := range packers {
					if util.IsVideo(packer.GetPayloadType()) {
						lost := packer.CalcLostRate()
						if j.pipeline != nil && j.pipeline.getPub() != nil {
							j.rembBW = j.pipeline.getPub().sendREMB(lost)
						}
					}
				}
			}
		}
	}()
}

func (j *jitterBuffer) GetPacket(ssrc uint32, sn uint16) *rtp.Packet {
	j.packerLock.RLock()
	packer := j.packers[ssrc]
	j.packerLock.RUnlock()
	if packer == nil {
		return nil
	}
	return packer.FindPacket(sn)
}

func (j *jitterBuffer) Stop() {
	if j.stop {
		return
	}
	j.stop = true
	close(j.nackCh)
}

func (j *jitterBuffer) Stat() string {
	j.packerLock.RLock()
	defer j.packerLock.RUnlock()
	out := fmt.Sprintf("remb: %d | ", j.rembBW)
	for ssrc, packer := range j.packers {
		out += fmt.Sprintf("ssrc:%d payload:%d | %s", ssrc, packer.GetPayloadType(), packer.GetStat())
	}
	return out
}
