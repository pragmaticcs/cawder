package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/core"
)

func invokeAgent(agent *core.AgentLoop, prompt string) tea.Cmd {
	return func() tea.Msg {
		ch, err := agent.Run(context.Background(), prompt)
		if err != nil {
			return core.AgentEventError{Error: err}
		}
		return core.AgentEventInvoke{
			Ch: ch,
		}
	}
}

func listenForAgent(ch <-chan core.AgentEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return core.AgentEventDone{}
		}
		return event
	}
}
