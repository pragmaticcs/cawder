package memory

import (
	"fmt"
	"slices"

	"github.com/openai/openai-go"
	"github.com/pragmaticcs/cawder/internal/core/tools"
)

const ReservedBuffer int64 = 8096

type Turn struct {
	Messages []Message
}

func NewTurn(messages ...Message) Turn {
	turn := Turn{}
	for _, msg := range messages {
		turn.Append(msg)
	}
	return turn
}

func (t *Turn) Append(msg Message) {
	t.Messages = append(t.Messages, msg)
}

type ContextManager struct {
	History []Turn
	System  Message
	Session *Session
	tools   *tools.Registry

	// Tokens
	MaxTokens           int64 // Context window
	lastPromptCharCount int64 // Char count of the previous context given to the model
	reservedTokenCount  int64
	estimator           *TokenEstimator
}

func NewContextManager(system string, maxTokens int, session *Session, tools *tools.Registry) *ContextManager {
	history := loadTurns(session.history)
	return &ContextManager{
		History:            history,
		System:             Message{Role: RoleSystem, Content: system},
		Session:            session,
		MaxTokens:          int64(maxTokens),
		estimator:          NewTokenEstimator(),
		reservedTokenCount: ReservedBuffer,
	}
}

func loadTurns(messages []Message) []Turn {
	var turns []Turn
	var currTurn Turn
	started := false
	for _, msg := range messages {
		if msg.Role == RoleUser {
			if started {
				turns = append(turns, currTurn)
			}
			currTurn, started = NewTurn(), true
		}
		currTurn.Append(msg)
	}
	if started {
		turns = append(turns, currTurn)
	}
	return turns
}

func (m *ContextManager) add(msg Message) error {
	if msg.Role == RoleUser {
		m.History = append(m.History, NewTurn(msg))
	} else {
		m.History[len(m.History)-1].Append(msg)
	}
	if m.Session != nil {
		if err := m.Session.Append(msg); err != nil {
			return fmt.Errorf("persist message: %w", err)
		}
	}
	return nil
}

func (m *ContextManager) AddUserMessage(content string) error {
	return m.add(Message{
		Role:    RoleUser,
		Content: content,
	})
}

func (m *ContextManager) AddAgentMessage(content string, toolCalls []ToolCall) error {
	return m.add(Message{
		Role:      RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	})
}

func (m *ContextManager) AddToolMessage(toolCallID, content string) error {
	return m.add(Message{
		Role:       RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
	})
}

func (m *ContextManager) UpdateUsage(usage openai.CompletionUsage) {
	if usage.PromptTokens > 0 && m.lastPromptCharCount > 0 {
		m.estimator.Update(int(m.lastPromptCharCount), int(usage.PromptTokens))
	}
}

func (m *ContextManager) Context() ([]openai.ChatCompletionMessageParamUnion, error) {
	ctx, err := m.assembleContext()
	if err != nil {
		return nil, err
	}

	charCount := 0
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(ctx))
	for _, msg := range ctx {
		charCount += msg.CharCount()
		p, err := msg.ToOpenAI()
		if err != nil {
			return nil, fmt.Errorf("convert message (role=%s): %w", msg.Role, err)
		}
		out = append(out, p)
	}

	m.lastPromptCharCount = int64(charCount)
	return out, nil
}

func (m *ContextManager) assembleContext() ([]Message, error) {
	// gather ids of messages
	var ids []int
	used := m.estimator.EstimateMessage(m.System)
	msgCount := 1 // account for system message
	budget := m.MaxTokens - m.reservedTokenCount
	for i, turn := range slices.Backward(m.History) {
		tokenCount := m.estimator.EstimateTurn(turn)
		if used+tokenCount > int(budget) {
			break
		}
		ids = append(ids, i)
		msgCount += len(turn.Messages)
		used += tokenCount
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("context too small for prompt")
	}

	ctx := make([]Message, 0, msgCount)
	ctx = append(ctx, m.System)
	for _, id := range slices.Backward(ids) {
		turn := m.History[id]
		for _, msg := range turn.Messages {
			ctx = append(ctx, msg)
		}
	}
	return ctx, nil
}

func (m *ContextManager) GetTokenCount() int {
	total := 0
	for _, turn := range m.History {
		total += m.estimator.EstimateTurn(turn)
	}
	total += m.estimator.EstimateMessage(m.System)
	// TODO: count tool schema tokens and add to total
	return total
}

func (m *ContextManager) EstimateTokens(chars int) int {
	return m.estimator.Estimate(chars)
}

func (m *ContextManager) EstimateString(s string) int {
	return m.estimator.EstimateString(s)
}

func (m *ContextManager) LastPromptTokenEstimate() int {
	return m.estimator.Estimate(int(m.lastPromptCharCount))
}
