package srtp

import (
	"fmt"
	"sync"

	"github.com/sssgun/ion/rtp"
	"github.com/sssgun/ion/transport/packetio"
)

// Limit the buffer size to 1MB
const srtpBufferSize = 1000 * 1000

// ReadStreamSRTP handles decryption for a single RTP SSRC
type ReadStreamSRTP struct {
	mu sync.Mutex

	isInited bool
	isClosed chan bool

	session *SessionSRTP
	ssrc    uint32

	buffer *packetio.Buffer
}

// Used by getOrCreateReadStream
func newReadStreamSRTP() readStream {
	return &ReadStreamSRTP{}
}

func (r *ReadStreamSRTP) init(child streamSession, ssrc uint32) error {
	sessionSRTP, ok := child.(*SessionSRTP)

	r.mu.Lock()
	defer r.mu.Unlock()

	if !ok {
		return fmt.Errorf("ReadStreamSRTP init failed type assertion")
	} else if r.isInited {
		return fmt.Errorf("ReadStreamSRTP has already been inited")
	}

	r.session = sessionSRTP
	r.ssrc = ssrc
	r.isInited = true
	r.isClosed = make(chan bool)

	// Create a buffer with a 1MB limit
	r.buffer = packetio.NewBuffer()
	r.buffer.SetLimitSize(srtpBufferSize)

	return nil
}

func (r *ReadStreamSRTP) write(buf []byte) (n int, err error) {
	n, err = r.buffer.Write(buf)

	if err == packetio.ErrFull {
		// Silently drop data when the buffer is full.
		return len(buf), nil
	}

	return n, err
}

// Read reads and decrypts full RTP packet from the nextConn
func (r *ReadStreamSRTP) Read(buf []byte) (int, error) {
	return r.buffer.Read(buf)
}

// ReadRTP reads and decrypts full RTP packet and its header from the nextConn
func (r *ReadStreamSRTP) ReadRTP(buf []byte) (int, *rtp.Header, error) {
	n, err := r.Read(buf)
	if err != nil {
		return 0, nil, err
	}

	header := &rtp.Header{}

	err = header.Unmarshal(buf[:n])
	if err != nil {
		return 0, nil, err
	}

	return n, header, nil
}

// Close removes the ReadStream from the session and cleans up any associated state
func (r *ReadStreamSRTP) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isInited {
		return fmt.Errorf("ReadStreamSRTP has not been inited")
	}

	select {
	case <-r.isClosed:
		return fmt.Errorf("ReadStreamSRTP is already closed")
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
func (r *ReadStreamSRTP) GetSSRC() uint32 {
	return r.ssrc
}

// WriteStreamSRTP is stream for a single Session that is used to encrypt RTP
type WriteStreamSRTP struct {
	session *SessionSRTP
}

// WriteRTP encrypts a RTP packet and writes to the connection
func (w *WriteStreamSRTP) WriteRTP(header *rtp.Header, payload []byte) (int, error) {
	return w.session.writeRTP(header, payload)
}

// Write encrypts and writes a full RTP packets to the nextConn
func (w *WriteStreamSRTP) Write(b []byte) (int, error) {
	return w.session.write(b)
}
