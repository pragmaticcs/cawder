package chat

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/ui/components/message"
	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

type Model struct {
	viewport  viewport.Model
	messages  []message.Model
	streaming *message.Model
	ready     bool

	toolIndex map[string]int

	renderers styles.Renderers
}

func New() Model {
	return Model{
		messages:  []message.Model{},
		toolIndex: make(map[string]int),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetSize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	if !m.ready {
		m.viewport = viewport.New(width, height)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height
	}
	m.renderers = styles.NewRenderers(width - 4)
}
