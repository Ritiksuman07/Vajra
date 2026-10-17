package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ollama's /api/tags returns an envelope with a "models" array. This test
// guards against the classic bug of parsing it as a bare array.
func TestOllamaModelsParsesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "llama3.1:8b", "model": "llama3.1:8b", "size": 4661234567},
				{"name": "qwen2.5-coder:7b", "model": "qwen2.5-coder:7b", "size": 4400000000},
			},
		})
	}))
	defer srv.Close()

	o := NewOllama(srv.URL)
	models, err := o.Models(context.Background())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Name != "llama3.1:8b" {
		t.Errorf("name = %q", models[0].Name)
	}
	if models[0].Backend != "ollama" {
		t.Errorf("backend = %q", models[0].Backend)
	}
}

func TestOllamaHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if !NewOllama(srv.URL).Healthy(context.Background()) {
		t.Fatal("expected healthy")
	}

	dead := NewOllama("http://127.0.0.1:1")
	if dead.Healthy(context.Background()) {
		t.Fatal("expected unhealthy for a dead server")
	}
}

func TestOllamaChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		chunks := []map[string]any{
			{"message": map[string]string{"role": "assistant", "content": "Hel"}},
			{"message": map[string]string{"role": "assistant", "content": "lo"}},
			{"message": map[string]string{"content": "!"}, "done": true, "eval_count": 3},
		}
		enc := json.NewEncoder(w)
		for _, c := range chunks {
			_ = enc.Encode(c)
		}
	}))
	defer srv.Close()

	var got string
	resp, err := NewOllama(srv.URL).Chat(context.Background(), ChatRequest{
		Model: "test",
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
		},
		Stream: func(tok string) error { got += tok; return nil },
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got != "Hello!" {
		t.Errorf("streamed = %q, want Hello!", got)
	}
	if resp.Content != "Hello!" {
		t.Errorf("content = %q", resp.Content)
	}
	if resp.EvalTokens != 3 {
		t.Errorf("tokens = %d, want 3", resp.EvalTokens)
	}
}

func TestOllamaChatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := NewOllama(srv.URL).Chat(context.Background(), ChatRequest{Model: "x"})
	if err == nil {
		t.Fatal("expected an error for a 404")
	}
}

func TestSplitSystem(t *testing.T) {
	system, turns := splitSystem([]Message{
		{Role: RoleSystem, Content: "be brief"},
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, Content: "hello"},
	})
	if system != "be brief" {
		t.Errorf("system = %q", system)
	}
	if len(turns) != 2 {
		t.Errorf("turns = %d, want 2 (system must be extracted)", len(turns))
	}
}
