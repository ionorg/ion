package packetio

import (
	"errors"
)

// netError implements net.Error
type netError struct {
	error
	timeout, temporary bool
}

func (e *netError) Timeout() bool {
	return e.timeout
}

func (e *netError) Temporary() bool {
	return e.temporary
}

// ErrFull is returned when the buffer has hit the configured limits.
var ErrFull = errors.New("packetio.Buffer is full, discarding write")

var errTimeout = errors.New("i/o timeout")
