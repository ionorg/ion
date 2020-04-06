package speech

import (
	"context"
	"encoding/json"
	"io"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
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

// Options Speech options
type Options struct {
	Sampling       uint32
	Channels       uint16
	StreamingLimit uint
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

func setInterval(someFunc func(), milliseconds uint, async bool) chan bool {

	// How often to fire the passed in function
	// in milliseconds
	interval := time.Duration(milliseconds) * time.Millisecond

	// Setup the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)
	clear := make(chan bool)

	// Put the selection in a go routine
	// so that the for loop is none blocking
	go func() {
		for {

			select {
			case <-ticker.C:
				if async {
					// This won't block
					go someFunc()
				} else {
					// This will block
					someFunc()
				}
			case <-clear:
				ticker.Stop()
				return
			}

		}
	}()

	// We return the channel so we can pass in
	// a value to it to clear the interval
	return clear

}

type SpeechWriter struct {
	start     time.Time
	stream    speechpb.Speech_StreamingRecognizeClient
	rid       string
	broadcast func(rid string, msg map[string]interface{}) *nprotoo.Error
	pw        *io.PipeWriter
	pr        *io.PipeReader
}

func (sw *SpeechWriter) init() {
	if sw.stream != nil {
		log.Infof("Closed speech writer")
		sw.stream.CloseSend()
	}
	log.Infof("Creating speech writer")
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Warnf("Error creating speech writer %s", err.Error())
		return
	}

	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Warnf(err.Error())
		return
	}

	// Send the initial configuration message.
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:                   speechpb.RecognitionConfig_OGG_OPUS,
					SampleRateHertz:            int32(cfg.Sampling),
					LanguageCode:               "en-US",
					MaxAlternatives:            3,
					EnableAutomaticPunctuation: true,
				},
			},
		},
	}); err != nil {
		log.Infof(err.Error())
		return
	}

	sw.stream = stream
}

func NewSpeechWriter(rid string, broadcast func(rid string, msg map[string]interface{}) *nprotoo.Error) *SpeechWriter {
	sw := &SpeechWriter{
		broadcast: broadcast,
	}
	sw.pr, sw.pw = io.Pipe()

	sw.init()
	// setInterval(func() {
	// 	sw.init()
	// }, cfg.StreamingLimit, true)

	go func() {
		// Pipe stdin to the API.
		buf := make([]byte, 1024)
		for {
			n, err := sw.pr.Read(buf)
			if n > 0 {

				if err := sw.stream.Send(&speechpb.StreamingRecognizeRequest{
					StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
						AudioContent: buf[:n],
					},
				}); err != nil {
					log.Infof("Could not send audio: %v", err)
				}
			}
			if err == io.EOF {
				// Nothing else to pipe, close the stream.
				if err := sw.stream.CloseSend(); err != nil {
					log.Warnf("Could not close stream: %v", err)
				}
				break
			}
			if err != nil {
				log.Infof("Could not read from stdin: %v", err)
				continue
			}
		}
	}()

	go func() {
		for {
			resp, err := sw.stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warnf("Cannot stream results: %v", err)
			}
			if err := resp.Error; err != nil {
				// Workaround while the API doesn't give a more informative error.
				// if err.Code == 3 || err.Code == 11 {
				// 	log.Infof("WARNING: Speech recognition request exceeded limit of 60 seconds.")
				// }
				log.Warnf("Speech error: %v", err)
				break
			}

			var data map[string]interface{}
			s, err := json.Marshal(resp)
			json.Unmarshal(s, &data)
			sw.broadcast(sw.rid, data)
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
func NewSpeech(rid string, mid string, broadcast func(rid string, msg map[string]interface{}) *nprotoo.Error) *processor.Processor {
	log.Infof("NewSpeech with config %v", cfg)
	p := &processor.Processor{
		ID: mid,
	}

	sw := NewSpeechWriter(rid, broadcast)

	if sw == nil {
		log.Warnf("init-disk-writers: error creating speech writer")
		return nil
	}

	oggWriter, err := oggwriterr.NewWith(sw, cfg.Sampling, cfg.Channels)
	if err != nil {
		log.Warnf("init-disk-writers: error creating audio writer")
		return nil
	}

	p.AudioWriter = processor.RTPWriter(oggWriter)

	return p
}
