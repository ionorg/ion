package recorder

import (
	"context"
	"path"

	"cloud.google.com/go/storage"
	"github.com/mitchellh/mapstructure"
	"github.com/pion/ion/pkg/log"
	processor "github.com/pion/ion/pkg/node/avp/processors"
	"github.com/pion/ion/pkg/node/avp/processors/oggwriterr"
	"github.com/pion/webrtc/v2/pkg/media/ivfwriter"
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
	Dest    string
	Outpath string
	Bucket  string
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

type writerError struct {
	reason string
}

func (e *writerError) Error() string {
	return e.reason
}

func intiDiskWriters(id string) (processor.RTPWriter, processor.RTPWriter, error) {
	audioOutputPath := path.Join(cfg.Outpath, id+".ogg")
	videoOutputPath := path.Join(cfg.Outpath, id+".ivf")
	oggWriter, err := oggwriterr.New(audioOutputPath, cfg.Audio.Sampling, cfg.Audio.Channels)
	if err != nil {
		return nil, nil, &writerError{"init-disk-writers: error creating audio writer"}
	}

	ivfWriter, err := ivfwriter.New(videoOutputPath)
	if err != nil {
		log.Errorf("New Recoder: failed to create video writer", err)
		return nil, nil, &writerError{"init-disk-writers: error creating video writer"}
	}

	return oggWriter, ivfWriter, nil
}

func initGCWriters(id string) (processor.RTPWriter, processor.RTPWriter, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	videoOutputPath := path.Join(cfg.Outpath, id+".ivf")
	vw := client.Bucket(cfg.Bucket).Object(videoOutputPath).NewWriter(ctx)
	ivfWriter, err := ivfwriter.NewWith(vw)
	if err != nil {
		log.Errorf("New Recoder: failed to create video writer", err)
		return nil, nil, &writerError{"init-disk-writers: error creating video writer"}
	}

	audioOutputPath := path.Join(cfg.Outpath, id+".ogg")
	aw := client.Bucket(cfg.Bucket).Object(audioOutputPath).NewWriter(ctx)
	oggWriter, err := oggwriterr.NewWith(aw, cfg.Audio.Sampling, cfg.Audio.Channels)
	if err != nil {
		return nil, nil, &writerError{"init-disk-writers: error creating audio writer"}
	}

	return oggWriter, ivfWriter, nil
}

// NewRecorder Creates a Recorder
func NewRecorder(id string) *processor.Processor {
	log.Infof("NewRecorder with config %v", cfg)
	r := &processor.Processor{
		ID: id,
	}

	var oggWriter processor.RTPWriter
	var ivfWriter processor.RTPWriter
	var err error

	if cfg.Dest == "gcloud" {
		oggWriter, ivfWriter, err = initGCWriters(id)
	} else {
		oggWriter, ivfWriter, err = intiDiskWriters(id)
	}

	if err != nil {
		log.Errorf(err.Error())
	}

	r.AudioWriter = processor.RTPWriter(oggWriter)
	r.VideoWriter = processor.RTPWriter(ivfWriter)

	return r
}
