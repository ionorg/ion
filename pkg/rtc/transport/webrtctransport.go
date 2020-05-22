package transport

import (
	"errors"
	"io"

	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	maxChanSize = 100
	IOSH264Fmtp = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"
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

// InitWebRTC init WebRTCTransport setting
func InitWebRTC(iceServers []webrtc.ICEServer, icePortStart, icePortEnd uint16) error {
	var err error
	if icePortStart != 0 || icePortEnd != 0 {
		err = setting.SetEphemeralUDPPortRange(icePortStart, icePortEnd)
	}

	cfg.ICEServers = iceServers
	return err
}

// WebRTCTransport contains pc incoming and outgoing tracks
type WebRTCTransport struct {
	mediaEngine  webrtc.MediaEngine
	api          *webrtc.API
	id           string
	pc           *webrtc.PeerConnection
	outTracks    map[uint32]*webrtc.Track
	outTrackLock sync.RWMutex
	inTracks     map[uint32]*webrtc.Track
	inTrackLock  sync.RWMutex
	writeErrCnt  int

	rtpCh             chan *rtp.Packet
	rtcpCh            chan rtcp.Packet
	stop              bool
	pendingCandidates []*webrtc.ICECandidate
	candidateLock     sync.RWMutex
	candidateCh       chan *webrtc.ICECandidate
	alive             bool
	bandwidth         int
	isPub             bool
	shutdownChan      chan string
}

func (w *WebRTCTransport) SetShutdownChan(ch chan string) {
	w.shutdownChan = ch
}

func (w *WebRTCTransport) init(options map[string]interface{}) error {
	w.mediaEngine = webrtc.MediaEngine{}
	w.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBGoogREMB,
		},
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBCCM,
		},
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBNACK,
		},
		webrtc.RTCPFeedback{
			Type: "nack pli",
		},
	}
	publish := KvOK(options, "publish", "true")
	tcc := KvOK(options, "transport-cc", "true")
	dc := KvOK(options, "data-channel", "true")
	codec := GetUpperString(options, "codec")
	bandwidth, err := GetInt(options, "bandwidth")
	if err == nil {
		if publish {
			w.bandwidth = bandwidth
		}
	}

	if tcc {
		rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBTransportCC,
		})
	}

	if publish {
		if codec == webrtc.H264 {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
		} else if codec == webrtc.VP8 {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
		} else if codec == webrtc.VP9 {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
		} else {
			w.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
		}
	} else {
		w.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(webrtc.DefaultPayloadTypeH264, 90000, rtcpfb, IOSH264Fmtp))
		w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))
		w.mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
	}

	if !dc {
		setting.DetachDataChannels()
	}
	w.api = webrtc.NewAPI(webrtc.WithMediaEngine(w.mediaEngine), webrtc.WithSettingEngine(setting))
	return nil
}

// NewWebRTCTransport create a WebRTCTransport
// options:
//   "video" = webrtc.H264[default] webrtc.VP8  webrtc.VP9
//   "audio" = webrtc.Opus[default] webrtc.PCMA webrtc.PCMU webrtc.G722
//   "transport-cc"  = "true" or "false"[default]
//   "data-channel"  = "true" or "false"[default]
func NewWebRTCTransport(id string, options map[string]interface{}) *WebRTCTransport {
	w := &WebRTCTransport{
		id:          id,
		outTracks:   make(map[uint32]*webrtc.Track),
		inTracks:    make(map[uint32]*webrtc.Track),
		rtpCh:       make(chan *rtp.Packet, maxChanSize),
		rtcpCh:      make(chan rtcp.Packet, maxChanSize),
		candidateCh: make(chan *webrtc.ICECandidate, maxChanSize),
		alive:       true,
	}
	err := w.init(options)
	if err != nil {
		log.Errorf("NewWebRTCTransport init %v", err)
		return nil
	}

	w.pc, err = w.api.NewPeerConnection(cfg)
	if err != nil {
		log.Errorf("NewWebRTCTransport api.NewPeerConnection %v", err)
		return nil
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver video %v", err)
		return nil
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver audio %v", err)
		return nil
	}

	w.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		remoteSDP := w.pc.RemoteDescription()
		if remoteSDP == nil {
			w.candidateLock.Lock()
			defer w.candidateLock.Unlock()
			w.pendingCandidates = append(w.pendingCandidates, c)
			log.Infof("w.pc.OnICECandidate remoteSDP == nil c=%v", c)
		} else {
			log.Infof("w.pc.OnICECandidate remoteSDP != nil c=%v", c)
			w.candidateCh <- c
		}
	})

	w.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		switch connectionState {
		case webrtc.ICEConnectionStateDisconnected:
			log.Errorf("webrtc ice disconnected")
		case webrtc.ICEConnectionStateFailed:
			log.Errorf("webrtc ice failed")
			w.alive = false
			w.shutdownChan <- id
		}
	})

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

// Offer return a offer
func (w *WebRTCTransport) Offer() (webrtc.SessionDescription, error) {
	if w.pc == nil {
		return webrtc.SessionDescription{}, errInvalidPC
	}
	offer, err := w.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	err = w.pc.SetLocalDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	return offer, nil
}

// SetRemoteSDP after Offer()
func (w *WebRTCTransport) SetRemoteSDP(sdp webrtc.SessionDescription) error {
	if w.pc == nil {
		return errInvalidPC
	}
	return w.pc.SetRemoteDescription(sdp)
}

// AddTrack add track to pc
func (w *WebRTCTransport) AddSendTrack(ssrc uint32, pt uint8, streamID string, trackID string) (*webrtc.Track, error) {
	if w.pc == nil {
		return nil, errInvalidPC
	}
	track, err := w.pc.NewTrack(pt, ssrc, trackID, streamID)
	if err != nil {
		return nil, err
	}

	_, err = w.pc.AddTransceiverFromTrack(track, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly})
	if err != nil {
		return nil, err
	}

	w.outTrackLock.Lock()
	w.outTracks[ssrc] = track
	w.outTrackLock.Unlock()
	return track, nil
}

// AddCandidate add candidate to pc
func (w *WebRTCTransport) AddCandidate(candidate string) error {
	if w.pc == nil {
		return errInvalidPC
	}

	err := w.pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)})
	if err != nil {
		return err
	}
	return nil
}

// Answer answer to pub or sub
func (w *WebRTCTransport) Answer(offer webrtc.SessionDescription, options map[string]interface{}) (webrtc.SessionDescription, error) {
	w.isPub = KvOK(options, "publish", "true")
	if w.isPub {
		w.pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
			w.inTrackLock.Lock()
			w.inTracks[remoteTrack.SSRC()] = remoteTrack
			w.inTrackLock.Unlock()
			// TODO replace with broadcast when receiving rtp failed
			// etcdKeepFunc(remoteTrack.SSRC(), remoteTrack.PayloadType())
			w.receiveInTrackRTP(remoteTrack)
		})
	} else {
		ssrcPT := options["ssrcpt"]
		if ssrcPT == nil {
			return webrtc.SessionDescription{}, errInvalidOptions
		}
		ssrcPTMap, _ := ssrcPT.(map[uint32]uint8)
		if len(ssrcPTMap) == 0 {
			return webrtc.SessionDescription{}, errInvalidOptions
		}

		for ssrc, pt := range ssrcPTMap {
			if _, found := w.outTracks[ssrc]; !found {
				track, _ := w.pc.NewTrack(pt, ssrc, "pion", "pion")
				if track != nil {
					_, err := w.pc.AddTrack(track)
					if err == nil {
						w.outTrackLock.Lock()
						w.outTracks[ssrc] = track
						w.outTrackLock.Unlock()
					} else {
						log.Errorf("w.pc.AddTrack err=%v", err)
					}
				}
			}
		}
		w.receiveOutTrackRTCP()
	}

	err := w.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Errorf("pc.SetRemoteDescription %v", err)
		return webrtc.SessionDescription{}, err
	}

	answer, err := w.pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("pc.CreateAnswer answer=%v err=%v", answer, err)
		return webrtc.SessionDescription{}, err
	}

	err = w.pc.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("pc.SetLocalDescription answer=%v err=%v", answer, err)
	}
	go func() {
		w.candidateLock.Lock()
		defer w.candidateLock.Unlock()
		for _, candidate := range w.pendingCandidates {
			log.Infof("WebRTCTransport.Answer candidate=%v", candidate)
			w.candidateCh <- candidate
		}
		w.pendingCandidates = nil
	}()
	return answer, err
}

// receiveInTrackRTP receive all incoming tracks' rtp and sent to one channel
func (w *WebRTCTransport) receiveInTrackRTP(remoteTrack *webrtc.Track) {
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

// WriteRTP send rtp packet to outgoing tracks
func (w *WebRTCTransport) WriteRTP(pkt *rtp.Packet) error {
	if pkt == nil {
		return errInvalidPacket
	}
	w.outTrackLock.RLock()
	track := w.outTracks[pkt.SSRC]
	w.outTrackLock.RUnlock()

	if track == nil {
		log.Errorf("WebRTCTransport.WriteRTP track==nil pkt.SSRC=%d", pkt.SSRC)
		return errInvalidTrack
	}

	log.Debugf("WebRTCTransport.WriteRTP pkt=%v", pkt)
	err := track.WriteRTP(pkt)
	if err != nil {
		log.Errorf("WebRTCTransport.WriteRTP => %s", err.Error())
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

// receive rtcp from outgoing tracks
func (w *WebRTCTransport) receiveOutTrackRTCP() {
	go func() {
		for _, sender := range w.pc.GetSenders() {
			for {
				if w.stop {
					return
				}

				pkts, err := sender.ReadRTCP()
				if err != nil {
					if err == io.EOF {
						return
					}
					log.Errorf("rtcp err => %v", err)
				}

				for _, pkt := range pkts {
					w.rtcpCh <- pkt
				}
			}

		}
	}()
}

// GetInTracks return incoming tracks
func (w *WebRTCTransport) GetInTracks() map[uint32]*webrtc.Track {
	w.inTrackLock.RLock()
	defer w.inTrackLock.RUnlock()
	return w.inTracks
}

// GetOutTracks return incoming tracks
func (w *WebRTCTransport) GetOutTracks() map[uint32]*webrtc.Track {
	w.outTrackLock.RLock()
	defer w.outTrackLock.RUnlock()
	return w.outTracks
}

// WriteRTCP write rtcp packet to pc
func (w *WebRTCTransport) WriteRTCP(pkt rtcp.Packet) error {
	if w.pc == nil {
		return errInvalidPC
	}
	// log.Infof("WebRTCTransport.WriteRTCP pkt=%+v", pkt)
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

// GetCandidateChan return a candidate channel
func (w *WebRTCTransport) GetCandidateChan() chan *webrtc.ICECandidate {
	return w.candidateCh
}

// GetBandwidth return bandwidth
func (w *WebRTCTransport) GetBandwidth() int {
	return w.bandwidth
}
