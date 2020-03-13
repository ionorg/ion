package plugins

import (
	"errors"
	"fmt"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const (
	// bandwidth range(kbps)
	minBandwidth = 200
	maxBandwidth = 2000
)

// JitterBufferConfig .
type JitterBufferConfig struct {
	RembCycle int
	PliCycle  int
	Bandwidth int
}

// JitterBuffer core buffer module
type JitterBuffer struct {
	id        string
	buffers   map[uint32]*Buffer
	rtcpCh    chan rtcp.Packet
	stop      bool
	bandwidth uint64
	lostRate  float64

	config JitterBufferConfig
}

// NewJitterBuffer return new JitterBuffer
func NewJitterBuffer(id string) *JitterBuffer {
	j := &JitterBuffer{
		buffers: make(map[uint32]*Buffer),
		id:      id,
		rtcpCh:  make(chan rtcp.Packet, 100),
	}
	j.rembLoop()
	j.pliLoop()
	return j
}

// Init jitterbuffer config
func (j *JitterBuffer) Init(ssrc uint32, pt uint8, config JitterBufferConfig) {
	log.Infof("JitterBuffer.Init ssrc=%d pt=%d config=%v", ssrc, pt, config)
	j.config = config
	if j.config.RembCycle > 5 {
		j.config.RembCycle = 5
	}

	if j.config.PliCycle > 5 {
		j.config.PliCycle = 5
	}

	if j.config.Bandwidth > maxBandwidth {
		j.config.Bandwidth = maxBandwidth
	}

	if j.config.Bandwidth < minBandwidth {
		j.config.Bandwidth = minBandwidth
	}

	if j.GetBuffer(ssrc) == nil {
		log.Infof("JitterBuffer.Init j.AddBuffer %d", ssrc)
		j.AddBuffer(ssrc).SetSSRCPT(ssrc, pt)
	}

	log.Infof("JitterBuffer.Init ok  j.config=%v", j.config)
}

// ID return id
func (j *JitterBuffer) ID() string {
	return j.id
}

// GetRTCPChan get response rtcp channel
func (j *JitterBuffer) GetRTCPChan() chan rtcp.Packet {
	return j.rtcpCh
}

// AddBuffer add a buffer by ssrc
func (j *JitterBuffer) AddBuffer(ssrc uint32) *Buffer {
	b := NewBuffer()
	j.buffers[ssrc] = b
	j.nackLoop(b)
	return b
}

// GetBuffer get a buffer by ssrc
func (j *JitterBuffer) GetBuffer(ssrc uint32) *Buffer {
	return j.buffers[ssrc]
}

// GetBuffers get all buffers
func (j *JitterBuffer) GetBuffers() map[uint32]*Buffer {
	return j.buffers
}

// PushRTP push rtp packet which from pub
func (j *JitterBuffer) PushRTP(pkt *rtp.Packet) error {
	ssrc := pkt.SSRC

	buffer := j.GetBuffer(ssrc)
	if buffer == nil {
		buffer = j.AddBuffer(ssrc)
	}
	if buffer == nil {
		return errors.New("buffer is nil")
	}

	buffer.Push(pkt)
	return nil
}

// PushRTCP push rtcp packet which from sub
func (j *JitterBuffer) PushRTCP(pkt rtcp.Packet) error {
	// log.Infof("JitterBuffer.PushRTCP %v", pkt)
	return nil
}

func (j *JitterBuffer) nackLoop(b *Buffer) {
	go func() {
		for nack := range b.GetRTCPChan() {
			if j.stop {
				return
			}
			j.rtcpCh <- nack
		}
	}()
}

func (j *JitterBuffer) rembLoop() {
	go func() {
		for {
			if j.stop {
				return
			}

			if j.config.RembCycle <= 0 {
				time.Sleep(time.Second)
				continue
			}

			time.Sleep(time.Duration(j.config.RembCycle) * time.Second)
			for _, buffer := range j.GetBuffers() {
				j.lostRate, j.bandwidth = buffer.GetLostRateBandwidth(uint64(j.config.RembCycle))
				var bw uint64
				if j.lostRate == 0 && j.bandwidth == 0 {
					bw = uint64(j.config.Bandwidth)
				} else if j.lostRate >= 0 && j.lostRate < 0.1 {
					bw = uint64(j.bandwidth * 2)
				} else {
					bw = uint64(float64(j.bandwidth) * (1 - j.lostRate))
				}

				if bw < minBandwidth {
					bw = minBandwidth
				}

				if bw > uint64(j.config.Bandwidth) {
					bw = uint64(j.config.Bandwidth)
				}

				remb := &rtcp.ReceiverEstimatedMaximumBitrate{
					SenderSSRC: buffer.GetSSRC(),
					Bitrate:    bw * 1000,
					SSRCs:      []uint32{buffer.GetSSRC()},
				}
				j.rtcpCh <- remb
			}
		}
	}()
}

func (j *JitterBuffer) pliLoop() {
	go func() {
		for {
			if j.stop {
				return
			}

			if j.config.PliCycle <= 0 {
				time.Sleep(time.Second)
				continue
			}
			time.Sleep(time.Duration(j.config.PliCycle) * time.Second)
			for _, buffer := range j.GetBuffers() {
				if util.IsVideo(buffer.GetPayloadType()) {
					pli := &rtcp.PictureLossIndication{SenderSSRC: buffer.GetSSRC(), MediaSSRC: buffer.GetSSRC()}
					j.rtcpCh <- pli
				}
			}
		}
	}()
}

// GetPacket get packet from buffer
func (j *JitterBuffer) GetPacket(ssrc uint32, sn uint16) *rtp.Packet {
	buffer := j.buffers[ssrc]
	if buffer == nil {
		return nil
	}
	return buffer.GetPacket(sn)
}

// Stop stop all buffer
func (j *JitterBuffer) Stop() {
	if j.stop {
		return
	}
	j.stop = true
	for _, buffer := range j.buffers {
		buffer.Stop()
	}
}

// Stat get stat from buffers
func (j *JitterBuffer) Stat() string {
	out := ""
	for ssrc, buffer := range j.buffers {
		out += fmt.Sprintf("ssrc:%d payload:%d | lostRate:%.2f | bandwidth:%dkbps | %s", ssrc, buffer.GetPayloadType(), j.lostRate, j.bandwidth, buffer.GetStat())
	}
	return out
}
