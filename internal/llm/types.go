// Package llm provides local-first language-model backends.
//
// Vajra never phones home. By default every request goes to a local
// Ollama server. Cloud providers are only used when the user adds an
// API key with `vajra connect`.
package llm

import (
	"context"
	"time"
)

// Message roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message is a single conversation turn in a neutral format shared by
// every backend.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Model describes an available model.
type Model struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	Size       int64  `json:"size,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

// StreamFunc receives incremental output. Returning an error aborts the
// stream.
type StreamFunc func(token string) error

// ChatRequest is a backend-agnostic completion request.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	// Stream, when non-nil, receives tokens as they are produced.
	Stream StreamFunc
	// NumCtx optionally overrides the context window (Ollama only).
	NumCtx int
}

// Response is a completed completion.
type Response struct {
	Content      string
	Model        string
	Backend      string
	PromptTokens int
	EvalTokens   int
	Duration     time.Duration
}

// Backend is the interface every provider implements.
type Backend interface {
	// Name is a short human label, e.g. "ollama" or "openai".
	Name() string
	// Chat performs a completion, streaming when req.Stream is set.
	Chat(ctx context.Context, req ChatRequest) (*Response, error)
	// Models lists models this backend can serve.
	Models(ctx context.Context) ([]Model, error)
	// Healthy reports whether the backend is reachable.
	Healthy(ctx context.Context) bool
}
