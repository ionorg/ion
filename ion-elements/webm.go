package elements

import (
	"github.com/at-wat/ebml-go/webm"

	"github.com/sssgun/ion/pkg/log"
	"github.com/sssgun/ion/pkg/process"
	"github.com/sssgun/ion/pkg/process/samples"
)

const (
	// TypeWebmSaver .
	TypeWebmSaver = "WebmSaver"
)

// WebmSaverConfig .
type WebmSaverConfig struct {
	ID string
}

// WebmSaver Module for saving rtp streams to webm
type WebmSaver struct {
	id                             string
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioTimestamp, videoTimestamp uint32
	sampleWriter                   *SampleWriter
}

// NewWebmSaver Initialize a new webm saver
func NewWebmSaver(config WebmSaverConfig) *WebmSaver {
	return &WebmSaver{
		id:           config.ID,
		sampleWriter: NewSampleWriter(),
	}
}

// Type of element
func (s *WebmSaver) Type() string {
	return TypeWebmSaver
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

// Attach attach a child element
func (s *WebmSaver) Attach(e process.Element) error {
	return s.sampleWriter.Attach(e)
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
	ws, err := webm.NewSimpleBlockWriter(s.sampleWriter,
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

// SampleWriter for writing samples
type SampleWriter struct {
	childElements map[string]process.Element
}

// NewSampleWriter creates a new sample writer
func NewSampleWriter() *SampleWriter {
	return &SampleWriter{
		childElements: make(map[string]process.Element),
	}
}

// Attach a child element
func (w *SampleWriter) Attach(e process.Element) error {
	if w.childElements[e.Type()] == nil {
		log.Infof("Transcribe.Attach element => %s", e.Type())
		w.childElements[e.Type()] = e
		return nil
	}
	return ErrElementAlreadyAttached
}

// Write sample
func (w *SampleWriter) Write(p []byte) (n int, err error) {
	for _, e := range w.childElements {
		sample := &samples.Sample{
			Type:    TypeBinary,
			Payload: p,
		}
		err := e.Write(sample)
		if err != nil {
			log.Errorf("SampleWriter.Write error => %s", err)
		}
	}
	return len(p), nil
}

// Close writer
func (w *SampleWriter) Close() error {
	for _, e := range w.childElements {
		e.Close()
	}
	return nil
}
