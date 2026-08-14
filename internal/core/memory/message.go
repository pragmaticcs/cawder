package memory

import (
	"fmt"
	"unicode/utf8"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

type Role string

const (
	RoleDeveloper Role = "developer"
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (m Message) ToOpenAI() (openai.ChatCompletionMessageParamUnion, error) {
	switch m.Role {
	case RoleDeveloper:
		return openai.DeveloperMessage(m.Content), nil
	case RoleSystem:
		return openai.SystemMessage(m.Content), nil
	case RoleUser:
		return openai.UserMessage(m.Content), nil
	case RoleTool:
		return openai.ToolMessage(m.Content, m.ToolCallID), nil
	case RoleAssistant:
		var toolCalls []openai.ChatCompletionMessageToolCallParam
		if len(m.ToolCalls) > 0 {
			toolCalls = make([]openai.ChatCompletionMessageToolCallParam, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
		}
		return openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(m.Content),
				},
				ToolCalls: toolCalls,
			},
		}, nil
	default:
		return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("unknown role %q", m.Role)
	}
}

func FromOpenAI(u openai.ChatCompletionMessageParamUnion) (Message, error) {
	optString := func(o param.Opt[string]) string {
		return o.Value
	}

	switch {
	case u.OfDeveloper != nil:
		return Message{Role: RoleDeveloper, Content: optString(u.OfDeveloper.Content.OfString)}, nil

	case u.OfSystem != nil:
		return Message{Role: RoleSystem, Content: optString(u.OfSystem.Content.OfString)}, nil

	case u.OfUser != nil:
		return Message{Role: RoleUser, Content: optString(u.OfUser.Content.OfString)}, nil

	case u.OfTool != nil:
		return Message{
			Role:       RoleTool,
			Content:    optString(u.OfTool.Content.OfString),
			ToolCallID: u.OfTool.ToolCallID,
		}, nil

	case u.OfAssistant != nil:
		a := u.OfAssistant
		msg := Message{Role: RoleAssistant, Content: optString(a.Content.OfString)}
		if len(a.ToolCalls) > 0 {
			msg.ToolCalls = make([]ToolCall, 0, len(a.ToolCalls))
			for _, tc := range a.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}
		return msg, nil

	default:
		return Message{}, fmt.Errorf("openai message union has no recognized variant set")
	}
}

func (m Message) CharCount() int {
	// overhead for message framing (~16 chars)
	count := 16 + utf8.RuneCountInString(m.Content) + utf8.RuneCountInString(m.ToolCallID)
	for _, tc := range m.ToolCalls {
		count += utf8.RuneCountInString(tc.ID) + utf8.RuneCountInString(tc.Name) + utf8.RuneCountInString(tc.Arguments)
	}
	return count
}
