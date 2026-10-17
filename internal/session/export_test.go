package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleSession() *Session {
	s := New("llama3.1:8b", "assistant")
	s.Title = "How do retries work?"
	s.Add(Message{
		Role:        "user",
		Content:     "How do retries work?",
		Attachments: []string{"notes.md", "retries.go"},
	})
	s.Add(Message{
		Role:    "assistant",
		Content: "They use exponential backoff.",
		Model:   "llama3.1:8b",
		Agent:   "swarm",
		Tokens:  42,
		Sources: []string{"retries.md:12", "retries.md"},
		Verification: &Verification{
			Score: 94, Verified: true, Summary: "grounded in the retrieved passage",
		},
		Swarm: []SwarmTurn{
			{From: "researcher", To: "all", Kind: "reply", Text: "Facts: backoff grows 2^n."},
			{From: "executor", To: "coder", Kind: "handoff", Text: "please add a test"},
			{From: "coder", To: "executor", Kind: "reply", Text: "```go\nfunc TestRetry(t *testing.T) {}\n```"},
			{From: "chief", To: "user", Kind: "synthesis", Text: "Final answer."},
		},
	})
	return s
}

func TestMarkdownExport(t *testing.T) {
	md := sampleSession().Markdown()

	for _, want := range []string{
		"# How do retries work?",
		"`llama3.1:8b`",
		"## 1 · You",
		"## 2 · Vajra · swarm",
		"42 tokens",
		"✓ verified · 94%",
		"**Attached:** `notes.md`, `retries.go`",
		"**Sources:** `retries.md:12`, `retries.md`",
		"Agent bus — 4 message(s)",
		"executor → coder",
		"handoff",
		"please add a test",
		"chief → user",
		"Final answer.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown export missing %q\n---\n%s", want, md)
		}
	}
}

func TestMarkdownSwarmFenceWidens(t *testing.T) {
	s := New("m", "a")
	s.Add(Message{Role: "user", Content: "hi"})
	s.Add(Message{
		Role: "assistant", Content: "answer",
		Swarm: []SwarmTurn{{From: "coder", To: "all", Kind: "reply", Text: "text with ``` fence ``` inside"}},
	})
	md := s.Markdown()
	if !strings.Contains(md, "````") {
		t.Errorf("a nested fence must widen the outer fence:\n%s", md)
	}
}

func TestMarkdownNeedsReviewBadge(t *testing.T) {
	s := New("m", "a")
	s.Add(Message{Role: "assistant", Content: "x", Verification: &Verification{Score: 30, NeedsReview: true}})
	if md := s.Markdown(); !strings.Contains(md, "⚠ needs review · 30%") {
		t.Errorf("review badge missing:\n%s", md)
	}
}

func TestMarkdownEmptySession(t *testing.T) {
	md := New("m", "a").Markdown()
	if !strings.Contains(md, "_No messages._") {
		t.Errorf("empty session should say so:\n%s", md)
	}
}

func TestJSONExportRoundTrips(t *testing.T) {
	s := sampleSession()
	data, ext, err := s.Export("json")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if ext != "json" {
		t.Errorf("ext = %q", ext)
	}
	var back Session
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Messages) != len(s.Messages) {
		t.Fatalf("messages = %d, want %d", len(back.Messages), len(s.Messages))
	}
	last := back.Messages[len(back.Messages)-1]
	if len(last.Swarm) != 4 || len(last.Sources) != 2 || len(back.Messages[0].Attachments) != 2 {
		t.Errorf("round trip lost metadata: %+v", last)
	}
}

func TestExportFormatSelection(t *testing.T) {
	s := sampleSession()
	for _, f := range []string{"", "md", "markdown", "MD"} {
		data, ext, err := s.Export(f)
		if err != nil {
			t.Fatalf("Export(%q): %v", f, err)
		}
		if ext != "md" {
			t.Errorf("Export(%q) ext = %q, want md", f, ext)
		}
		if !strings.HasPrefix(string(data), "# ") {
			t.Errorf("Export(%q) does not look like markdown", f)
		}
	}
	if _, _, err := s.Export("pdf"); err == nil {
		t.Error("expected an error for an unknown format")
	}
}

func TestFilenameIsSafe(t *testing.T) {
	s := sampleSession()
	s.Title = "How do retries/backoff & jitter work? (v2)"
	name := s.Filename("md")
	if strings.ContainsAny(name, `/\&?()`) {
		t.Errorf("filename %q contains unsafe characters", name)
	}
	if !strings.HasPrefix(name, "vajra-"+s.ID) || !strings.HasSuffix(name, ".md") {
		t.Errorf("filename = %q", name)
	}
	if got := New("m", "a").Filename("json"); !strings.HasSuffix(got, ".json") {
		t.Errorf("filename for an untitled session = %q", got)
	}
}

func TestVerificationStillSerialises(t *testing.T) {
	// Guards the existing on-disk shape: adding fields to Message must not
	// break verification persistence.
	s := sampleSession()
	data, err := s.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"verification"`) || !strings.Contains(string(data), `"swarm"`) {
		t.Errorf("json missing verification/swarm:\n%s", data)
	}
}
