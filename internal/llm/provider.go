package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Cloud is a backend for OpenAI-compatible and Anthropic APIs.
// OpenAI, OpenRouter, Groq and Together all speak the OpenAI wire format.
type Cloud struct {
	kind    string
	name    string
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewCloud builds a cloud backend. baseURL may be empty to use the
// provider default.
func NewCloud(kind, name, apiKey, baseURL string) *Cloud {
	if name == "" {
		name = kind
	}
	return &Cloud{
		kind:    strings.ToLower(kind),
		name:    name,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 0,
		},
	}
}

// Name implements Backend.
func (c *Cloud) Name() string { return c.name }

func (c *Cloud) endpoint() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	switch c.kind {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "together":
		return "https://api.together.xyz/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// Healthy reports whether the provider answers a lightweight request.
func (c *Cloud) Healthy(ctx context.Context) bool {
	if c.apiKey == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	_, err := c.Models(ctx)
	return err == nil
}

// ---------------------------------------------------------------------------
// OpenAI-compatible chat
// ---------------------------------------------------------------------------

func (c *Cloud) Chat(ctx context.Context, req ChatRequest) (*Response, error) {
	if c.kind == "anthropic" {
		return c.anthropicChat(ctx, req)
	}
	return c.openAIChat(ctx, req)
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *Cloud) openAIChat(ctx context.Context, req ChatRequest) (*Response, error) {
	stream := req.Stream != nil
	payload := map[string]any{
		"model":    req.Model,
		"messages": toOpenAIMessages(req.Messages),
		"stream":   stream,
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	raw, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint()+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.kind == "openrouter" {
		httpReq.Header.Set("HTTP-Referer", "https://github.com/Ritiksuman07/Vajra")
		httpReq.Header.Set("X-Title", "Vajra")
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %d: %s", c.name, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	out := &Response{Model: req.Model, Backend: c.name}
	if !stream {
		var body struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, err
		}
		if len(body.Choices) > 0 {
			out.Content = body.Choices[0].Message.Content
		}
		out.PromptTokens = body.Usage.PromptTokens
		out.EvalTokens = body.Usage.CompletionTokens
		return out, nil
	}

	// Server-sent events: "data: {...}\n\n" ... "data: [DONE]"
	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			piece := chunk.Choices[0].Delta.Content
			sb.WriteString(piece)
			if err := req.Stream(piece); err != nil {
				return nil, err
			}
		}
		if chunk.Usage != nil {
			out.PromptTokens = chunk.Usage.PromptTokens
			out.EvalTokens = chunk.Usage.CompletionTokens
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	out.Content = sb.String()
	return out, nil
}

func toOpenAIMessages(msgs []Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, openAIMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

// ---------------------------------------------------------------------------
// Anthropic
// ---------------------------------------------------------------------------

func (c *Cloud) anthropicChat(ctx context.Context, req ChatRequest) (*Response, error) {
	system, turns := splitSystem(req.Messages)
	payload := map[string]any{
		"model":      req.Model,
		"messages":   turns,
		"max_tokens": 8192,
	}
	if system != "" {
		payload["system"] = system
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	raw, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint()+"/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %d: %s", c.name, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var body struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := &Response{Model: req.Model, Backend: c.name}
	var sb strings.Builder
	for _, block := range body.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	out.Content = sb.String()
	out.PromptTokens = body.Usage.InputTokens
	out.EvalTokens = body.Usage.OutputTokens
	if req.Stream != nil {
		if err := req.Stream(out.Content); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func splitSystem(msgs []Message) (string, []Message) {
	var system strings.Builder
	turns := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem {
			if system.Len() > 0 {
				system.WriteString("\n\n")
			}
			system.WriteString(m.Content)
			continue
		}
		turns = append(turns, m)
	}
	return system.String(), turns
}

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

// Models lists models from the provider.
func (c *Cloud) Models(ctx context.Context) ([]Model, error) {
	if c.kind == "anthropic" {
		return []Model{
			{Name: "claude-sonnet-4-20250514", Backend: c.name},
			{Name: "claude-3-5-haiku-20241022", Backend: c.name},
		}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint()+"/models", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: /models returned %d", c.name, resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(body.Data))
	for _, m := range body.Data {
		out = append(out, Model{Name: m.ID, Backend: c.name})
	}
	return out, nil
}
