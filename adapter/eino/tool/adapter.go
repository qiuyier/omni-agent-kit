// Package einotool adapts framework-neutral kit tools to Eino tools.
package einotool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"

	kittool "github.com/qiuyier/omni-agent-kit/tool"
)

// Adapter implements Eino's InvokableTool using a framework-neutral kit Tool.
type Adapter struct {
	source kittool.Tool
}

var _ tool.InvokableTool = (*Adapter)(nil)

// Wrap creates an Eino InvokableTool. The source definition is resolved in
// Info so context-dependent definition failures remain visible to Eino.
func Wrap(source kittool.Tool) (tool.InvokableTool, error) {
	if source == nil {
		return nil, errors.New("einotool: nil source tool")
	}
	return &Adapter{source: source}, nil
}

// Info converts the kit JSON Schema definition to Eino's ToolInfo.
func (a *Adapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	definition, err := a.source.Definition(ctx)
	if err != nil {
		return nil, fmt.Errorf("einotool: load definition: %w", err)
	}
	if err := kittool.ValidateDefinition(definition); err != nil {
		return nil, fmt.Errorf("einotool: invalid definition: %w", err)
	}

	info := &schema.ToolInfo{Name: definition.Name, Desc: definition.Description}
	inputSchema := bytes.TrimSpace(definition.InputSchema)
	if len(inputSchema) == 0 {
		return info, nil
	}

	var parsed jsonschema.Schema
	if err := json.Unmarshal(inputSchema, &parsed); err != nil {
		return nil, fmt.Errorf("einotool: decode input schema: %w", err)
	}
	info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&parsed)
	return info, nil
}

// InvokableRun executes the source tool. Eino options are intentionally not
// translated: product-specific behavior belongs in the Tool implementation or
// Eino middleware, not in this protocol adapter.
func (a *Adapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	arguments := bytes.TrimSpace([]byte(argumentsInJSON))
	if len(arguments) == 0 {
		arguments = []byte(`{}`)
	}
	if !json.Valid(arguments) {
		return "", errors.New("einotool: arguments are not valid JSON")
	}

	result, err := a.source.Invoke(ctx, json.RawMessage(arguments))
	if err != nil {
		return "", err
	}
	if !json.Valid(result) {
		return "", errors.New("einotool: source returned invalid JSON")
	}
	return string(result), nil
}
