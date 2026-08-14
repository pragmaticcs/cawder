package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const defaultReadFileLines = 400
const maxReadFileLineBytes = 10 * 1024 * 1024

type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Reads a file's contents as numbered lines (1-based, like `cat -n`: a right-aligned line number, a tab, then the line). Large files return only a window - pass offset (1-based start line) and limit (max lines, default 400; <=0 reads to end) to page through the rest."
}

func (t *ReadFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to read.",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "1-based line number to start from (default 1).",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Max lines to return (default 400; <=0 reads to the end of the file).",
			},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *ReadFileTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  *int   `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{Content: "Failed to parse arguments: " + err.Error(), Error: true}, nil
	}
	if input.Path == "" {
		return ToolCallResult{Content: "path is required", Error: true}, nil
	}

	f, err := os.Open(input.Path)
	if err != nil {
		return ToolCallResult{Content: "Failed to read file: " + err.Error(), Error: true}, nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxReadFileLineBytes)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return ToolCallResult{Content: "Failed to read file: " + err.Error(), Error: true}, nil
	}

	total := len(lines)
	if total == 0 {
		return ToolCallResult{Content: fmt.Sprintf("[%s: empty file]", input.Path), MainArg: input.Path}, nil
	}

	start := input.Offset - 1
	start = max(start, 0)
	if start >= total {
		return ToolCallResult{
			Content: fmt.Sprintf("[%s: %d lines; offset %d is past end of file]", input.Path, total, start+1),
			Error:   true,
		}, nil
	}

	limit := defaultReadFileLines
	if input.Limit != nil {
		limit = *input.Limit
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	var b strings.Builder
	if start > 0 || end < total {
		fmt.Fprintf(&b, "[%s: lines %d-%d of %d; call read_file with offset/limit to page]\n", input.Path, start+1, end, total)
	}
	for i := start; i < end; i++ {
		fmt.Fprintf(&b, "%6d\t%s\n", i+1, lines[i])
	}

	return ToolCallResult{Content: b.String(), MainArg: input.Path}, nil
}
