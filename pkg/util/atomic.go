package util

import "sync/atomic"

// AtomicBool represents a atomic bool
type AtomicBool struct {
	val int32
}

// Set atomic bool
func (b *AtomicBool) Set(value bool) {
	var i int32
	if value {
		i = 1
	}

	atomic.StoreInt32(&(b.val), i)
}

// Get atomic bool
func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&(b.val)) != 0
}
