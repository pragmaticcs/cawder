package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type WriteFileTool struct{}

func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Writes the full content to a file, replacing any existing content. " +
		"Parent directories must already exist."
}

func (t *WriteFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Full file content.",
			},
		},
		"required":             []string{"path", "content"},
		"additionalProperties": false,
	}
}

func (t *WriteFileTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{
			Content: "error: invalid arguments: " + err.Error(),
			Error:   true,
		}, nil
	}

	if input.Path == "" {
		return ToolCallResult{
			Content: "error: path is required",
			Error:   true,
		}, nil
	}

	mode := os.FileMode(0o644)

	if info, err := os.Stat(input.Path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to stat %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	if err := atomicWriteFile(input.Path, []byte(input.Content), mode); err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to write %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	return ToolCallResult{
		Content: fmt.Sprintf("wrote %d bytes to %s", len(input.Content), input.Path),
		MainArg: input.Path,
	}, nil
}
