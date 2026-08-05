package main

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/qiuyier/omni-agent-kit/adapter/eino/tool"
	kittool "github.com/qiuyier/omni-agent-kit/tool"
)

func main() {
	source, err := kittool.New(kittool.Definition{
		Name:        "lookup_order",
		Description: "Look up an order by identifier.",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"order_id":{"type":"string"}},
			"required":["order_id"]
		}`),
	}, func(_ context.Context, arguments json.RawMessage) (json.RawMessage, error) {
		var input map[string]string
		if err := json.Unmarshal(arguments, &input); err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{
			"order_id": input["order_id"],
			"status":   "paid",
		})
	})
	if err != nil {
		panic(err)
	}

	adapted, err := einotool.Wrap(source)
	if err != nil {
		panic(err)
	}
	result, err := adapted.InvokableRun(context.Background(), `{"order_id":"A-42"}`)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}
