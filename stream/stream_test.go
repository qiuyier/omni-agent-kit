package stream_test

import (
	"errors"
	"io"
	"testing"

	"github.com/qiuyier/omni-agent-kit/stream"
)

func TestFromSlice(t *testing.T) {
	reader := stream.FromSlice([]int{2, 3, 5})
	defer reader.Close()

	for _, want := range []int{2, 3, 5} {
		got, err := reader.Recv()
		if err != nil {
			t.Fatalf("Recv() error = %v", err)
		}
		if got != want {
			t.Fatalf("Recv() = %d, want %d", got, want)
		}
	}
	if _, err := reader.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final Recv() error = %v, want io.EOF", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	closeCalls := 0
	reader := stream.NewReader(
		func() (string, error) { return "", io.EOF },
		func() error {
			closeCalls++
			return nil
		},
	)

	if err := reader.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if closeCalls != 1 {
		t.Fatalf("close function called %d times, want 1", closeCalls)
	}
	if _, err := reader.Recv(); !errors.Is(err, stream.ErrClosed) {
		t.Fatalf("Recv() after Close() error = %v, want ErrClosed", err)
	}
}

func TestCloseReturnsSameError(t *testing.T) {
	want := errors.New("close failed")
	reader := stream.NewReader(
		func() (int, error) { return 0, io.EOF },
		func() error { return want },
	)

	if err := reader.Close(); !errors.Is(err, want) {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := reader.Close(); !errors.Is(err, want) {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestNewReaderRejectsNilReceive(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewReader did not panic for a nil receive function")
		}
	}()
	stream.NewReader[int](nil, nil)
}
