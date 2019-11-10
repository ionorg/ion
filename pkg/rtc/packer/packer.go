package packer

import (
	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const (
	maxSN = 65536
)

// Packer contains all packets
type Packer struct {
	maxLate    uint16
	nackBuffer [maxSN]*rtp.Packet
	// nackBufferMap  map[uint16]*rtp.Packet
	nackBufferLock sync.RWMutex
	lastNackSeq    uint16

	// Last seqnum that has been added to buffer
	lastPush uint16

	ssrc        uint32
	payloadType uint8

	//calc lost rate
	receivedPkt int
	lostPkt     int

	//response nack channel
	nackCh chan *rtcp.TransportLayerNack

	notify chan struct{}
}

// New constructs a new Packer
func New(maxLate uint16, nackCh chan *rtcp.TransportLayerNack) *Packer {
	return &Packer{
		maxLate: maxLate,
		nackCh:  nackCh,
		notify:  make(chan struct{}),
	}
}

// Distance between two seqnums
func seqnumDistance(x, y uint16) uint16 {
	if x > y {
		return x - y
	}

	return y - x
}

// Push adds a RTP Packet
func (s *Packer) Push(p *rtp.Packet) {
	// log.Infof("Packer.Push pt=%v sn=%v ts=%v", p.PayloadType, p.SequenceNumber, p.Timestamp)
	s.receivedPkt++
	if s.ssrc == 0 || s.payloadType == 0 {
		s.ssrc = p.SSRC
		s.payloadType = p.PayloadType
	}
	s.lastPush = p.SequenceNumber
	// clear old packet
	s.nackBufferLock.Lock()
	s.nackBuffer[p.SequenceNumber] = p
	if s.nackBuffer[p.SequenceNumber-s.maxLate] != nil {
		s.nackBuffer[p.SequenceNumber-s.maxLate] = nil
		// log.Infof("p.SequenceNumber-nackBufferMaxSize=%d", p.SequenceNumber-nackBufferMaxSize)
	}
	s.nackBufferLock.Unlock()

	//////////////
	nackPairs := []rtcp.NackPair{}
	if s.lastNackSeq == 0 {
		s.lastNackSeq = s.lastPush
	}
	// log.Infof("s.lastPush=%d s.lastNackSeq=%d", s.lastPush, s.lastNackSeq)
	//if overflow , uint16(-1)=65535  63355/8 > 2
	if s.lastPush-s.lastNackSeq >= 16 && (s.lastPush-s.lastNackSeq)/8 <= 2 {
		s.nackBufferLock.RLock()
		// calc [lastNackSeq, lastpush-8] if has keyframe
		nackPair, lostPkt := util.NackPair(s.nackBuffer, s.lastNackSeq, s.lastPush-8, true)
		s.lostPkt += lostPkt
		s.lastNackSeq += 8
		s.nackBufferLock.RUnlock()
		if nackPair != nil {
			nackPairs = append(nackPairs, *nackPair)
			nack := &rtcp.TransportLayerNack{
				//origin ssrc
				SenderSSRC: s.ssrc,
				MediaSSRC:  s.ssrc,
				Nacks:      nackPairs,
			}
			// log.Infof("nackPairs=%+v", nackPairs)
			// log.Infof("nack=%+v", nack)
			s.nackCh <- nack
		}
	}

}

func (s *Packer) CalcLostRate() float64 {
	total := s.receivedPkt + s.lostPkt
	if total == 0 {
		s.receivedPkt, s.lostPkt = 0, 0
		return 0
	}
	lostRate := float64(s.lostPkt) / float64(total)
	log.Debugf("Packer.CalcLostRate s.receivedPkt=%d s.lostPkt=%d lostRate=%v", s.receivedPkt, s.lostPkt, lostRate)
	s.receivedPkt, s.lostPkt = 0, 0
	return lostRate
}

func (s *Packer) FindPacket(sn uint16) *rtp.Packet {
	s.nackBufferLock.RLock()
	defer s.nackBufferLock.RUnlock()
	return s.nackBuffer[sn]
}

func (s *Packer) Close() {
	close(s.notify)
}

func (s *Packer) GetPayloadType() uint8 {
	return s.payloadType
}
