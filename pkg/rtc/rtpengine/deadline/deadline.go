// Package deadline provides deadline timer used to implement
// net.Conn compatible connection
package deadline

import (
	"sync"
	"time"
)

// Deadline signals updatable deadline timer.
type Deadline struct {
	exceeded chan struct{}
	stop     chan struct{}
	stopped  chan bool
	mu       sync.RWMutex
}

// New creates new deadline timer.
func New() *Deadline {
	d := &Deadline{
		exceeded: make(chan struct{}),
		stop:     make(chan struct{}),
		stopped:  make(chan bool, 1),
	}
	d.stopped <- true
	return d
}

// Set new deadline. Zero value means no deadline.
func (d *Deadline) Set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	close(d.stop)

	select {
	case <-d.exceeded:
		d.exceeded = make(chan struct{})
	default:
		stopped := <-d.stopped
		if !stopped {
			d.exceeded = make(chan struct{})
		}
	}
	d.stop = make(chan struct{})
	d.stopped = make(chan bool, 1)

	if t.IsZero() {
		d.stopped <- true
		return
	}

	if dur := time.Until(t); dur > 0 {
		exceeded := d.exceeded
		stopped := d.stopped
		go func() {
			select {
			case <-time.After(dur):
				close(exceeded)
				stopped <- false
			case <-d.stop:
				stopped <- true
			}
		}()
		return
	}

	close(d.exceeded)
	d.stopped <- false
}

// Done receives deadline signal.
func (d *Deadline) Done() <-chan struct{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.exceeded
}
