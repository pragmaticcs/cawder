package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const SystemPrompt = "You are a helpful coding agent named Cawder working in the user's " +
	"current directory under a {{OS}} environment. Use the provided tools to inspect and modify code. " +
	"Take one concrete step at a time. When invoking tools always use the correct tool call format."

const osPlaceholder string = "{{OS}}"

func LoadSystemPrompt(dirName, fileName string) (string, error) {
	dirPath := filepath.Join(".", dirName)
	filePath := filepath.Join(dirPath, fileName)
	_, err := os.Stat(filePath)

	if err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", fileName, err)
		}

		system := SystemPrompt + fmt.Sprintf("\nHere are additional instructions:\n%s", string(data))

		return resolveSystemPrompt(system), nil
	}
	return resolveSystemPrompt(SystemPrompt), nil
}

func resolveSystemPrompt(raw string) string {
	return strings.TrimSpace(strings.ReplaceAll(raw, osPlaceholder, runtime.GOOS))
}
