package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var lineNumPrefixRe = regexp.MustCompile(`^ *\d+\t`)

func stripLineNumbers(text string) string {
	lines := strings.Split(text, "\n")
	sawNonEmpty := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		sawNonEmpty = true
		if !lineNumPrefixRe.MatchString(ln) {
			return text
		}
	}
	if !sawNonEmpty {
		return text
	}
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = lineNumPrefixRe.ReplaceAllString(ln, "")
	}
	return strings.Join(out, "\n")
}

type EditFileTool struct{}

func NewEditFileTool() *EditFileTool {
	return &EditFileTool{}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Replaces one exact occurrence of `old` with `new` in a file. `old` must match the file's raw text exactly and uniquely; if you paste read_file's numbered lines instead, the `<n>\\t` line-number prefixes are stripped automatically before matching."
}

func (t *EditFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to edit.",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "The exact text to replace. Must occur exactly once in the file.",
			},
			"new": map[string]any{
				"type":        "string",
				"description": "The text to replace it with.",
			},
		},
		"required":             []string{"path", "old", "new"},
		"additionalProperties": false,
	}
}

func (t *EditFileTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Path string `json:"path"`
		Old  string `json:"old"`
		New  string `json:"new"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{
			Content: "Failed to parse arguments: " + err.Error(), Error: true}, nil
	}
	if input.Path == "" {
		return ToolCallResult{Content: "path is required", Error: true}, nil
	}

	data, err := os.ReadFile(input.Path)
	if err != nil {
		return ToolCallResult{Content: "Failed to read file: " + err.Error(), Error: true}, nil
	}
	src := string(data)

	oldStr, newStr := input.Old, input.New
	count := strings.Count(src, oldStr)
	if count != 1 {
		if stripped := stripLineNumbers(oldStr); stripped != oldStr && strings.Count(src, stripped) == 1 {
			oldStr, newStr = stripped, stripLineNumbers(newStr)
			count = 1
		}
	}
	if count != 1 {
		return ToolCallResult{
			Content: fmt.Sprintf("`old` matched %d times (need exactly 1)", strings.Count(src, oldStr)),
			Error:   true,
		}, nil
	}

	updated := strings.Replace(src, oldStr, newStr, 1)
	if err := os.WriteFile(input.Path, []byte(updated), 0o644); err != nil {
		return ToolCallResult{Content: "Failed to write file: " + err.Error(), Error: true}, nil
	}

	return ToolCallResult{Content: fmt.Sprintf("edited %s", input.Path), MainArg: input.Path}, nil
}
