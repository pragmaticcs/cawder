package ui

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/core"
	"github.com/pragmaticcs/cawder/internal/core/config"
	"github.com/pragmaticcs/cawder/internal/core/memory"
	"github.com/pragmaticcs/cawder/internal/ui/components/chat"
	"github.com/pragmaticcs/cawder/internal/ui/components/input"
	"github.com/pragmaticcs/cawder/internal/ui/components/picker"
	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

type TUIMode int

const (
	TUIModeChat TUIMode = iota
	TUIModeSessionSelect
	TUIModeModelSelect
)

type Model struct {
	// Agent
	agent         *core.AgentLoop
	agentCh       <-chan core.AgentEvent
	config        config.AgentConfig
	selectedModel config.AgentModel

	// Session
	sessions *memory.SessionManager

	// Visual components
	chat          chat.Model
	input         input.Model
	spinner       spinner.Model
	modelPicker   picker.Model[config.AgentModel]
	sessionPicker picker.Model[SessionItem]

	// State
	mode       TUIMode
	responding bool
	startTime  time.Time
	elapsed    time.Duration
	tokenCount int
	err        error

	// Config
	width  int
	height int
}

func New(c config.AgentConfig, s *memory.SessionManager) (Model, error) {
	spinner := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.StatusStyle),
	)

	if len(c.Selected) == 0 {
		return Model{}, errors.New("select a model before launching cawder")
	}
	selectedModel, ok := c.Models[c.Selected]
	if !ok {
		return Model{}, fmt.Errorf("selected model (%s) is not declared ", c.Selected)
	}

	return Model{
		agent:         nil,
		sessions:      s,
		config:        c,
		selectedModel: selectedModel,
		chat:          chat.New(),
		input:         input.New("Build me a startup. Do not hallucinate. Do not make mistakes."),
		spinner:       spinner,
		mode:          TUIModeChat,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.input.Init()
}
