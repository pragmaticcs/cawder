package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/core/config"
	"github.com/pragmaticcs/cawder/internal/core/memory"
	tui "github.com/pragmaticcs/cawder/internal/ui"
)

const (
	configDir  = ".cawder"
	configFile = "conf.toml"
	sessionDir = "sessions"
	systemFile = "SYSTEM.md"
)

func main() {
	cfg, err := config.LoadConfig(configDir, configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running cawder:", err)
		os.Exit(1)
	}

	system, err := config.LoadSystemPrompt(configDir, systemFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running cawder:", err)
		os.Exit(1)
	}
	cfg.System = system

	sessions, err := memory.NewSessionManager(filepath.Join(configDir, sessionDir))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running cawder:", err)
		os.Exit(1)
	}
	defer func() {
		if err := sessions.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "error closing session store:", err)
		}
	}()

	app, err := tui.New(*cfg, sessions)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running cawder:", err)
		os.Exit(1)
	}
	program := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running cawder:", err)
		os.Exit(1)
	}
}
