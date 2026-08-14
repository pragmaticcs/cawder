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
	return "Writes (overwrites) a file with the given content, creating it (and any missing parent behavior aside - the directory must already exist) if it doesn't exist."
}

func (t *WriteFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The full content to write to the file.",
			},
		},
		"required":             []string{"path", "content"},
		"additionalProperties": false,
	}
}

func (t *WriteFileTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{Content: "Failed to parse arguments: " + err.Error(), Error: true}, nil
	}
	if input.Path == "" {
		return ToolCallResult{Content: "path is required", Error: true}, nil
	}

	if err := os.WriteFile(input.Path, []byte(input.Content), 0o644); err != nil {
		return ToolCallResult{Content: "Failed to write file: " + err.Error(), Error: true}, nil
	}

	return ToolCallResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(input.Content), input.Path), MainArg: input.Path}, nil
}
