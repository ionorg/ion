package plugins

import (
	"fmt"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
)

const (
	maxSN      = 65536
	maxPktSize = 1000

	// kProcessIntervalMs=20 ms
	//https://chromium.googlesource.com/external/webrtc/+/ad34dbe934/webrtc/modules/video_coding/nack_module.cc#28

	// vp8 vp9 h264 clock rate 90000Hz
	videoClock = 90000

	//1+16(FSN+BLP) https://tools.ietf.org/html/rfc2032#page-9
	maxNackLostSize = 17

	//default buffer time by ms
	defaultBufferTime = 1000

	tccExtMapID = 3
	//64ms = 64000us = 250 << 8
	//https://webrtc.googlesource.com/src/webrtc/+/f54860e9ef0b68e182a01edc994626d21961bc4b/modules/rtp_rtcp/source/rtcp_packet/transport_feedback.cc#41
	baseScaleFactor = 64000
	//https://webrtc.googlesource.com/src/webrtc/+/f54860e9ef0b68e182a01edc994626d21961bc4b/modules/rtp_rtcp/source/rtcp_packet/transport_feedback.cc#43
	timeWrapPeriodUs = (int64(1) << 24) * baseScaleFactor

	//experiment cycle
	tccCycle = 10 * time.Millisecond
)

func tsDelta(x, y uint32) uint32 {
	if x > y {
		return x - y
	}
	return y - x
}

type rtpExtInfo struct {
	//transport sequence num
	TSN       uint16
	Timestamp int64
}

// Buffer contains all packets
type Buffer struct {
	pktBuffer   [maxSN]*rtp.Packet
	lastNackSN  uint16
	lastClearTS uint32
	lastClearSN uint16

	// Last seqnum that has been added to buffer
	lastPushSN uint16

	ssrc        uint32
	payloadType uint8

	//calc lost rate
	receivedPkt int
	lostPkt     int

	//response nack channel
	rtcpCh chan rtcp.Packet

	//calc bandwidth
	totalByte uint64

	//buffer time
	maxBufferTS uint32

	stop bool

	feedbackPacketCount uint8

	rtpExtInfoChan chan rtpExtInfo
	// lastTCCSN      uint16
	// bufferStartTS time.Time
}

type BufferOptions struct {
	TCCOn      bool
	BufferTime int
}

// NewBuffer constructs a new Buffer
func NewBuffer(o BufferOptions) *Buffer {
	b := &Buffer{
		rtcpCh:         make(chan rtcp.Packet, maxPktSize),
		rtpExtInfoChan: make(chan rtpExtInfo, maxPktSize),
	}

	if o.TCCOn {
		b.calcTCCLoop()
	}

	if o.BufferTime <= 0 {
		o.BufferTime = defaultBufferTime
	}
	b.maxBufferTS = uint32(o.BufferTime) * videoClock / 1000
	// b.bufferStartTS = time.Now()
	log.Infof("NewBuffer BufferOptions=%v", o)
	return b
}

func (b *Buffer) calcTCCLoop() {
	go func() {
		t := time.NewTicker(tccCycle)
		defer t.Stop()
		for {
			if b.stop {
				return
			}
			<-t.C
			b.calcTCC()
		}
	}()
}

func (b *Buffer) calcTCC() {
	cap := len(b.rtpExtInfoChan)
	if cap == 0 {
		return
	}

	//get all rtp extension infos from channel
	rtpExtInfo := make(map[uint16]int64)
	for i := 0; i < cap; i++ {
		info := <-b.rtpExtInfoChan
		rtpExtInfo[info.TSN] = info.Timestamp
	}

	//find the min and max transport sn
	var minTSN, maxTSN uint16
	for tsn := range rtpExtInfo {

		//init
		if minTSN == 0 {
			minTSN = tsn
		}

		if minTSN > tsn {
			minTSN = tsn
		}

		if maxTSN < tsn {
			maxTSN = tsn
		}
	}

	//force small deta rtcp.RunLengthChunk
	chunk := &rtcp.RunLengthChunk{
		Type:               rtcp.TypeTCCRunLengthChunk,
		PacketStatusSymbol: rtcp.TypeTCCPacketReceivedSmallDelta,
		RunLength:          maxTSN - minTSN + 1,
	}

	//gather deltas
	var recvDeltas []*rtcp.RecvDelta
	var refTime uint32
	var lastTS int64
	var baseTimeTicks int64
	for i := minTSN; i <= maxTSN; i++ {
		ts, ok := rtpExtInfo[i]

		//lost packet
		if !ok {
			recvDelta := &rtcp.RecvDelta{
				Type: rtcp.TypeTCCPacketReceivedSmallDelta,
			}
			recvDeltas = append(recvDeltas, recvDelta)
			continue
		}

		// init lastTS
		if lastTS == 0 {
			lastTS = ts
		}

		//received packet
		if baseTimeTicks == 0 {
			baseTimeTicks = (ts % timeWrapPeriodUs) / baseScaleFactor
		}

		var delta int64
		if lastTS == ts {
			delta = ts%timeWrapPeriodUs - baseTimeTicks*baseScaleFactor
		} else {
			delta = (ts - lastTS) % timeWrapPeriodUs
		}

		if refTime == 0 {
			refTime = uint32(baseTimeTicks) & 0x007FFFFF
		}

		recvDelta := &rtcp.RecvDelta{
			Type:  rtcp.TypeTCCPacketReceivedSmallDelta,
			Delta: delta,
		}
		recvDeltas = append(recvDeltas, recvDelta)
	}
	rtcpTCC := &rtcp.TransportLayerCC{
		Header: rtcp.Header{
			Padding: false,
			Count:   rtcp.FormatTCC,
			Type:    rtcp.TypeTransportSpecificFeedback,
			// Length:  5, //need calc
		},
		// SenderSSRC:         b.ssrc,
		MediaSSRC:          b.ssrc,
		BaseSequenceNumber: minTSN,
		PacketStatusCount:  maxTSN - minTSN + 1,
		ReferenceTime:      refTime,
		FbPktCount:         b.feedbackPacketCount,
		RecvDeltas:         recvDeltas,
		PacketChunks:       []rtcp.PacketStatusChunk{chunk},
	}
	rtcpTCC.Header.Length = rtcpTCC.Len()/4 - 1
	if !b.stop {
		b.rtcpCh <- rtcpTCC
		b.feedbackPacketCount++
	}
}

// Push adds a RTP Packet, out of order, new packet may be arrived later
func (b *Buffer) Push(p *rtp.Packet) {
	b.receivedPkt++
	b.totalByte += uint64(p.MarshalSize())

	// init ssrc payloadType
	if b.ssrc == 0 || b.payloadType == 0 {
		b.ssrc = p.SSRC
		b.payloadType = p.PayloadType
	}

	// init lastClearTS
	if b.lastClearTS == 0 {
		b.lastClearTS = p.Timestamp
	}

	// init lastClearSN
	if b.lastClearSN == 0 {
		b.lastClearSN = p.SequenceNumber
	}

	// init lastNackSN
	if b.lastNackSN == 0 {
		b.lastNackSN = p.SequenceNumber
	}

	b.pktBuffer[p.SequenceNumber] = p
	b.lastPushSN = p.SequenceNumber

	//store arrival time
	timestampUs := time.Now().UnixNano() / 1000
	rtpTCC := rtp.TransportCCExtension{}
	err := rtpTCC.Unmarshal(p.GetExtension(tccExtMapID))
	if err == nil {
		// if time.Now().Sub(b.bufferStartTS) > time.Second {

		//only calc the packet which rtpTCC.TransportSequence > b.lastTCCSN
		//https://webrtc.googlesource.com/src/webrtc/+/f54860e9ef0b68e182a01edc994626d21961bc4b/modules/rtp_rtcp/source/rtcp_packet/transport_feedback.cc#353
		// if rtpTCC.TransportSequence > b.lastTCCSN {
		b.rtpExtInfoChan <- rtpExtInfo{
			TSN:       rtpTCC.TransportSequence,
			Timestamp: timestampUs,
		}
		// b.lastTCCSN = rtpTCC.TransportSequence
		// }
	}
	// }

	// clear old packet by timestamp
	b.clearOldPkt(p.Timestamp, p.SequenceNumber)

	// limit nack range
	if b.lastPushSN-b.lastNackSN >= maxNackLostSize {
		b.lastNackSN = b.lastPushSN - maxNackLostSize
	}

	if b.lastPushSN-b.lastNackSN >= maxNackLostSize {
		// calc [lastNackSN, lastpush-8] if has keyframe
		nackPair, lostPkt := b.GetNackPair(b.pktBuffer, b.lastNackSN, b.lastPushSN)
		b.lastNackSN = b.lastPushSN
		// log.Infof("b.lastNackSN=%v, b.lastPushSN=%v, lostPkt=%v, nackPair=%v", b.lastNackSN, b.lastPushSN, lostPkt, nackPair)
		if lostPkt > 0 {
			b.lostPkt += lostPkt
			nack := &rtcp.TransportLayerNack{
				//origin ssrc
				// SenderSSRC: b.ssrc,
				MediaSSRC: b.ssrc,
				Nacks: []rtcp.NackPair{
					nackPair,
				},
			}
			b.rtcpCh <- nack
		}
	}
}

// clearOldPkt clear old packet
func (b *Buffer) clearOldPkt(pushPktTS uint32, pushPktSN uint16) {
	clearTS := b.lastClearTS
	clearSN := b.lastClearSN
	// log.Infof("clearOldPkt pushPktTS=%d pushPktSN=%d     clearTS=%d  clearSN=%d ", pushPktTS, pushPktSN, clearTS, clearSN)
	if tsDelta(pushPktTS, clearTS) >= b.maxBufferTS {
		//pushPktSN will loop from 0 to 65535
		if pushPktSN == 0 {
			//make sure clear the old packet from 655xx to 65535
			pushPktSN = maxSN - 1
		}
		var skipCount int
		for i := clearSN + 1; i <= pushPktSN; i++ {
			if b.pktBuffer[i] == nil {
				skipCount++
				continue
			}
			if tsDelta(pushPktTS, b.pktBuffer[i].Timestamp) >= b.maxBufferTS {
				b.lastClearTS = b.pktBuffer[i].Timestamp
				b.lastClearSN = i
				b.pktBuffer[i] = nil
			} else {
				break
			}
		}
		if skipCount > 0 {
			log.Infof("b.pktBuffer nil count : %d", skipCount)
		}
		if pushPktSN == maxSN-1 {
			b.lastClearSN = 0
			b.lastNackSN = 0
		}
	}
}

// FindPacket find packet from buffer
func (b *Buffer) FindPacket(sn uint16) *rtp.Packet {
	return b.pktBuffer[sn]
}

// Stop buffer
func (b *Buffer) Stop() {
	b.stop = true
	close(b.rtcpCh)
	b.clear()
}

func (b *Buffer) clear() {
	for i := range b.pktBuffer {
		b.pktBuffer[i] = nil
	}
}

// GetPayloadType get payloadtype
func (b *Buffer) GetPayloadType() uint8 {
	return b.payloadType
}

// GetStat get status from buffer
func (b *Buffer) GetStat() string {
	out := fmt.Sprintf("buffer:[%d, %d] | lastNackSN:%d | lostRate:%.2f |\n", b.lastClearSN, b.lastPushSN, b.lastNackSN, float64(b.lostPkt)/float64(b.receivedPkt+b.lostPkt))
	return out
}

// GetNackPair calc nackpair
func (b *Buffer) GetNackPair(buffer [65536]*rtp.Packet, begin, end uint16) (rtcp.NackPair, int) {

	var lostPkt int

	//size is <= 17
	if end-begin > maxNackLostSize {
		return rtcp.NackPair{}, lostPkt
	}

	//Bitmask of following lost packets (BLP)
	blp := uint16(0)
	lost := uint16(0)

	//find first lost pkt
	for i := begin; i < end; i++ {
		if buffer[i] == nil {
			lost = i
			lostPkt++
			break
		}
	}

	//no packet lost
	if lost == 0 {
		return rtcp.NackPair{}, lostPkt
	}

	//calc blp
	for i := lost; i < end; i++ {
		//calc from next lost packet
		if i > lost && buffer[i] == nil {
			blp = blp | (1 << (i - lost - 1))
			lostPkt++
		}
	}
	log.Debugf("NackPair begin=%v end=%v buffer=%v\n", begin, end, buffer[begin:end])
	return rtcp.NackPair{PacketID: lost, LostPackets: rtcp.PacketBitmap(blp)}, lostPkt
}

// SetSSRCPT set ssrc payloadtype
func (b *Buffer) SetSSRCPT(ssrc uint32, pt uint8) {
	b.ssrc = ssrc
	b.payloadType = pt
}

// GetSSRC get ssrc
func (b *Buffer) GetSSRC() uint32 {
	return b.ssrc
}

// GetRTCPChan return rtcp channel
func (b *Buffer) GetRTCPChan() chan rtcp.Packet {
	return b.rtcpCh
}

// GetLostRateBandwidth calc lostRate and bandwidth by cycle
func (b *Buffer) GetLostRateBandwidth(cycle uint64) (float64, uint64) {
	lostRate := float64(b.lostPkt) / float64(b.receivedPkt+b.lostPkt)
	byteRate := b.totalByte / cycle
	log.Debugf("Buffer.CalcLostRateByteRate b.receivedPkt=%d b.lostPkt=%d   lostRate=%v byteRate=%v", b.receivedPkt, b.lostPkt, lostRate, byteRate)
	b.receivedPkt, b.lostPkt, b.totalByte = 0, 0, 0
	return lostRate, byteRate * 8 / 1000
}

// GetPacket get packet by sequence number
func (b *Buffer) GetPacket(sn uint16) *rtp.Packet {
	return b.pktBuffer[sn]
}

// IsVP8KeyFrame check key frame
func IsVP8KeyFrame(pkt *rtp.Packet) bool {
	if pkt != nil && pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 {
		vp8 := &codecs.VP8Packet{}
		_, err := vp8.Unmarshal(pkt.Payload)
		if err != nil {
			return false
		}
		// start of a frame, there is a payload header  when S == 1
		if vp8.S == 1 && vp8.Payload[0]&0x01 == 0 {
			//key frame
			// log.Infof("vp8.Payload[0]=%b pkt=%v", vp8.Payload[0], pkt)
			return true
		}
	}
	return false
}
