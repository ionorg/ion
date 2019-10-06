package muxrtp

import (
	"fmt"
	"net"

	"github.com/pion/rtp"
)

type SessionRTP struct {
	session
	writeStream *WriteStreamRTP
}

func NewSessionRTP(conn net.Conn) (*SessionRTP, error) {
	s := &SessionRTP{
		session: session{
			nextConn:    conn,
			readStreams: map[uint32]readStream{},
			newStream:   make(chan readStream),
			started:     make(chan interface{}),
			closed:      make(chan interface{}),
		},
	}
	s.writeStream = &WriteStreamRTP{s}

	err := s.session.start(s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Start initializes and allows reading/writing to begin
// func (s *SessionRTP) Start(nextConn net.Conn) error {
// s.session.nextConn = nextConn
// return s.session.start(s)
// }

// OpenWriteStream returns the global write stream for the Session
func (s *SessionRTP) OpenWriteStream() (*WriteStreamRTP, error) {
	return s.writeStream, nil
}

// OpenReadStream opens a read stream for the given SSRC, it can be used
// if you want a certain SSRC, but don't want to wait for AcceptStream
func (s *SessionRTP) OpenReadStream(SSRC uint32) (*ReadStreamRTP, error) {
	r, _ := s.session.getOrCreateReadStream(SSRC, s, newReadStreamRTP)

	if readStream, ok := r.(*ReadStreamRTP); ok {
		return readStream, nil
	}

	return nil, fmt.Errorf("failed to open ReadStreamSRCTP, type assertion failed")
}

// AcceptStream returns a stream to handle RTCP for a single SSRC
func (s *SessionRTP) AcceptStream() (*ReadStreamRTP, uint32, error) {
	stream, ok := <-s.newStream
	if !ok {
		return nil, 0, fmt.Errorf("SessionRTP has been closed")
	}

	readStream, ok := stream.(*ReadStreamRTP)
	if !ok {
		return nil, 0, fmt.Errorf("newStream was found, but failed type assertion")
	}

	return readStream, stream.GetSSRC(), nil
}

// Close ends the session
func (s *SessionRTP) Close() error {
	return s.session.close()
}

func (s *SessionRTP) write(b []byte) (int, error) {
	packet := &rtp.Packet{}

	err := packet.Unmarshal(b)
	if err != nil {
		return 0, nil
	}

	return s.writeRTP(&packet.Header, packet.Payload)
}

func (s *SessionRTP) writeRTP(header *rtp.Header, payload []byte) (int, error) {
	if _, ok := <-s.session.started; ok {
		return 0, fmt.Errorf("started channel used incorrectly, should only be closed")
	}
	p := rtp.Packet{Header: *header, Payload: payload}
	bin, err := p.Marshal()
	if err != nil {
		return 0, err
	}

	return s.session.nextConn.Write(bin)
}

func (s *SessionRTP) handle(buf []byte) error {
	h := &rtp.Header{}
	if err := h.Unmarshal(buf); err != nil {
		return err
	}

	r, isNew := s.session.getOrCreateReadStream(h.SSRC, s, newReadStreamRTP)
	if r == nil {
		return nil // Session has been closed
	} else if isNew {
		s.session.newStream <- r // Notify AcceptStream
	}

	readStream, ok := r.(*ReadStreamRTP)
	if !ok {
		return fmt.Errorf("failed to get/create ReadStreamSRTP")
	}

	_, err := readStream.write(buf)
	if err != nil {
		return err
	}

	return nil
}
