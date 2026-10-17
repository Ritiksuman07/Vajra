// Package session persists chat sessions as human-readable JSON files
// under ~/.vajra/sessions. No database, no daemon — just files you can
// read, diff and back up.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Verification mirrors the outcome of the anti-hallucination pipeline
// for a single assistant message.
type Verification struct {
	Score       float64  `json:"score"`
	Verified    bool     `json:"verified"`
	NeedsReview bool     `json:"needs_review"`
	Layers      []string `json:"layers,omitempty"`
	Summary     string   `json:"summary,omitempty"`
}

// Message is one turn in a session.
type Message struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Model        string        `json:"model,omitempty"`
	Agent        string        `json:"agent,omitempty"`
	Tokens       int           `json:"tokens,omitempty"`
	At           int64         `json:"at"`
	Verification *Verification `json:"verification,omitempty"`
}

// Session is a conversation with metadata.
type Session struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Model    string    `json:"model"`
	Agent    string    `json:"agent"`
	Messages []Message `json:"messages"`
	Created  int64     `json:"created"`
	Updated  int64     `json:"updated"`
}

// Meta is a lightweight session summary for listings.
type Meta struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
	Agent   string `json:"agent"`
	Count   int    `json:"count"`
	Updated int64  `json:"updated"`
}

// NewID returns a short random session id.
func NewID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("s%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// New creates an empty session.
func New(model, agent string) *Session {
	now := time.Now().Unix()
	return &Session{
		ID:       NewID(),
		Title:    "New chat",
		Model:    model,
		Agent:    agent,
		Messages: []Message{},
		Created:  now,
		Updated:  now,
	}
}

// Add appends a message and bumps the timestamp.
func (s *Session) Add(m Message) {
	if m.At == 0 {
		m.At = time.Now().Unix()
	}
	s.Messages = append(s.Messages, m)
	s.Updated = m.At
	if s.Title == "New chat" && m.Role == "user" {
		s.Title = titleFrom(m.Content)
	}
}

// History returns prior turns (excluding system) for the model.
func (s *Session) History(limit int) []Message {
	var out []Message
	for _, m := range s.Messages {
		if m.Role == "user" || m.Role == "assistant" {
			out = append(out, m)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func titleFrom(content string) string {
	t := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if len(t) > 48 {
		t = strings.TrimSpace(t[:48]) + "…"
	}
	if t == "" {
		return "New chat"
	}
	return t
}

// Store persists sessions in a directory.
type Store struct {
	dir string
}

// NewStore returns a store rooted at dir.
func NewStore(dir string) *Store { return &Store{dir: dir} }

// Dir returns the backing directory.
func (st *Store) Dir() string { return st.dir }

func (st *Store) path(id string) string {
	return filepath.Join(st.dir, id+".json")
}

// Save writes a session to disk atomically.
func (st *Store) Save(s *Session) error {
	if err := os.MkdirAll(st.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := st.path(s.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path(s.ID))
}

// Load reads a session by id.
func (st *Store) Load(id string) (*Session, error) {
	data, err := os.ReadFile(st.path(id))
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete removes a session.
func (st *Store) Delete(id string) error {
	err := os.Remove(st.path(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// List returns session summaries, newest first.
func (st *Store) List() ([]Meta, error) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Meta{}, nil
		}
		return nil, err
	}
	var out = make([]Meta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(st.dir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		out = append(out, Meta{
			ID: s.ID, Title: s.Title, Model: s.Model, Agent: s.Agent,
			Count: len(s.Messages), Updated: s.Updated,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated > out[j].Updated })
	return out, nil
}
