package samplebuilder

import (
	"errors"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/samplebuilder"
)

const (
	maxSize = 100
)

var (
	// ErrCodecNotSupported is returned when a rtp packed it pushed with an unsupported codec
	ErrCodecNotSupported = errors.New("codec not supported")
)

// Sample constructed from rtp packets
type Sample struct {
	Type           int
	Payload        []byte
	Timestamp      uint32
	SequenceNumber uint16
}

// Config .
type Config struct {
	ID           string
	On           bool
	AudioMaxLate uint16
	VideoMaxLate uint16
}

// SampleBuilder Module for building video/audio samples from rtp streams
type SampleBuilder struct {
	id                           string
	stop                         bool
	audioBuilder, videoBuilder   *samplebuilder.SampleBuilder
	audioSequence, videoSequence uint16
	outChan                      chan *Sample
}

// NewSampleBuilder Initialize a new sample builder
func NewSampleBuilder(config Config) *SampleBuilder {
	log.Infof("NewSampleBuilder with config %+v", config)
	s := &SampleBuilder{
		id:           config.ID,
		audioBuilder: samplebuilder.New(config.AudioMaxLate, &codecs.OpusPacket{}),
		videoBuilder: samplebuilder.New(config.VideoMaxLate, &codecs.VP8Packet{}),
		outChan:      make(chan *Sample, maxSize),
	}

	samplebuilder.WithPartitionHeadChecker(&codecs.OpusPartitionHeadChecker{})(s.audioBuilder)
	samplebuilder.WithPartitionHeadChecker(&codecs.VP8PartitionHeadChecker{})(s.videoBuilder)

	return s
}

// ID SampleBuilder id
func (s *SampleBuilder) ID() string {
	return s.id
}

// WriteRTP Write RTP packet to SampleBuilder
func (s *SampleBuilder) WriteRTP(pkt *rtp.Packet) error {
	if pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 {
		s.pushVP8(pkt)
		return nil
	} else if pkt.PayloadType == webrtc.DefaultPayloadTypeOpus {
		s.pushOpus(pkt)
		return nil
	}
	return ErrCodecNotSupported
}

// Read sample
func (s *SampleBuilder) Read() *Sample {
	return <-s.outChan
}

// Stop stop all buffer
func (s *SampleBuilder) Stop() {
	if s.stop {
		return
	}
	s.stop = true
}

func (s *SampleBuilder) pushOpus(pkt *rtp.Packet) {
	s.audioBuilder.Push(pkt)

	for {
		sample, timestamp := s.audioBuilder.PopWithTimestamp()
		if sample == nil {
			return
		}
		s.outChan <- &Sample{
			Type:           webrtc.DefaultPayloadTypeOpus,
			SequenceNumber: s.audioSequence,
			Timestamp:      timestamp,
			Payload:        sample.Data,
		}
		s.audioSequence++
	}
}

func (s *SampleBuilder) pushVP8(pkt *rtp.Packet) {
	s.videoBuilder.Push(pkt)
	for {
		sample, timestamp := s.videoBuilder.PopWithTimestamp()
		if sample == nil {
			return
		}

		s.outChan <- &Sample{
			Type:           webrtc.DefaultPayloadTypeVP8,
			SequenceNumber: s.videoSequence,
			Timestamp:      timestamp,
			Payload:        sample.Data,
		}
		s.videoSequence++
	}
}
