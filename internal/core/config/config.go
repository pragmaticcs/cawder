package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type (
	AgentConfig struct {
		Selected string                `toml:"selected"`
		Models   map[string]AgentModel `toml:"models"`
		System   string                `toml:"-"`
	}
	AgentModel struct {
		Name    string `toml:"name"`
		Url     string `toml:"url"`
		Key     string `toml:"key"`
		Context int    `toml:"context"`
	}
)

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Selected: "",
		Models: map[string]AgentModel{
			"local": {
				Name:    "MODEL_NAME",
				Url:     "URL",
				Key:     "API_KEY",
				Context: 32000,
			},
		},
	}
}

func LoadConfig(dirName, fileName string) (*AgentConfig, error) {
	dirPath := filepath.Join(".", dirName)
	filePath := filepath.Join(dirPath, fileName)

	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create config directory: %w", err)
			}

			cfg := DefaultAgentConfig()

			data, err := toml.Marshal(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal default config: %w", err)
			}

			if err := os.WriteFile(filePath, data, 0644); err != nil {
				return nil, fmt.Errorf("failed to write default config file: %w", err)
			}

			return &cfg, nil
		}

		return nil, fmt.Errorf("error accessing config file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg AgentConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
