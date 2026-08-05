// Package sse bridges a pull-based kit stream to Hertz Server-Sent Events.
package sse

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hertzsse "github.com/cloudwego/hertz/pkg/protocol/sse"

	"github.com/qiuyier/omni-agent-kit/fault"
	kitstream "github.com/qiuyier/omni-agent-kit/stream"
)

// Event is a transport-ready Server-Sent Event.
type Event struct {
	ID   string
	Type string
	Data []byte
}

// Encoder maps a product-specific stream item to an SSE event.
type Encoder[T any] func(ctx context.Context, item T) (Event, error)

type eventWriter interface {
	WriteEvent(id, eventType string, data []byte) error
	Close() error
}

// Write synchronously drains reader into a Hertz response and returns whether
// streaming started. The handler must stay active until Write returns.
//
// If an error occurs before the first event, started is false and the caller
// can still write a normal HTTP problem response. If an error occurs after the
// first event, Write emits a terminal "error" event when possible and returns
// started=true so the caller does not attempt to replace the response body.
func Write[T any](ctx context.Context, request *app.RequestContext, reader kitstream.Reader[T], encode Encoder[T]) (started bool, err error) {
	if request == nil {
		return false, errors.New("sse: nil request context")
	}
	return write(ctx, reader, encode, func() eventWriter {
		return hertzsse.NewWriter(request)
	})
}

func write[T any](ctx context.Context, reader kitstream.Reader[T], encode Encoder[T], newWriter func() eventWriter) (started bool, err error) {
	if ctx == nil {
		return false, errors.New("sse: nil context")
	}
	if reader == nil {
		return false, errors.New("sse: nil stream reader")
	}
	if encode == nil {
		return false, errors.New("sse: nil event encoder")
	}

	var writer eventWriter
	defer func() {
		readerErr := reader.Close()
		var writerErr error
		if writer != nil {
			writerErr = writer.Close()
		}
		err = errors.Join(err, readerErr, writerErr)
	}()

	item, recvErr := receive(ctx, reader)
	if errors.Is(recvErr, io.EOF) {
		return false, nil
	}
	if recvErr != nil {
		return false, recvErr
	}
	event, encodeErr := encode(ctx, item)
	if encodeErr != nil {
		return false, encodeErr
	}
	if validateErr := validateEvent(event); validateErr != nil {
		return false, validateErr
	}

	writer = newWriter()
	started = true
	if writeErr := writer.WriteEvent(event.ID, event.Type, event.Data); writeErr != nil {
		return true, writeErr
	}

	for {
		item, recvErr = receive(ctx, reader)
		if errors.Is(recvErr, io.EOF) {
			return true, nil
		}
		if recvErr != nil {
			return true, errors.Join(recvErr, writeStreamError(writer, recvErr))
		}

		event, encodeErr = encode(ctx, item)
		if encodeErr != nil {
			return true, errors.Join(encodeErr, writeStreamError(writer, encodeErr))
		}
		if validateErr := validateEvent(event); validateErr != nil {
			return true, errors.Join(validateErr, writeStreamError(writer, validateErr))
		}
		if writeErr := writer.WriteEvent(event.ID, event.Type, event.Data); writeErr != nil {
			return true, writeErr
		}
	}
}

func receive[T any](ctx context.Context, reader kitstream.Reader[T]) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	return reader.Recv()
}

func validateEvent(event Event) error {
	if strings.ContainsAny(event.ID, "\r\n") {
		return errors.New("sse: event ID contains a line break")
	}
	if strings.ContainsAny(event.Type, "\r\n") {
		return errors.New("sse: event type contains a line break")
	}
	return nil
}

func writeStreamError(writer eventWriter, err error) error {
	payload, marshalErr := json.Marshal(struct {
		Code    fault.Code `json:"code"`
		Message string     `json:"message"`
	}{
		Code:    fault.CodeOf(err),
		Message: fault.SafeMessage(err),
	})
	if marshalErr != nil {
		return marshalErr
	}
	return writer.WriteEvent("", "error", payload)
}
