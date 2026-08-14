package message

import (
	"strings"

	"github.com/pragmaticcs/cawder/internal/core/memory"
)

type MessageType int

const (
	MessageUserResponse MessageType = iota
	MessageAgentResponse
	MessageAgentReasoning
	MessageToolCall
	MessageError
)

type ToolStatus int

const (
	ToolStatusRunning ToolStatus = iota
	ToolStatusDone
	ToolStatusError
)

type Model struct {
	Type    MessageType
	Content string   // Type == MessageUserResponse or MessageAgentResponse or MessageAgentReasoning
	Tool    ToolInfo // Type == MessageToolCall
}

func New(msgType MessageType, content string) Model {
	return Model{
		Type:    msgType,
		Content: content,
	}
}

func (m *Model) StreamToken(token string) {
	m.Content += token
}

func FromMessages(msgs []memory.Message) []Model {
	models := make([]Model, 0, len(msgs))
	pending := make(map[string]int)

	for _, m := range msgs {
		switch m.Role {
		case memory.RoleUser:
			models = append(models, New(MessageUserResponse, m.Content))

		case memory.RoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				models = append(models, New(MessageAgentResponse, m.Content))
			}
			for _, tc := range m.ToolCalls {
				pending[tc.ID] = len(models)
				models = append(models, NewToolCall(tc.ID, tc.Name))
			}

		case memory.RoleTool:
			if idx, ok := pending[m.ToolCallID]; ok {
				models[idx].CompleteToolCall("", m.Content, false)
				delete(pending, m.ToolCallID)
			} else {
				tm := NewToolCall(m.ToolCallID, "")
				tm.CompleteToolCall("", m.Content, false)
				models = append(models, tm)
			}

		case memory.RoleSystem, memory.RoleDeveloper:
			// Skip

		default:
			models = append(models, New(MessageError, "unknown role: "+string(m.Role)))
		}
	}

	return models
}

func FromMessage(m memory.Message) []Model {
	switch m.Role {
	case memory.RoleUser:
		return []Model{New(MessageUserResponse, m.Content)}

	case memory.RoleAssistant:
		var models []Model
		if strings.TrimSpace(m.Content) != "" {
			models = append(models, New(MessageAgentResponse, m.Content))
		}
		for _, tc := range m.ToolCalls {
			models = append(models, NewToolCall(tc.ID, tc.Name))
		}
		return models

	case memory.RoleSystem, memory.RoleDeveloper, memory.RoleTool:
		return nil

	default:
		return []Model{New(MessageError, "unknown role: "+string(m.Role))}
	}
}
