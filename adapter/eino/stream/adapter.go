// Package einostream adapts Eino streams to omni-agent-kit streams.
package einostream

import (
	"errors"

	"github.com/cloudwego/eino/schema"

	kitstream "github.com/qiuyier/omni-agent-kit/stream"
)

// Wrap adapts an Eino StreamReader. The returned reader owns source and closes
// it when Close is called.
func Wrap[T any](source *schema.StreamReader[T]) (kitstream.Reader[T], error) {
	if source == nil {
		return nil, errors.New("einostream: nil source reader")
	}
	return kitstream.NewReader(source.Recv, func() error {
		source.Close()
		return nil
	}), nil
}
