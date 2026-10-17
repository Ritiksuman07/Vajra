package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Ollama is a client for a local Ollama server (https://ollama.com).
type Ollama struct {
	host string
	http *http.Client
}

// NewOllama creates an Ollama backend. An empty host defaults to
// http://localhost:11434.
func NewOllama(host string) *Ollama {
	if host == "" {
		host = "http://localhost:11434"
	}
	host = strings.TrimRight(host, "/")
	return &Ollama{
		host: host,
		http: &http.Client{Timeout: 0}, // streams can be long-lived
	}
}

// Host returns the configured base URL.
func (o *Ollama) Host() string { return o.host }

// Name implements Backend.
func (o *Ollama) Name() string { return "ollama" }

// ---------------------------------------------------------------------------
// Wire types
// ---------------------------------------------------------------------------

type ollamaTags struct {
	Models []struct {
		Name       string `json:"name"`
		Model      string `json:"model"`
		Size       int64  `json:"size"`
		ModifiedAt string `json:"modified_at"`
	} `json:"models"`
}

type ollamaChatChunk struct {
	Model    string   `json:"model"`
	Message  *Message `json:"message"`
	Response string   `json:"response"`
	Done     bool     `json:"done"`
	// Stats are only present on the final chunk.
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	EvalDuration    int64  `json:"eval_duration"` // nanoseconds
	Error           string `json:"error"`
}

type ollamaChatBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  *struct {
		Temperature float64 `json:"temperature,omitempty"`
		NumCtx      int     `json:"num_ctx,omitempty"`
	} `json:"options,omitempty"`
}

// ---------------------------------------------------------------------------
// Health / models
// ---------------------------------------------------------------------------

// Healthy implements Backend.
func (o *Ollama) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.host+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

// Models implements Backend. It correctly parses Ollama's
// {"models":[{...}]} envelope.
func (o *Ollama) Models(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.host+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: /api/tags returned %d", resp.StatusCode)
	}
	var tags ollamaTags
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(tags.Models))
	for _, m := range tags.Models {
		name := m.Name
		if name == "" {
			name = m.Model
		}
		out = append(out, Model{
			Name:       name,
			Backend:    "ollama",
			Size:       m.Size,
			ModifiedAt: m.ModifiedAt,
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Chat
// ---------------------------------------------------------------------------

// Chat implements Backend with true token streaming.
func (o *Ollama) Chat(ctx context.Context, req ChatRequest) (*Response, error) {
	body := ollamaChatBody{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}
	opts := struct {
		Temperature float64 `json:"temperature,omitempty"`
		NumCtx      int     `json:"num_ctx,omitempty"`
	}{Temperature: req.Temperature, NumCtx: req.NumCtx}
	if opts.Temperature != 0 || opts.NumCtx != 0 {
		body.Options = &opts
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.host+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")

	resp, err := o.http.Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama: %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	out := &Response{Model: req.Model, Backend: "ollama"}
	start := time.Now()
	var sb strings.Builder

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var chunk ollamaChatChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		if chunk.Error != "" {
			return nil, fmt.Errorf("ollama: %s", chunk.Error)
		}
		piece := ""
		if chunk.Message != nil {
			piece = chunk.Message.Content
		}
		if piece == "" {
			piece = chunk.Response
		}
		if piece != "" {
			sb.WriteString(piece)
			if req.Stream != nil {
				if err := req.Stream(piece); err != nil {
					return nil, err
				}
			}
		}
		if chunk.PromptEvalCount > 0 {
			out.PromptTokens = chunk.PromptEvalCount
		}
		if chunk.EvalCount > 0 {
			out.EvalTokens = chunk.EvalCount
		}
		if chunk.EvalDuration > 0 {
			out.Duration = time.Duration(chunk.EvalDuration)
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if out.Duration == 0 {
		out.Duration = time.Since(start)
	}
	out.Content = sb.String()
	return out, nil
}

// ---------------------------------------------------------------------------
// Embeddings (used by the verification pipeline)
// ---------------------------------------------------------------------------

// Embed returns a vector embedding for text using the given model
// (e.g. "nomic-embed-text" or "all-minilm"). The /api/embed endpoint is
// the modern one; we fall back to /api/embeddings for older servers.
func (o *Ollama) Embed(ctx context.Context, model, text string) ([]float64, error) {
	if model == "" {
		model = "nomic-embed-text"
	}
	// Modern endpoint.
	if vec, err := o.embedModern(ctx, model, text); err == nil {
		return vec, nil
	}
	return o.embedLegacy(ctx, model, text)
}

func (o *Ollama) embedModern(ctx context.Context, model, text string) ([]float64, error) {
	body := map[string]any{"model": model, "input": text}
	raw, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.host+"/api/embed", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed: %d", resp.StatusCode)
	}
	var out struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) == 0 || len(out.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("embed: empty")
	}
	return out.Embeddings[0], nil
}

func (o *Ollama) embedLegacy(ctx context.Context, model, text string) ([]float64, error) {
	body := map[string]any{"model": model, "prompt": text}
	raw, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.host+"/api/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embeddings: %d", resp.StatusCode)
	}
	var out struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("embeddings: empty")
	}
	return out.Embedding, nil
}

// ---------------------------------------------------------------------------
// Pull
// ---------------------------------------------------------------------------

// Pull downloads a model, reporting progress (0..100) through progress.
func (o *Ollama) Pull(ctx context.Context, name string, progress func(pct float64, status string)) error {
	raw, _ := json.Marshal(map[string]any{"name": name, "stream": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.host+"/api/pull", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull: %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var chunk struct {
			Status    string `json:"status"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
			Error     string `json:"error"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		if chunk.Error != "" {
			return fmt.Errorf("pull: %s", chunk.Error)
		}
		pct := 0.0
		if chunk.Total > 0 {
			pct = float64(chunk.Completed) / float64(chunk.Total) * 100
		}
		if progress != nil {
			progress(pct, chunk.Status)
		}
		if chunk.Status == "success" {
			return nil
		}
	}
	return scanner.Err()
}

// Version returns the Ollama server version string.
func (o *Ollama) Version(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.host+"/api/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Version, nil
}
