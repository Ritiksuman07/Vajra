// Package config manages Vajra's on-disk configuration.
//
// The config lives at ~/.vajra/config.json by default. Everything is
// local-first: the file never leaves the machine unless the user
// explicitly adds a cloud provider via `vajra connect`.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Known provider kinds. "ollama" and "lmstudio" are local;
// the rest require an API key.
const (
	KindOllama     = "ollama"
	KindLMStudio   = "lmstudio"
	KindOpenAI     = "openai"
	KindAnthropic  = "anthropic"
	KindOpenRouter = "openrouter"
	KindGroq       = "groq"
	KindTogether   = "together"
	KindCustom     = "custom"
)

// Provider is a configured LLM backend.
type Provider struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Config is the root configuration document.
type Config struct {
	// Model is the default model used when none is specified.
	Model string `json:"model"`
	// Agent is the default agent role (see internal/agent).
	Agent string `json:"agent"`
	// Theme is "dark" or "light".
	Theme string `json:"theme"`
	// Verify toggles the 4-layer anti-hallucination pipeline.
	Verify bool `json:"verify"`
	// Stream toggles token streaming for local models.
	Stream bool `json:"stream"`
	// Temperature controls sampling (0..1).
	Temperature float64 `json:"temperature"`
	// MaxContext is the context window Vajra budgets for, in tokens. Older
	// turns are trimmed to fit (default 8192).
	MaxContext int `json:"max_context"`
	// SystemPrompt is prepended to every conversation when no agent is set.
	SystemPrompt string `json:"system_prompt,omitempty"`
	// OllamaHost is the base URL of the local Ollama server.
	OllamaHost string `json:"ollama_host"`
	// EmbedModel is the local embedding model used for semantic checks.
	EmbedModel string `json:"embed_model,omitempty"`
	// JudgeModel is the model used for LLM-as-a-judge verification.
	JudgeModel string `json:"judge_model,omitempty"`
	// Port is the default HTTP server / web UI port.
	Port int `json:"port"`
	// Providers holds configured backends.
	Providers []Provider `json:"providers"`

	// path remembers where this config was loaded from (not serialised).
	path string `json:"-"`
}

// Default returns a fresh configuration with sensible local-first defaults.
func Default() *Config {
	return &Config{
		Model:       "llama3.1:8b",
		Agent:       "assistant",
		Theme:       "dark",
		Verify:      false,
		Stream:      true,
		Temperature: 0.7,
		MaxContext:  8192,
		OllamaHost:  "http://localhost:11434",
		Port:        4096,
		Providers: []Provider{
			{Name: "ollama", Kind: KindOllama, BaseURL: "http://localhost:11434", Enabled: true},
		},
	}
}

// Dir returns the configuration directory (~/.vajra), creating nothing.
func Dir() string {
	if d := os.Getenv("VAJRA_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".vajra"
	}
	return filepath.Join(home, ".vajra")
}

// Path returns the full path to config.json.
func Path() string { return filepath.Join(Dir(), "config.json") }

// SessionsDir returns the directory used to persist chat sessions.
func SessionsDir() string { return filepath.Join(Dir(), "sessions") }

// IndexDir returns the directory used to persist RAG indexes.
func IndexDir() string { return filepath.Join(Dir(), "index") }

// ExportsDir returns the directory session exports are written to.
func ExportsDir() string { return filepath.Join(Dir(), "exports") }

// Load reads the config from disk, falling back to defaults. Environment
// variables override the persisted values so containers and CI can tweak
// behaviour without rewriting the file.
func Load() *Config {
	cfg := Default()
	cfg.path = Path()

	if data, err := os.ReadFile(cfg.path); err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	cfg.applyEnv()
	cfg.normalise()
	return cfg
}

// Save writes the config to disk atomically.
func (c *Config) Save() error {
	if c.path == "" {
		c.path = Path()
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

func (c *Config) applyEnv() {
	if v := os.Getenv("VAJRA_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("VAJRA_AGENT"); v != "" {
		c.Agent = v
	}
	if v := os.Getenv("VAJRA_PORT"); v != "" {
		if n, err := atoi(v); err == nil && n > 0 {
			c.Port = n
		}
	}
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		c.OllamaHost = ensureScheme(v)
	}
	if v := os.Getenv("VAJRA_OLLAMA_HOST"); v != "" {
		c.OllamaHost = ensureScheme(v)
	}
	if v := os.Getenv("VAJRA_VERIFY"); v == "1" || strings.EqualFold(v, "true") {
		c.Verify = true
	}
	if v := os.Getenv("VAJRA_MAX_CONTEXT"); v != "" {
		if n, err := atoi(v); err == nil {
			c.MaxContext = n
		}
	}
}

func (c *Config) normalise() {
	if c.Model == "" {
		c.Model = "llama3.1:8b"
	}
	if c.Agent == "" {
		c.Agent = "assistant"
	}
	if c.Theme == "" {
		c.Theme = "dark"
	}
	if c.OllamaHost == "" {
		c.OllamaHost = "http://localhost:11434"
	}
	if c.Port == 0 {
		c.Port = 4096
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxContext <= 0 {
		c.MaxContext = 8192
	}
	if len(c.Providers) == 0 {
		c.Providers = []Provider{
			{Name: "ollama", Kind: KindOllama, BaseURL: c.OllamaHost, Enabled: true},
		}
	}
}

// GetEmbedModel returns the embedding model, defaulting to a small,
// widely-available Ollama model.
func (c *Config) GetEmbedModel() string {
	if c.EmbedModel != "" {
		return c.EmbedModel
	}
	return "nomic-embed-text"
}

// GetJudgeModel returns the verification judge model, defaulting to the
// active chat model.
func (c *Config) GetJudgeModel() string {
	if c.JudgeModel != "" {
		return c.JudgeModel
	}
	return c.Model
}

// GetProvider returns the named provider if enabled.
func (c *Config) GetProvider(name string) (Provider, bool) {
	for _, p := range c.Providers {
		if strings.EqualFold(p.Name, name) {
			return p, p.Enabled
		}
	}
	return Provider{}, false
}

// AddProvider inserts or updates a provider by name.
func (c *Config) AddProvider(p Provider) {
	if p.Kind == "" {
		p.Kind = KindCustom
	}
	p.Enabled = true
	for i := range c.Providers {
		if strings.EqualFold(c.Providers[i].Name, p.Name) {
			c.Providers[i] = p
			return
		}
	}
	c.Providers = append(c.Providers, p)
}

// RemoveProvider deletes a provider by name.
func (c *Config) RemoveProvider(name string) bool {
	out := c.Providers[:0]
	removed := false
	for _, p := range c.Providers {
		if strings.EqualFold(p.Name, name) {
			removed = true
			continue
		}
		out = append(out, p)
	}
	c.Providers = out
	return removed
}

// EnabledProviders returns providers that are enabled.
func (c *Config) EnabledProviders() []Provider {
	var out []Provider
	for _, p := range c.Providers {
		if p.Enabled {
			out = append(out, p)
		}
	}
	return out
}

func ensureScheme(v string) string {
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		return "http://" + v
	}
	return v
}

func atoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
