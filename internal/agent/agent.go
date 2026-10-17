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

// Budget returns the context budget Vajra applies to every request.
//
// The budget is enforced here rather than in each caller, so the TUI, the web
// server, the CLI and the swarm all trim identically and none of them can
// accidentally overflow a local model's context window.
func (r *Runner) Budget() llm.Budget {
	return llm.DefaultBudget(r.router.Config().MaxContext)
}

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

// build assembles [system, history…, user] and fits it to the context budget.
func (r *Runner) build(roleID string, history []llm.Message, input string) []llm.Message {
	role := resolve(roleID)
	msgs := make([]llm.Message, 0, len(history)+2)
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: role.System})
	msgs = append(msgs, history...)
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: strings.TrimSpace(input)})
	return r.Budget().Trim(msgs).Messages
}

// Run sends input to the role, prepending the role's system prompt and
// the conversation history. When stream is non-nil tokens are delivered
// incrementally.
func (r *Runner) Run(ctx context.Context, roleID string, history []llm.Message, input string, stream llm.StreamFunc) (*llm.Response, error) {
	return r.chat(ctx, roleID, r.router.Config().Model, history, input, stream)
}

// RunModel is like Run but with an explicit model.
func (r *Runner) RunModel(ctx context.Context, roleID, model string, history []llm.Message, input string, stream llm.StreamFunc) (*llm.Response, error) {
	if model == "" {
		model = r.router.Config().Model
	}
	return r.chat(ctx, roleID, model, history, input, stream)
}

func (r *Runner) chat(ctx context.Context, roleID, model string, history []llm.Message, input string, stream llm.StreamFunc) (*llm.Response, error) {
	return r.router.Chat(ctx, llm.ChatRequest{
		Model:    model,
		Messages: r.build(roleID, history, input),
		Stream:   stream,
	})
}
