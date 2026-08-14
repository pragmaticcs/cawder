package ui

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/core"
	"github.com/pragmaticcs/cawder/internal/core/config"
	"github.com/pragmaticcs/cawder/internal/core/memory"
	"github.com/pragmaticcs/cawder/internal/ui/components/chat"
	"github.com/pragmaticcs/cawder/internal/ui/components/input"
	"github.com/pragmaticcs/cawder/internal/ui/components/message"
	"github.com/pragmaticcs/cawder/internal/ui/components/picker"
	"github.com/pragmaticcs/cawder/internal/ui/keys"
)

type timerTickMsg time.Time

func tickTimer() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return timerTickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.chat.SetSize(m.width-4, m.height-8)

		switch m.mode {
		case TUIModeModelSelect:
			m.modelPicker.SetSize(m.width, m.height)
		case TUIModeSessionSelect:
			m.sessionPicker.SetSize(m.width, m.height)
		}

	case timerTickMsg:
		if m.responding {
			m.elapsed = time.Since(m.startTime)
			cmds = append(cmds, tickTimer())
		}

	case tea.KeyMsg:
		if m.mode == TUIModeChat {
			switch {
			case key.Matches(msg, keys.Keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, keys.Keys.Send):
				if !m.responding && m.input.Value() != "" {
					prompt := m.input.Value()

					var cmd tea.Cmd
					m.chat, cmd = m.chat.Update(chat.ChatEventUserMessage{Content: prompt})
					cmds = append(cmds, cmd)

					m.input, cmd = m.input.Update(input.ClearMsg{})
					cmds = append(cmds, cmd)

					m.responding = true
					m.startTime = time.Now()
					m.elapsed = 0

					if m.agent == nil {
						newSession, err := m.sessions.Create()
						if err != nil {
							var errCmd tea.Cmd
							m, errCmd = m.reportError(fmt.Errorf("create session: %w", err))
							cmds = append(cmds, errCmd)
							break
						}
						m.agent = core.NewAgentLoop(m.selectedModel, m.config.System, newSession)
						m.tokenCount = m.agent.TokenCount()
					}

					cmds = append(cmds, invokeAgent(m.agent, prompt), m.spinner.Tick, tickTimer())
				}
			case key.Matches(msg, keys.Keys.SelectModel):
				m.mode = TUIModeModelSelect
				values := slices.Collect(maps.Values(m.config.Models))
				m.modelPicker = picker.New(
					"Select your model",
					values,
					func(a config.AgentModel) string {
						return a.Name
					},
					func(a config.AgentModel) string {
						return fmt.Sprintf("context: %d", a.Context)
					},
					m.width,
					m.height,
				)
			case key.Matches(msg, keys.Keys.SelectSession):
				m.mode = TUIModeSessionSelect
				sessions, err := m.sessions.List()
				if err != nil {
					var errCmd tea.Cmd
					m, errCmd = m.reportError(fmt.Errorf("list sessions: %w", err))
					cmds = append(cmds, errCmd)
					break
				}
				sessionChoices := []SessionItem{
					{
						Title: "Start a new session...",
						IsNew: true,
					},
				}
				for _, sessionMeta := range sessions {
					sessionChoices = append(sessionChoices, SessionItem{
						Title: sessionMeta.Snippet,
						Meta:  sessionMeta,
						IsNew: false,
					})
				}
				m.sessionPicker = picker.New(
					"Resume session",
					sessionChoices,
					func(s SessionItem) string {
						return s.Title
					},
					func(s SessionItem) string {
						return fmt.Sprintf("Last modified: %s", s.Meta.ModTime.Format("2006-01-02 15:04:05"))
					},
					m.width,
					m.height,
				)
			}
		}

	case core.AgentEventInvoke:
		m.agentCh = msg.Ch
		cmds = append(cmds, listenForAgent(m.agentCh))

	case core.AgentEventResponseChunk:
		var cmd tea.Cmd
		m.tokenCount += msg.TokenCount
		m.chat, cmd = m.chat.Update(chat.ChatEventAgentChunk{
			Type:  message.MessageAgentResponse,
			Chunk: msg.Chunk,
		})
		cmds = append(cmds, tea.Batch(cmd, listenForAgent(m.agentCh)))

	case core.AgentEventReasoningChunk:
		var cmd tea.Cmd
		m.tokenCount += msg.TokenCount
		m.chat, cmd = m.chat.Update(chat.ChatEventAgentChunk{
			Type:  message.MessageAgentReasoning,
			Chunk: msg.Chunk,
		})
		cmds = append(cmds, tea.Batch(cmd, listenForAgent(m.agentCh)))

	case core.AgentEventToolCallStart:
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(chat.ChatEventToolCallStart{
			ID:   msg.Call.ID,
			Name: msg.Call.Name,
		})
		cmds = append(cmds, tea.Batch(cmd, listenForAgent(m.agentCh)))

	case core.AgentEventToolResult:
		var cmd tea.Cmd
		m.tokenCount += msg.TokenCount
		m.chat, cmd = m.chat.Update(chat.ChatEventToolCallResult{
			ID:      msg.Call.ID,
			Content: msg.Result.Content,
			MainArg: msg.Result.MainArg,
			Error:   msg.Result.Error,
		})
		cmds = append(cmds, tea.Batch(cmd, listenForAgent(m.agentCh)))

	case core.AgentEventDone:
		m.responding = false
		m.tokenCount = msg.TokenCount
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(chat.ChatEventAgentDone{})
		cmds = append(cmds, cmd)

	case core.AgentEventError:
		m.responding = false
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(chat.ChatEventAgentError{Error: msg.Error})
		cmds = append(cmds, cmd)

	case picker.SelectedMsg[config.AgentModel]:
		m.mode = TUIModeChat
		m.selectedModel = msg.Value
		if m.agent != nil {
			m.agent.SetModel(msg.Value)
		}

	case picker.SelectedMsg[SessionItem]:
		var session *memory.Session
		var err error
		if msg.Value.IsNew {
			session, err = m.sessions.Create()
		} else {
			session, err = m.sessions.Open(msg.Value.Meta.ID)
		}
		if err != nil {
			m.mode = TUIModeChat
			var errCmd tea.Cmd
			m, errCmd = m.reportError(fmt.Errorf("open session: %w", err))
			cmds = append(cmds, errCmd)
			break
		}

		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(chat.ChatEventLoadSession{
			History: message.FromMessages(session.History()),
		})
		cmds = append(cmds, cmd)

		m.agent = core.NewAgentLoop(m.selectedModel, m.config.System, session)
		m.tokenCount = m.agent.TokenCount()
		m.mode = TUIModeChat

	case picker.CancelledMsg:
		m.mode = TUIModeChat
	}

	var cmd tea.Cmd

	switch m.mode {
	case TUIModeChat:
		m.chat, cmd = m.chat.Update(msg)
		cmds = append(cmds, cmd)

		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case TUIModeModelSelect:
		m.modelPicker, cmd = m.modelPicker.Update(msg)
		cmds = append(cmds, cmd)

	case TUIModeSessionSelect:
		m.sessionPicker, cmd = m.sessionPicker.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.responding {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// helper method for reporting errors in chat
func (m Model) reportError(err error) (Model, tea.Cmd) {
	m.responding = false
	var cmd tea.Cmd
	m.chat, cmd = m.chat.Update(chat.ChatEventAgentError{Error: err})
	return m, cmd
}
