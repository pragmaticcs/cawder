package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	// Custom HTTP client that cleans SSE keep-alive comments from local servers
	httpClient := &http.Client{
		Transport: &sseFilterTransport{
			base: http.DefaultTransport,
		},
	}

	api := openai.NewClient(
		option.WithBaseURL(baseUrl),
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(httpClient),
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
			common.TrySend(ctx, out, OpenAIResponse{
				Error: fmt.Errorf("openai stream decode error: %w", err),
			})
		}
	}()
	return out
}

type sseFilterTransport struct {
	base http.RoundTripper
}

func (t *sseFilterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		resp.Body = newSSEFilterReader(resp.Body)
	}
	return resp, nil
}

type sseFilterReader struct {
	pipeR *io.PipeReader
	rc    io.ReadCloser
}

func (r *sseFilterReader) Read(p []byte) (int, error) {
	return r.pipeR.Read(p)
}

func (r *sseFilterReader) Close() error {
	_ = r.pipeR.Close()
	return r.rc.Close()
}

func newSSEFilterReader(rc io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer rc.Close()

		r := bufio.NewReaderSize(rc, 32*1024)
		var eventBuffer bytes.Buffer
		hasData := false

		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				trimmed := bytes.TrimSpace(line)
				if len(trimmed) > 0 && trimmed[0] == ':' {
				} else if len(trimmed) == 0 {
					if hasData {
						eventBuffer.WriteByte('\n')
						if _, wErr := pw.Write(eventBuffer.Bytes()); wErr != nil {
							return
						}
						eventBuffer.Reset()
						hasData = false
					}
				} else {
					if bytes.HasPrefix(trimmed, []byte("data:")) {
						payload := bytes.TrimSpace(trimmed[5:])
						if len(payload) == 0 {
							continue
						}
					}

					eventBuffer.Write(trimmed)
					eventBuffer.WriteByte('\n')
					hasData = true
				}
			}

			if err != nil {
				if err != io.EOF {
					pw.CloseWithError(err)
				} else {
					// Flush any trailing event buffer if connection closed without a final newline
					if eventBuffer.Len() > 0 {
						eventBuffer.WriteByte('\n')
						_, _ = pw.Write(eventBuffer.Bytes())
					}
					pw.Close()
				}
				return
			}
		}
	}()

	return &sseFilterReader{
		pipeR: pr,
		rc:    rc,
	}
}
