package speech

import (
	"context"
	"io"

	"github.com/mitchellh/mapstructure"
	"github.com/pion/ion/pkg/log"
	processor "github.com/pion/ion/pkg/node/avp/processors"
	"github.com/pion/ion/pkg/node/avp/processors/oggwriterr"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var (
	cfg Options
)

// AudioOptions audio options
type AudioOptions struct {
	Sampling uint32
	Channels uint16
}

// Options Speech options
type Options struct {
	Audio AudioOptions
}

// Init initialize speech processor
func Init(conf map[string]interface{}) {
	var decoded Options
	err := mapstructure.Decode(conf, &decoded)

	if err != nil {
		log.Errorf("Speech: Error decoding config: %v", conf)
	}

	log.Infof("Initializing Speech with config %#v", decoded)
	cfg = decoded
}

type writerError struct {
	reason string
}

func (e *writerError) Error() string {
	return e.reason
}

type SpeechWriter struct {
	stream speechpb.Speech_StreamingRecognizeClient
	pw     *io.PipeWriter
	pr     *io.PipeReader
}

func NewSpeechWriter() *SpeechWriter {
	sw := &SpeechWriter{}
	sw.pr, sw.pw = io.Pipe()
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Warnf("Error creating speech writer %s", err.Error())
		return nil
	}

	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Warnf(err.Error())
		return nil
	}

	// Send the initial configuration message.
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_OGG_OPUS,
					SampleRateHertz: int32(cfg.Audio.Sampling),
					LanguageCode:    "en-US",
				},
			},
		},
	}); err != nil {
		log.Infof(err.Error())
		return nil
	}

	go func() {
		// Pipe stdin to the API.
		buf := make([]byte, 1024)
		for {
			n, err := sw.pr.Read(buf)
			if n > 0 {
				if err := stream.Send(&speechpb.StreamingRecognizeRequest{
					StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
						AudioContent: buf[:n],
					},
				}); err != nil {
					log.Infof("Could not send audio: %v", err)
				}
			}
			if err == io.EOF {
				// Nothing else to pipe, close the stream.
				if err := stream.CloseSend(); err != nil {
					log.Warnf("Could not close stream: %v", err)
				}
				return
			}
			if err != nil {
				log.Infof("Could not read from stdin: %v", err)
				continue
			}
		}
	}()

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warnf("Cannot stream results: %v", err)
			}
			if err := resp.Error; err != nil {
				// Workaround while the API doesn't give a more informative error.
				if err.Code == 3 || err.Code == 11 {
					log.Infof("WARNING: Speech recognition request exceeded limit of 60 seconds.")
				}
				log.Warnf("Could not recognize: %v", err)
			}
			for _, result := range resp.Results {
				log.Infof("Result: %+v\n", result)
			}
		}
	}()

	return sw
}

func (sw *SpeechWriter) Close() error {
	return sw.pw.Close()
}

func (sw *SpeechWriter) Write(p []byte) (n int, err error) {
	return sw.pw.Write(p)
}

// NewSpeech Creates a Speech processor
func NewSpeech(id string) *processor.Processor {
	log.Infof("NewSpeech with config %v", cfg)
	p := &processor.Processor{
		ID: id,
	}

	sw := NewSpeechWriter()

	if sw == nil {
		log.Warnf("init-disk-writers: error creating speech writer")
		return nil
	}

	oggWriter, err := oggwriterr.NewWith(sw, cfg.Audio.Sampling, cfg.Audio.Channels)
	if err != nil {
		log.Warnf("init-disk-writers: error creating audio writer")
		return nil
	}

	p.AudioWriter = processor.RTPWriter(oggWriter)

	return p
}
