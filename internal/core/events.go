package core

import (
	"github.com/pragmaticcs/cawder/internal/core/parser"
	"github.com/pragmaticcs/cawder/internal/core/tools"
)

type AgentEvent interface{ isAgentEvent() }

type AgentEventResponseChunk struct {
	Chunk      string
	TokenCount int
}

func (AgentEventResponseChunk) isAgentEvent() {}

type AgentEventReasoningChunk struct {
	Chunk      string
	TokenCount int
}

func (AgentEventReasoningChunk) isAgentEvent() {}

type AgentEventToolCallStart struct{ Call parser.ToolCall }

func (AgentEventToolCallStart) isAgentEvent() {}

type AgentEventToolResult struct {
	Call       parser.ToolCall
	Result     tools.ToolCallResult
	TokenCount int
}

func (AgentEventToolResult) isAgentEvent() {}

type AgentEventStatus struct{ Message string }

func (AgentEventStatus) isAgentEvent() {}

type AgentEventDone struct {
	TokenCount int
}

func (AgentEventDone) isAgentEvent() {}

type AgentEventError struct{ Error error }

func (AgentEventError) isAgentEvent() {}

type AgentEventInvoke struct{ Ch <-chan AgentEvent }

func (AgentEventInvoke) isAgentEvent() {}
