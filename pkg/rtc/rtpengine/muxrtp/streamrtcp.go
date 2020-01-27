package muxrtp

import (
	"fmt"
	"sync"

	"github.com/pion/ion/pkg/rtc/rtpengine/packetio"
	"github.com/pion/rtcp"
)

// Limit the buffer size to 100KB
const rtcpBufferSize = 100 * 1000

// ReadStreamRTCP handles decryption for a single RTCP SSRC
type ReadStreamRTCP struct {
	mu sync.Mutex

	isInited bool
	isClosed chan bool

	session *SessionRTCP
	ssrc    uint32

	buffer *packetio.Buffer
}

func (r *ReadStreamRTCP) write(buf []byte) (n int, err error) {
	return r.buffer.Write(buf)
}

// Used by getOrCreateReadStream
func newReadStreamRTCP() readStream {
	return &ReadStreamRTCP{}
}

// ReadRTCP reads and decrypts full RTCP packet and its header from the nextConn
// func (r *ReadStreamRTCP) ReadRTCP(buf []byte) (int, *rtcp.Header, error) {
// n, err := r.Read(buf)
// if err != nil {
// return 0, nil, err
// }

// header := &rtcp.Header{}
// err = header.Unmarshal(buf[:n])
// if err != nil {
// return 0, nil, err
// }

// return n, header, nil
// }

// ReadRTCP reads full RTCP packet
func (r *ReadStreamRTCP) ReadRTCP(buf []byte) ([]rtcp.Packet, error) {
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}

	return rtcp.Unmarshal(buf[:n])
}

// Read reads and decrypts full RTCP packet from the nextConn
func (r *ReadStreamRTCP) Read(b []byte) (int, error) {
	n, err := r.buffer.Read(b)

	if err == packetio.ErrFull {
		// Silently drop data when the buffer is full.
		return len(b), nil
	}

	return n, err
}

// Close removes the ReadStream from the session and cleans up any associated state
func (r *ReadStreamRTCP) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isInited {
		return fmt.Errorf("ReadStreamRTCP has not been inited")
	}

	select {
	case <-r.isClosed:
		return fmt.Errorf("ReadStreamRTCP is already closed")
	default:
		err := r.buffer.Close()
		if err != nil {
			return err
		}

		r.session.removeReadStream(r.ssrc)
		return nil
	}
}

func (r *ReadStreamRTCP) init(child streamSession, ssrc uint32) error {
	sessionRTCP, ok := child.(*SessionRTCP)

	r.mu.Lock()
	defer r.mu.Unlock()
	if !ok {
		return fmt.Errorf("ReadStreamRTCP init failed type assertion")
	} else if r.isInited {
		return fmt.Errorf("ReadStreamRTCP has already been inited")
	}

	r.session = sessionRTCP
	r.ssrc = ssrc
	r.isInited = true
	r.isClosed = make(chan bool)

	// Create a buffer and limit it to 100KB
	r.buffer = packetio.NewBuffer()
	r.buffer.SetLimitSize(rtcpBufferSize)

	return nil
}

// GetSSRC returns the SSRC we are demuxing for
func (r *ReadStreamRTCP) GetSSRC() uint32 {
	return r.ssrc
}

// WriteStreamRTCP is stream for a single Session that is used to encrypt RTCP
type WriteStreamRTCP struct {
	session *SessionRTCP
}

// Write a RTCP header and its payload to the nextConn
func (w *WriteStreamRTCP) WriteRTCP(header *rtcp.Header, payload []byte) (int, error) {
	headerRaw, err := header.Marshal()
	if err != nil {
		return 0, err
	}

	return w.session.write(append(headerRaw, payload...))
}

// Write a RTCP header and its payload to the nextConn
func (w *WriteStreamRTCP) WriteRawRTCP(data []byte) (int, error) {
	return w.session.write(data)
}

// Write encrypts and writes a full RTCP packets to the nextConn
func (w *WriteStreamRTCP) Write(b []byte) (int, error) {
	return w.session.write(b)
}
