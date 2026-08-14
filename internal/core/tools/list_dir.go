package tools

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
)

type ListDirTool struct{}

func NewListDirTool() *ListDirTool {
	return &ListDirTool{}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "Lists the entries of a directory, sorted by name. Defaults to the current directory."
}

func (t *ListDirTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to list (default: current directory).",
			},
		},
		"additionalProperties": false,
	}
}

func (t *ListDirTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Path string `json:"path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return ToolCallResult{Content: "Failed to parse arguments: " + err.Error(), Error: true}, nil
		}
	}

	path := input.Path
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolCallResult{Content: "Failed to list directory: " + err.Error(), Error: true}, nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	return ToolCallResult{Content: strings.Join(names, "\n"), MainArg: path}, nil
}
