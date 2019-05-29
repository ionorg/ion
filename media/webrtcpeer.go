package media

import (
	"time"

	"github.com/pion/ion/log"
	"github.com/pion/webrtc/v2"
)

var (
	webrtcEngine *WebRTCEngine
)

func init() {
	webrtcEngine = NewWebRTCEngine()
}

type WebRTCPeer struct {
	ID         string
	PC         *webrtc.PeerConnection
	VideoTrack *webrtc.Track
	AudioTrack *webrtc.Track
	stop       chan int
	pli        chan int
}

func NewWebRTCPeer(id string) *WebRTCPeer {
	return &WebRTCPeer{
		ID:   id,
		stop: make(chan int),
		pli:  make(chan int),
	}
}

func (p *WebRTCPeer) Stop() {
	close(p.stop)
	close(p.pli)
}

func (p *WebRTCPeer) AnswerSender(offer webrtc.SessionDescription) (answer webrtc.SessionDescription, err error) {
	log.Infof("WebRTCPeer.AnswerSender")
	return webrtcEngine.CreateReceiver(offer, &p.PC, &p.VideoTrack, &p.AudioTrack, p.stop, p.pli)
}

func (p *WebRTCPeer) AnswerReceiver(offer webrtc.SessionDescription, addVideoTrack **webrtc.Track, addAudioTrack **webrtc.Track) (answer webrtc.SessionDescription, err error) {
	log.Infof("WebRTCPeer.AnswerReceiver")
	return webrtcEngine.CreateSender(offer, &p.PC, addVideoTrack, addAudioTrack, p.stop)
}

func (p *WebRTCPeer) SendPLI() {
	go func() {
		defer func() {
			// recover from panic caused by writing to a closed channel
			if r := recover(); r != nil {
				log.Errorf("%v", r)
				return
			}
		}()
		ticker := time.NewTicker(time.Second)
		i := 0
		for {
			select {
			case <-ticker.C:
				p.pli <- 1
				if i > 3 {
					return
				}
				i++
			case <-p.stop:
				return
			}
		}
	}()
}
