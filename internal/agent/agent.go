package agent

import (
	"context"
	"strings"

	"github.com/Ritiksuman07/Vajra/internal/llm"
)

// Runner executes prompts through a role's persona.
type Runner struct {
	router *llm.Router
}

// NewRunner creates a runner bound to a model router.
func NewRunner(r *llm.Router) *Runner { return &Runner{router: r} }

// Router exposes the underlying router.
func (r *Runner) Router() *llm.Router { return r.router }

// Role returns the configured role or the default assistant.
func resolve(roleID string) Role {
	if roleID == "" {
		roleID = "assistant"
	}
	if role, ok := Get(roleID); ok {
		return role
	}
	assistant, _ := Get("assistant")
	return assistant
}

// Run sends input to the role, prepending the role's system prompt and
// the conversation history. When stream is non-nil tokens are delivered
// incrementally.
func (r *Runner) Run(ctx context.Context, roleID string, history []llm.Message, input string, stream llm.StreamFunc) (*llm.Response, error) {
	role := resolve(roleID)
	msgs := make([]llm.Message, 0, len(history)+2)
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: role.System})
	msgs = append(msgs, history...)
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: strings.TrimSpace(input)})

	return r.router.Chat(ctx, llm.ChatRequest{
		Model:    r.router.Config().Model,
		Messages: msgs,
		Stream:   stream,
	})
}

// RunModel is like Run but with an explicit model.
func (r *Runner) RunModel(ctx context.Context, roleID, model string, history []llm.Message, input string, stream llm.StreamFunc) (*llm.Response, error) {
	if model == "" {
		model = r.router.Config().Model
	}
	role := resolve(roleID)
	msgs := make([]llm.Message, 0, len(history)+2)
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: role.System})
	msgs = append(msgs, history...)
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: strings.TrimSpace(input)})

	return r.router.Chat(ctx, llm.ChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   stream,
	})
}
