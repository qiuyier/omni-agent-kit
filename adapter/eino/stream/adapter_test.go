package einostream_test

import (
	"errors"
	"io"
	"testing"

	"github.com/cloudwego/eino/schema"

	einostream "github.com/qiuyier/omni-agent-kit/adapter/eino/stream"
	kitstream "github.com/qiuyier/omni-agent-kit/stream"
)

func TestWrap(t *testing.T) {
	reader, err := einostream.Wrap(schema.StreamReaderFromArray([]string{"a", "b"}))
	if err != nil {
		t.Fatalf("Wrap() error = %v", err)
	}

	for _, want := range []string{"a", "b"} {
		got, recvErr := reader.Recv()
		if recvErr != nil {
			t.Fatalf("Recv() error = %v", recvErr)
		}
		if got != want {
			t.Fatalf("Recv() = %q, want %q", got, want)
		}
	}
	if _, recvErr := reader.Recv(); !errors.Is(recvErr, io.EOF) {
		t.Fatalf("final Recv() error = %v, want io.EOF", recvErr)
	}
	if closeErr := reader.Close(); closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if _, recvErr := reader.Recv(); !errors.Is(recvErr, kitstream.ErrClosed) {
		t.Fatalf("Recv() after close error = %v, want ErrClosed", recvErr)
	}
}

func TestWrapRejectsNil(t *testing.T) {
	if _, err := einostream.Wrap[string](nil); err == nil {
		t.Fatal("Wrap(nil) error = nil")
	}
}
