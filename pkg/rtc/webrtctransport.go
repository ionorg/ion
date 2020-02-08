package rtc

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
	cfg webrtc.Configuration

	errChanClosed    = errors.New("channel closed")
	errInvalidTrack  = errors.New("track is nil")
	errInvalidPacket = errors.New("packet is nil")
	errInvalidPC     = errors.New("pc is nil")
)

func initICE(ices []string) {
	cfg = webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: ices,
			},
		},
	}
}

// WebRTCTransport ..
type WebRTCTransport struct {
	id          string
	pc          *webrtc.PeerConnection
	track       map[uint32]*webrtc.Track
	trackLock   sync.RWMutex
	stop        bool
	rtpCh       chan *rtp.Packet
	ssrcPT      map[uint32]uint8
	ssrcPTLock  sync.RWMutex
	byteRate    uint64
	hasVideo    bool
	hasAudio    bool
	hasScreen   bool
	writeErrCnt int
}

func newWebRTCTransport(id string) *WebRTCTransport {
	w := &WebRTCTransport{
		id:     id,
		track:  make(map[uint32]*webrtc.Track),
		rtpCh:  make(chan *rtp.Packet, 1000),
		ssrcPT: make(map[uint32]uint8),
	}

	return w
}

// ID return id
func (t *WebRTCTransport) ID() string {
	return t.id
}

// AnswerPublish answer to pub
func (t *WebRTCTransport) AnswerPublish(rid string, offer webrtc.SessionDescription, options map[string]interface{}, fn func(ssrc uint32, pt uint8)) (answer webrtc.SessionDescription, err error) {
	if options == nil {
		return webrtc.SessionDescription{}, errors.New("invalid options")
	}
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// only register one video codec which client need
	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBTransportCC,
		},
	}
	if codec, ok := options["codec"]; ok {
		codecStr := codec.(string)
		if strings.EqualFold(codecStr, "h264") {
			mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb))
		} else if strings.EqualFold(codecStr, "vp9") {
			mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
		} else {
			// mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb))
			mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
		}
	}

	//check video audio screen
	if v, ok := options["video"].(bool); ok {
		t.hasVideo = v
	}
	if a, ok := options["audio"].(bool); ok {
		t.hasAudio = a
	}
	if s, ok := options["screen"].(bool); ok {
		t.hasScreen = s
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	t.pc, err = api.NewPeerConnection(cfg)
	if err != nil {
		log.Errorf("api.NewPeerConnection %v", err)
		return webrtc.SessionDescription{}, err
	}

	// Allow us to receive 1 video track
	_, err = t.pc.AddTransceiver(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("pc.AddTransceiver video %v", err)
		return webrtc.SessionDescription{}, err
	}

	// Allow us to receive 1 audio track
	_, err = t.pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("pc.AddTransceiver audio %v", err)
		return webrtc.SessionDescription{}, err
	}

	t.pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		t.ssrcPTLock.Lock()
		t.ssrcPT[remoteTrack.SSRC()] = remoteTrack.PayloadType()
		t.ssrcPTLock.Unlock()
		fn(remoteTrack.SSRC(), remoteTrack.PayloadType())
		t.receiveRTP(remoteTrack)
	})

	err = t.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Errorf("pc.SetRemoteDescription %v", err)
		return webrtc.SessionDescription{}, err
	}

	answer, err = t.pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("SetLocalDescription answer=%v err=%v", answer, err)
	}
	err = t.pc.SetLocalDescription(answer)
	//TODO recently not use, fix panic?
	// t.pubReceiveRTCP()

	if err != nil {
		log.Errorf("SetLocalDescription answer=%v err=%v", answer, err)
	}
	return answer, err
}

// AnswerSubscribe answer to sub
func (t *WebRTCTransport) AnswerSubscribe(offer webrtc.SessionDescription, ssrcPT map[uint32]uint8, mid string) (answer webrtc.SessionDescription, err error) {
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBTransportCC,
		},
	}
	// mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb))
	// mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb))

	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	t.pc, err = api.NewPeerConnection(cfg)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	var track *webrtc.Track
	for ssrc, pt := range ssrcPT {
		if pt == webrtc.DefaultPayloadTypeVP8 ||
			pt == webrtc.DefaultPayloadTypeVP9 ||
			pt == webrtc.DefaultPayloadTypeH264 {
			track, _ = t.pc.NewTrack(pt, ssrc, "video", "pion")
		} else {
			track, _ = t.pc.NewTrack(pt, ssrc, "audio", "pion")
		}
		if track != nil {
			t.pc.AddTrack(track)
			t.trackLock.Lock()
			t.track[ssrc] = track
			t.trackLock.Unlock()
		}
	}

	err = t.pc.SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = t.pc.CreateAnswer(nil)
	err = t.pc.SetLocalDescription(answer)
	t.subReadRTCP(mid)
	return answer, err
}

func (t *WebRTCTransport) receiveRTP(remoteTrack *webrtc.Track) {
	for {
		if t.stop {
			return
		}

		rtp, err := remoteTrack.ReadRTP()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Errorf("rtp err => %v", err)
		}
		t.rtpCh <- rtp
	}
}

// ReadRTP read rtp packet
func (t *WebRTCTransport) ReadRTP() (*rtp.Packet, error) {
	rtp, ok := <-t.rtpCh
	if !ok {
		return nil, errChanClosed
	}
	return rtp, nil
}

// WriteRTP send rtp packet
func (t *WebRTCTransport) WriteRTP(pkt *rtp.Packet) error {
	if pkt == nil {
		return errInvalidPacket
	}
	t.trackLock.RLock()
	track := t.track[pkt.SSRC]
	t.trackLock.RUnlock()

	if track == nil {
		log.Errorf("WebRTCTransport.WriteRTP track==nil pkt.SSRC=%d", pkt.SSRC)
		return errInvalidTrack
	}

	log.Debugf("WebRTCTransport.WriteRTP pkt=%v", pkt)
	// log.Infof("rtp.Extension=%t rtp.ExtensionProfile=%x rtp.ExtensionPayload=%x", pkt.Extension, pkt.ExtensionProfile, pkt.ExtensionPayload)
	err := track.WriteRTP(pkt)
	if err != nil {
		log.Errorf(err.Error())
		t.writeErrCnt++
		return err
	}
	return nil
}

// Close all
func (t *WebRTCTransport) Close() {
	if t.stop {
		return
	}
	log.Infof("WebRTCTransport.Close t.ID()=%v", t.ID())
	// close pc first, otherwise remoteTrack.ReadRTP will be blocked
	t.pc.Close()
	t.stop = true
}

func (t *WebRTCTransport) subReadRTCP(mid string) {
	senders := t.pc.GetSenders()
	for i := 0; i < len(senders); i++ {
		go func(i int) {
			for {
				select {
				default:
					if t.stop {
						return
					}

					p := getPipeline(mid)
					if p == nil {
						continue
					}

					pkt, err := senders[i].ReadRTCP()
					if err != nil {
						if err == io.EOF {
							return
						}
						log.Errorf("rtcp err => %v", err)
					}

					for i := 0; i < len(pkt); i++ {
						// log.Infof("pkt[i]=%v", pkt[i])
						switch pkt[i].(type) {
						case *rtcp.TransportLayerNack:
							// log.Infof("WebRTCTransport.subReadRTCP rtcp.TransportLayerNack pkt[i]=%v", pkt[i])
							nack := pkt[i].(*rtcp.TransportLayerNack)
							// for _, nackPair := range nack.Nacks {
							// sns := nackPair.PacketList()
							// for _, sn := range sns {
							// if !p.writeRTP(t.id, nack.MediaSSRC, sn) {
							// // log.Errorf("p.writePacket failed t.id=%v sn=%v", t.id, sn)
							// if pub := p.getPub(); pub != nil {
							// n := &rtcp.TransportLayerNack{
							// //origin ssrc
							// SenderSSRC: nack.SenderSSRC,
							// MediaSSRC:  nack.MediaSSRC,
							// Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: sn}},
							// }
							// pub.WriteRTCP(n)
							// }
							// }
							// }
							// }
							p.getPub().WriteRTCP(nack)
						default:
							p.PushRTCP(pkt[i])
						}
					}
				}
			}
		}(i)
	}
}

// SSRCPT get SSRC and PayloadType
func (t *WebRTCTransport) SSRCPT() map[uint32]uint8 {
	t.ssrcPTLock.RLock()
	defer t.ssrcPTLock.RUnlock()
	return t.ssrcPT
}

func (t *WebRTCTransport) WriteRTCP(pkt rtcp.Packet) error {
	if t.pc == nil {
		return errInvalidPC
	}
	return t.pc.WriteRTCP([]rtcp.Packet{pkt})
}

func (t *WebRTCTransport) writeErrTotal() int {
	return t.writeErrCnt
}

func (t *WebRTCTransport) writeErrReset() {
	t.writeErrCnt = 0
}
