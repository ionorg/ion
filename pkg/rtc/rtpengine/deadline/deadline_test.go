package deadline

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Deadline", func(t *testing.T) {
		now := time.Now()

		ctx0, cancel0 := context.WithDeadline(ctx, now.Add(40*time.Millisecond))
		defer cancel0()
		ctx1, cancel1 := context.WithDeadline(ctx, now.Add(60*time.Millisecond))
		defer cancel1()
		d := New()
		d.Set(now.Add(50 * time.Millisecond))

		ch := make(chan byte)
		go sendOnDone(ctx, ctx0.Done(), ch, 0)
		go sendOnDone(ctx, ctx1.Done(), ch, 1)
		go sendOnDone(ctx, d.Done(), ch, 2)

		calls := collectCh(ch, 3, 100*time.Millisecond)
		expectedCalls := []byte{0, 2, 1}
		if !bytes.Equal(calls, expectedCalls) {
			t.Errorf("Wrong order of deadline signal, expected: %v, got: %v", expectedCalls, calls)
		}
	})

	t.Run("DeadlineExtend", func(t *testing.T) {
		now := time.Now()

		ctx0, cancel0 := context.WithDeadline(ctx, now.Add(40*time.Millisecond))
		defer cancel0()
		ctx1, cancel1 := context.WithDeadline(ctx, now.Add(60*time.Millisecond))
		defer cancel1()
		d := New()
		d.Set(now.Add(50 * time.Millisecond))
		d.Set(now.Add(70 * time.Millisecond))

		ch := make(chan byte)
		go sendOnDone(ctx, ctx0.Done(), ch, 0)
		go sendOnDone(ctx, ctx1.Done(), ch, 1)
		go sendOnDone(ctx, d.Done(), ch, 2)

		calls := collectCh(ch, 3, 100*time.Millisecond)
		expectedCalls := []byte{0, 1, 2}
		if !bytes.Equal(calls, expectedCalls) {
			t.Errorf("Wrong order of deadline signal, expected: %v, got: %v", expectedCalls, calls)
		}
	})

	t.Run("DeadlinePretend", func(t *testing.T) {
		now := time.Now()

		ctx0, cancel0 := context.WithDeadline(ctx, now.Add(40*time.Millisecond))
		defer cancel0()
		ctx1, cancel1 := context.WithDeadline(ctx, now.Add(60*time.Millisecond))
		defer cancel1()
		d := New()
		d.Set(now.Add(50 * time.Millisecond))
		d.Set(now.Add(30 * time.Millisecond))

		ch := make(chan byte)
		go sendOnDone(ctx, ctx0.Done(), ch, 0)
		go sendOnDone(ctx, ctx1.Done(), ch, 1)
		go sendOnDone(ctx, d.Done(), ch, 2)

		calls := collectCh(ch, 3, 100*time.Millisecond)
		expectedCalls := []byte{2, 0, 1}
		if !bytes.Equal(calls, expectedCalls) {
			t.Errorf("Wrong order of deadline signal, expected: %v, got: %v", expectedCalls, calls)
		}
	})

	t.Run("DeadlineCancel", func(t *testing.T) {
		now := time.Now()

		ctx0, cancel0 := context.WithDeadline(ctx, now.Add(40*time.Millisecond))
		defer cancel0()
		d := New()
		d.Set(now.Add(50 * time.Millisecond))
		d.Set(time.Time{})

		ch := make(chan byte)
		go sendOnDone(ctx, ctx0.Done(), ch, 0)
		go sendOnDone(ctx, d.Done(), ch, 1)

		calls := collectCh(ch, 2, 60*time.Millisecond)
		expectedCalls := []byte{0}
		if !bytes.Equal(calls, expectedCalls) {
			t.Errorf("Wrong order of deadline signal, expected: %v, got: %v", expectedCalls, calls)
		}
	})
}

func sendOnDone(ctx context.Context, done <-chan struct{}, dest chan byte, val byte) {
	select {
	case <-done:
	case <-ctx.Done():
		return
	}
	dest <- val
}
func collectCh(ch <-chan byte, n int, timeout time.Duration) []byte {
	a := time.After(timeout)
	var calls []byte
	for len(calls) < n {
		select {
		case call := <-ch:
			calls = append(calls, call)
		case <-a:
			return calls
		}
	}
	return calls
}
