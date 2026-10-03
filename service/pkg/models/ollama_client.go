package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaClient connects to a local Ollama instance.
type OllamaClient struct {
	host string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(host string) *OllamaClient {
	if host == "" {
		host = "http://localhost:11434"
	}
	// Strip trailing slash
	host = strings.TrimSuffix(host, "/")
	return &OllamaClient{
		host: host,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for model inference
		},
	}
}

func (c *OllamaClient) Name() string {
	return "ollama"
}

func (c *OllamaClient) Type() ProviderType {
	return ProviderOllama
}

// IsAvailable checks if Ollama is reachable.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.host+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ListModels lists models available from Ollama.
func (c *OllamaClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.host+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name      string    `json:"name"`
			Size      int64     `json:"size"`
			ModifiedAt time.Time `json:"modified_at"`
			Details   struct {
				Family       string `json:"family"`
				ParameterSize string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	infos := make([]ModelInfo, 0, len(result.Models))
	for _, m := range result.Models {
		infos = append(infos, ModelInfo{
			Name:         m.Name,
			Size:         m.Size,
			ModifiedAt:   m.ModifiedAt,
			Quant:        m.Details.QuantizationLevel,
			Capabilities: []string{"chat", "tools"},
			Context:      4096,
		})
	}
	return infos, nil
}

// Generate sends a chat completion request to Ollama.
func (c *OllamaClient) Generate(ctx context.Context, opts GenerateOptions) (*Response, error) {
	payload := map[string]any{
		"model":    opts.Model,
		"stream":   false,
		"messages": c.formatMessages(opts.Messages),
		"options": map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
		},
	}

	if len(opts.Stop) > 0 {
		payload["options"] = map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
			"stop":        opts.Stop,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderConnectionError{Name: "ollama", Err: err}
	}
	defer resp.Body.Close()

	var result struct {
		Message struct {
			Role       string `json:"role"`
			Content    string `json:"content"`
			ToolCalls  []struct {
				Function struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		TotalPromptTokens     int `json:"total_prompt_tokens"`
		TotalCompletionTokens int `json:"total_completion_tokens"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	toolCalls := make([]FunctionCall, 0, len(result.Message.ToolCalls))
	for _, tc := range result.Message.ToolCalls {
		toolCalls = append(toolCalls, FunctionCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return &Response{
		Model:        opts.Model,
		Content:      result.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     result.TotalPromptTokens,
			CompletionTokens: result.TotalCompletionTokens,
		},
	}, nil
}

// GenerateStream sends a streaming chat completion request.
func (c *OllamaClient) GenerateStream(ctx context.Context, opts GenerateOptions) (<-chan *StreamChunk, error) {
	payload := map[string]any{
		"model":    opts.Model,
		"stream":   true,
		"messages": c.formatMessages(opts.Messages),
		"options": map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
		},
	}

	if len(opts.Stop) > 0 {
		payload["options"] = map[string]any{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
			"stop":        opts.Stop,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderConnectionError{Name: "ollama", Err: err}
	}
	// Note: don't close resp.Body yet, we need to read from it in the goroutine

	ch := make(chan *StreamChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		for {
			var chunk struct {
				Message struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Function struct {
							Name      string         `json:"name"`
							Arguments map[string]any `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				Done bool `json:"done"`
			}

			if err := dec.Decode(&chunk); err == io.EOF {
				break
			} else if err != nil {
				ch <- &StreamChunk{
					Content: "error: " + err.Error(),
					Done:    true,
				}
				break
			}

			toolCalls := make([]FunctionCall, 0, len(chunk.Message.ToolCalls))
			for _, tc := range chunk.Message.ToolCalls {
				toolCalls = append(toolCalls, FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}

			ch <- &StreamChunk{
				Content:    chunk.Message.Content,
				ToolCalls:  toolCalls,
				Done:       chunk.Done,
			}

			if chunk.Done {
				break
			}
		}
	}()

	return ch, nil
}

// formatMessages converts our Message format to Ollama's expected format.
func (c *OllamaClient) formatMessages(msgs []Message) []map[string]any {
	ollamaMsgs := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		om := map[string]any{"role": m.Role}
		switch m.Content.(type) {
		case string:
			om["content"] = m.Content
		default:
			om["content"] = m.Content
		}
		if m.Name != "" {
			om["name"] = m.Name
		}
		ollamaMsgs = append(ollamaMsgs, om)
	}
	return ollamaMsgs
}

