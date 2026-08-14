package tools

import (
	"github.com/openai/openai-go"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) ToOpenAI() []openai.ChatCompletionToolParam {
	out := make([]openai.ChatCompletionToolParam, 0, len(r.tools))

	for _, t := range r.tools {
		out = append(out, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        t.Name(),
				Description: openai.String(t.Description()),
				Parameters:  t.Schema(),
			},
		})
	}

	return out
}
