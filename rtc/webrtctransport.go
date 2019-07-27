package rtc

import (
	"errors"
	"io"
	"time"

	"sync"

	"github.com/pion/ion/conf"
	"github.com/pion/ion/gslb"
	"github.com/pion/ion/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	pliDuration = 1 * time.Second
)

var (
	cfg         webrtc.Configuration
	mediaEngine webrtc.MediaEngine
	api         *webrtc.API

	errChanClosed   = errors.New("channel closed")
	errInvalidTrack = errors.New("track not found")
)

func init() {
	cfg = webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: conf.WebRTC.Ices,
			},
		},
	}
	mediaEngine = webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	api = webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
}

type WebRTCTransport struct {
	id              string
	pc              *webrtc.PeerConnection
	track           map[uint32]*webrtc.Track
	trackLock       sync.RWMutex
	notify          chan struct{}
	pli             chan int
	rtpCh           chan *rtp.Packet
	wg              sync.WaitGroup
	payloadSSRC     map[uint8]uint32
	payloadSSRCLock sync.RWMutex
}

func newWebRTCTransport(id string) *WebRTCTransport {
	return &WebRTCTransport{
		id:          id,
		track:       make(map[uint32]*webrtc.Track),
		notify:      make(chan struct{}),
		pli:         make(chan int),
		rtpCh:       make(chan *rtp.Packet, 1000),
		payloadSSRC: make(map[uint8]uint32),
	}
}

func (t *WebRTCTransport) ID() string {
	return t.id
}

func (t *WebRTCTransport) AnswerPublish(rid string, offer webrtc.SessionDescription) (answer webrtc.SessionDescription, err error) {
	t.pc, err = api.NewPeerConnection(cfg)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// Allow us to receive 1 video track
	_, err = t.pc.AddTransceiver(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// Allow us to receive 1 audio track
	_, err = t.pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	t.pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		t.payloadSSRCLock.Lock()
		t.payloadSSRC[remoteTrack.PayloadType()] = remoteTrack.SSRC()
		t.payloadSSRCLock.Unlock()
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			t.wg.Add(1)
			go func() {
				for {
					select {
					case <-t.pli:
						// log.Infof("WriteRTCP PLI %v", remoteTrack.SSRC())
						t.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}})
					case <-t.notify:
						log.Infof("OnTrack t.wg.Done()")
						t.wg.Done()
						return
					}
				}
			}()
			gslb.KeepMediaInfo(rid, t.ID(), remoteTrack.PayloadType(), remoteTrack.SSRC())
			t.receiveRTP(remoteTrack)
		} else {
			// t.AudioTrack, err = t.pc.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "audio", remoteTrack.Label())
			gslb.KeepMediaInfo(rid, t.ID(), remoteTrack.PayloadType(), remoteTrack.SSRC())
			t.receiveRTP(remoteTrack)
		}
	})

	err = t.pc.SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = t.pc.CreateAnswer(nil)
	err = t.pc.SetLocalDescription(answer)
	return answer, err

}

func (t *WebRTCTransport) AnswerSubscribe(offer webrtc.SessionDescription, payloadSSRC map[uint8]uint32, pid string) (answer webrtc.SessionDescription, err error) {
	t.pc, err = api.NewPeerConnection(cfg)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	log.Infof("payloadSSRC=%v", payloadSSRC)
	var track *webrtc.Track
	for payloadType, ssrc := range payloadSSRC {
		if payloadType == webrtc.DefaultPayloadTypeVP8 ||
			payloadType == webrtc.DefaultPayloadTypeVP9 ||
			payloadType == webrtc.DefaultPayloadTypeH264 {
			track, _ = t.pc.NewTrack(payloadType, ssrc, "video", "pion")
			// t.track[ssrc], _ = t.pc.NewTrack(payloadType, ssrc, "pion", "")
		} else {
			track, _ = t.pc.NewTrack(payloadType, ssrc, "audio", "pion")
			// t.track[ssrc], _ = t.pc.NewTrack(payloadType, ssrc, "pion", "")
		}
		t.pc.AddTrack(track)
		t.trackLock.Lock()
		t.track[ssrc] = track
		t.trackLock.Unlock()
	}

	//track is ready
	err = t.pc.SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = t.pc.CreateAnswer(nil)
	err = t.pc.SetLocalDescription(answer)
	// TODO nack
	// t.receiveRTCP(pid)
	return answer, err
}

func (t *WebRTCTransport) sendPLI() {
	go func() {
		ticker := time.NewTicker(pliDuration)
		t.wg.Add(1)
		for {
			select {
			case <-ticker.C:
				t.pli <- 1
			case <-t.notify:
				t.wg.Done()
				return
			}
		}
	}()
}

func (t *WebRTCTransport) receiveRTP(remoteTrack *webrtc.Track) {
	t.wg.Add(1)
	for {
		select {
		case <-t.notify:
			t.wg.Done()
			return
		default:
			rtp, err := remoteTrack.ReadRTP()
			if err != nil {
				if err == io.EOF {
					t.wg.Done()
					return
				}
				log.Errorf("rtp err => %v", err)
			}
			t.rtpCh <- rtp
		}
	}
}

func (t *WebRTCTransport) ReadRTP() (*rtp.Packet, error) {
	rtp, ok := <-t.rtpCh
	if !ok {
		return nil, errChanClosed
	}
	return rtp, nil
}

func (t *WebRTCTransport) WriteRTP(pkt *rtp.Packet) error {
	t.trackLock.RLock()
	track := t.track[pkt.SSRC]
	t.trackLock.RUnlock()
	if track != nil {
		track.WriteRTP(pkt)
		return nil
	}
	log.Errorf("t.track=%v", t.track)
	log.Errorf("WebRTCTransport.WriteRTP pkt.ssrc=%d err", pkt.SSRC)
	return errInvalidTrack
}

func (t *WebRTCTransport) Close() {
	// close pc first, otherwise remoteTrack.ReadRTP will be blocked
	t.pc.Close()
	// close notify before rtpCh, otherwise panic: send on closed channel
	close(t.notify)
	t.wg.Wait()
	close(t.rtpCh)
	close(t.pli)
}

func (t *WebRTCTransport) receiveRTCP(pid string) {
	senders := t.pc.GetSenders()
	for i := 0; i < len(senders); i++ {
		t.wg.Add(1)
		go func(i int) {
			for {
				select {
				case <-t.notify:
					t.wg.Done()
					return
				default:
					pkt, err := senders[i].ReadRTCP()
					if err != nil {
						if err == io.EOF {
							t.wg.Done()
							return
						}
						log.Errorf("rtcp err => %v", err)
					}
					for i := 0; i < len(pkt); i++ {
						switch pkt[i].(type) {
						case *rtcp.PictureLossIndication:
							// pub is already sending PLI now
							// SendPLI(pid)
						case *rtcp.TransportLayerNack:
							log.Debugf("rtcp.TransportLayerNack pkt[i]=%v", pkt[i])
						case *rtcp.ReceiverEstimatedMaximumBitrate:
						case *rtcp.ReceiverReport:
						default:
							log.Debugf("rtcp type = %v", pkt[i])
						}
					}
				}
			}
		}(i)
	}
}

func (t *WebRTCTransport) PayloadSSRC() map[uint8]uint32 {
	t.payloadSSRCLock.RLock()
	defer t.payloadSSRCLock.RUnlock()
	return t.payloadSSRC
}
