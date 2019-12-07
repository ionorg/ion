package rtc

import (
	"errors"
	"io"
	"strings"
	"time"

	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
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

type WebRTCTransport struct {
	id           string
	pc           *webrtc.PeerConnection
	track        map[uint32]*webrtc.Track
	trackLock    sync.RWMutex
	notify       chan struct{}
	pli          chan int
	rtpCh        chan *rtp.Packet
	wg           sync.WaitGroup
	ssrcPT       map[uint32]uint8
	ssrcPTLock   sync.RWMutex
	byteRate     uint64
	isLostPacket bool
	hasVideo     bool
	hasAudio     bool
	hasScreen    bool
}

func newWebRTCTransport(id string) *WebRTCTransport {
	w := &WebRTCTransport{
		id:     id,
		track:  make(map[uint32]*webrtc.Track),
		notify: make(chan struct{}),
		pli:    make(chan int),
		rtpCh:  make(chan *rtp.Packet, 1000),
		ssrcPT: make(map[uint32]uint8),
	}

	return w
}

func (t *WebRTCTransport) ID() string {
	return t.id
}

func (t *WebRTCTransport) AnswerPublish(rid string, offer webrtc.SessionDescription, options map[string]interface{}, fn func(ssrc uint32, pt uint8)) (answer webrtc.SessionDescription, err error) {
	if options == nil {
		return webrtc.SessionDescription{}, errors.New("invalid options")
	}
	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// only register one video codec which client need
	if codec, ok := options["codec"]; ok {
		codecStr := codec.(string)
		if strings.EqualFold(codecStr, "h264") {
			mediaEngine.RegisterCodec(webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000))
		} else if strings.EqualFold(codecStr, "vp9") {
			mediaEngine.RegisterCodec(webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000))
		} else {
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
		t.ssrcPTLock.Lock()
		t.ssrcPT[remoteTrack.SSRC()] = remoteTrack.PayloadType()
		t.ssrcPTLock.Unlock()
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			t.wg.Add(1)
			go func() {
				for {
					select {
					case <-t.pli:
						log.Debugf("WebRTCTransport.AnswerPublish WriteRTCP PLI %v", remoteTrack.SSRC())
						t.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{SenderSSRC: remoteTrack.SSRC(), MediaSSRC: remoteTrack.SSRC()}})
					case <-t.notify:
						t.wg.Done()
						return
					}
				}
			}()
			fn(remoteTrack.SSRC(), remoteTrack.PayloadType())
			t.receiveRTP(remoteTrack)
		} else {
			fn(remoteTrack.SSRC(), remoteTrack.PayloadType())
			t.receiveRTP(remoteTrack)
		}
	})

	err = t.pc.SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	answer, err = t.pc.CreateAnswer(nil)
	err = t.pc.SetLocalDescription(answer)
	//TODO recently not use, fix panic?
	// t.pubReceiveRTCP()

	t.sendPLI()
	return answer, err
}

func (t *WebRTCTransport) AnswerSubscribe(offer webrtc.SessionDescription, ssrcPT map[uint32]uint8, pid string) (answer webrtc.SessionDescription, err error) {

	mediaEngine := webrtc.MediaEngine{}
	mediaEngine.RegisterDefaultCodecs()
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
	t.SubReceiveRTCP(pid)
	return answer, err
}

func (t *WebRTCTransport) sendPLI() {
	if t.hasVideo || t.hasScreen {
		go func() {
			ticker := time.NewTicker(pliDuration)
			defer ticker.Stop()
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
}

func (t *WebRTCTransport) receiveRTP(remoteTrack *webrtc.Track) {
	t.wg.Add(1)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	total := uint64(0)
	for {
		select {
		case <-t.notify:
			t.wg.Done()
			return
		case <-ticker.C:
			t.byteRate = total / 3
			total = 0
			t.isLostPacket = false
		default:
			rtp, err := remoteTrack.ReadRTP()
			if err != nil {
				if err == io.EOF {
					t.wg.Done()
					return
				}
				log.Errorf("rtp err => %v", err)
			}
			total += uint64(rtp.MarshalSize())
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
	if pkt == nil {
		return errInvalidPacket
	}
	t.trackLock.RLock()
	track := t.track[pkt.SSRC]
	t.trackLock.RUnlock()
	if track != nil {
		log.Debugf("WebRTCTransport.WriteRTP pkt=%v", pkt)
		return track.WriteRTP(pkt)
	}
	log.Errorf("WebRTCTransport.WriteRTP track==nil pkt.SSRC=%d", pkt.SSRC)
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

func (t *WebRTCTransport) pubReceiveRTCP() {
	receivers := t.pc.GetReceivers()
	for i := 0; i < len(receivers); i++ {
		t.wg.Add(1)
		go func(i int) {
			for {
				select {
				case <-t.notify:
					t.wg.Done()
					return
				default:
					pkt, err := receivers[i].ReadRTCP()
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
							// SendPLI(t.id)
						case *rtcp.TransportLayerNack:
							log.Debugf("pub rtcp.TransportLayerNack pkt[i]=%v", pkt[i])
							// nack := pkt[i].(*rtcp.TransportLayerNack)
							// for _, nackPair := range nack.Nacks {
							// // sns := util.GetLostSN(nackPair.PacketID, uint16(nackPair.LostPackets))
							// sns := nackPair.PacketList()
							// for _, sn := range sns {
							// if !getPipeline(t.id).WritePacket(t.id, nack.MediaSSRC, sn) {
							// n := &rtcp.TransportLayerNack{
							// //origin ssrc
							// SenderSSRC: nack.SenderSSRC,
							// MediaSSRC:  nack.MediaSSRC,
							// Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: sn}},
							// }
							// log.Infof("sendNack to ion %v", n)
							// getPipeline(t.id).GetPub().sendNack(n)
							// }
							// }
							// }
						case *rtcp.ReceiverEstimatedMaximumBitrate:
						case *rtcp.ReceiverReport:
						case *rtcp.SenderReport:
							log.Debugf("pub rtcp.ReceiverReport = %+v", pkt[i])
							rr := pkt[i].(*rtcp.SenderReport)
							for _, report := range rr.Reports {
								log.Debugf("report=%+v", report)
							}

						default:
							log.Debugf("rtcp type = %v", pkt[i])
						}
					}
				}
			}
		}(i)
	}
}

func (t *WebRTCTransport) SubReceiveRTCP(pid string) {
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
							nack := pkt[i].(*rtcp.TransportLayerNack)
							for _, nackPair := range nack.Nacks {
								sns := nackPair.PacketList()
								for _, sn := range sns {
									if !getPipeline(pid).writePacket(t.id, nack.MediaSSRC, sn) {
										n := &rtcp.TransportLayerNack{
											//origin ssrc
											SenderSSRC: nack.SenderSSRC,
											MediaSSRC:  nack.MediaSSRC,
											Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: sn}},
										}
										log.Debugf("sendNack to pub %v", n)
										getPipeline(pid).getPub().sendNack(n)
									}
								}
							}
						case *rtcp.ReceiverEstimatedMaximumBitrate:
						case *rtcp.ReceiverReport:
						default:
							log.Debugf("WebRTCTransport.SubReceiveRTCP rtcp type = %v", pkt[i])
						}
					}
				}
			}
		}(i)
	}
}

func (t *WebRTCTransport) SsrcPT() map[uint32]uint8 {
	t.ssrcPTLock.RLock()
	defer t.ssrcPTLock.RUnlock()
	return t.ssrcPT
}

func (t *WebRTCTransport) sendNack(nack *rtcp.TransportLayerNack) {
	if t.pc == nil {
		return
	}
	t.isLostPacket = true
	t.pc.WriteRTCP([]rtcp.Packet{nack})
}

func (t *WebRTCTransport) sendREMB(lostRate float64) {
	if lostRate > 1 || lostRate < 0 {
		return
	}
	var videoSSRC uint32
	t.trackLock.RLock()
	for ssrc, track := range t.track {
		if track.PayloadType() == webrtc.DefaultPayloadTypeVP8 ||
			track.PayloadType() == webrtc.DefaultPayloadTypeH264 ||
			track.PayloadType() == webrtc.DefaultPayloadTypeVP9 {
			videoSSRC = ssrc
		}
	}
	t.trackLock.RUnlock()

	var bw uint64
	if lostRate == 0 && t.byteRate == 0 {
		bw = rembHighBW
	} else if lostRate >= 0 && lostRate < 0.1 {
		bw = t.byteRate * 2
	} else {
		bw = uint64(float64(t.byteRate) * (1 - lostRate))
	}

	if bw < rembLowBW {
		bw = rembLowBW
	}

	if bw > rembHighBW {
		bw = rembHighBW
	}

	log.Debugf("WebRTCTransport.sendREMB lostRate=%v bw=%v", lostRate, bw*8)
	remb := &rtcp.ReceiverEstimatedMaximumBitrate{
		SenderSSRC: videoSSRC,
		Bitrate:    bw * 8,
		SSRCs:      []uint32{videoSSRC},
	}
	t.pc.WriteRTCP([]rtcp.Packet{remb})
}
