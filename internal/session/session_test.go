package session

import (
	"testing"
)

func TestNewAndAdd(t *testing.T) {
	s := New("llama3.1:8b", "assistant")
	if s.ID == "" {
		t.Fatal("expected an id")
	}
	if s.Title != "New chat" {
		t.Errorf("title = %q", s.Title)
	}
	s.Add(Message{Role: "user", Content: "Write a haiku about go routines"})
	if s.Title == "New chat" {
		t.Error("title should be derived from the first user message")
	}
	s.Add(Message{Role: "assistant", Content: "..."})
	if len(s.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(s.Messages))
	}
	if s.Messages[0].At == 0 {
		t.Error("expected a timestamp")
	}
}

func TestTitleTruncation(t *testing.T) {
	long := ""
	for i := 0; i < 20; i++ {
		long += "word "
	}
	s := New("m", "a")
	s.Add(Message{Role: "user", Content: long})
	if len([]rune(s.Title)) > 49 {
		t.Errorf("title too long: %q", s.Title)
	}
}

func TestHistoryLimit(t *testing.T) {
	s := New("m", "a")
	for i := 0; i < 10; i++ {
		s.Add(Message{Role: "user", Content: "q"})
		s.Add(Message{Role: "assistant", Content: "a"})
	}
	s.Add(Message{Role: "system", Content: "note"})
	h := s.History(5)
	if len(h) != 5 {
		t.Fatalf("history = %d, want 5", len(h))
	}
	for _, m := range h {
		if m.Role == "system" {
			t.Error("system messages must be excluded from history")
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	s := New("m", "coder")
	s.Add(Message{Role: "user", Content: "hello"})
	if err := st.Save(s); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := st.Load(s.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "hello" {
		t.Errorf("unexpected messages: %+v", got.Messages)
	}

	metas, err := st.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(metas) != 1 || metas[0].ID != s.ID {
		t.Errorf("unexpected metas: %+v", metas)
	}

	if err := st.Delete(s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	metas, _ = st.List()
	if len(metas) != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", len(metas))
	}
}

func TestStoreListEmpty(t *testing.T) {
	st := NewStore(t.TempDir() + "/does-not-exist")
	metas, err := st.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if metas == nil {
		t.Error("expected a non-nil empty slice (JSON [] rather than null)")
	}
	if len(metas) != 0 {
		t.Errorf("expected 0, got %d", len(metas))
	}
}
