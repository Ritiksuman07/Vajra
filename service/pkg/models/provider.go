// Package models provides local LLM provider integration
// supporting Ollama, LM Studio, and llama.cpp via a unified interface.
package models

import (
	"context"
	"time"
)

// ProviderType indicates which local/remote model backend is used.
type ProviderType string

const (
	ProviderOllama   ProviderType = "ollama"
	ProviderLMStudio ProviderType = "lmstudio"
	ProviderLlamaCPP ProviderType = "llamacpp"
	ProviderOpenAI   ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
)

// Message represents a single chat message in the conversation.
type Message struct {
	Role    string      `json:"role"`    // system, user, assistant, tool
	Content interface{} `json:"content"` // string or []ContentPart for multimodal
	Name    string      `json:"name,omitempty"`
}

// ContentPart represents a part of a multimodal message.
type ContentPart struct {
	Type     string `json:"type"`              // "text" | "image_url" | "input_audio"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// FunctionCall represents a tool call requested by the model.
type FunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Tool represents a function that can be called by the model.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}

// Response is the output of a Generate call.
type Response struct {
	ID           string
	Model        string
	Content      string
	ToolCalls    []FunctionCall
	FinishReason string // stop, length, tool_calls
	Usage        Usage
}

// Usage contains token usage statistics.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// GenerateOptions configures the model generation request.
type GenerateOptions struct {
	Model       string
	Temperature float64
	TopP        float64
	MaxTokens   int
	Stop        []string
	Tools       []Tool
	Messages    []Message
	Stream      bool
}

// Provider is the interface all model backends implement.
type Provider interface {
	// Name returns the provider name (e.g., "ollama").
	Name() string

	// Type returns the provider type.
	Type() ProviderType

	// IsAvailable checks if the backend is reachable.
	IsAvailable(ctx context.Context) bool

	// ListModels lists models available from this backend.
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// Generate sends a chat completion request.
	Generate(ctx context.Context, opts GenerateOptions) (*Response, error)

	// GenerateStream sends a streaming chat completion request.
	GenerateStream(ctx context.Context, opts GenerateOptions) (<-chan *StreamChunk, error)
}

// ModelInfo describes a model available from a provider.
type ModelInfo struct {
	Name         string
	Size         int64
	ModifiedAt   time.Time
	Capabilities []string // e.g., ["chat", "tools", "reasoning", "code"]
	Context      int     // context length in tokens
	Quant        string  // quantization level, e.g. "Q4_K_M"
}

// StreamChunk is a single chunk of a streaming response.
type StreamChunk struct {
	Content      string
	ToolCalls    []FunctionCall
	FinishReason string
	Done         bool
}

// ProviderNotFoundError is returned when a requested provider is not configured.
type ProviderNotFoundError struct {
	Name string
}

func (e *ProviderNotFoundError) Error() string {
	return "provider not found: " + e.Name
}

// ProviderConnectionError is returned when a provider is unreachable.
type ProviderConnectionError struct {
	Name string
	Err  error
}

func (e *ProviderConnectionError) Error() string {
	return "provider connection error (" + e.Name + "): " + e.Err.Error()
}

