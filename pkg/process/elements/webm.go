package elements

import (
	"fmt"
	"os"
	"path"

	"github.com/at-wat/ebml-go/webm"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/process/samples"
)

const (
	// TypeWebmSaver .
	TypeWebmSaver = "WebmSaver"
)

var (
	config WebmSaverConfig
)

// WebmSaverConfig .
type WebmSaverConfig struct {
	Enabled   bool
	Togglable bool
	DefaultOn bool
	Path      string
}

// WebmSaver Module for saving rtp streams to webm
type WebmSaver struct {
	id                             string
	path                           string
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioTimestamp, videoTimestamp uint32
}

// InitWebmSaver sets initial config
func InitWebmSaver(c WebmSaverConfig) {
	config = c
}

// NewWebmSaver Initialize a new webm saver
func NewWebmSaver(id string) *WebmSaver {
	return &WebmSaver{
		id:   id,
		path: config.Path,
	}
}

// Write sample to webmsaver
func (s *WebmSaver) Write(sample *samples.Sample) error {
	if sample.Type == samples.TypeVP8 {
		s.pushVP8(sample)
	} else if sample.Type == samples.TypeOpus {
		s.pushOpus(sample)
	}
	return nil
}

func (s *WebmSaver) Read() <-chan *samples.Sample {
	return nil
}

// Close Close the WebmSaver
func (s *WebmSaver) Close() {
	log.Infof("WebmSaver.Close() => %s", s.id)
	if s.audioWriter != nil {
		if err := s.audioWriter.Close(); err != nil {
			panic(err)
		}
	}
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) pushOpus(sample *samples.Sample) {
	if s.audioWriter != nil {
		if s.audioTimestamp == 0 {
			s.audioTimestamp = sample.Timestamp
		}
		t := (sample.Timestamp - s.audioTimestamp) / 48
		if _, err := s.audioWriter.Write(true, int64(t), sample.Payload); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) pushVP8(sample *samples.Sample) {
	// Read VP8 header.
	videoKeyframe := (sample.Payload[0]&0x1 == 0)

	if videoKeyframe {
		// Keyframe has frame information.
		raw := uint(sample.Payload[6]) | uint(sample.Payload[7])<<8 | uint(sample.Payload[8])<<16 | uint(sample.Payload[9])<<24
		width := int(raw & 0x3FFF)
		height := int((raw >> 16) & 0x3FFF)

		if s.videoWriter == nil || s.audioWriter == nil {
			// Initialize WebM saver using received frame size.
			s.initWriter(width, height)
		}
	}

	if s.videoWriter != nil {
		if s.videoTimestamp == 0 {
			s.videoTimestamp = sample.Timestamp
		}
		t := (sample.Timestamp - s.videoTimestamp) / 90
		if _, err := s.videoWriter.Write(videoKeyframe, int64(t), sample.Payload); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) initWriter(width, height int) {
	w, err := os.OpenFile(path.Join(s.path, fmt.Sprintf("%s.webm", s.id)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}

	ws, err := webm.NewSimpleBlockWriter(w,
		[]webm.TrackEntry{
			{
				Name:            "Audio",
				TrackNumber:     1,
				TrackUID:        12345,
				CodecID:         "A_OPUS",
				TrackType:       2,
				DefaultDuration: 20000000,
				Audio: &webm.Audio{
					SamplingFrequency: 48000.0,
					Channels:          2,
				},
			}, {
				Name:            "Video",
				TrackNumber:     2,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 33333333,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		})
	if err != nil {
		panic(err)
	}
	log.Infof("WebM saver has started with video width=%d, height=%d\n", width, height)
	s.audioWriter = ws[0]
	s.videoWriter = ws[1]
}
