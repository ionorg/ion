// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"encoding/json"
	"os"
	"time"
	"github.com/cloudwebrtc/go-protoo/client"
	"github.com/cloudwebrtc/go-protoo/logger"
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/google/uuid"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

const (
	address       = "wss://ion.streamhuddle.com/ws"
	audioFileName = "output.ogg"
	videoFileName = "output.ivf"
	rid           = "test"
	username      = "pub-from-disk"
)

// AnswerJSEP is part of Answer JSON reply
type AnswerJSEP struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

// Answer is a JSON reply
type Answer struct {
	JSEP AnswerJSEP `json:"jsep"`
	mid  string
}


func main() {
	// Assert that we have an audio or video file
	_, err := os.Stat(videoFileName)
	haveVideoFile := !os.IsNotExist(err)

	_, err = os.Stat(audioFileName)
	haveAudioFile := !os.IsNotExist(err)

	if !haveAudioFile && !haveVideoFile {
		panic("Could not find `" + audioFileName + "` or `" + videoFileName + "`")
	}

	// We make our own mediaEngine so we can place the sender's codecs in it.  This because we must use the
	// dynamic media type from the sender in our answer. This is not required if we are the offerer
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterDefaultCodecs()

	// Create a new RTCPeerConnection
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	if haveVideoFile {
		// Create a video track
		videoTrack, addTrackErr := peerConnection.NewTrack(getPayloadType(mediaEngine, webrtc.RTPCodecTypeVideo, "VP8"), rand.Uint32(), "video", "pion")
		if addTrackErr != nil {
			panic(addTrackErr)
		}
		if _, addTrackErr = peerConnection.AddTrack(videoTrack); err != nil {
			panic(addTrackErr)
		}

		go func() {
			// Open a IVF file and start reading using our IVFReader
			file, ivfErr := os.Open(videoFileName)
			if ivfErr != nil {
				panic(ivfErr)
			}

			ivf, header, ivfErr := ivfreader.NewWith(file)
			if ivfErr != nil {
				panic(ivfErr)
			}

			// Wait for connection established
			<-iceConnectedCtx.Done()

			// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
			// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
			sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
			for {
				frame, _, ivfErr := ivf.ParseNextFrame()
				if ivfErr == io.EOF {
					fmt.Printf("All video frames parsed and sent")
					peerConnection.Close()
					os.Exit(0)
				}

				if ivfErr != nil {
					panic(ivfErr)
				}

				time.Sleep(sleepTime)
				if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Samples: 90000}); ivfErr != nil {
					panic(ivfErr)
				}
			}
		}()
	}

	if haveAudioFile {
		// Create a audio track
		audioTrack, addTrackErr := peerConnection.NewTrack(getPayloadType(mediaEngine, webrtc.RTPCodecTypeAudio, "opus"), rand.Uint32(), "audio", "pion")
		if addTrackErr != nil {
			panic(addTrackErr)
		}
		if _, addTrackErr = peerConnection.AddTrack(audioTrack); err != nil {
			panic(addTrackErr)
		}

		go func() {
			// Open a IVF file and start reading using our IVFReader
			file, oggErr := os.Open(audioFileName)
			if oggErr != nil {
				panic(oggErr)
			}

			// Open on oggfile in non-checksum mode.
			ogg, _, oggErr := oggreader.NewWith(file)
			if oggErr != nil {
				panic(oggErr)
			}

			// Wait for connection established
			<-iceConnectedCtx.Done()

			// Keep track of last granule, the difference is the amount of samples in the buffer
			var lastGranule uint64
			for {
				pageData, pageHeader, oggErr := ogg.ParseNextPage()
				if oggErr == io.EOF {
					fmt.Printf("All audio pages parsed and sent")
					os.Exit(0)
				}

				if oggErr != nil {
					panic(oggErr)
				}

				// The amount of samples is the difference between the last and current timestamp
				sampleCount := float64((pageHeader.GranulePosition - lastGranule))
				lastGranule = pageHeader.GranulePosition

				if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Samples: uint32(sampleCount)}); oggErr != nil {
					panic(oggErr)
				}

				// Convert seconds to Milliseconds, Sleep doesn't accept floats
				time.Sleep(time.Duration((sampleCount/48000)*1000) * time.Millisecond)
			}
		}()
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
		}
	})

	peerID := uuid.New().String()


	client.NewClient(address+"?peer="+peerID, func(con *transport.WebSocketTransport) {
		logger.Infof("handleWebSocketOpen")

		pr := peer.NewPeer(peerID, con)

		handleRequest := func(request peer.Request, accept peer.RespondFunc, reject peer.RejectFunc) {
			method := request.Method
			logger.Infof("handleRequest =>  (%s) ", method)
			if method == "kick" {
				reject(486, "Busy Here")
			} else {
				accept(nil)
			}
		}

		handleNotification := func(notification peer.Notification) {
			logger.Infof("handleNotification => %s", notification.Method)
		}

		handleClose := func(err transport.TransportErr) {
			logger.Infof("handleClose => peer (%s) [%d] %s", pr.ID(), err.Code, err.Text)
		}

		go func() {
			for {
				select {
				case msg := <-pr.OnNotification:
					log.Println(msg)
					handleNotification(msg)
				case msg := <-pr.OnRequest:
					handleRequest(msg.Request, msg.Accept, msg.Reject)
				case msg := <-pr.OnClose:
					handleClose(msg)
				}
			}
		}()

		pr.Request("join", json.RawMessage(`{"rid":"`+rid+`","info":{"name":"`+username+`"}}`),
			func(result json.RawMessage) {
				logger.Infof("join success: =>  %s", result)

				offer, err := peerConnection.CreateOffer(nil)
				if err != nil {
					panic(err)
				}

				publishInfo := map[string]interface{}{
					"rid": "test",
					"jsep": map[string]interface{}{
						"sdp":  string(offer.SDP),
						"type": "offer",
					},
					"options": map[string]interface{}{
						"codec":     "VP8",
						"bandwidth": 1024,
					},
				}

				publish, err := json.Marshal(publishInfo)

				logger.Infof("Publish Message: %s\n", publish)

				pr.Request("publish", publishInfo,
					func(result json.RawMessage) {
						logger.Infof("publish success: =>  %s", result)
						var answer Answer
						json.Unmarshal(result, &answer)

						peerConnection.SetRemoteDescription(webrtc.SessionDescription{
							Type: webrtc.SDPTypeAnswer,
							SDP:  answer.JSEP.SDP,
						})
					},
					func(code int, err string) {
						logger.Infof("publish reject: %d => %s", code, err)
					})

			},
			func(code int, err string) {
				logger.Infof("login reject: %d => %s", code, err)
			})

	},
	)

	for {
		// wait until end of file and exit
	}
}

// Search for Codec PayloadType
//
// Since we are answering we need to match the remote PayloadType
func getPayloadType(m webrtc.MediaEngine, codecType webrtc.RTPCodecType, codecName string) uint8 {
	for _, codec := range m.GetCodecsByKind(codecType) {
		if codec.Name == codecName {
			return codec.PayloadType
		}
	}
	panic(fmt.Sprintf("Remote peer does not support %s", codecName))
}
