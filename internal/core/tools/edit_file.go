package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	return "Replaces exactly one occurrence of old with new in a file. " +
		"old must match exactly once. read_file line-number prefixes may be included and are stripped automatically."
}

func (t *EditFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to edit.",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "Exact text to replace. It must occur exactly once.",
			},
			"new": map[string]any{
				"type":        "string",
				"description": "Replacement text.",
			},
		},
		"required":             []string{"path", "old", "new"},
		"additionalProperties": false,
	}
}

func (t *EditFileTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	var input struct {
		Path string `json:"path"`
		Old  string `json:"old"`
		New  string `json:"new"`
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

	data, err := os.ReadFile(input.Path)
	if err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to read %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	src := string(data)

	oldStr := input.Old
	newStr := input.New

	count := strings.Count(src, oldStr)

	if count != 1 {
		if stripped := stripLineNumbers(oldStr); stripped != oldStr {
			if strippedCount := strings.Count(src, stripped); strippedCount == 1 {
				oldStr = stripped
				newStr = stripLineNumbers(newStr)
				count = 1
			}
		}
	}

	if count != 1 {
		return ToolCallResult{
			Content: fmt.Sprintf(
				"error: old text matched %d times; expected exactly 1",
				count,
			),
			Error: true,
		}, nil
	}

	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: operation cancelled",
			Error:   true,
		}, nil
	}

	updated := strings.Replace(src, oldStr, newStr, 1)

	info, err := os.Stat(input.Path)
	if err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to stat %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	if err := atomicWriteFile(input.Path, []byte(updated), info.Mode().Perm()); err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to write %s: %v", input.Path, err),
			Error:   true,
		}, nil
	}

	return ToolCallResult{
		Content: fmt.Sprintf("edited %s", input.Path),
		MainArg: input.Path,
	}, nil
}

func atomicWriteFile(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".agent-edit-*")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
