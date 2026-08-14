package tools

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error)
}

type ToolCallResult struct {
	Content string
	MainArg string
	Error   bool
}
