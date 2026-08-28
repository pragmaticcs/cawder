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
	return "Lists the entries of a directory sorted by name. Directories have a trailing '/'. " +
		"Defaults to the current directory."
}

func (t *ListDirTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to list. Defaults to the current directory.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *ListDirTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	var input struct {
		Path string `json:"path"`
	}

	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return ToolCallResult{
				Content: "error: invalid arguments: " + err.Error(),
				Error:   true,
			}, nil
		}
	}

	path := input.Path
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolCallResult{
			Content: "error: failed to list " + path + ": " + err.Error(),
			Error:   true,
		}, nil
	}

	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	if len(entries) == 0 {
		return ToolCallResult{
			Content: "[" + path + ": empty directory]",
			MainArg: path,
		}, nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}

	sort.Strings(names)

	return ToolCallResult{
		Content: strings.Join(names, "\n"),
		MainArg: path,
	}, nil
}
