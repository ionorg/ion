package muxrtp

import (
	"errors"
	"fmt"
	"net"

	"github.com/pion/rtcp"
)

var (
	// ErrSessionRTCPClosed is returned when a RTCP session has been closed
	ErrSessionRTCPClosed = errors.New("SessionRTCP has been closed")
)

type SessionRTCP struct {
	session
	writeStream *WriteStreamRTCP
}

// NewSessionRTCP creates a RTCP session using conn as the underlying transport.
func NewSessionRTCP(conn net.Conn) (*SessionRTCP, error) {
	s := &SessionRTCP{
		session: session{
			nextConn:    conn,
			readStreams: map[uint32]readStream{},
			newStream:   make(chan readStream),
			started:     make(chan interface{}),
			closed:      make(chan interface{}),
		},
	}
	s.writeStream = &WriteStreamRTCP{s}

	err := s.session.start(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// OpenWriteStream returns the global write stream for the Session
func (s *SessionRTCP) OpenWriteStream() (*WriteStreamRTCP, error) {
	return s.writeStream, nil
}

// OpenReadStream opens a read stream for the given SSRC, it can be used
// if you want a certain SSRC, but don't want to wait for AcceptStream
func (s *SessionRTCP) OpenReadStream(SSRC uint32) (*ReadStreamRTCP, error) {
	r, _ := s.session.getOrCreateReadStream(SSRC, s, newReadStreamRTCP)

	if readStream, ok := r.(*ReadStreamRTCP); ok {
		return readStream, nil
	}
	return nil, fmt.Errorf("failed to open ReadStreamSRCTP, type assertion failed")
}

// AcceptStream returns a stream to handle RTCP for a single SSRC
func (s *SessionRTCP) AcceptStream() (*ReadStreamRTCP, uint32, error) {
	stream, ok := <-s.newStream
	if !ok {
		return nil, 0, ErrSessionRTCPClosed
	}

	readStream, ok := stream.(*ReadStreamRTCP)
	if !ok {
		return nil, 0, fmt.Errorf("newStream was found, but failed type assertion")
	}

	return readStream, stream.GetSSRC(), nil
}

// Close ends the session
func (s *SessionRTCP) Close() error {
	return s.session.close()
}

// Private
func (s *SessionRTCP) write(buf []byte) (int, error) {
	if _, ok := <-s.session.started; ok {
		return 0, fmt.Errorf("started channel used incorrectly, should only be closed")
	}
	// p := rtcp.RawPacket(buf)
	// bin, err := p.Marshal()
	// if err != nil {
	// return 0, err
	// }

	return s.session.nextConn.Write(buf)
}

//create a list of Destination SSRCs
//that's a superset of all Destinations in the slice.
func destinationSSRC(pkts []rtcp.Packet) []uint32 {
	ssrcSet := make(map[uint32]struct{})
	for _, p := range pkts {
		for _, ssrc := range p.DestinationSSRC() {
			ssrcSet[ssrc] = struct{}{}
		}
	}

	out := make([]uint32, 0, len(ssrcSet))
	for ssrc := range ssrcSet {
		out = append(out, ssrc)
	}

	return out
}

func (s *SessionRTCP) handle(buf []byte) error {
	pkt, err := rtcp.Unmarshal(buf)
	if err != nil {
		return err
	}

	for _, ssrc := range destinationSSRC(pkt) {
		r, isNew := s.session.getOrCreateReadStream(ssrc, s, newReadStreamRTCP)
		if r == nil {
			return nil // Session has been closed
		} else if isNew {
			s.session.newStream <- r // Notify AcceptStream
		}

		readStream, ok := r.(*ReadStreamRTCP)
		if !ok {
			return fmt.Errorf("failed to get/create ReadStreamRTP")
		}

		_, err = readStream.write(buf)
		if err != nil {
			return err
		}
	}

	return nil
}
