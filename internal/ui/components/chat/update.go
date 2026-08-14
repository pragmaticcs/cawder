package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pragmaticcs/cawder/internal/ui/components/message"
	"github.com/pragmaticcs/cawder/internal/ui/styles"
)

type ChatEventUserMessage struct {
	Content string
}
type ChatEventAgentChunk struct {
	Type  message.MessageType
	Chunk string
}
type ChatEventAgentDone struct{}
type ChatEventSize struct {
	Width, Height int
}
type ChatEventAgentError struct {
	Error error
}
type ChatEventToolCallStart struct {
	ID   string
	Name string
}
type ChatEventToolCallResult struct {
	ID      string
	Content string
	MainArg string
	Error   bool
}
type ChatEventLoadSession struct {
	History []message.Model
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ChatEventSize:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
		m.renderers = styles.NewRenderers(msg.Width - 4)
		m.sync()
		return m, nil

	case ChatEventUserMessage:
		m.messages = append(m.messages, message.New(message.MessageUserResponse, msg.Content))
		m.sync()
		return m, nil

	case ChatEventAgentError:
		m.messages = append(m.messages, message.New(message.MessageError, msg.Error.Error()))
		m.sync()
		return m, nil

	case ChatEventAgentChunk:
		if m.streaming != nil {
			if m.streaming.Type == message.MessageAgentReasoning && msg.Type == message.MessageAgentResponse {
				m.flushStreaming()
				newMsg := message.New(msg.Type, msg.Chunk)
				m.streaming = &newMsg
			} else {
				m.streaming.StreamToken(msg.Chunk)
			}
		} else {
			newMsg := message.New(msg.Type, msg.Chunk)
			m.streaming = &newMsg
		}
		m.sync()
		return m, nil

	case ChatEventToolCallStart:
		m.flushStreaming()
		tc := message.NewToolCall(msg.ID, msg.Name)
		m.toolIndex[msg.ID] = len(m.messages)
		m.messages = append(m.messages, tc)
		m.sync()
		return m, nil

	case ChatEventToolCallResult:
		if idx, ok := m.toolIndex[msg.ID]; ok && idx < len(m.messages) {
			m.messages[idx].CompleteToolCall(msg.MainArg, msg.Content, msg.Error)
		}
		m.sync()
		return m, nil

	case ChatEventAgentDone:
		m.flushStreaming()
		return m, nil

	case ChatEventLoadSession:
		m.messages = msg.History
		m.sync()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) flushStreaming() {
	if m.streaming == nil {
		return
	}
	m.messages = append(m.messages, *m.streaming)
	m.streaming = nil
}

func (m *Model) sync() {
	if !m.ready {
		return
	}
	var b strings.Builder
	for i, msg := range m.messages {
		if i != 0 {
			b.WriteString("\n")
		}
		b.WriteString(msg.View(m.renderers))
	}
	if m.streaming != nil {
		b.WriteString(m.streaming.View(m.renderers))
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}
