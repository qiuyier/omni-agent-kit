// Package tool defines a framework-neutral AI tool contract based on JSON
// Schema and JSON values.
package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Definition describes a tool without exposing runtime credentials or other
// secret configuration. InputSchema and OutputSchema use JSON Schema.
type Definition struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
}

// Tool is the stable boundary between product code and framework adapters.
// Arguments and results must be valid JSON values. An error represents an
// execution or infrastructure failure; a business outcome the model can act on
// should normally be returned as structured JSON instead.
type Tool interface {
	Definition(ctx context.Context) (Definition, error)
	Invoke(ctx context.Context, arguments json.RawMessage) (json.RawMessage, error)
}

// InvokeFunc implements a tool operation.
type InvokeFunc func(ctx context.Context, arguments json.RawMessage) (json.RawMessage, error)

type funcTool struct {
	definition Definition
	invoke     InvokeFunc
}

// New creates a Tool from a static definition and an invocation function.
func New(definition Definition, invoke InvokeFunc) (Tool, error) {
	if err := ValidateDefinition(definition); err != nil {
		return nil, err
	}
	if invoke == nil {
		return nil, errors.New("tool: nil invoke function")
	}
	return &funcTool{definition: cloneDefinition(definition), invoke: invoke}, nil
}

// ValidateDefinition validates the transport-independent parts of a tool
// definition. It checks that schemas are JSON objects, but intentionally does
// not implement full JSON Schema validation.
func ValidateDefinition(definition Definition) error {
	if strings.TrimSpace(definition.Name) == "" {
		return errors.New("tool: name is required")
	}
	if definition.Name != strings.TrimSpace(definition.Name) {
		return errors.New("tool: name must not have surrounding whitespace")
	}
	if strings.IndexFunc(definition.Name, unicode.IsControl) >= 0 {
		return errors.New("tool: name must not contain control characters")
	}
	if strings.TrimSpace(definition.Description) == "" {
		return errors.New("tool: description is required")
	}
	if err := validateSchema("input schema", definition.InputSchema); err != nil {
		return err
	}
	if err := validateSchema("output schema", definition.OutputSchema); err != nil {
		return err
	}
	return nil
}

func validateSchema(name string, schema json.RawMessage) error {
	if len(bytes.TrimSpace(schema)) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(schema, &value); err != nil {
		return fmt.Errorf("tool: invalid %s: %w", name, err)
	}
	if value == nil {
		return fmt.Errorf("tool: %s must be a JSON object", name)
	}
	return nil
}

func (t *funcTool) Definition(context.Context) (Definition, error) {
	return cloneDefinition(t.definition), nil
}

func (t *funcTool) Invoke(ctx context.Context, arguments json.RawMessage) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	arguments = bytes.TrimSpace(arguments)
	if len(arguments) == 0 {
		arguments = json.RawMessage(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal(arguments, &object); err != nil {
		return nil, fmt.Errorf("tool: arguments must be a JSON object: %w", err)
	}
	if object == nil {
		return nil, errors.New("tool: arguments must be a JSON object")
	}

	result, err := t.invoke(ctx, append(json.RawMessage(nil), arguments...))
	if err != nil {
		return nil, err
	}
	result = bytes.TrimSpace(result)
	if len(result) == 0 {
		result = json.RawMessage(`null`)
	}
	if !json.Valid(result) {
		return nil, errors.New("tool: result must be valid JSON")
	}
	return append(json.RawMessage(nil), result...), nil
}

func cloneDefinition(definition Definition) Definition {
	definition.InputSchema = append(json.RawMessage(nil), definition.InputSchema...)
	definition.OutputSchema = append(json.RawMessage(nil), definition.OutputSchema...)
	return definition
}
