package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxSearchResults    = 100
	maxSearchOutputSize = 100 * 1024
	maxSearchLineBytes  = 10 * 1024 * 1024
)

type SearchTool struct{}

func NewSearchTool() *SearchTool {
	return &SearchTool{}
}

func (t *SearchTool) Name() string {
	return "search"
}

func (t *SearchTool) Description() string {
	return "Searches files for a literal text string. " +
		"Path may be a file or directory. Directory searches are recursive. " +
		"Returns matching file paths and 1-based line numbers, up to 100 matches."
}

func (t *SearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Literal text to search for.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search. Defaults to the current directory.",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matches to return. Defaults to 100.",
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}

func (t *SearchTool) Call(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Query      string `json:"query"`
		Path       string `json:"path"`
		MaxResults *int   `json:"max_results"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{
			Content: "error: invalid arguments: " + err.Error(),
			Error:   true,
		}, nil
	}

	if input.Query == "" {
		return ToolCallResult{
			Content: "error: query is required",
			Error:   true,
		}, nil
	}

	root := input.Path
	if root == "" {
		root = "."
	}

	maxResults := maxSearchResults
	if input.MaxResults != nil {
		if *input.MaxResults <= 0 {
			return ToolCallResult{
				Content: "error: max_results must be greater than 0",
				Error:   true,
			}, nil
		}
		maxResults = min(*input.MaxResults, maxSearchResults)
	}

	info, err := os.Stat(root)
	if err != nil {
		return ToolCallResult{
			Content: fmt.Sprintf("error: failed to access %s: %v", root, err),
			Error:   true,
		}, nil
	}

	var files []string

	if info.IsDir() {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if err := ctx.Err(); err != nil {
				return err
			}

			if d.IsDir() {
				switch d.Name() {
				case ".git", ".hg", ".svn", "node_modules", "vendor":
					if path != root {
						return filepath.SkipDir
					}
				}
				return nil
			}

			if d.Type().IsRegular() {
				files = append(files, path)
			}

			return nil
		})
		if err != nil {
			if ctx.Err() != nil {
				return ToolCallResult{
					Content: "error: search cancelled",
					Error:   true,
				}, nil
			}

			return ToolCallResult{
				Content: fmt.Sprintf("error: failed to walk %s: %v", root, err),
				Error:   true,
			}, nil
		}
	} else {
		files = []string{root}
	}

	var b strings.Builder
	matches := 0
	truncated := false

	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return ToolCallResult{
				Content: "error: search cancelled",
				Error:   true,
			}, nil
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), maxSearchLineBytes)

		lineNo := 0

		for scanner.Scan() {
			lineNo++

			line := scanner.Text()
			if !strings.Contains(line, input.Query) {
				continue
			}

			resultPath := path
			if info.IsDir() {
				resultPath, _ = filepath.Rel(root, path)
				if resultPath == "." {
					resultPath = path
				}
			}

			entry := fmt.Sprintf("%s:%d:%s\n", resultPath, lineNo, line)

			if matches >= maxResults ||
				b.Len()+len(entry) > maxSearchOutputSize {
				truncated = true
				break
			}

			b.WriteString(entry)
			matches++
		}

		_ = f.Close()

		if truncated {
			break
		}
	}

	if err := ctx.Err(); err != nil {
		return ToolCallResult{
			Content: "error: search cancelled",
			Error:   true,
		}, nil
	}

	if matches == 0 {
		return ToolCallResult{
			Content: fmt.Sprintf("no matches for %q", input.Query),
			MainArg: root,
		}, nil
	}

	if truncated {
		b.WriteString(fmt.Sprintf(
			"[results truncated at %d matches or %d bytes]",
			maxResults,
			maxSearchOutputSize,
		))
	}

	return ToolCallResult{
		Content: b.String(),
		MainArg: root,
	}, nil
}
