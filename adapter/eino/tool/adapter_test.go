package einotool_test

import (
	"context"
	"encoding/json"
	"testing"

	einotool "github.com/qiuyier/omni-agent-kit/adapter/eino/tool"
	kittool "github.com/qiuyier/omni-agent-kit/tool"
)

func TestWrap(t *testing.T) {
	source, err := kittool.New(kittool.Definition{
		Name:        "lookup_order",
		Description: "Look up an order by identifier.",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"order_id":{"type":"string"}},
			"required":["order_id"]
		}`),
	}, func(_ context.Context, arguments json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"received":` + string(arguments) + `}`), nil
	})
	if err != nil {
		t.Fatalf("kittool.New() error = %v", err)
	}

	adapted, err := einotool.Wrap(source)
	if err != nil {
		t.Fatalf("Wrap() error = %v", err)
	}
	info, err := adapted.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "lookup_order" {
		t.Fatalf("Info().Name = %q", info.Name)
	}
	parsed, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema() error = %v", err)
	}
	if parsed == nil || len(parsed.Required) != 1 || parsed.Required[0] != "order_id" {
		t.Fatalf("required schema fields = %#v", parsed)
	}

	result, err := adapted.InvokableRun(context.Background(), `{"order_id":"A-42"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if result != `{"received":{"order_id":"A-42"}}` {
		t.Fatalf("InvokableRun() = %s", result)
	}
}

func TestNoInputSchema(t *testing.T) {
	source, err := kittool.New(kittool.Definition{
		Name:        "ping",
		Description: "Return a health response.",
	}, func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	})
	if err != nil {
		t.Fatalf("kittool.New() error = %v", err)
	}

	adapted, err := einotool.Wrap(source)
	if err != nil {
		t.Fatalf("Wrap() error = %v", err)
	}
	info, err := adapted.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.ParamsOneOf != nil {
		t.Fatal("ParamsOneOf is non-nil for a tool without an input schema")
	}
}

func TestWrapRejectsNil(t *testing.T) {
	if _, err := einotool.Wrap(nil); err == nil {
		t.Fatal("Wrap(nil) error = nil")
	}
}
