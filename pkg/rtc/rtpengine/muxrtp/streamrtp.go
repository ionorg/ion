package muxrtp

import (
	"fmt"
	"sync"

	"github.com/pion/ion/pkg/rtc/rtpengine/packetio"
	"github.com/pion/rtp"
)

// Limit the buffer size to 1MB
const rtpBufferSize = 1000 * 1000

// ReadStreamRTP handles decryption for a single RTP SSRC
type ReadStreamRTP struct {
	mu sync.Mutex

	isInited bool
	isClosed chan bool

	session *SessionRTP
	ssrc    uint32

	buffer *packetio.Buffer
}

// Used by getOrCreateReadStream
func newReadStreamRTP() readStream {
	return &ReadStreamRTP{}
}

func (r *ReadStreamRTP) init(child streamSession, ssrc uint32) error {
	sessionRTP, ok := child.(*SessionRTP)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !ok {
		return fmt.Errorf("ReadStreamRTP init failed type assertion")
	} else if r.isInited {
		return fmt.Errorf("ReadStreamRTP has already been inited")
	}

	r.session = sessionRTP
	r.ssrc = ssrc
	r.isInited = true
	r.isClosed = make(chan bool)

	// Create a buffer with a 1MB limit
	r.buffer = packetio.NewBuffer()
	r.buffer.SetLimitSize(rtpBufferSize)

	return nil
}

func (r *ReadStreamRTP) write(buf []byte) (n int, err error) {
	n, err = r.buffer.Write(buf)

	if err == packetio.ErrFull {
		// Silently drop data when the buffer is full.
		return len(buf), nil
	}

	return n, err
}

// Read reads and decrypts full RTP packet from the nextConn
func (r *ReadStreamRTP) Read(buf []byte) (int, error) {
	return r.buffer.Read(buf)
}

// ReadRTP reads and decrypts full RTP packet and its header from the nextConn
func (r *ReadStreamRTP) ReadRTP(buf []byte) (int, *rtp.Packet, error) {
	n, err := r.Read(buf)
	if err != nil {
		return 0, nil, err
	}

	pkt := &rtp.Packet{}

	err = pkt.Unmarshal(buf[:n])
	if err != nil {
		return 0, nil, err
	}

	return n, pkt, nil
}

// Close removes the ReadStream from the session and cleans up any associated state
func (r *ReadStreamRTP) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isInited {
		return fmt.Errorf("ReadStreamRTP has not been inited")
	}

	select {
	case <-r.isClosed:
		return fmt.Errorf("ReadStreamRTP is already closed")
	default:
		err := r.buffer.Close()
		if err != nil {
			return err
		}

		r.session.removeReadStream(r.ssrc)
		return nil
	}

}

// GetSSRC returns the SSRC we are demuxing for
func (r *ReadStreamRTP) GetSSRC() uint32 {
	return r.ssrc
}

// WriteStreamRTP is stream for a single Session that is used to encrypt RTP
type WriteStreamRTP struct {
	session *SessionRTP
}

// WriteRTP encrypts a RTP packet and writes to the connection
func (w *WriteStreamRTP) WriteRTP(header *rtp.Header, payload []byte) (int, error) {
	return w.session.writeRTP(header, payload)
}
