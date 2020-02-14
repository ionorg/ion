package transport

import (
	"errors"
	"io"
	"strings"

	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

var (
	// only support unified plan
	cfg = webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	setting webrtc.SettingEngine

	errChanClosed     = errors.New("channel closed")
	errInvalidTrack   = errors.New("track is nil")
	errInvalidPacket  = errors.New("packet is nil")
	errInvalidPC      = errors.New("pc is nil")
	errInvalidOptions = errors.New("invalid options")
)

// InitICE init ice urls
func InitICE(ices []string) {
	cfg.ICEServers = []webrtc.ICEServer{
		{
			URLs: ices,
		},
	}
}

// WebRTCTransport ..
type WebRTCTransport struct {
	mediaEngine webrtc.MediaEngine
	api         *webrtc.API
	id          string
	pc          *webrtc.PeerConnection
	track       map[uint32]*webrtc.Track
	trackLock   sync.RWMutex
	stop        bool
	rtpCh       chan *rtp.Packet
	ssrcPT      map[uint32]uint8
	ssrcPTLock  sync.RWMutex
	writeErrCnt int

	rtcpCh chan rtcp.Packet
}

func (w *WebRTCTransport) init(options map[string]interface{}) {
	w.mediaEngine = webrtc.MediaEngine{}
	w.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBTransportCC,
		},
	}
	_, tccOK := options["transport-cc"]
	// only register one video codec which client need
	if codec, ok := options["codec"]; ok {
		codecStr, ok := codec.(string)
		if !ok {
			log.Errorf("NewWebRTCTransport err=%v", errInvalidOptions)
			return
		}
		if strings.EqualFold(codecStr, "h264") && tccOK {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb))
		} else if strings.EqualFold(codecStr, "vp8") && tccOK {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb))
		} else if strings.EqualFold(codecStr, "h264") {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000))
		} else if strings.EqualFold(codecStr, "vp8") {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
		} else if strings.EqualFold(codecStr, "vp9") {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
		} else {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
		}
	}

	w.api = webrtc.NewAPI(webrtc.WithMediaEngine(w.mediaEngine))
}

// NewWebRTCTransport create a WebRTCTransport
func NewWebRTCTransport(id string, options map[string]interface{}) *WebRTCTransport {
	w := &WebRTCTransport{
		id:     id,
		track:  make(map[uint32]*webrtc.Track),
		rtpCh:  make(chan *rtp.Packet, 1000),
		ssrcPT: make(map[uint32]uint8),
		rtcpCh: make(chan rtcp.Packet, 100),
	}
	w.init(options)
	return w
}

// ID return id
func (w *WebRTCTransport) ID() string {
	return w.id
}

// Type return type of transport
func (w *WebRTCTransport) Type() int {
	return TypeWebRTCTransport
}

// Publish answer to pub
func (w *WebRTCTransport) Publish(offer webrtc.SessionDescription) (answer webrtc.SessionDescription, err error) {
	// func (w *WebRTCTransport) AnswerPublish(offer webrtc.SessionDescription, etcdKeepFunc func(ssrc uint32, pt uint8)) (answer webrtc.SessionDescription, err error) {
	w.pc, err = w.api.NewPeerConnection(cfg)
	if err != nil {
		log.Errorf("WebRTCTransport api.NewPeerConnection %v", err)
		return webrtc.SessionDescription{}, err
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver video %v", err)
		return webrtc.SessionDescription{}, err
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver audio %v", err)
		return webrtc.SessionDescription{}, err
	}

	w.pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		w.ssrcPTLock.Lock()
		w.ssrcPT[remoteTrack.SSRC()] = remoteTrack.PayloadType()
		w.ssrcPTLock.Unlock()
		// TODO replace with broacast when receiving rtp failed
		// etcdKeepFunc(remoteTrack.SSRC(), remoteTrack.PayloadType())
		w.receiveRTP(remoteTrack)
	})

	err = w.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Errorf("pc.SetRemoteDescription %v", err)
		return webrtc.SessionDescription{}, err
	}

	answer, err = w.pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("pc.CreateAnswer answer=%v err=%v", answer, err)
		return webrtc.SessionDescription{}, err
	}

	err = w.pc.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("pc.SetLocalDescription answer=%v err=%v", answer, err)
	}
	return answer, err
}

// Subscribe answer to sub
func (w *WebRTCTransport) Subscribe(offer webrtc.SessionDescription, ssrcPT map[uint32]uint8) (answer webrtc.SessionDescription, err error) {
	w.pc, err = w.api.NewPeerConnection(cfg)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	var track *webrtc.Track
	for ssrc, pt := range ssrcPT {
		if pt == webrtc.DefaultPayloadTypeVP8 ||
			pt == webrtc.DefaultPayloadTypeVP9 ||
			pt == webrtc.DefaultPayloadTypeH264 {
			track, _ = w.pc.NewTrack(pt, ssrc, "video", "pion")
		} else {
			track, _ = w.pc.NewTrack(pt, ssrc, "audio", "pion")
		}
		if track != nil {
			w.pc.AddTrack(track)
			w.trackLock.Lock()
			w.track[ssrc] = track
			w.trackLock.Unlock()
		}
	}

	err = w.pc.SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = w.pc.CreateAnswer(nil)
	err = w.pc.SetLocalDescription(answer)
	w.subReadRTCP()
	return answer, err
}

// ReceiveRTP receive all tracks' rtp and sent to one channel
func (w *WebRTCTransport) receiveRTP(remoteTrack *webrtc.Track) {
	for {
		if w.stop {
			return
		}

		rtp, err := remoteTrack.ReadRTP()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Errorf("rtp err => %v", err)
		}
		w.rtpCh <- rtp
	}
}

// ReadRTP read rtp packet
func (w *WebRTCTransport) ReadRTP() (*rtp.Packet, error) {
	rtp, ok := <-w.rtpCh
	if !ok {
		return nil, errChanClosed
	}
	return rtp, nil
}

// WriteRTP send rtp packet
func (w *WebRTCTransport) WriteRTP(pkt *rtp.Packet) error {
	if pkt == nil {
		return errInvalidPacket
	}
	w.trackLock.RLock()
	track := w.track[pkt.SSRC]
	w.trackLock.RUnlock()

	if track == nil {
		log.Errorf("WebRTCTransport.WriteRTP track==nil pkt.SSRC=%d", pkt.SSRC)
		return errInvalidTrack
	}

	log.Debugf("WebRTCTransport.WriteRTP pkt=%v", pkt)
	err := track.WriteRTP(pkt)
	if err != nil {
		log.Errorf(err.Error())
		w.writeErrCnt++
		return err
	}
	return nil
}

// Close all
func (w *WebRTCTransport) Close() {
	if w.stop {
		return
	}
	log.Infof("WebRTCTransport.Close t.ID()=%v", w.ID())
	// close pc first, otherwise remoteTrack.ReadRTP will be blocked
	w.pc.Close()
	w.stop = true
}

func (w *WebRTCTransport) subReadRTCP() {
	senders := w.pc.GetSenders()
	for i := 0; i < len(senders); i++ {
		go func(i int) {
			for {
				select {
				default:
					if w.stop {
						return
					}

					pkt, err := senders[i].ReadRTCP()
					if err != nil {
						if err == io.EOF {
							return
						}
						log.Errorf("rtcp err => %v", err)
					}

					for i := 0; i < len(pkt); i++ {
						w.rtcpCh <- pkt[i]
					}
				}
			}
		}(i)
	}
}

// SSRCPT get SSRC and PayloadType
func (w *WebRTCTransport) SSRCPT() map[uint32]uint8 {
	w.ssrcPTLock.RLock()
	defer w.ssrcPTLock.RUnlock()
	return w.ssrcPT
}

// WriteRTCP write rtcp packet to pc
func (w *WebRTCTransport) WriteRTCP(pkt rtcp.Packet) error {
	if w.pc == nil {
		return errInvalidPC
	}
	return w.pc.WriteRTCP([]rtcp.Packet{pkt})
}

// WriteErrTotal return write error
func (w *WebRTCTransport) WriteErrTotal() int {
	return w.writeErrCnt
}

// WriteErrReset reset write error
func (w *WebRTCTransport) WriteErrReset() {
	w.writeErrCnt = 0
}

// GetRTCPChan return a rtcp channel
func (w *WebRTCTransport) GetRTCPChan() chan rtcp.Packet {
	return w.rtcpCh
}
