package util

import "sync/atomic"

// AtomicBool represents a atomic bool
type AtomicBool struct {
	val int32
}

// Set atomic bool
func (b *AtomicBool) Set(value bool) (swapped bool) {
	if value {
		return atomic.SwapInt32(&(b.val), 1) == 0
	}
	return atomic.SwapInt32(&(b.val), 0) == 1
}

// Get atomic bool
func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&(b.val)) != 0
}
