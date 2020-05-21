package plugins

import (
	"fmt"
	"os"
	"path"

	"github.com/at-wat/ebml-go/webm"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

// WebmSaverConfig .
type WebmSaverConfig struct {
	ID   string
	MID  string
	On   bool
	Path string
}

// WebmSaver Module for saving rtp streams to webm
type WebmSaver struct {
	id                             string
	mid                            string
	path                           string
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioTimestamp, videoTimestamp uint32
	outRTPChan                     chan *rtp.Packet
}

// NewWebmSaver Initialize a new webm saver
func NewWebmSaver(config WebmSaverConfig) *WebmSaver {
	return &WebmSaver{
		id:         config.ID,
		mid:        config.MID,
		path:       config.Path,
		outRTPChan: make(chan *rtp.Packet, maxSize),
	}
}

// ID WebmSaver id
func (s *WebmSaver) ID() string {
	return s.id
}

// WriteRTP Write RTP packet to webmsaver
func (s *WebmSaver) WriteRTP(pkt *rtp.Packet) error {
	if pkt.PayloadType == webrtc.DefaultPayloadTypeVP8 {
		s.pushVP8(pkt)
	} else if pkt.PayloadType == webrtc.DefaultPayloadTypeOpus {
		s.pushOpus(pkt)
	}
	s.outRTPChan <- pkt
	return nil
}

// ReadRTP Forward rtp packet which from pub
func (s *WebmSaver) ReadRTP() <-chan *rtp.Packet {
	return s.outRTPChan
}

// Stop Close the WebmSaver
func (s *WebmSaver) Stop() {
	fmt.Printf("Finalizing webm...\n")
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

func (s *WebmSaver) pushOpus(pkt *rtp.Packet) {
	if s.audioWriter != nil {
		if s.audioTimestamp == 0 {
			s.audioTimestamp = pkt.Timestamp
		}
		t := (pkt.Timestamp - s.audioTimestamp) / 48
		if _, err := s.audioWriter.Write(true, int64(t), pkt.Payload); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) pushVP8(pkt *rtp.Packet) {
	// Read VP8 header.
	videoKeyframe := (pkt.Payload[0]&0x1 == 0)

	if videoKeyframe {
		// Keyframe has frame information.
		raw := uint(pkt.Payload[6]) | uint(pkt.Payload[7])<<8 | uint(pkt.Payload[8])<<16 | uint(pkt.Payload[9])<<24
		width := int(raw & 0x3FFF)
		height := int((raw >> 16) & 0x3FFF)

		if s.videoWriter == nil || s.audioWriter == nil {
			// Initialize WebM saver using received frame size.
			s.initWriter(width, height)
		}
	}

	if s.videoWriter != nil {
		if s.videoTimestamp == 0 {
			s.videoTimestamp = pkt.Timestamp
		}
		t := (pkt.Timestamp - s.videoTimestamp) / 90
		if _, err := s.videoWriter.Write(videoKeyframe, int64(t), pkt.Payload); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) initWriter(width, height int) {
	w, err := os.OpenFile(path.Join(s.path, fmt.Sprintf("%s.webm", s.mid)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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
