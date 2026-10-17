package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/agent"
	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/rag"
	"github.com/Ritiksuman07/Vajra/internal/session"
)

func TestCollectAttachmentsFromPendingListAndMentions(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(notes, []byte("# notes\nhello"), 0o600); err != nil {
		t.Fatal(err)
	}
	code := filepath.Join(dir, "code.go")
	if err := os.WriteFile(code, []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := &Model{attached: []string{notes}}

	// A mention of a real file is attached; an @role mention (which is not a
	// file) is ignored, as is a non-existent path.
	files := m.collectAttachments("explain @" + code + " and also @coder: check this @nope.txt")
	if len(files) != 2 {
		t.Fatalf("attached %d files, want 2 (%+v)", len(files), files)
	}
	for _, f := range files {
		if !f.Attached() {
			t.Errorf("%s was not attached: %s", f.Name, f.Skipped)
		}
	}
}

func TestCollectAttachmentsNone(t *testing.T) {
	m := &Model{}
	if got := m.collectAttachments("just a question, no files"); got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestAttachNamesAndNotice(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	_ = os.WriteFile(good, []byte("hi"), 0o600)
	bad := filepath.Join(dir, "bad.bin")
	_ = os.WriteFile(bad, []byte{0, 1, 2}, 0o600)

	m := &Model{attached: []string{good, bad}}
	files := m.collectAttachments("summarise")
	names := attachNames(files)
	if len(names) != 1 || names[0] != "good.txt" {
		t.Fatalf("names = %v", names)
	}
	notice := attachNotice(files)
	if !strings.Contains(notice, "1 attached") || !strings.Contains(notice, "skipped") {
		t.Errorf("notice = %q — skipped files must be reported", notice)
	}
}

func TestSwarmTurnsAndReplay(t *testing.T) {
	bus := []agent.BusMessage{
		{ID: "m1", From: "researcher", To: "all", Kind: "reply", Text: "facts"},
		{ID: "m2", From: "executor", To: "coder", Kind: "handoff", Text: "add a test"},
		{ID: "m3", From: "chief", To: "user", Kind: "synthesis", Text: "final"},
	}
	msgs := []session.Message{
		{Role: "user", Content: "do it"},
		{
			Role: "assistant", Content: "final", Swarm: swarmTurns(bus),
			Attachments: []string{"notes.md"}, Sources: []string{"a.md:1"},
		},
	}

	back := lastSwarmBus(msgs)
	if len(back) != 3 {
		t.Fatalf("replayed %d messages, want 3", len(back))
	}
	if back[1].From != "executor" || back[1].To != "coder" || back[1].Kind != "handoff" {
		t.Errorf("handoff lost in replay: %+v", back[1])
	}
	if got := lastSwarmBus([]session.Message{{Role: "assistant", Content: "plain"}}); got != nil {
		t.Errorf("expected nil for a session without a swarm, got %v", got)
	}
}

func TestCtxMeterAndBadges(t *testing.T) {
	m := &Model{cfg: &config.Config{MaxContext: 8192}}
	if got := m.ctxMeter(); got != "0/8.2k" {
		t.Errorf("ctxMeter = %q", got)
	}
	m.messages = []session.Message{{Role: "user", Content: strings.Repeat("x", 4000)}}
	got := m.ctxMeter()
	if !strings.HasPrefix(got, "1.0k/") {
		t.Errorf("ctxMeter after 4000 chars = %q, want ~1.0k used", got)
	}
	if b := m.contextBadges(); b != "" {
		t.Errorf("badges = %q, want empty", b)
	}
	m.attached = []string{"a", "b"}
	m.ragDir = "/tmp/docs"
	m.ragChunks = 42
	if b := m.contextBadges(); !strings.Contains(b, "📎2") || !strings.Contains(b, "📚42") {
		t.Errorf("badges = %q", b)
	}
	if s := m.ragStatus(); !strings.Contains(s, "docs") || !strings.Contains(s, "42") {
		t.Errorf("ragStatus = %q", s)
	}
}

func TestHumanTokens(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1.0k", 8192: "8.2k"}
	for in, want := range cases {
		if got := humanTokens(in); got != want {
			t.Errorf("humanTokens(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestExportCmdWritesFile(t *testing.T) {
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	m := &Model{sess: session.New("llama3.1:8b", "assistant")}
	m.sess.Add(session.Message{Role: "user", Content: "hello"})
	m.sess.Add(session.Message{
		Role: "assistant", Content: "hi",
		Swarm: []session.SwarmTurn{{From: "coder", To: "all", Kind: "reply", Text: "code"}},
	})

	msg, ok := m.exportCmd("md")().(exportedMsg)
	if !ok {
		t.Fatal("exportCmd did not return an exportedMsg")
	}
	if msg.err != nil {
		t.Fatalf("export: %v", msg.err)
	}
	data, err := os.ReadFile(msg.path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !strings.Contains(string(data), "# hello") && !strings.Contains(string(data), "## 1 · You") {
		t.Errorf("export does not look like markdown:\n%s", data)
	}
	if !strings.Contains(string(data), "coder → all") {
		t.Errorf("export dropped the swarm transcript:\n%s", data)
	}
	if !strings.HasPrefix(filepath.Base(msg.path), "vajra-") {
		t.Errorf("export path = %q", msg.path)
	}
}

func TestExportCmdUnknownFormat(t *testing.T) {
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	m := &Model{sess: session.New("m", "a")}
	msg := m.exportCmd("pdf")().(exportedMsg)
	if msg.err == nil {
		t.Error("expected an error for an unsupported format")
	}
}

func TestRagSlashRejectsNonFolder(t *testing.T) {
	m := &Model{cfg: &config.Config{}}
	if cmd := m.ragSlash("/definitely/not/here"); cmd != nil {
		t.Error("expected no command for a missing folder")
	}
	if m.err == nil {
		t.Error("expected an error to be recorded")
	}
}

func TestRagSlashOff(t *testing.T) {
	m := &Model{cfg: &config.Config{}, ragDir: "/tmp/x", ragChunks: 3}
	if cmd := m.ragSlash("off"); cmd != nil {
		t.Error("expected no command when turning retrieval off")
	}
	if m.ragDir != "" || m.ragChunks != 0 {
		t.Errorf("retrieval not cleared: %q %d", m.ragDir, m.ragChunks)
	}
}

func TestRagSlashUsesExistingIndex(t *testing.T) {
	t.Setenv("VAJRA_CONFIG_DIR", t.TempDir())
	dir := t.TempDir()
	m := &Model{cfg: &config.Config{}}

	// Prime the store with a tiny index so /rag does not need to embed.
	store := m.indexStore()
	idx := &rag.Index{
		Dir: dir, Model: "test", Dim: 2, Files: 1,
		Chunks: []rag.Chunk{{Path: "a.md", Line: 1, Text: "hello", Vec: []float64{1, 0}}},
	}
	if err := store.Save(idx); err != nil {
		t.Fatal(err)
	}
	if cmd := m.ragSlash(dir); cmd != nil {
		t.Fatal("an existing index should be reused, not rebuilt")
	}
	if m.ragDir != dir || m.ragChunks != 1 {
		t.Errorf("ragDir=%q ragChunks=%d", m.ragDir, m.ragChunks)
	}
}
