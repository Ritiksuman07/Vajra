package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Ritiksuman07/Vajra/internal/config"
)

// Router resolves model names to backends and aggregates them.
//
// Resolution order:
//  1. an exact match against a configured provider's default model
//  2. a name prefix/pattern (claude-*, gpt-*, */* → provider)
//  3. otherwise the local Ollama server
//
// This keeps Vajra local-first: unknown models always go to Ollama.
type Router struct {
	cfg    *config.Config
	ollama *Ollama
	clouds map[string]*Cloud
}

// NewRouter builds a router from configuration.
func NewRouter(cfg *config.Config) *Router {
	r := &Router{
		cfg:    cfg,
		ollama: NewOllama(cfg.OllamaHost),
		clouds: map[string]*Cloud{},
	}
	for _, p := range cfg.EnabledProviders() {
		if c := cloudFor(p); c != nil {
			r.clouds[strings.ToLower(p.Name)] = c
		}
	}
	return r
}

// Config returns the router's configuration.
func (r *Router) Config() *config.Config { return r.cfg }

// Ollama exposes the local Ollama client (used by the verifier for
// embeddings).
func (r *Router) Ollama() *Ollama { return r.ollama }

// cloudFor maps a provider config to a Cloud backend, returning nil for
// purely local providers that the Ollama client already covers.
func cloudFor(p config.Provider) *Cloud {
	base := p.BaseURL
	switch strings.ToLower(p.Kind) {
	case config.KindOllama:
		return nil // handled by the Ollama client
	case config.KindLMStudio:
		if base == "" {
			base = "http://localhost:1234/v1"
		}
		return NewCloud("openai", p.Name, "lm-studio", base)
	case config.KindOpenAI:
		return NewCloud("openai", p.Name, p.APIKey, base)
	case config.KindAnthropic:
		return NewCloud("anthropic", p.Name, p.APIKey, base)
	case config.KindOpenRouter:
		return NewCloud("openrouter", p.Name, p.APIKey, base)
	case config.KindGroq:
		return NewCloud("groq", p.Name, p.APIKey, base)
	case config.KindTogether:
		return NewCloud("together", p.Name, p.APIKey, base)
	case config.KindCustom:
		return NewCloud("openai", p.Name, p.APIKey, base)
	default:
		return NewCloud("openai", p.Name, p.APIKey, base)
	}
}

// Resolve returns the backend that should serve the given model.
func (r *Router) Resolve(model string) Backend {
	lower := strings.ToLower(strings.TrimSpace(model))

	// 1. Explicit provider model mapping.
	for _, p := range r.cfg.EnabledProviders() {
		if p.Model != "" && strings.EqualFold(p.Model, model) {
			if c, ok := r.clouds[strings.ToLower(p.Name)]; ok {
				return c
			}
		}
	}

	// 2. Heuristics by name.
	pick := func(name string) (Backend, bool) {
		c, ok := r.clouds[name]
		return c, ok
	}
	if strings.HasPrefix(lower, "claude") {
		if c, ok := pick("anthropic"); ok {
			return c
		}
	}
	if strings.HasPrefix(lower, "gpt") || strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") ||
		strings.HasPrefix(lower, "text-embedding") {
		if c, ok := pick("openai"); ok {
			return c
		}
	}
	if strings.Contains(lower, "/") {
		for name, c := range r.clouds {
			if strings.Contains(name, "openrouter") {
				return c
			}
		}
	}

	// 3. Local-first default.
	return r.ollama
}

// Chat resolves the model and performs the completion.
func (r *Router) Chat(ctx context.Context, req ChatRequest) (*Response, error) {
	b := r.Resolve(req.Model)
	if req.Temperature == 0 {
		req.Temperature = r.cfg.Temperature
	}
	// Ask local servers to allocate the window we are actually budgeting for;
	// otherwise they may allocate a smaller one and silently truncate.
	if req.NumCtx == 0 {
		req.NumCtx = r.cfg.MaxContext
	}
	return b.Chat(ctx, req)
}

// Models lists every model across healthy backends, deduplicated.
func (r *Router) Models(ctx context.Context) ([]Model, error) {
	seen := map[string]bool{}
	var out []Model
	add := func(ms []Model) {
		for _, m := range ms {
			if m.Name == "" || seen[m.Name] {
				continue
			}
			seen[m.Name] = true
			out = append(out, m)
		}
	}
	local, err := r.ollama.Models(ctx)
	if err == nil {
		add(local)
	}
	for _, c := range r.clouds {
		if ms, err := c.Models(ctx); err == nil {
			add(ms)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// LocalModels lists models available from the local Ollama server.
func (r *Router) LocalModels(ctx context.Context) ([]Model, error) {
	return r.ollama.Models(ctx)
}

// Embed produces an embedding via the local Ollama server.
func (r *Router) Embed(ctx context.Context, text string) ([]float64, error) {
	return r.ollama.Embed(ctx, r.cfg.GetEmbedModel(), text)
}

// Health reports reachability for each backend.
func (r *Router) Health(ctx context.Context) map[string]bool {
	out := map[string]bool{"ollama": r.ollama.Healthy(ctx)}
	for name, c := range r.clouds {
		out[name] = c.Healthy(ctx)
	}
	return out
}

// Backend describes a resolved backend for display.
func (r *Router) BackendName(model string) string {
	return r.Resolve(model).Name()
}

// String renders a short description of the routing table.
func (r *Router) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ollama → %s\n", r.cfg.OllamaHost)
	names := make([]string, 0, len(r.clouds))
	for n := range r.clouds {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(&b, "%s → %s\n", n, r.clouds[n].endpoint())
	}
	return b.String()
}
