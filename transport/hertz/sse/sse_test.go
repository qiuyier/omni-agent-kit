package sse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/qiuyier/omni-agent-kit/fault"
	kitstream "github.com/qiuyier/omni-agent-kit/stream"
)

type recordedEvent struct {
	id        string
	eventType string
	data      []byte
}

type recordingWriter struct {
	events   []recordedEvent
	closed   bool
	writeErr error
}

func (w *recordingWriter) WriteEvent(id, eventType string, data []byte) error {
	if w.writeErr != nil {
		return w.writeErr
	}
	w.events = append(w.events, recordedEvent{id: id, eventType: eventType, data: bytes.Clone(data)})
	return nil
}

func (w *recordingWriter) Close() error {
	w.closed = true
	return nil
}

func TestWriteNormalStream(t *testing.T) {
	writer := &recordingWriter{}
	reader := kitstream.FromSlice([]string{"hello", "world"})

	started, err := write(context.Background(), reader, stringEncoder, func() eventWriter { return writer })
	if err != nil {
		t.Fatalf("write() error = %v", err)
	}
	if !started {
		t.Fatal("write() started = false")
	}
	if len(writer.events) != 2 {
		t.Fatalf("event count = %d, want 2", len(writer.events))
	}
	if string(writer.events[1].data) != `{"text":"world"}` {
		t.Fatalf("second event data = %s", writer.events[1].data)
	}
	if !writer.closed {
		t.Fatal("writer was not closed")
	}
}

func TestInitialFailureKeepsHTTPResponseAvailable(t *testing.T) {
	want := fault.New(fault.CodeUnavailable, "model unavailable")
	reader := kitstream.NewReader(func() (string, error) { return "", want }, nil)
	writerCreated := false

	started, err := write(context.Background(), reader, stringEncoder, func() eventWriter {
		writerCreated = true
		return &recordingWriter{}
	})
	if !errors.Is(err, want) {
		t.Fatalf("write() error = %v", err)
	}
	if started {
		t.Fatal("write() started = true")
	}
	if writerCreated {
		t.Fatal("SSE writer was created before the first event")
	}
}

func TestFailureAfterStartWritesSafeTerminalEvent(t *testing.T) {
	secretCause := errors.New("provider token=secret")
	index := 0
	reader := kitstream.NewReader(func() (string, error) {
		if index == 0 {
			index++
			return "first", nil
		}
		return "", fault.Wrap(fault.CodeUnavailable, "model unavailable", secretCause)
	}, nil)
	writer := &recordingWriter{}

	started, err := write(context.Background(), reader, stringEncoder, func() eventWriter { return writer })
	if err == nil {
		t.Fatal("write() error = nil")
	}
	if !started {
		t.Fatal("write() started = false")
	}
	if len(writer.events) != 2 || writer.events[1].eventType != "error" {
		t.Fatalf("events = %#v", writer.events)
	}
	if bytes.Contains(writer.events[1].data, []byte("secret")) {
		t.Fatalf("terminal event leaked cause: %s", writer.events[1].data)
	}
	var payload struct {
		Code    fault.Code `json:"code"`
		Message string     `json:"message"`
	}
	if decodeErr := json.Unmarshal(writer.events[1].data, &payload); decodeErr != nil {
		t.Fatalf("decode terminal event: %v", decodeErr)
	}
	if payload.Code != fault.CodeUnavailable || payload.Message != "model unavailable" {
		t.Fatalf("terminal event = %#v", payload)
	}
}

func TestEmptyStreamDoesNotStartSSE(t *testing.T) {
	reader := kitstream.FromSlice([]int(nil))
	started, err := write(context.Background(), reader, func(context.Context, int) (Event, error) {
		return Event{}, nil
	}, func() eventWriter {
		t.Fatal("writer should not be created")
		return nil
	})
	if err != nil {
		t.Fatalf("write() error = %v", err)
	}
	if started {
		t.Fatal("write() started = true")
	}
}

func TestCancelledBeforeFirstRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := kitstream.NewReader(func() (string, error) {
		t.Fatal("Recv was called after cancellation")
		return "", io.EOF
	}, nil)

	started, err := write(ctx, reader, stringEncoder, func() eventWriter { return &recordingWriter{} })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("write() error = %v, want context.Canceled", err)
	}
	if started {
		t.Fatal("write() started = true")
	}
}

func stringEncoder(_ context.Context, item string) (Event, error) {
	payload, err := json.Marshal(map[string]string{"text": item})
	return Event{Type: "message", Data: payload}, err
}
