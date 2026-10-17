package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/config"
)

func newTestServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	cfg := config.Load()
	Version = "test"
	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func post(t *testing.T, url, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestHealth(t *testing.T) {
	ts, _ := newTestServer(t)
	code, body := get(t, ts.URL+"/api/health")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body = %s", body)
	}
}

func TestAgentsEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)
	_, body := get(t, ts.URL+"/api/agents")
	if !strings.Contains(body, "assistant") || !strings.Contains(body, "coder") {
		t.Errorf("expected built-in agents, got %s", body)
	}
}

func TestWebUIServed(t *testing.T) {
	ts, _ := newTestServer(t)
	_, body := get(t, ts.URL+"/")
	if !strings.Contains(body, "Vajra") {
		t.Errorf("web UI missing, got %d bytes", len(body))
	}
	_, css := get(t, ts.URL+"/style.css")
	if len(css) < 100 {
		t.Error("stylesheet not served")
	}
	_, js := get(t, ts.URL+"/app.js")
	if len(js) < 100 {
		t.Error("javascript not served")
	}
}

func TestSessionLifecycle(t *testing.T) {
	ts, _ := newTestServer(t)
	code, body := post(t, ts.URL+"/api/sessions", `{"model":"m","agent":"coder"}`)
	if code != 201 {
		t.Fatalf("create status = %d (%s)", code, body)
	}
	var sess map[string]any
	if err := json.Unmarshal([]byte(body), &sess); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	id, _ := sess["id"].(string)
	if id == "" {
		t.Fatal("expected an id")
	}

	code, body = get(t, ts.URL+"/api/sessions/"+id)
	if code != 200 {
		t.Fatalf("get status = %d", code)
	}
	if !strings.Contains(body, `"agent":"coder"`) {
		t.Errorf("unexpected session body %s", body)
	}

	code, body = get(t, ts.URL+"/api/sessions")
	if code != 200 || !strings.Contains(body, id) {
		t.Errorf("session not listed (%d): %s", code, body)
	}
}

func TestConfigSet(t *testing.T) {
	ts, srv := newTestServer(t)
	code, _ := post(t, ts.URL+"/api/config", `{"model":"new-model","verify":true}`)
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if srv.Config().Model != "new-model" {
		t.Errorf("model = %q", srv.Config().Model)
	}
	if !srv.Config().Verify {
		t.Error("verify should be enabled")
	}
}

func TestProvidersEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)
	_, body := get(t, ts.URL+"/api/providers")
	if !strings.Contains(body, "ollama") {
		t.Errorf("expected the default ollama provider, got %s", body)
	}

	code, _ := post(t, ts.URL+"/api/providers", `{"kind":"openai","name":"openai","api_key":"sk-x"}`)
	if code != 200 {
		t.Fatalf("add provider status = %d", code)
	}
	_, body = get(t, ts.URL+"/api/providers")
	if !strings.Contains(body, `"has_key":true`) {
		t.Errorf("expected a key to be recorded, got %s", body)
	}
	// The key itself must never be echoed back.
	if strings.Contains(body, "sk-x") {
		t.Error("API key leaked to the client")
	}
}

func TestCORS(t *testing.T) {
	ts, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/api/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}
