package rtc

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/pion/ion/log"
	"github.com/pion/ion/rtc/mux"
	"github.com/pion/ion/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	receiveMTU  = 8192
	extSentInit = 30
)

type RTPTransport struct {
	rtpSession      *SessionRTP
	rtcpSession     *SessionRTCP
	rtpEndpoint     *mux.Endpoint
	rtcpEndpoint    *mux.Endpoint
	conn            net.Conn
	mux             *mux.Mux
	rtpCh           chan *rtp.Packet
	payloadSSRC     map[uint8]uint32
	payloadSSRCLock sync.RWMutex
	notify          chan struct{}
	extSent         int
	id              string
	pid             string
	idLock          sync.RWMutex
	addr            string
}

func newRTPTransport(conn net.Conn) *RTPTransport {
	t := &RTPTransport{
		conn:        conn,
		rtpCh:       make(chan *rtp.Packet, 1000),
		notify:      make(chan struct{}),
		payloadSSRC: make(map[uint8]uint32),
		extSent:     extSentInit,
	}
	config := mux.Config{
		Conn:       conn,
		BufferSize: receiveMTU,
	}
	t.mux = mux.NewMux(config)
	t.rtpEndpoint = t.newEndpoint(mux.MatchRTP)
	t.rtcpEndpoint = t.newEndpoint(mux.MatchRTCP)
	var err error
	t.rtpSession, err = NewSessionRTP(t.rtpEndpoint)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	t.rtcpSession, err = NewSessionRTCP(t.rtcpEndpoint)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	return t
}

func newPubRTPTransport(id, pid, addr string) *RTPTransport {
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
	t.pid = pid
	t.addr = addr
	t.receiveRTCP()
	log.Infof("NewSubRTPTransport %s %d", ip, port)
	return t
}

func (t *RTPTransport) ID() string {
	return t.id
}

func (t *RTPTransport) Close() {
	close(t.notify)
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
			select {
			case <-t.notify:
				log.Infof("ReceiveRTP case  <-t.notify!!!!!!!!")
				return
			default:
				readStream, ssrc, err := t.rtpSession.AcceptStream()
				if err != nil {
					log.Warnf("Failed to accept stream %v ", err)
					return
				}
				go func() {
					rtpBuf := make([]byte, receiveMTU)
					for {
						_, pkt, err := readStream.ReadRTP(rtpBuf)
						if err != nil {
							log.Warnf("Failed to read rtp %v %d ", err, ssrc)
							return
						}
						if t.getPID() == "" {
							t.idLock.Lock()
							t.pid = util.GetIDFromRTP(pkt)
							t.idLock.Unlock()
						}
						t.rtpCh <- pkt
						t.payloadSSRCLock.Lock()
						t.payloadSSRC[pkt.Header.PayloadType] = pkt.Header.SSRC
						t.payloadSSRCLock.Unlock()

						log.Debugf("got RTP: %+v", pkt.Header)
					}
				}()
			}
		}
	}()
}

func (t *RTPTransport) ReadRTP() (*rtp.Packet, error) {
	return <-t.rtpCh, nil
}

// rtp pub receive rtcp
func (t *RTPTransport) receiveRTCP() {
	go func() {
		for {
			select {
			case <-t.notify:
				log.Infof("ReceiveRTCP case  <-t.notify!!!!!!!!")
				return
			default:
				readStream, ssrc, err := t.rtcpSession.AcceptStream()
				if err != nil {
					log.Warnf("Failed to accept RTCP %v ", err)
					return
				}

				go func() {
					rtcpBuf := make([]byte, receiveMTU)
					for {
						_, header, err := readStream.ReadRTCP(rtcpBuf)
						if err != nil {
							log.Warnf("Failed to read rtcp %v %d ", err, ssrc)
							return
						}
						log.Debugf("got RTCP: %+v", header)
						switch header.Type {
						case rtcp.TypePayloadSpecificFeedback:
							if header.Count == rtcp.FormatPLI {
								//send a RTP PLI packet
								// log.Infof("got pli TODO pipeline send key frame!")
								// getPipeline(t.pid).SendKeyFrame(t.id)
								// getPipeline(t.pid).SendPLI()
							}
						}
					}
				}()
			}
		}
	}()
}

func (t *RTPTransport) WriteRTP(rtp *rtp.Packet) error {
	writeStream, err := t.rtpSession.OpenWriteStream()
	if err != nil {
		return err
	}
	if t.extSent > 0 {
		util.SetIDToRTP(rtp, t.pid)
	}

	_, err = writeStream.WriteRTP(&rtp.Header, rtp.Payload)
	if err == nil && t.extSent > 0 {
		t.extSent--
	}
	return err
}

func (t *RTPTransport) WriteRawRTCP(data []byte) (int, error) {
	writeStream, err := t.rtcpSession.OpenWriteStream()
	if err != nil {
		return 0, err
	}
	return writeStream.WriteRawRTCP(data)
}

func (t *RTPTransport) WriteRTCP(header *rtcp.Header, payload []byte) (int, error) {
	writeStream, err := t.rtcpSession.OpenWriteStream()
	if err != nil {
		return 0, err
	}
	return writeStream.WriteRTCP(header, payload)
}

// used by rtp pub, tell remote ion to send key frame
func (t *RTPTransport) sendPLI() {
	t.payloadSSRCLock.RLock()
	ssrc := t.payloadSSRC[webrtc.DefaultPayloadTypeVP8]
	if ssrc == 0 {
		ssrc = t.payloadSSRC[webrtc.DefaultPayloadTypeH264]
	}
	if ssrc == 0 {
		ssrc = t.payloadSSRC[webrtc.DefaultPayloadTypeVP9]
	}
	log.Infof("RTPTransport.SendPLI ssrc=%d payloadSSRC=%v", ssrc, t.payloadSSRC)
	t.payloadSSRCLock.RUnlock()
	pli := rtcp.PictureLossIndication{MediaSSRC: ssrc}
	data, err := pli.Marshal()
	if err != nil {
		log.Warnf("pli marshal failed: %v", err)
		return
	}
	t.WriteRawRTCP(data)
}

// PeekPayloadSSRC playload type and ssrc
func (t *RTPTransport) PayloadSSRC() map[uint8]uint32 {
	t.payloadSSRCLock.RLock()
	defer t.payloadSSRCLock.RUnlock()
	return t.payloadSSRC
}

func (t *RTPTransport) getPID() string {
	t.idLock.RLock()
	defer t.idLock.RUnlock()
	return t.pid
}

func (t *RTPTransport) getAddr() string {
	return t.addr
}

func (t *RTPTransport) ResetExtSent() {
	t.extSent = extSentInit
}
