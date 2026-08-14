package client

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pragmaticcs/cawder/internal/core/common"
)

type OpenAIClient struct {
	api openai.Client
}

type OpenAIResponse struct {
	Content openai.ChatCompletionChunk
	Usage   *openai.CompletionUsage
	Error   error
}

func NewOpenAIClient(baseUrl string, apiKey string) *OpenAIClient {
	api := openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithAPIKey(apiKey),
	)
	return &OpenAIClient{
		api: api,
	}
}

func (c *OpenAIClient) Invoke(ctx context.Context, modelName string, messages []openai.ChatCompletionMessageParamUnion, tools []openai.ChatCompletionToolParam) <-chan OpenAIResponse {
	stream := c.api.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    modelName,
		Messages: messages,
		Tools:    tools,
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	})

	out := make(chan OpenAIResponse)
	go func() {
		defer close(out)
		for stream.Next() {
			chunk := stream.Current()
			var usage *openai.CompletionUsage
			if chunk.Usage.TotalTokens > 0 {
				usage = &chunk.Usage
			}
			common.TrySend(ctx, out, OpenAIResponse{
				Content: chunk,
				Usage:   usage,
			})
		}
		if err := stream.Err(); err != nil {
			common.TrySend(ctx, out, OpenAIResponse{Error: err})
		}
	}()
	return out
}
