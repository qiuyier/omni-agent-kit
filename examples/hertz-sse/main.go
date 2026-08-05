package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/qiuyier/omni-agent-kit/stream"
	"github.com/qiuyier/omni-agent-kit/transport/hertz/problem"
	kitsse "github.com/qiuyier/omni-agent-kit/transport/hertz/sse"
)

type answerEvent struct {
	Text string
}

func main() {
	h := server.Default(server.WithSenseClientDisconnection(true))
	h.GET("/answers/stream", streamAnswer)
	h.Spin()
}

func streamAnswer(ctx context.Context, c *app.RequestContext) {
	reader := stream.FromSlice([]answerEvent{
		{Text: "Hello"},
		{Text: " from"},
		{Text: " omni-agent-kit"},
	})

	started, err := kitsse.Write(ctx, c, reader, func(_ context.Context, item answerEvent) (kitsse.Event, error) {
		payload, marshalErr := json.Marshal(item)
		return kitsse.Event{Type: "answer.delta", Data: payload}, marshalErr
	})
	if err == nil {
		return
	}
	if !started {
		_ = problem.Write(c, err)
		return
	}
	log.Printf("stream ended with error: %v", err)
}
