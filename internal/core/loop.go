package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmaticcs/cawder/internal/core/client"
	"github.com/pragmaticcs/cawder/internal/core/common"
	"github.com/pragmaticcs/cawder/internal/core/config"
	"github.com/pragmaticcs/cawder/internal/core/memory"
	"github.com/pragmaticcs/cawder/internal/core/parser"
	"github.com/pragmaticcs/cawder/internal/core/tools"
)

type AgentLoop struct {
	client *client.OpenAIClient
	parser *parser.Parser
	tools  *tools.Registry
	memory *memory.ContextManager
	model  config.AgentModel
}

func NewAgentLoop(model config.AgentModel, system string, session *memory.Session) *AgentLoop {
	registry := tools.NewRegistry()
	registry.Register(tools.NewListDirTool())
	registry.Register(tools.NewReadFileTool())
	registry.Register(tools.NewWriteFileTool())
	registry.Register(tools.NewEditFileTool())
	registry.Register(tools.NewExecCommandTool())
	registry.Register(tools.NewSearchTool())

	mem := memory.NewContextManager(
		system,
		model.Context,
		session,
		registry,
	)
	return &AgentLoop{
		client: client.NewOpenAIClient(model.Url, model.Key),
		parser: parser.NewParser(parser.DefaultConventions...),
		tools:  registry,
		memory: mem,
		model:  model,
	}
}

func (a *AgentLoop) TokenCount() int {
	return a.memory.GetTokenCount()
}

func (a *AgentLoop) SetModel(model config.AgentModel) {
	a.model = model
	a.client = client.NewOpenAIClient(model.Url, model.Key)
	a.memory.MaxTokens = int64(model.Context)
}

type agentTurn struct {
	content   strings.Builder
	reasoning strings.Builder
	toolCalls []parser.ToolCall
	err       string
}

func (t *agentTurn) absorb(ev parser.Event) parser.Event {
	switch ev.Type {
	case parser.ParserEventText:
		t.content.WriteString(ev.Text)
	case parser.ParserEventReasoning:
		t.reasoning.WriteString(ev.Text)
	case parser.ParserEventToolCall:
		if ev.ToolCall.ID == "" {
			ev.ToolCall.ID = fmt.Sprintf("tool_call_%s_%d", ev.ToolCall.Convention, ev.ToolCall.Index)
		}
		t.toolCalls = append(t.toolCalls, ev.ToolCall)
	case parser.ParserEventToolCallError:
		t.content.WriteString(ev.Text)
		t.err = ev.Err
	}
	return ev
}

func (a *AgentLoop) Run(ctx context.Context, prompt string) (<-chan AgentEvent, error) {
	if err := a.memory.AddUserMessage(prompt); err != nil {
		return nil, err
	}

	out := make(chan AgentEvent)
	go func() {
		defer close(out)

		for {
			toolCalls, err := a.runOnce(ctx, out)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				common.TrySend[AgentEvent](ctx, out, AgentEventError{Error: err})
				return
			}
			if len(toolCalls) == 0 {
				common.TrySend[AgentEvent](ctx, out, AgentEventDone{
					TokenCount: a.memory.GetTokenCount(),
				})
				return
			}
			if !a.executeToolCalls(ctx, toolCalls, out) {
				return
			}
		}
	}()
	return out, nil
}

func (a *AgentLoop) runOnce(ctx context.Context, out chan<- AgentEvent) ([]parser.ToolCall, error) {
	a.parser.Reset()

	messages, err := a.memory.Context()
	if err != nil {
		return nil, err
	}
	responseCh := a.client.Invoke(ctx, a.model.Name, messages, a.tools.ToOpenAI())

	turn := agentTurn{}
	emit := func(ev parser.Event) bool {
		if ev.Type == parser.ParserEventDone {
			return true
		}
		ev = turn.absorb(ev)
		if aev := a.toAgentEvent(ev); aev != nil {
			return common.TrySend(ctx, out, aev)
		}
		return true
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case response, ok := <-responseCh:
			if !ok {
				for _, ev := range a.parser.Finalize() {
					if !emit(ev) {
						return nil, ctx.Err()
					}
				}
				if err := a.memory.AddAgentMessage(turn.content.String(), toMemoryToolCalls(turn.toolCalls)); err != nil {
					return nil, err
				}
				return turn.toolCalls, nil
			}
			if response.Usage != nil {
				a.memory.UpdateUsage(*response.Usage)
			}
			if response.Error != nil {
				return nil, response.Error
			}
			for _, ev := range a.parser.Feed(response.Content) {
				if !emit(ev) {
					return nil, ctx.Err()
				}
			}
		}
	}
}

func (a *AgentLoop) executeToolCalls(ctx context.Context, calls []parser.ToolCall, out chan<- AgentEvent) bool {
	for _, tc := range calls {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		result, err := a.callTool(ctx, tc)
		if err != nil {
			result = tools.ToolCallResult{
				Content: err.Error(),
				Error:   true,
			}
		}

		if !common.TrySend(ctx, out, AgentEvent(AgentEventToolResult{
			Call:       tc,
			Result:     result,
			TokenCount: a.memory.EstimateString(result.Content),
		})) {
			return false
		}
		if err := a.memory.AddToolMessage(tc.ID, result.Content); err != nil {
			common.TrySend(ctx, out, AgentEvent(AgentEventError{Error: err}))
			return false
		}
	}
	return true
}

func (a *AgentLoop) callTool(ctx context.Context, tc parser.ToolCall) (tools.ToolCallResult, error) {
	t, ok := a.tools.Get(tc.Name)
	if !ok {
		return tools.ToolCallResult{}, fmt.Errorf("unknown tool %q", tc.Name)
	}
	return t.Call(ctx, json.RawMessage(tc.Arguments))
}

func toMemoryToolCalls(calls []parser.ToolCall) []memory.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]memory.ToolCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, memory.ToolCall{
			ID:        c.ID,
			Name:      c.Name,
			Arguments: c.Arguments,
		})
	}
	return out
}

func (a *AgentLoop) toAgentEvent(ev parser.Event) AgentEvent {
	switch ev.Type {
	case parser.ParserEventText:
		return AgentEventResponseChunk{
			Chunk:      ev.Text,
			TokenCount: a.memory.EstimateString(ev.Text),
		}
	case parser.ParserEventReasoning:
		return AgentEventReasoningChunk{
			Chunk:      ev.Text,
			TokenCount: a.memory.EstimateString(ev.Text),
		}
	case parser.ParserEventToolCallStart:
		return AgentEventToolCallStart{Call: ev.ToolCall}
	default:
		return nil
	}
}
