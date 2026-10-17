// Package server exposes Vajra over HTTP: a small REST API plus the
// embedded web UI. It is optional — the TUI and CLI work without it.
package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Ritiksuman07/Vajra/internal/agent"
	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/llm"
	"github.com/Ritiksuman07/Vajra/internal/session"
	"github.com/Ritiksuman07/Vajra/internal/verify"
)

//go:embed web
var webFS embed.FS

// Server bundles the dependencies shared by every handler.
type Server struct {
	cfg    *config.Config
	router *llm.Router
	runner *agent.Runner
	store  *session.Store
}

// New constructs a server.
func New(cfg *config.Config) *Server {
	router := llm.NewRouter(cfg)
	return &Server{
		cfg:    cfg,
		router: router,
		runner: agent.NewRunner(router),
		store:  session.NewStore(config.SessionsDir()),
	}
}

// Router exposes the model router (used by the TUI and CLI).
func (s *Server) Router() *llm.Router { return s.router }

// Config exposes the configuration.
func (s *Server) Config() *config.Config { return s.cfg }

// Handler returns the fully-wired HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	mux.HandleFunc("GET /api/models", s.handleModels)
	mux.HandleFunc("GET /api/agents", s.handleAgents)

	mux.HandleFunc("GET /api/config", s.handleConfigGet)
	mux.HandleFunc("POST /api/config", s.handleConfigSet)

	mux.HandleFunc("GET /api/providers", s.handleProvidersList)
	mux.HandleFunc("POST /api/providers", s.handleProviderAdd)
	mux.HandleFunc("DELETE /api/providers/{name}", s.handleProviderDelete)

	mux.HandleFunc("GET /api/tools", s.handleTools)
	mux.HandleFunc("GET /api/status", s.handleStatus)

	mux.HandleFunc("GET /api/sessions", s.handleSessionsList)
	mux.HandleFunc("POST /api/sessions", s.handleSessionCreate)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleSessionGet)
	mux.HandleFunc("DELETE /api/sessions/{id}", s.handleSessionDelete)

	mux.HandleFunc("POST /api/chat", s.handleChat)
	mux.HandleFunc("POST /api/chat/stream", s.handleChatStream)
	mux.HandleFunc("POST /api/team", s.handleSwarm)
	mux.HandleFunc("POST /api/swarm", s.handleSwarm)
	mux.HandleFunc("POST /api/verify", s.handleVerify)

	// Embedded web UI.
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(sub)))

	return withCORS(mux)
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

// ---------------------------------------------------------------------------
// Simple handlers
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"name":    "vajra",
		"version": Version,
	})
}

// Version is injected by main at startup.
var Version = "dev"

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	models, err := s.router.Models(ctx)
	if err != nil && len(models) == 0 {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"agents": agent.All()})
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": []any{}})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{
		"config":  s.cfg,
		"health":  s.router.Health(ctx),
		"version": Version,
	})
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.cfg)
}

func (s *Server) handleConfigSet(w http.ResponseWriter, r *http.Request) {
	var patch struct {
		Model       *string  `json:"model"`
		Agent       *string  `json:"agent"`
		Theme       *string  `json:"theme"`
		Verify      *bool    `json:"verify"`
		Stream      *bool    `json:"stream"`
		Temperature *float64 `json:"temperature"`
	}
	if err := decode(r, &patch); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if patch.Model != nil {
		s.cfg.Model = *patch.Model
	}
	if patch.Agent != nil {
		s.cfg.Agent = *patch.Agent
	}
	if patch.Theme != nil {
		s.cfg.Theme = *patch.Theme
	}
	if patch.Verify != nil {
		s.cfg.Verify = *patch.Verify
	}
	if patch.Stream != nil {
		s.cfg.Stream = *patch.Stream
	}
	if patch.Temperature != nil {
		s.cfg.Temperature = *patch.Temperature
	}
	_ = s.cfg.Save()
	writeJSON(w, http.StatusOK, s.cfg)
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

type providerView struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	BaseURL string `json:"base_url,omitempty"`
	Model   string `json:"model,omitempty"`
	Enabled bool   `json:"enabled"`
	HasKey  bool   `json:"has_key"`
}

func (s *Server) handleProvidersList(w http.ResponseWriter, r *http.Request) {
	out := make([]providerView, 0, len(s.cfg.Providers))
	for _, p := range s.cfg.Providers {
		out = append(out, providerView{
			Name: p.Name, Kind: p.Kind, BaseURL: p.BaseURL,
			Model: p.Model, Enabled: p.Enabled, HasKey: p.APIKey != "",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}

func (s *Server) handleProviderAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Kind    string `json:"kind"`
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Kind == "" {
		writeErr(w, http.StatusBadRequest, "kind is required")
		return
	}
	if req.Name == "" {
		req.Name = req.Kind
	}
	s.cfg.AddProvider(config.Provider{
		Name: req.Name, Kind: req.Kind, APIKey: req.APIKey,
		BaseURL: req.BaseURL, Model: req.Model, Enabled: true,
	})
	_ = s.cfg.Save()
	s.reload()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleProviderDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.cfg.RemoveProvider(name)
	_ = s.cfg.Save()
	s.reload()
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// reload rebuilds the router after configuration changes.
func (s *Server) reload() {
	s.router = llm.NewRouter(s.cfg)
	s.runner = agent.NewRunner(s.router)
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func (s *Server) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	metas, err := s.store.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": metas})
}

func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model string `json:"model"`
		Agent string `json:"agent"`
	}
	_ = decode(r, &req)
	if req.Model == "" {
		req.Model = s.cfg.Model
	}
	if req.Agent == "" {
		req.Agent = s.cfg.Agent
	}
	sess := session.New(req.Model, req.Agent)
	_ = s.store.Save(sess)
	writeJSON(w, http.StatusCreated, sess)
}

func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.Load(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.Delete(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------------------------------------------------------------------------
// Chat
// ---------------------------------------------------------------------------

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Model     string `json:"model"`
	Agent     string `json:"agent"`
	Verify    bool   `json:"verify"`

	// Swarm tuning (optional; sensible defaults apply).
	MinAgents  int   `json:"min_agents,omitempty"`
	MaxAgents  int   `json:"max_agents,omitempty"`
	Handoffs   int   `json:"handoffs,omitempty"`
	Concurrent int   `json:"concurrent,omitempty"`
	Review     *bool `json:"review,omitempty"`
}

func (s *Server) prepare(req *chatRequest) (*session.Session, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, errors.New("message is required")
	}
	if req.Model == "" {
		req.Model = s.cfg.Model
	}
	if req.Agent == "" {
		req.Agent = s.cfg.Agent
	}
	var sess *session.Session
	if req.SessionID != "" {
		if loaded, err := s.store.Load(req.SessionID); err == nil {
			sess = loaded
		}
	}
	if sess == nil {
		sess = session.New(req.Model, req.Agent)
	}
	return sess, nil
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := s.prepare(&req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	history := toLLMMessages(sess.History(20))
	resp, err := s.runner.RunModel(ctx, req.Agent, req.Model, history, req.Message, nil)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}

	sess.Model = req.Model
	sess.Agent = req.Agent
	sess.Add(session.Message{Role: "user", Content: req.Message})
	msg := session.Message{
		Role: "assistant", Content: resp.Content, Model: req.Model,
		Agent: req.Agent, Tokens: resp.EvalTokens,
	}
	var report *verify.Report
	if req.Verify || s.cfg.Verify {
		report = verify.Run(ctx, resp.Content, verify.Options{
			Router: s.router, JudgeModel: s.cfg.GetJudgeModel(), MinScore: 70,
		})
		msg.Verification = toSessionVerification(report)
	}
	sess.Add(msg)
	_ = s.store.Save(sess)

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id":   sess.ID,
		"content":      resp.Content,
		"model":        resp.Model,
		"backend":      resp.Backend,
		"tokens":       resp.EvalTokens,
		"verification": report,
	})
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := s.prepare(&req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sse := func(payload any) {
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	sse(map[string]any{"type": "start", "session_id": sess.ID, "model": req.Model, "agent": req.Agent})

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	history := toLLMMessages(sess.History(20))
	var sb strings.Builder
	resp, err := s.runner.RunModel(ctx, req.Agent, req.Model, history, req.Message, func(tok string) error {
		sb.WriteString(tok)
		sse(map[string]any{"type": "token", "text": tok})
		return nil
	})
	if err != nil {
		sse(map[string]any{"type": "error", "error": err.Error()})
		return
	}

	content := resp.Content
	if content == "" {
		content = sb.String()
	}

	sess.Model = req.Model
	sess.Agent = req.Agent
	sess.Add(session.Message{Role: "user", Content: req.Message})
	msg := session.Message{Role: "assistant", Content: content, Model: req.Model, Agent: req.Agent, Tokens: resp.EvalTokens}

	if req.Verify || s.cfg.Verify {
		sse(map[string]any{"type": "verifying"})
		report := verify.Run(ctx, content, verify.Options{
			Router: s.router, JudgeModel: s.cfg.GetJudgeModel(), MinScore: 70,
		})
		msg.Verification = toSessionVerification(report)
		sse(map[string]any{"type": "verification", "report": report})
	}
	sess.Add(msg)
	_ = s.store.Save(sess)

	sse(map[string]any{"type": "done", "session_id": sess.ID, "tokens": resp.EvalTokens})
}

// handleSwarm streams a multi-agent swarm run as server-sent events.
//
// Every event is a serialised agent.SwarmEvent, so clients receive the plan,
// each agent's streamed tokens, the agent-to-agent bus traffic, the reviewer
// critique and the final synthesis.
func (s *Server) handleSwarm(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeErr(w, http.StatusBadRequest, "message is required")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	var mu sync.Mutex
	sse := func(payload any) {
		b, _ := json.Marshal(payload)
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	opts := agent.SwarmOptions{
		MinAgents:  req.MinAgents,
		MaxAgents:  req.MaxAgents,
		Handoffs:   req.Handoffs,
		Concurrent: req.Concurrent,
	}
	if req.Review != nil {
		opts.DisableReview = !*req.Review
	}

	_, err := s.runner.RunSwarm(ctx, req.Message, opts, func(ev agent.SwarmEvent) {
		sse(ev)
	})
	if err != nil {
		sse(map[string]any{"type": "error", "error": err.Error()})
		return
	}
	sse(map[string]any{"type": "done"})
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Response string   `json:"response"`
		Context  []string `json:"context"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	report := verify.Run(ctx, req.Response, verify.Options{
		Router: s.router, Context: req.Context, JudgeModel: s.cfg.GetJudgeModel(), MinScore: 70,
	})
	writeJSON(w, http.StatusOK, report)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toLLMMessages(msgs []session.Message) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, llm.Message{Role: m.Role, Content: m.Content})
	}
	return out
}

func toSessionVerification(rep *verify.Report) *session.Verification {
	if rep == nil {
		return nil
	}
	layers := make([]string, 0, len(rep.Layers))
	for _, l := range rep.Layers {
		layers = append(layers, l.Name)
	}
	return &session.Verification{
		Score:       rep.Score,
		Verified:    rep.Verified,
		NeedsReview: rep.NeedsReview,
		Layers:      layers,
		Summary:     rep.Summary,
	}
}
