// Package stream defines a small pull-based stream contract that can bridge
// AI frameworks and transports without coupling either side to the other.
package stream

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// ErrClosed is returned when Recv is called after the reader has been closed.
var ErrClosed = errors.New("stream: reader closed")

// Reader is a pull-based stream. Recv returns io.EOF on normal completion.
//
// The producer must observe the context used when the reader was created so a
// blocked Recv can be released after cancellation. The consumer owns Reader
// and must call Close exactly once; implementations must tolerate repeated
// Close calls.
type Reader[T any] interface {
	Recv() (T, error)
	Close() error
}

type funcReader[T any] struct {
	recv     func() (T, error)
	close    func() error
	closed   atomic.Bool
	closeErr error
	closeOne sync.Once
}

// NewReader adapts receive and close functions to Reader. The receive function
// must not be nil. A nil close function is treated as a no-op.
func NewReader[T any](recv func() (T, error), closeFunc func() error) Reader[T] {
	if recv == nil {
		panic("stream: nil receive function")
	}
	if closeFunc == nil {
		closeFunc = func() error { return nil }
	}
	return &funcReader[T]{recv: recv, close: closeFunc}
}

func (r *funcReader[T]) Recv() (T, error) {
	if r.closed.Load() {
		var zero T
		return zero, ErrClosed
	}
	return r.recv()
}

func (r *funcReader[T]) Close() error {
	r.closeOne.Do(func() {
		r.closed.Store(true)
		r.closeErr = r.close()
	})
	return r.closeErr
}

// FromSlice returns a reader over a snapshot of values. It is primarily useful
// for examples, deterministic workflows, and tests.
func FromSlice[T any](values []T) Reader[T] {
	items := append([]T(nil), values...)
	var mu sync.Mutex
	index := 0

	return NewReader(func() (T, error) {
		mu.Lock()
		defer mu.Unlock()

		if index == len(items) {
			var zero T
			return zero, io.EOF
		}
		value := items[index]
		index++
		return value, nil
	}, nil)
}
