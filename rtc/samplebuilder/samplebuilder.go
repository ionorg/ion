package samplebuilder

import (
	"github.com/pion/rtp"
)

// Sample contains media, and the amount of samples in it
type Sample struct {
	Data           []byte
	Samples        uint32
	SequenceNumber uint16
}

// SampleBuilder contains all packets
// maxLate determines how long we should wait until we get a valid Sample
// The larger the value the less packet loss you will see, but higher latency
type SampleBuilder struct {
	maxLate uint16
	buffer  [65536]*rtp.Packet

	// Interface that allows us to take RTP packets to samples
	depacketizer rtp.Depacketizer

	// Last seqnum that has been added to buffer
	lastPush uint16

	// Last seqnum that has been successfully popped
	// isContiguous is false when we start or when we have a gap
	// that is older then maxLate
	isContiguous     bool
	lastPopSeq       uint16
	lastPopTimestamp uint32

	lastContiguousSeq uint16
}

// New constructs a new SampleBuilder
func New(maxLate uint16, depacketizer rtp.Depacketizer) *SampleBuilder {
	return &SampleBuilder{maxLate: maxLate, depacketizer: depacketizer}
}

// Distance between two seqnums
func seqnumDistance(x, y uint16) uint16 {
	if x > y {
		return x - y
	}

	return y - x
}

// Push adds a RTP Packet to the sample builder
func (s *SampleBuilder) Push(p *rtp.Packet, gapSeq chan uint16) {
	// log.Infof("SampleBuilder.Push pt=%v sn=%v ts=%v", p.PayloadType, p.SequenceNumber, p.Timestamp)
	s.buffer[p.SequenceNumber] = p
	// if s.lastPush != p.SequenceNumber-1 && s.lastPush != 0 {
	// // some caps after lastPush
	// for i := s.lastPush + 1; i < p.SequenceNumber; i++ {
	// gapSeq <- i
	// log.Infof("gap !!!!!!gapSeq=%d", i)
	// }
	// }
	s.lastPush = p.SequenceNumber
	// clear old packet
	s.buffer[p.SequenceNumber-s.maxLate] = nil
}

// We have a valid collection of RTP Packets
// walk forwards building a sample if everything looks good clear and update buffer+values
func (s *SampleBuilder) buildSample(firstBuffer uint16) *Sample {
	data := []byte{}

	for i := firstBuffer; s.buffer[i] != nil; i++ {
		if s.buffer[i].Timestamp != s.buffer[firstBuffer].Timestamp {
			lastTimeStamp := s.lastPopTimestamp
			if !s.isContiguous && s.buffer[firstBuffer-1] != nil {
				// firstBuffer-1 should always pass, but just to be safe if there is a bug in Pop()
				lastTimeStamp = s.buffer[firstBuffer-1].Timestamp
			}

			samples := s.buffer[i-1].Timestamp - lastTimeStamp
			s.lastPopSeq = i - 1
			s.isContiguous = true
			s.lastPopTimestamp = s.buffer[i-1].Timestamp
			for j := firstBuffer; j < i; j++ {
				s.buffer[j] = nil
			}
			return &Sample{Data: data, Samples: samples, SequenceNumber: firstBuffer}
		}

		p, err := s.depacketizer.Unmarshal(s.buffer[i].Payload)
		if err != nil {
			return nil
		}

		data = append(data, p...)
	}
	return nil
}

// Pop scans buffer for valid samples, returns nil when no valid samples have been found
func (s *SampleBuilder) Pop() *Sample {
	var i uint16
	if !s.isContiguous {
		i = s.lastPush - s.maxLate
	} else {
		if seqnumDistance(s.lastPopSeq, s.lastPush) > s.maxLate {
			i = s.lastPush - s.maxLate
			s.isContiguous = false
		} else {
			i = s.lastPopSeq + 1
		}
	}

	for ; i != s.lastPush; i++ {
		curr := s.buffer[i]
		// current rtp == nil
		if curr == nil {

			//previous rtp != nil, gap break
			if s.buffer[i-1] != nil {
				break // there is a gap, we can't proceed
			}

			continue // we haven't hit a buffer yet, keep moving
		}

		// current rtp != nil
		if !s.isContiguous {
			// previous rtp  == nil
			if s.buffer[i-1] == nil {
				continue // We have never popped a buffer, so we can't assert that the first RTP packet we encounter is valid
			} else if s.buffer[i-1].Timestamp == curr.Timestamp {
				continue // We have the same timestamps, so it is data that spans multiple RTP packets
			}
		}

		// Initial validity checks have passed, walk forward
		return s.buildSample(i)
	}
	return nil
}

// We have a valid collection of RTP Packets
func (s *SampleBuilder) buildPackets(firstBuffer uint16) []*rtp.Packet {
	pkts := []*rtp.Packet{}

	for i := firstBuffer; s.buffer[i] != nil; i++ {
		if s.buffer[i].Timestamp != s.buffer[firstBuffer].Timestamp {
			// lastTimeStamp := s.lastPopTimestamp
			// if !s.isContiguous && s.buffer[firstBuffer-1] != nil {
			// firstBuffer-1 should always pass, but just to be safe if there is a bug in Pop()
			// lastTimeStamp = s.buffer[firstBuffer-1].Timestamp
			// }

			// samples := s.buffer[i-1].Timestamp - lastTimeStamp
			s.lastPopSeq = i - 1
			s.isContiguous = true
			s.lastPopTimestamp = s.buffer[i-1].Timestamp
			for j := firstBuffer; j < i; j++ {
				s.buffer[j] = nil
			}
			// log.Infof("pop pkts=%v", pkts)
			return pkts
		}
		pkts = append(pkts, s.buffer[i])
	}
	// log.Infof("buildPackets pop nil")
	return nil
}

// Pop scans buffer for valid samples, returns nil when no valid samples have been found
func (s *SampleBuilder) PopPackets() []*rtp.Packet {
	var i uint16
	if !s.isContiguous {
		i = s.lastPush - s.maxLate
		// log.Infof("!s.isContiguous i = %d", i)
	} else {
		if seqnumDistance(s.lastPopSeq, s.lastPush) > s.maxLate {
			i = s.lastPush - s.maxLate
			s.isContiguous = false
			// log.Infof("seqnumDistance(s.lastPopSeq, s.lastPush) > s.maxLate s.lastPopSeq=%d  s.lastPush=%d  i=%d  s.isContiguous=false s.maxLate=%d", s.lastPopSeq, s.lastPush, i, s.maxLate)
		} else {
			i = s.lastPopSeq + 1
			// log.Infof("i=%d s.lastPopSeq=%d", i, s.lastPopSeq)
		}
	}

	for ; i != s.lastPush; i++ {
		curr := s.buffer[i]
		// current rtp == nil
		if curr == nil {

			//previous rtp != nil, gap break
			if s.buffer[i-1] != nil {
				break // there is a gap, we can't proceed
			}

			continue // we haven't hit a buffer yet, keep moving
		}

		// log.Infof("s.isContiguous=%v", s.isContiguous)
		// current rtp != nil
		if !s.isContiguous {
			// previous rtp  == nil
			if s.buffer[i-1] == nil {
				// log.Infof("s.buffer[i-1] == nil")
				continue // We have never popped a buffer, so we can't assert that the first RTP packet we encounter is valid
			} else if s.buffer[i-1].Timestamp == curr.Timestamp {
				// log.Infof("s.buffer[i-1].Timestamp=%d, curr.Timestamp=%d", s.buffer[i-1].Timestamp, curr.Timestamp)
				continue // We have the same timestamps, so it is data that spans multiple RTP packets
			}
		}

		// Initial validity checks have passed, walk forward
		return s.buildPackets(i)
	}
	return nil
}
