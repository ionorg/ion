package packetio

import (
	"errors"
	"io"
	"sync"
)

// ErrFull is returned when the buffer has hit the configured limits.
var ErrFull = errors.New("full buffer")

// Buffer allows writing packets to an intermediate buffer, which can then be read form.
// This is verify similar to bytes.Buffer but avoids combining multiple writes into a single read.
type Buffer struct {
	mutex   sync.Mutex
	packets [][]byte

	notify chan struct{}
	subs   bool
	closed bool

	// The number of buffered packets in bytes.
	size int

	// The limit on Write in packet count and total size.
	limitCount int
	limitSize  int
}

// NewBuffer creates a new Buffer object.
func NewBuffer() *Buffer {
	return &Buffer{
		notify: make(chan struct{}),
	}
}

// Write appends a copy of the packet data to the buffer.
// If any defined limits are hit, returns ErrFull.
func (b *Buffer) Write(packet []byte) (n int, err error) {
	// Copy the packet before adding it.
	packet = append([]byte{}, packet...)

	b.mutex.Lock()

	// Make sure we're not closed.
	if b.closed {
		b.mutex.Unlock()
		return 0, io.ErrClosedPipe
	}

	// Check if there is available capacity
	if b.limitCount != 0 && len(b.packets)+1 > b.limitCount {
		b.mutex.Unlock()
		return 0, ErrFull
	}

	// Check if there is available capacity
	if b.limitSize != 0 && b.size+len(packet) > b.limitSize {
		b.mutex.Unlock()
		return 0, ErrFull
	}

	var notify chan struct{}

	// Decide if we need to wake up any readers.
	if b.subs {
		// If so, close the notify channel and make a new one.
		// This effectively behaves like a broadcast, waking up any blocked goroutines.
		// We close after we release the lock to reduce contention.
		notify = b.notify
		b.notify = make(chan struct{})

		// Reset the subs marker.
		b.subs = false
	}

	// Add the packet to the queue.
	b.packets = append(b.packets, packet)
	b.size += len(packet)
	b.mutex.Unlock()

	// Actually close the notify channel down here.
	if notify != nil {
		close(notify)
	}

	return len(packet), nil
}

// Read populates the given byte slice, returning the number of bytes read.
// Blocks until data is available or the buffer is closed.
// Returns io.ErrShortBuffer is the packet is too small to copy the Write.
// Returns io.EOF if the buffer is closed.
func (b *Buffer) Read(packet []byte) (n int, err error) {
	for {
		b.mutex.Lock()

		// See if there are any packets in the queue.
		if len(b.packets) > 0 {
			first := b.packets[0]

			// This is a packet-based reader/writer so we can't truncate.
			if len(first) > len(packet) {
				b.mutex.Unlock()
				return 0, io.ErrShortBuffer
			}

			// Remove our packet and continue.
			b.packets = b.packets[1:]
			b.size -= len(first)

			b.mutex.Unlock()

			// Actually transfer the data.
			n := copy(packet, first)
			return n, nil
		}

		// Make sure the reader isn't actually closed.
		// This is done after checking packets to fully read the buffer.
		if b.closed {
			b.mutex.Unlock()
			return 0, io.EOF
		}

		// Get the current notify channel.
		// This will be closed when there is new data available, waking us up.
		notify := b.notify

		// Set the subs marker, telling the writer we're waiting.
		b.subs = true
		b.mutex.Unlock()

		// Wake for the broadcast.
		<-notify
	}
}

// Close will unblock any readers and prevent future writes.
// Data in the buffer can still be read, returning io.EOF when fully depleted.
func (b *Buffer) Close() (err error) {
	// note: We don't use defer so we can close the notify channel after unlocking.
	// This will unblock goroutines that can grab the lock immediately, instead of blocking again.
	b.mutex.Lock()

	if b.closed {
		b.mutex.Unlock()
		return nil
	}

	notify := b.notify

	b.closed = true
	b.mutex.Unlock()

	close(notify)

	return nil
}

// Count returns the number of packets in the buffer.
func (b *Buffer) Count() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return len(b.packets)
}

// SetLimitCount controls the maximum number of packets that can be buffered.
// Causes Write to return ErrFull when this limit is reached.
// A zero value will disable this limit.
func (b *Buffer) SetLimitCount(limit int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.limitCount = limit
}

// Size returns the total byte size of packets in the buffer.
func (b *Buffer) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.size
}

// SetLimitSize controls the maximum number of bytes that can be buffered.
// Causes Write to return ErrFull when this limit is reached.
// A zero value will disable this limit.
func (b *Buffer) SetLimitSize(limit int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.limitSize = limit
}
