package transport

import (
	"crypto/sha1"
	"errors"
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
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

const (
	extSentInit = 30
	receiveMTU  = 8192
	maxPktSize  = 1024
)

var (
	errInvalidConn = errors.New("invalid conn")
	errInvalidAddr = errors.New("invalid addr")
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
	id           string
	idLock       sync.RWMutex
	writeErrCnt  int
	rtcpCh       chan rtcp.Packet
	bandwidth    int
	shutdownChan chan string
}

func (r *RTPTransport) SetShutdownChan(ch chan string) {
	r.shutdownChan = ch
}

// NewRTPTransport create a RTPTransport by net.Conn
func NewRTPTransport(conn net.Conn) *RTPTransport {
	if conn == nil {
		log.Errorf("NewRTPTransport err=%v", errInvalidConn)
		return nil
	}
	t := &RTPTransport{
		conn:    conn,
		rtpCh:   make(chan *rtp.Packet, maxPktSize),
		ssrcPT:  make(map[uint32]uint8),
		extSent: extSentInit,
		rtcpCh:  make(chan rtcp.Packet, maxPktSize),
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
	t.receiveRTP()
	return t
}

// NewOutRTPTransport new a outgoing RTPTransport
func NewOutRTPTransport(id, addr string) *RTPTransport {
	n := strings.Index(addr, ":")
	if n == 0 {
		log.Errorf("NewOutRTPTransport err=%v", errInvalidAddr)
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
	r := NewRTPTransport(conn)
	r.receiveRTCP()
	log.Infof("NewOutRTPTransport %s %s", id, addr)
	r.idLock.Lock()
	defer r.idLock.Unlock()
	r.id = id
	return r
}

// NewOutRTPTransportWithKCP  new a outgoing RTPTransport by kcp
func NewOutRTPTransportWithKCP(id, addr string, kcpKey, kcpSalt string) *RTPTransport {
	key := pbkdf2.Key([]byte(kcpKey), []byte(kcpSalt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	// dial to the echo server
	conn, err := kcp.DialWithOptions(addr, block, 10, 3)
	if err != nil {
		log.Errorf("NewOutRTPTransportWithKCP err=%v", err)
	}
	r := NewRTPTransport(conn)
	r.receiveRTCP()
	log.Infof("NewOutRTPTransportWithKCP %s %s", id, addr)
	r.idLock.Lock()
	defer r.idLock.Unlock()
	r.id = id
	return r
}

// ID return id
func (r *RTPTransport) ID() string {
	r.idLock.RLock()
	defer r.idLock.RUnlock()
	return r.id
}

// Type return type of transport
func (r *RTPTransport) Type() int {
	return TypeRTPTransport
}

// Close release all
func (r *RTPTransport) Close() {
	if r.stop {
		return
	}
	r.stop = true
	r.rtpSession.Close()
	r.rtcpSession.Close()
	r.rtpEndpoint.Close()
	r.rtcpEndpoint.Close()
	r.mux.Close()
	r.conn.Close()
}

// newEndpoint registers a new endpoint on the underlying mux.
func (r *RTPTransport) newEndpoint(f mux.MatchFunc) *mux.Endpoint {
	return r.mux.NewEndpoint(f)
}

// ReceiveRTP receive rtp
func (r *RTPTransport) receiveRTP() {
	go func() {
		for {
			readStream, ssrc, err := r.rtpSession.AcceptStream()
			if err != nil {
				log.Warnf("Failed to accept stream %v ", err)
				//for non-blocking ReadRTP()
				r.rtpCh <- nil
				continue
			}
			go func() {
				rtpBuf := make([]byte, receiveMTU)
				for {
					if r.stop {
						return
					}
					_, pkt, err := readStream.ReadRTP(rtpBuf)
					if err != nil {
						log.Warnf("Failed to read rtp %v %d ", err, ssrc)
						//for non-blocking ReadRTP()
						r.rtpCh <- nil
						continue
						// return
					}

					log.Debugf("RTPTransport.receiveRTP pkt=%v", pkt)
					r.idLock.Lock()
					if r.id == "" {
						r.id = util.GetIDFromRTP(pkt)
					}
					r.idLock.Unlock()

					r.rtpCh <- pkt
					r.ssrcPTLock.Lock()
					r.ssrcPT[pkt.Header.SSRC] = pkt.Header.PayloadType
					r.ssrcPTLock.Unlock()
					// log.Debugf("got RTP: %+v", pkt.Header)
				}
			}()
		}
	}()
}

// ReadRTP read rtp from transport
func (r *RTPTransport) ReadRTP() (*rtp.Packet, error) {
	return <-r.rtpCh, nil
}

// rtp sub receive rtcp
func (r *RTPTransport) receiveRTCP() {
	go func() {
		for {
			readStream, ssrc, err := r.rtcpSession.AcceptStream()
			if err != nil {
				log.Warnf("Failed to accept RTCP %v ", err)
				return
			}

			go func() {
				rtcpBuf := make([]byte, receiveMTU)
				for {
					if r.stop {
						return
					}
					rtcps, err := readStream.ReadRTCP(rtcpBuf)
					if err != nil {
						log.Warnf("Failed to read rtcp %v %d ", err, ssrc)
						return
					}
					log.Debugf("got RTCPs: %+v ", rtcps)
					for _, pkt := range rtcps {
						switch pkt.(type) {
						case *rtcp.PictureLossIndication:
							log.Debugf("got pli, not need send key frame!")
						case *rtcp.TransportLayerNack:
							log.Debugf("rtptransport got nack: %+v", pkt)
							r.rtcpCh <- pkt
						}
					}
				}
			}()
		}
	}()
}

// WriteRTP send rtp packet
func (r *RTPTransport) WriteRTP(rtp *rtp.Packet) error {
	log.Debugf("RTPTransport.WriteRTP rtp=%v", rtp)
	writeStream, err := r.rtpSession.OpenWriteStream()
	if err != nil {
		r.writeErrCnt++
		return err
	}

	if r.extSent > 0 {
		r.idLock.Lock()
		util.SetIDToRTP(rtp, r.id)
		r.idLock.Unlock()
	}

	_, err = writeStream.WriteRTP(&rtp.Header, rtp.Payload)
	if err == nil && r.extSent > 0 {
		r.extSent--
	}
	if err != nil {
		log.Errorf(err.Error())
		r.writeErrCnt++
	}
	return err
}

// WriteRawRTCP write rtcp data
func (r *RTPTransport) WriteRawRTCP(data []byte) (int, error) {
	writeStream, err := r.rtcpSession.OpenWriteStream()
	if err != nil {
		return 0, err
	}
	return writeStream.WriteRawRTCP(data)
}

// SSRCPT playload type and ssrc
func (r *RTPTransport) SSRCPT() map[uint32]uint8 {
	r.ssrcPTLock.RLock()
	defer r.ssrcPTLock.RUnlock()
	return r.ssrcPT
}

// WriteRTCP write rtcp
func (r *RTPTransport) WriteRTCP(pkt rtcp.Packet) error {
	bin, err := pkt.Marshal()
	if err != nil {
		return err
	}
	_, err = r.WriteRawRTCP(bin)
	if err != nil {
		return err
	}
	return err
}

// WriteErrTotal return write error
func (r *RTPTransport) WriteErrTotal() int {
	return r.writeErrCnt
}

// WriteErrReset reset write error
func (r *RTPTransport) WriteErrReset() {
	r.writeErrCnt = 0
}

// GetRTCPChan return a rtcp channel
func (r *RTPTransport) GetRTCPChan() chan rtcp.Packet {
	return r.rtcpCh
}

// RemoteAddr return remote addr
func (r *RTPTransport) RemoteAddr() net.Addr {
	if r.conn == nil {
		log.Errorf("RemoteAddr err=%v", errInvalidConn)
		return nil
	}
	return r.conn.RemoteAddr()
}

// GetBandwidth get bindwitdh setting
func (r *RTPTransport) GetBandwidth() int {
	return r.bandwidth
}
