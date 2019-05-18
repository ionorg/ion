package media

import (
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/log"
	"github.com/pion/webrtc/v2"
)

var defaultPeerCfg = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.stunprotocol.org:3478"},
		},
	},
}

const (
	rtcpPLIInterval = time.Second * 1
)

type WebRTCEngine struct {
	// PeerConnection config
	cfg webrtc.Configuration

	// Media engine
	mediaEngine webrtc.MediaEngine

	// API object
	api *webrtc.API
}

func NewWebRTCEngine() *WebRTCEngine {
	urls := conf.Cfg.Sfu.Ices

	w := &WebRTCEngine{
		mediaEngine: webrtc.MediaEngine{},
		cfg: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: urls,
				},
			},
		},
	}
	w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	w.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	w.api = webrtc.NewAPI(webrtc.WithMediaEngine(w.mediaEngine))
	return w
}

func (s WebRTCEngine) CreateSender(offer webrtc.SessionDescription, pc **webrtc.PeerConnection, addVideoTrack, addAudioTrack **webrtc.Track, stop chan int) (answer webrtc.SessionDescription, err error) {
	*pc, err = s.api.NewPeerConnection(s.cfg)
	log.Infof("WebRTCEngine.CreateSender pc=%p", *pc)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	//track is ready
	if *addVideoTrack != nil && *addAudioTrack != nil {
		(*pc).AddTrack(*addVideoTrack)
		(*pc).AddTrack(*addAudioTrack)
		err = (*pc).SetRemoteDescription(offer)
		if err != nil {
			return webrtc.SessionDescription{}, err
		}
	}

	answer, err = (*pc).CreateAnswer(nil)
	err = (*pc).SetLocalDescription(answer)
	log.Infof("WebRTCEngine.CreateSender ok")
	return answer, err
}

func (s WebRTCEngine) CreateReceiver(offer webrtc.SessionDescription, pc **webrtc.PeerConnection, videoTrack, audioTrack **webrtc.Track, stop chan int, pli chan int) (answer webrtc.SessionDescription, err error) {
	*pc, err = s.api.NewPeerConnection(s.cfg)
	log.Infof("WebRTCEngine.CreateReceiver pc=%p", *pc)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	_, err = (*pc).AddTransceiver(webrtc.RTPCodecTypeVideo)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	_, err = (*pc).AddTransceiver(webrtc.RTPCodecTypeAudio)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	(*pc).OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			*videoTrack, err = (*pc).NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")

			go func() {
				// ticker := time.NewTicker(rtcpPLIInterval)
				for {
					select {
					// case <-ticker.C:
					case <-pli:
						(*pc).WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}})
					case <-stop:
						return
					}
				}
			}()

			rtpBuf := make([]byte, 1400)
			for {
				select {
				case <-stop:
					return
				default:
					i, err := remoteTrack.Read(rtpBuf)
					if err == nil {
						(*videoTrack).Write(rtpBuf[:i])
					} else {
						log.Infof(err.Error())
					}
				}
			}
		} else {
			*audioTrack, err = (*pc).NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "audio", "pion")

			rtpBuf := make([]byte, 1400)
			for {
				select {
				case <-stop:
					return
				default:
					i, err := remoteTrack.Read(rtpBuf)
					if err == nil {
						(*audioTrack).Write(rtpBuf[:i])
					} else {
						log.Infof(err.Error())
					}
				}
			}
		}
	})

	err = (*pc).SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = (*pc).CreateAnswer(nil)
	err = (*pc).SetLocalDescription(answer)
	log.Infof("WebRTCEngine.CreateReceiver ok")
	return answer, err
}
