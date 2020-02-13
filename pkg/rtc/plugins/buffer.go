package plugins

import (
	"fmt"
	"sort"

	"time"

	"github.com/bluele/gcache"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
)

const (
	maxTTL = time.Millisecond * 1000
	//https://chromium.googlesource.com/external/webrtc/+/ad34dbe934/webrtc/modules/video_coding/nack_module.cc#28
	nackTTL     = time.Millisecond * 20
	maxPktSize  = 1000
	maxNackSize = 1000

	//1+16(FSN+BLP) https://tools.ietf.org/html/rfc2032#page-9
	maxNackLostSize = 17
)

type RtpExtensionInfo struct {
	RtpExtensionSN uint16
	Timestamp      int64
}

// Buffer contains all packets
type Buffer struct {
	maxLate      uint16
	pktBuffer    gcache.Cache
	rtpExtBuffer gcache.Cache
	nackBuffer   gcache.Cache
	lastPushSN   uint16

	ssrc        uint32
	payloadType uint8

	//calc lost rate
	receivedPkt int
	lostPkt     int

	//response nack channel
	rtcpCh chan rtcp.Packet

	//calc bindwidth
	totalByte uint64
	byteRate  uint64

	stop       bool
	lastNackSN uint16
}

// NewBuffer constructs a new Buffer
func NewBuffer(maxLate uint16) *Buffer {
	b := &Buffer{
		maxLate:      maxLate,
		rtcpCh:       make(chan rtcp.Packet, maxNackSize),
		pktBuffer:    gcache.New(maxPktSize).Simple().Build(),
		nackBuffer:   gcache.New(maxNackSize).Simple().Build(),
		rtpExtBuffer: gcache.New(maxPktSize).Simple().Build(),
	}
	b.GatherJitterInfoLoop()
	return b
}

func (b *Buffer) GatherJitterInfoLoop() {
	go func() {
		t := time.NewTicker(nackTTL)
		defer t.Stop()
		for {
			if b.stop {
				return
			}
			select {
			case <-t.C:
				b.GatherJitterInfo()
			}
		}
	}()
}

// GetRTCPChan return rtcp channel
func (b *Buffer) GetRTCPChan() chan rtcp.Packet {
	return b.rtcpCh
}

// Push adds a RTP Packet, out of order, new packet may be arrived later
func (b *Buffer) Push(pkt *rtp.Packet) {
	// log.Infof("Buffer.Push pt=%v sn=%v ts=%v", pkt.PayloadType, pkt.SequenceNumber, pkt.Timestamp)
	b.receivedPkt++
	b.totalByte += uint64(pkt.MarshalSize())
	if b.ssrc == 0 || b.payloadType == 0 {
		b.ssrc = pkt.SSRC
		b.payloadType = pkt.PayloadType
	}

	b.pktBuffer.SetWithExpire(pkt.SequenceNumber, pkt, maxTTL)
	b.nackBuffer.SetWithExpire(pkt.SequenceNumber, pkt, nackTTL)
	if pkt.Extension {
		rtpTCC := rtp.TransportCCExtension{}
		err := rtpTCC.Unmarshal(pkt.ExtensionPayload)
		if err == nil {
			rtpExt := RtpExtensionInfo{
				RtpExtensionSN: rtpTCC.TransportSequence,
				Timestamp:      time.Now().UnixNano() / 1e6,
			}
			b.rtpExtBuffer.SetWithExpire(pkt.SequenceNumber, rtpExt, maxTTL)
			log.Infof("rtpExt=%v", rtpExt)
		} else {
			log.Errorf("rtpTCC.Unmarshal err: %v", err)
		}
	}
	b.lastPushSN = pkt.SequenceNumber
}

func (b *Buffer) GatherJitterInfo() {
	// 1. find ordered sequence nums
	var sns []int
	keys := b.nackBuffer.Keys(true)

	for _, sn := range keys {
		sns = append(sns, int(sn.(uint16)))
	}

	if len(sns) == 0 {
		return
	}

	sort.Ints(sns)

	if b.lastNackSN == 0 {
		b.lastNackSN = uint16(sns[0])
	}

	// log.Infof("sns=%v b.lastNackSN=%v", sns, b.lastNackSN)
	// 2. caculate nacks and lostPkt
	for ; b.lastNackSN < uint16(sns[len(sns)-1]); b.lastNackSN++ {
		if b.GetPacket(b.lastNackSN) == nil {
			break
		}
	}
	nacks, lostPkt := b.GetNackPairsAndLostPkts(sns, b.lastNackSN, false)
	b.lastNackSN = uint16(sns[len(sns)-1])
	// log.Infof("b.lastNackSN=%v", b.lastNackSN)
	if nacks != nil {
		nack := &rtcp.TransportLayerNack{
			//origin ssrc
			SenderSSRC: b.ssrc,
			MediaSSRC:  b.ssrc,
			Nacks:      nacks,
		}
		if !b.stop {
			b.rtcpCh <- nack
		}
	}
	b.lostPkt += lostPkt
}

func (b *Buffer) CalcLostRateByteRate(cycle uint64) (float64, uint64) {
	lostRate := float64(b.lostPkt) / float64(b.receivedPkt+b.lostPkt)
	byteRate := b.totalByte / cycle
	log.Debugf("Buffer.CalcLostRateByteRate b.receivedPkt=%d b.lostPkt=%d lostRate=%v byteRate=%v", b.receivedPkt, b.lostPkt, lostRate, byteRate)
	b.receivedPkt, b.lostPkt, b.totalByte = 0, 0, 0
	return lostRate, byteRate
}

func (b *Buffer) GetPacket(sn uint16) *rtp.Packet {
	pkt, _ := b.pktBuffer.Get(sn)
	if pkt == nil {
		return nil
	}
	return pkt.(*rtp.Packet)
}

func (b *Buffer) existInNackBuf(sn uint16) bool {
	pkt, _ := b.nackBuffer.Get(sn)
	if pkt == nil {
		return false
	}
	return true
}

func (b *Buffer) GetPayloadType() uint8 {
	return b.payloadType
}

func (b *Buffer) SetSSRCPT(ssrc uint32, pt uint8) {
	b.ssrc = ssrc
	b.payloadType = pt
}

func (b *Buffer) GetSSRC() uint32 {
	return b.ssrc
}

func (b *Buffer) GetStat() string {
	out := fmt.Sprintf("buffer:lastPushSN:%d |\n", b.lastPushSN)
	return out
}

func (b *Buffer) Stop() {
	b.stop = true
	close(b.rtcpCh)
}

// GetNackPairsAndLostPkts get nack lost pkts
func (b *Buffer) GetNackPairsAndLostPkts(sns []int, begin uint16, keyFrame bool) ([]rtcp.NackPair, int) {
	// log.Infof("packer.GetNackPairsAndLostPkts sns=%v", sns)
	//Bitmask of following lost packets (BLP)
	blp := uint16(0)
	// begin := uint16(sns[0])
	end := uint16(sns[len(sns)-1])
	lostPkt := 0
	lostSN := uint16(0)
	hasKeyFrame := false

	// begin = b.lastNackSN + 1
	if end-begin > maxNackLostSize*3 {
		begin = end - maxNackLostSize*3
	}

	//find first lostSN
	for i := begin; i < end; i++ {
		if !hasKeyFrame && IsVP8KeyFrame(b.GetPacket(i)) {
			hasKeyFrame = true
		}

		if !b.existInNackBuf(i) {
			lostSN = i
			// find in the old buffer, prevent repeatly counting
			// example: [2,  5]  [1, 4]
			if b.GetPacket(i) == nil {
				lostPkt++
			}
			break
		}
	}

	//no packet lost
	if lostPkt == 0 {
		return nil, 0
	}

	var nacks []rtcp.NackPair

	// find looping
	for i := lostSN + 1; i < end; i++ {
		if !b.existInNackBuf(i) {
			//https://tools.ietf.org/html/rfc2032
			blp = blp | (1 << (i - lostSN - 1))
			if b.GetPacket(i) == nil {
				lostPkt++
			}
		}

		if (i-lostSN)%maxNackLostSize == 0 {
			nacks = append(nacks, rtcp.NackPair{PacketID: lostSN, LostPackets: rtcp.PacketBitmap(blp)})
			blp = 0
			//find next beginning packetID
			for j := i; j < end; j++ {
				if !b.existInNackBuf(j) {
					i, lostSN = j, j
					if b.GetPacket(i) == nil {
						lostPkt++
					}
					break
				}
			}
		}
	}

	// add last one
	if (end-lostSN)%maxNackLostSize != 0 {
		nacks = append(nacks, rtcp.NackPair{PacketID: lostSN, LostPackets: rtcp.PacketBitmap(blp)})
	}

	if keyFrame && !hasKeyFrame {
		return nil, 0
	}
	// log.Infof("packer.GetNackPairsAndLostPkts nacks=%v lostPkt=%v", nacks, lostPkt)
	return nacks, lostPkt
}

func IsVP8KeyFrame(pkt *rtp.Packet) bool {
	if pkt != nil && pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 {
		vp8 := &codecs.VP8Packet{}
		vp8.Unmarshal(pkt.Payload)
		// start of a frame, there is a payload header  when S == 1
		if vp8.S == 1 && vp8.Payload[0]&0x01 == 0 {
			//key frame
			// log.Infof("vp8.Payload[0]=%b pkt=%v", vp8.Payload[0], pkt)
			return true
		}
	}
	return false
}
