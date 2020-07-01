package srtp

import (
	"errors"
	"fmt"
	"net"

	"github.com/sssgun/ion/logging"
	"github.com/sssgun/ion/rtp"
)

const defaultSessionSRTPReplayProtectionWindow = 64

// SessionSRTP implements io.ReadWriteCloser and provides a bi-directional SRTP session
// SRTP itself does not have a design like this, but it is common in most applications
// for local/remote to each have their own keying material. This provides those patterns
// instead of making everyone re-implement
type SessionSRTP struct {
	session
	writeStream *WriteStreamSRTP
}

// NewSessionSRTP creates a SRTP session using conn as the underlying transport.
func NewSessionSRTP(conn net.Conn, config *Config) (*SessionSRTP, error) {
	if config == nil {
		return nil, errors.New("no config provided")
	} else if conn == nil {
		return nil, errors.New("no conn provided")
	}

	loggerFactory := config.LoggerFactory
	if loggerFactory == nil {
		loggerFactory = logging.NewDefaultLoggerFactory()
	}

	localOpts := append(
		[]ContextOption{},
		config.LocalOptions...,
	)
	remoteOpts := append(
		[]ContextOption{
			// Default options
			SRTPReplayProtection(defaultSessionSRTPReplayProtectionWindow),
		},
		config.RemoteOptions...,
	)

	s := &SessionSRTP{
		session: session{
			nextConn:      conn,
			localOptions:  localOpts,
			remoteOptions: remoteOpts,
			readStreams:   map[uint32]readStream{},
			newStream:     make(chan readStream),
			started:       make(chan interface{}),
			closed:        make(chan interface{}),
			log:           loggerFactory.NewLogger("srtp"),
		},
	}
	s.writeStream = &WriteStreamSRTP{s}

	err := s.session.start(
		config.Keys.LocalMasterKey, config.Keys.LocalMasterSalt,
		config.Keys.RemoteMasterKey, config.Keys.RemoteMasterSalt,
		config.Profile,
		s,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// OpenWriteStream returns the global write stream for the Session
func (s *SessionSRTP) OpenWriteStream() (*WriteStreamSRTP, error) {
	return s.writeStream, nil
}

// OpenReadStream opens a read stream for the given SSRC, it can be used
// if you want a certain SSRC, but don't want to wait for AcceptStream
func (s *SessionSRTP) OpenReadStream(SSRC uint32) (*ReadStreamSRTP, error) {
	r, _ := s.session.getOrCreateReadStream(SSRC, s, newReadStreamSRTP)

	if readStream, ok := r.(*ReadStreamSRTP); ok {
		return readStream, nil
	}

	return nil, fmt.Errorf("failed to open ReadStreamSRCTP, type assertion failed")
}

// AcceptStream returns a stream to handle RTCP for a single SSRC
func (s *SessionSRTP) AcceptStream() (*ReadStreamSRTP, uint32, error) {
	stream, ok := <-s.newStream
	if !ok {
		return nil, 0, fmt.Errorf("SessionSRTP has been closed")
	}

	readStream, ok := stream.(*ReadStreamSRTP)
	if !ok {
		return nil, 0, fmt.Errorf("newStream was found, but failed type assertion")
	}

	return readStream, stream.GetSSRC(), nil
}

// Close ends the session
func (s *SessionSRTP) Close() error {
	return s.session.close()
}

func (s *SessionSRTP) write(b []byte) (int, error) {
	packet := &rtp.Packet{}

	err := packet.Unmarshal(b)
	if err != nil {
		return 0, nil
	}

	return s.writeRTP(&packet.Header, packet.Payload)
}

func (s *SessionSRTP) writeRTP(header *rtp.Header, payload []byte) (int, error) {
	if _, ok := <-s.session.started; ok {
		return 0, fmt.Errorf("started channel used incorrectly, should only be closed")
	}

	s.session.localContextMutex.Lock()
	encrypted, err := s.localContext.encryptRTP(nil, header, payload)
	s.session.localContextMutex.Unlock()

	if err != nil {
		return 0, err
	}

	return s.session.nextConn.Write(encrypted)
}

func (s *SessionSRTP) decrypt(buf []byte) error {
	h := &rtp.Header{}
	if err := h.Unmarshal(buf); err != nil {
		return err
	}

	r, isNew := s.session.getOrCreateReadStream(h.SSRC, s, newReadStreamSRTP)
	if r == nil {
		return nil // Session has been closed
	} else if isNew {
		s.session.newStream <- r // Notify AcceptStream
	}

	readStream, ok := r.(*ReadStreamSRTP)
	if !ok {
		return fmt.Errorf("failed to get/create ReadStreamSRTP")
	}

	decrypted, err := s.remoteContext.decryptRTP(buf, buf, h)
	if err != nil {
		return err
	}

	_, err = readStream.write(decrypted)
	if err != nil {
		return err
	}

	return nil
}
