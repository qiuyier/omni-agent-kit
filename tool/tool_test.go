package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/qiuyier/omni-agent-kit/tool"
)

func TestFuncTool(t *testing.T) {
	definition := tool.Definition{
		Name:        "lookup_order",
		Description: "Look up an order by its identifier.",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"order_id":{"type":"string"}},
			"required":["order_id"]
		}`),
	}

	gotArguments := json.RawMessage(nil)
	wrapped, err := tool.New(definition, func(_ context.Context, arguments json.RawMessage) (json.RawMessage, error) {
		gotArguments = append(gotArguments, arguments...)
		return json.RawMessage(`{"status":"paid"}`), nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := wrapped.Invoke(context.Background(), json.RawMessage(`{"order_id":"A-42"}`))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if string(gotArguments) != `{"order_id":"A-42"}` {
		t.Fatalf("handler arguments = %s", gotArguments)
	}
	if string(result) != `{"status":"paid"}` {
		t.Fatalf("result = %s", result)
	}
}

func TestDefinitionIsCopied(t *testing.T) {
	inputSchema := json.RawMessage(`{"type":"object"}`)
	wrapped, err := tool.New(tool.Definition{
		Name:        "ping",
		Description: "Return a health response.",
		InputSchema: inputSchema,
	}, func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	inputSchema[2] = 'X'
	first, err := wrapped.Definition(context.Background())
	if err != nil {
		t.Fatalf("Definition() error = %v", err)
	}
	first.InputSchema[2] = 'Y'
	second, _ := wrapped.Definition(context.Background())
	if string(second.InputSchema) != `{"type":"object"}` {
		t.Fatalf("definition was mutated through an external slice: %s", second.InputSchema)
	}
}

func TestInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name       string
		definition tool.Definition
	}{
		{name: "missing name", definition: tool.Definition{Description: "desc"}},
		{name: "missing description", definition: tool.Definition{Name: "name"}},
		{name: "invalid schema", definition: tool.Definition{Name: "name", Description: "desc", InputSchema: json.RawMessage(`{`)}},
		{name: "non-object schema", definition: tool.Definition{Name: "name", Description: "desc", InputSchema: json.RawMessage(`[]`)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := tool.New(test.definition, func(context.Context, json.RawMessage) (json.RawMessage, error) {
				return nil, nil
			})
			if err == nil {
				t.Fatal("New() error = nil")
			}
		})
	}
}

func TestInvokeValidatesJSONBoundaries(t *testing.T) {
	wrapped, err := tool.New(tool.Definition{Name: "test", Description: "Test JSON validation."}, func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`not-json`), nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := wrapped.Invoke(context.Background(), json.RawMessage(`[]`)); err == nil {
		t.Fatal("Invoke() accepted non-object arguments")
	}
	if _, err := wrapped.Invoke(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("Invoke() accepted an invalid JSON result")
	}
}

func TestInvokePropagatesCancellation(t *testing.T) {
	called := false
	wrapped, err := tool.New(tool.Definition{Name: "test", Description: "Test cancellation."}, func(context.Context, json.RawMessage) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`null`), nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = wrapped.Invoke(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Invoke() error = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("handler was called after cancellation")
	}
}
