package llm

import (
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/config"
)

func TestRouterResolve(t *testing.T) {
	cfg := config.Default()
	cfg.AddProvider(config.Provider{Name: "openai", Kind: config.KindOpenAI, APIKey: "sk", Enabled: true})
	cfg.AddProvider(config.Provider{Name: "anthropic", Kind: config.KindAnthropic, APIKey: "sk", Enabled: true})
	cfg.AddProvider(config.Provider{Name: "openrouter", Kind: config.KindOpenRouter, APIKey: "sk", Enabled: true})
	r := NewRouter(cfg)

	cases := map[string]string{
		"llama3.1:8b":                       "ollama",
		"qwen2.5-coder:7b":                  "ollama",
		"gpt-4o":                            "openai",
		"claude-sonnet-4-20250514":          "anthropic",
		"meta-llama/llama-3.1-70b-instruct": "openrouter",
	}
	for model, want := range cases {
		if got := r.Resolve(model).Name(); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestRouterFallsBackToOllama(t *testing.T) {
	cfg := config.Default()
	r := NewRouter(cfg)
	// No cloud providers configured: everything must resolve locally.
	if got := r.Resolve("gpt-4o").Name(); got != "ollama" {
		t.Errorf("expected local fallback for gpt-4o, got %q", got)
	}
}

func TestRouterExplicitModelMapping(t *testing.T) {
	cfg := config.Default()
	cfg.AddProvider(config.Provider{
		Name: "groq", Kind: config.KindGroq, APIKey: "sk", Model: "my-custom-model", Enabled: true,
	})
	r := NewRouter(cfg)
	if got := r.Resolve("my-custom-model").Name(); got != "groq" {
		t.Errorf("explicit mapping failed, got %q", got)
	}
}
