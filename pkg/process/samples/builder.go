package samples

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

// BuilderConfig .
type BuilderConfig struct {
	ID           string
	On           bool
	AudioMaxLate uint16
	VideoMaxLate uint16
}

// Builder Module for building video/audio samples from rtp streams
type Builder struct {
	id                           string
	stop                         bool
	audioBuilder, videoBuilder   *samplebuilder.SampleBuilder
	audioSequence, videoSequence uint16
	outChan                      chan *Sample
}

// NewBuilder Initialize a new sample builder
func NewBuilder(config BuilderConfig) *Builder {
	log.Infof("NewBuilder with config %+v", config)
	s := &Builder{
		id:           config.ID,
		audioBuilder: samplebuilder.New(config.AudioMaxLate, &codecs.OpusPacket{}),
		videoBuilder: samplebuilder.New(config.VideoMaxLate, &codecs.VP8Packet{}),
		outChan:      make(chan *Sample, maxSize),
	}

	samplebuilder.WithPartitionHeadChecker(&codecs.OpusPartitionHeadChecker{})(s.audioBuilder)
	samplebuilder.WithPartitionHeadChecker(&codecs.VP8PartitionHeadChecker{})(s.videoBuilder)

	return s
}

// ID Builder id
func (s *Builder) ID() string {
	return s.id
}

// WriteRTP Write RTP packet to Builder
func (s *Builder) WriteRTP(pkt *rtp.Packet) error {
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
func (s *Builder) Read() *Sample {
	return <-s.outChan
}

// Stop stop all buffer
func (s *Builder) Stop() {
	if s.stop {
		return
	}
	s.stop = true
}

func (s *Builder) pushOpus(pkt *rtp.Packet) {
	s.audioBuilder.Push(pkt)

	for {
		sample, timestamp := s.audioBuilder.PopWithTimestamp()
		if sample == nil {
			return
		}
		s.outChan <- &Sample{
			Type:           TypeOpus,
			SequenceNumber: s.audioSequence,
			Timestamp:      timestamp,
			Payload:        sample.Data,
		}
		s.audioSequence++
	}
}

func (s *Builder) pushVP8(pkt *rtp.Packet) {
	s.videoBuilder.Push(pkt)
	for {
		sample, timestamp := s.videoBuilder.PopWithTimestamp()
		if sample == nil {
			return
		}

		s.outChan <- &Sample{
			Type:           TypeVP8,
			SequenceNumber: s.videoSequence,
			Timestamp:      timestamp,
			Payload:        sample.Data,
		}
		s.videoSequence++
	}
}
