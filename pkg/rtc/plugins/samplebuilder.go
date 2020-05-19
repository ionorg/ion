package plugins

import (
	"errors"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/samplebuilder"
)

var (
	// ErrCodecNotSupported is returned when a rtp packed it pushed with an unsupported codec
	ErrCodecNotSupported = errors.New("codec not supported")
)

// SampleBuilderConfig .
type SampleBuilderConfig struct {
	ID           string
	On           bool
	AudioMaxLate uint16
	VideoMaxLate uint16
}

// SampleBuilder Module for building video/audio samples from rtp streams
type SampleBuilder struct {
	id                             string
	stop                           bool
	audioBuilder, videoBuilder     *samplebuilder.SampleBuilder
	audioSequence, videoSequence   uint16
	audioTimestamp, videoTimestamp uint32
	outRTPChan                     chan *rtp.Packet
}

// NewSampleBuilder Initialize a new webm saver
func NewSampleBuilder(config SampleBuilderConfig) *SampleBuilder {
	s := &SampleBuilder{
		id:           config.ID,
		audioBuilder: samplebuilder.New(config.AudioMaxLate, &codecs.OpusPacket{}),
		videoBuilder: samplebuilder.New(config.VideoMaxLate, &codecs.VP8Packet{}),
		outRTPChan:   make(chan *rtp.Packet, maxSize),
	}

	samplebuilder.WithPartitionHeadChecker(&codecs.OpusPartitionHeadChecker{})(s.audioBuilder)
	samplebuilder.WithPartitionHeadChecker(&codecs.VP8PartitionHeadChecker{})(s.videoBuilder)

	return s
}

// ID SampleBuilder id
func (s *SampleBuilder) ID() string {
	return s.id
}

// AttachPub Attach pub stream
func (s *SampleBuilder) AttachPub(t transport.Transport) {
	go func(t transport.Transport) {
		for {
			if s.stop {
				return
			}
			pkt, err := t.ReadRTP()
			if err != nil {
				log.Errorf("AttachPub t.ReadRTP pkt=%+v", pkt)
				continue
			}
			err = s.WriteRTP(pkt)
			if err != nil {
				log.Errorf("AttachPub t.WriteRTP err=%+v", err)
				continue
			}
		}
	}(t)
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

// ReadRTP Forward rtp packet which from pub
func (s *SampleBuilder) ReadRTP() <-chan *rtp.Packet {
	return s.outRTPChan
}

// Stop stop all buffer
func (s *SampleBuilder) Stop() {
	log.Infof("stop sample builder")
	if s.stop {
		return
	}
	log.Infof("stop sample builder")
	s.stop = true
}

func (s *SampleBuilder) pushOpus(pkt *rtp.Packet) {
	s.audioBuilder.Push(pkt)

	for {
		sample, timestamp := s.audioBuilder.PopWithTimestamp()
		if sample == nil {
			return
		}
		if s.audioTimestamp == 0 {
			s.audioTimestamp = timestamp
		}
		t := (timestamp - s.audioTimestamp) / 48
		s.outRTPChan <- &rtp.Packet{
			Header: rtp.Header{
				Version:        pkt.Version,
				PayloadType:    webrtc.DefaultPayloadTypeOpus,
				SequenceNumber: s.audioSequence,
				Timestamp:      t,
			},
			Payload: sample.Data,
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
		if s.videoTimestamp == 0 {
			s.videoTimestamp = timestamp
		}
		t := (timestamp - s.videoTimestamp) / 90
		s.outRTPChan <- &rtp.Packet{
			Header: rtp.Header{
				Version:        pkt.Version,
				PayloadType:    webrtc.DefaultPayloadTypeVP8,
				SequenceNumber: s.videoSequence,
				Timestamp:      t,
			},
			Payload: sample.Data,
		}
		s.videoSequence++
	}
}
