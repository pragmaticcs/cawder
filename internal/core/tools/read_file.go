package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	defaultReadFileLines = 400
	maxReadFileLineBytes = 10 * 1024 * 1024
)

type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Reads up to " + string(defaultReadFileLines) + " lines from a text file with 1-based line numbers. " +
		"Use offset to start at a specific line and limit to control how many lines are returned. " +
		"Use another read_file call to page through the file."
}

func (t *ReadFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read.",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "1-based line number to start from. Defaults to 1.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Maximum number of lines to return. Defaults to %d and must be greater than 0.", defaultReadFileLines),
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

	offset := input.Offset
	if offset <= 0 {
		offset = 1
	}

	limit := defaultReadFileLines
	if input.Limit != nil {
		if *input.Limit <= 0 {
			return ToolCallResult{
				Content: "error: limit must be greater than 0",
				Error:   true,
			}, nil
		}
		limit = *input.Limit
	}

	f, err := os.Open(input.Path)
	if err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to open %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxReadFileLineBytes)

	var b strings.Builder

	lineNo := 0
	linesReturned := 0
	more := false

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return ToolCallResult{
				Content: "error: read cancelled",
				Error:   true,
			}, nil
		}

		lineNo++

		if lineNo < offset {
			continue
		}

		if linesReturned >= limit {
			more = true
			break
		}

		fmt.Fprintf(&b, "%6d\t%s\n", lineNo, scanner.Text())
		linesReturned++
	}

	if err := scanner.Err(); err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to read %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	if lineNo == 0 {
		return ToolCallResult{
			Content: fmt.Sprintf("[%s: empty file]", input.Path),
			MainArg: input.Path,
		}, nil
	}

	if linesReturned == 0 && offset > lineNo {
		return ToolCallResult{
			Content: fmt.Sprintf(
				"error: %s has %d lines; offset %d is past the end of the file",
				input.Path,
				lineNo,
				offset,
			),
			Error: true,
		}, nil
	}

	end := offset + linesReturned - 1

	var header strings.Builder
	fmt.Fprintf(
		&header,
		"[%s: lines %d-%d",
		input.Path,
		offset,
		end,
	)

	if more {
		header.WriteString("; more lines follow")
	}

	header.WriteString("]\n")

	return ToolCallResult{
		Content: header.String() + b.String(),
		MainArg: input.Path,
	}, nil
}
