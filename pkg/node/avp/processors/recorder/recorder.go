package recorder

import (
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/pion/ion/pkg/log"
	processor "github.com/pion/ion/pkg/node/avp/processors"
	"github.com/pion/webrtc/v2/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v2/pkg/media/oggwriter"
)

var (
	cfg Options
)

// AudioOptions Recorder audio options
type AudioOptions struct {
	Sampling uint32
	Channels uint16
}

// Options Recorder options
type Options struct {
	Outpath string
	Audio   AudioOptions
}

// Init initialize recorder
func Init(conf map[string]interface{}) {
	var decoded Options
	err := mapstructure.Decode(conf, &decoded)

	if err != nil {
		log.Errorf("Recorder: Error decoding config: %v", conf)
	}

	log.Infof("Initializing Recorder with config %#v", decoded)
	cfg = decoded
}

// NewRecorder Creates a Recorder
func NewRecorder(id string) *processor.Processor {
	log.Infof("NewRecorder with output at %s", cfg.Outpath)
	r := &processor.Processor{
		ID: id,
	}

	audioOutputPath := path.Join(cfg.Outpath, id+".ogg")
	videoOutputPath := path.Join(cfg.Outpath, id+".ivf")

	oggWriter, err := oggwriter.New(audioOutputPath, cfg.Audio.Sampling, cfg.Audio.Channels)
	if err != nil {
		log.Errorf("New Recoder: failed to create audio writer", err)
		return nil
	}

	ivfWriter, err := ivfwriter.New(videoOutputPath)
	if err != nil {
		log.Errorf("New Recoder: failed to create video writer", err)
		return nil
	}

	r.AudioWriter = processor.RTPWriter(oggWriter)
	r.VideoWriter = processor.RTPWriter(ivfWriter)

	return r
}
