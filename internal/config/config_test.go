package config

import (
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Model == "" {
		t.Fatal("expected a default model")
	}
	if cfg.OllamaHost == "" {
		t.Fatal("expected a default ollama host")
	}
	if len(cfg.Providers) == 0 {
		t.Fatal("expected the local ollama provider to be present")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VAJRA_CONFIG_DIR", dir)

	cfg := Load()
	cfg.Model = "test-model"
	cfg.Agent = "coder"
	cfg.AddProvider(Provider{Name: "openai", Kind: KindOpenAI, APIKey: "sk-test"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got := Load()
	if got.Model != "test-model" {
		t.Errorf("model = %q, want test-model", got.Model)
	}
	if got.Agent != "coder" {
		t.Errorf("agent = %q, want coder", got.Agent)
	}
	p, ok := got.GetProvider("openai")
	if !ok {
		t.Fatal("expected openai provider to persist")
	}
	if p.APIKey != "sk-test" {
		t.Errorf("api key = %q", p.APIKey)
	}
	if filepath.Base(Path()) != "config.json" {
		t.Errorf("unexpected config path %q", Path())
	}
}

func TestRemoveProvider(t *testing.T) {
	cfg := Default()
	cfg.AddProvider(Provider{Name: "groq", Kind: KindGroq})
	if len(cfg.EnabledProviders()) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfg.EnabledProviders()))
	}
	if !cfg.RemoveProvider("groq") {
		t.Fatal("expected removal to report true")
	}
	if cfg.RemoveProvider("groq") {
		t.Fatal("expected second removal to report false")
	}
}

func TestModelHelpers(t *testing.T) {
	cfg := Default()
	if cfg.GetEmbedModel() == "" {
		t.Error("expected default embed model")
	}
	cfg.Model = "llama3.1:8b"
	if cfg.GetJudgeModel() != "llama3.1:8b" {
		t.Errorf("judge model = %q, want the chat model", cfg.GetJudgeModel())
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	t.Setenv("VAJRA_MODEL", "env-model")
	t.Setenv("VAJRA_AGENT", "reviewer")
	cfg := Load()
	if cfg.Model != "env-model" {
		t.Errorf("model = %q, want env-model", cfg.Model)
	}
	if cfg.Agent != "reviewer" {
		t.Errorf("agent = %q, want reviewer", cfg.Agent)
	}
}
