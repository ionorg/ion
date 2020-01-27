package rtc

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/rtc/rtpengine/muxrtp"
	"github.com/pion/ion/pkg/rtc/rtpengine/muxrtp/mux"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	extSentInit = 30
	receiveMTU  = 8192
)

// RTPTransport ..
type RTPTransport struct {
	rtpSession   *muxrtp.SessionRTP
	rtcpSession  *muxrtp.SessionRTCP
	rtpEndpoint  *mux.Endpoint
	rtcpEndpoint *mux.Endpoint
	conn         net.Conn
	mux          *mux.Mux
	rtpCh        chan *rtp.Packet
	ssrcPT       map[uint32]uint8
	ssrcPTLock   sync.RWMutex
	stop         bool
	extSent      int
	// id == mid if this is a pub
	// id != mid if this is a sub
	id          string
	mid         string
	idLock      sync.RWMutex
	addr        string
	writeErrCnt int
}

func newRTPTransport(conn net.Conn) *RTPTransport {
	t := &RTPTransport{
		conn:    conn,
		rtpCh:   make(chan *rtp.Packet, 1000),
		ssrcPT:  make(map[uint32]uint8),
		extSent: extSentInit,
	}
	config := mux.Config{
		Conn:       conn,
		BufferSize: receiveMTU,
	}
	t.mux = mux.NewMux(config)
	t.rtpEndpoint = t.newEndpoint(mux.MatchRTP)
	t.rtcpEndpoint = t.newEndpoint(mux.MatchRTCP)
	var err error
	t.rtpSession, err = muxrtp.NewSessionRTP(t.rtpEndpoint)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	t.rtcpSession, err = muxrtp.NewSessionRTCP(t.rtcpEndpoint)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	return t
}

func newPubRTPTransport(id, mid, addr string) *RTPTransport {
	n := strings.Index(addr, ":")
	if n == 0 {
		return nil
	}
	ip := addr[:n]
	port, _ := strconv.Atoi(addr[n+1:])

	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: net.ParseIP(ip), Port: port}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	t := newRTPTransport(conn)
	t.id = id
	t.mid = mid
	t.addr = addr
	t.receiveRTCP()
	log.Infof("newSubRTPTransport %s %d", ip, port)
	return t
}

// ID return id
func (t *RTPTransport) ID() string {
	return t.id
}

// Close release all
func (t *RTPTransport) Close() {
	if t.stop {
		return
	}
	t.stop = true
	t.rtpSession.Close()
	t.rtcpSession.Close()
	t.rtpEndpoint.Close()
	t.rtcpEndpoint.Close()
	t.mux.Close()
	t.conn.Close()
}

// newEndpoint registers a new endpoint on the underlying mux.
func (t *RTPTransport) newEndpoint(f mux.MatchFunc) *mux.Endpoint {
	return t.mux.NewEndpoint(f)
}

func (t *RTPTransport) receiveRTP() {
	go func() {
		for {
			readStream, ssrc, err := t.rtpSession.AcceptStream()
			if err != nil {
				log.Warnf("Failed to accept stream %v ", err)
				//for non-blocking ReadRTP()
				t.rtpCh <- nil
				continue
			}
			go func() {
				rtpBuf := make([]byte, receiveMTU)
				for {
					_, pkt, err := readStream.ReadRTP(rtpBuf)
					if err != nil {
						log.Warnf("Failed to read rtp %v %d ", err, ssrc)
						//for non-blocking ReadRTP()
						t.rtpCh <- nil
						continue
						// return
					}
					log.Debugf("RTPTransport.receiveRTP pkt=%v", pkt)
					if t.getMID() == "" {
						t.idLock.Lock()
						t.mid = util.GetIDFromRTP(pkt)
						t.idLock.Unlock()
					}
					t.rtpCh <- pkt
					t.ssrcPTLock.Lock()
					t.ssrcPT[pkt.Header.SSRC] = pkt.Header.PayloadType
					t.ssrcPTLock.Unlock()

					// log.Debugf("got RTP: %+v", pkt.Header)
				}
			}()
		}
	}()
}

// ReadRTP read rtp from transport
func (t *RTPTransport) ReadRTP() (*rtp.Packet, error) {
	return <-t.rtpCh, nil
}

// rtp sub receive rtcp
func (t *RTPTransport) receiveRTCP() {
	go func() {
		for {
			readStream, ssrc, err := t.rtcpSession.AcceptStream()
			if err != nil {
				log.Warnf("Failed to accept RTCP %v ", err)
				return
			}

			go func() {
				rtcpBuf := make([]byte, receiveMTU)
				for {
					rtcps, err := readStream.ReadRTCP(rtcpBuf)
					if err != nil {
						log.Warnf("Failed to read rtcp %v %d ", err, ssrc)
						return
					}
					log.Debugf("got RTCPs: %+v ", rtcps)
					for _, pkt := range rtcps {
						switch pkt.(type) {
						case *rtcp.PictureLossIndication:
							log.Infof("got pli pipeline not need send key frame!")
						case *rtcp.TransportLayerNack:
							log.Debugf("rtptransport got nack: %+v", pkt)
							nack := pkt.(*rtcp.TransportLayerNack)
							for _, nackPair := range nack.Nacks {
								p := getPipeline(t.mid)
								if p != nil {
									if !p.writePacket(t.id, nack.MediaSSRC, nackPair.PacketID) {
										n := &rtcp.TransportLayerNack{
											//origin ssrc
											SenderSSRC: nack.SenderSSRC,
											MediaSSRC:  nack.MediaSSRC,
											Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: nackPair.PacketID}},
										}
										log.Debugf("getPipeline(t.mid).GetPub().sendNack(n) %v", n)
										p.getPub().sendNack(n)
									}
								}
							}
						}
					}
				}
			}()
		}
	}()
}

// WriteRTP send rtp packet
func (t *RTPTransport) WriteRTP(rtp *rtp.Packet) error {
	log.Debugf("RTPTransport.WriteRTP rtp=%v", rtp)
	writeStream, err := t.rtpSession.OpenWriteStream()
	if err != nil {
		t.writeErrCnt++
		return err
	}

	if t.extSent > 0 {
		util.SetIDToRTP(rtp, t.mid)
	}

	_, err = writeStream.WriteRTP(&rtp.Header, rtp.Payload)
	if err == nil && t.extSent > 0 {
		t.extSent--
	}
	if err != nil {
		log.Errorf(err.Error())
		t.writeErrCnt++
	}
	return err
}

// WriteRawRTCP write rtcp data
func (t *RTPTransport) WriteRawRTCP(data []byte) (int, error) {
	writeStream, err := t.rtcpSession.OpenWriteStream()
	if err != nil {
		return 0, err
	}
	return writeStream.WriteRawRTCP(data)
}

// WriteRTCP send rtp header and payload
func (t *RTPTransport) WriteRTCP(header *rtcp.Header, payload []byte) (int, error) {
	writeStream, err := t.rtcpSession.OpenWriteStream()
	if err != nil {
		return 0, err
	}
	return writeStream.WriteRTCP(header, payload)
}

// used by rtp pub, tell remote ion to send key frame
func (t *RTPTransport) sendPLI() {
	t.ssrcPTLock.RLock()
	for ssrc, pt := range t.ssrcPT {
		if pt == webrtc.DefaultPayloadTypeVP8 || pt == webrtc.DefaultPayloadTypeH264 || pt == webrtc.DefaultPayloadTypeVP9 {
			pli := rtcp.PictureLossIndication{MediaSSRC: ssrc}
			data, err := pli.Marshal()
			if err != nil {
				log.Warnf("pli marshal failed: %v", err)
				return
			}
			t.WriteRawRTCP(data)
			log.Infof("RTPTransport.SendPLI ssrc=%d pt=%v", ssrc, pt)
		}
	}
	t.ssrcPTLock.RUnlock()
}

// SSRCPT playload type and ssrc
func (t *RTPTransport) SSRCPT() map[uint32]uint8 {
	t.ssrcPTLock.RLock()
	defer t.ssrcPTLock.RUnlock()
	return t.ssrcPT
}

func (t *RTPTransport) getMID() string {
	t.idLock.RLock()
	defer t.idLock.RUnlock()
	return t.mid
}

func (t *RTPTransport) getAddr() string {
	return t.addr
}

func (t *RTPTransport) sendNack(nack *rtcp.TransportLayerNack) {
	bin, _ := nack.Marshal()
	t.WriteRawRTCP(bin)
}

func (t *RTPTransport) sendREMB(lostRate float64) {
	return
}

func (t *RTPTransport) writeErrTotal() int {
	return t.writeErrCnt
}

func (t *RTPTransport) writeErrReset() {
	t.writeErrCnt = 0
}
