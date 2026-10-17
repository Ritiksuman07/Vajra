package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ritiksuman07/Vajra/internal/agent"
	"github.com/Ritiksuman07/Vajra/internal/attach"
	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/rag"
	"github.com/Ritiksuman07/Vajra/internal/session"
)

// ── Attachments ─────────────────────────────────────────────────────────────

// mentionRe finds @path references in a prompt. Role mentions from the swarm
// (@coder: …) are ignored automatically: the path simply will not exist.
var mentionRe = regexp.MustCompile(`@([^\s@]+)`)

// collectAttachments resolves the pending attachment list plus any @path
// mentions that point at a real file.
func (m *Model) collectAttachments(prompt string) []attach.File {
	paths := append([]string(nil), m.attached...)
	for _, mt := range mentionRe.FindAllStringSubmatch(prompt, -1) {
		p := strings.TrimRight(mt[1], ".,;:!?)\"'`")
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return attach.Load(paths, attach.Limits{})
}

// attachNames lists the names of files that were actually attached.
func attachNames(files []attach.File) []string {
	var out []string
	for _, f := range files {
		if f.Attached() {
			out = append(out, f.Name)
		}
	}
	return out
}

// attachNotice describes what was attached and what was skipped, so nothing is
// dropped silently.
func attachNotice(files []attach.File) string {
	if len(files) == 0 {
		return ""
	}
	s := "📎 " + attach.Summary(files)
	if sk := attach.SkippedList(files); len(sk) > 0 {
		s += " · skipped " + strings.Join(sk, ", ")
	}
	return s
}

// ── Local retrieval (RAG) ───────────────────────────────────────────────────

func (m *Model) indexStore() *rag.Store { return rag.NewStore(config.IndexDir()) }

type ragIndexedMsg struct {
	dir    string
	chunks int
	files  int
	err    error
}

// indexCmd builds or rebuilds a folder index in the background.
func (m *Model) indexCmd(dir string) tea.Cmd {
	router := m.router
	store := m.indexStore()
	model := m.cfg.GetEmbedModel()
	return func() tea.Msg {
		idx, err := rag.Build(context.Background(), router, dir, rag.Options{Model: model})
		if err != nil {
			return ragIndexedMsg{dir: dir, err: err}
		}
		if err := store.Save(idx); err != nil {
			return ragIndexedMsg{dir: dir, err: err}
		}
		return ragIndexedMsg{dir: dir, chunks: len(idx.Chunks), files: idx.Files}
	}
}

// searchRAG loads a folder's index and retrieves the passages closest to the
// query. It never errors the run: retrieval problems are reported and the
// question is answered without context rather than not at all.
func (m *Model) searchRAG(ctx context.Context, dir, query string) ([]rag.Hit, error) {
	idx, err := m.indexStore().Load(dir)
	if err != nil {
		return nil, err
	}
	return rag.Search(ctx, m.router, idx, query, rag.SearchOptions{})
}

// ragSlash implements /rag [off | build <folder> | <folder>].
func (m *Model) ragSlash(arg string) tea.Cmd {
	arg = strings.TrimSpace(arg)
	switch {
	case arg == "":
		if m.ragDir == "" {
			m.notice = "retrieval off — /rag <folder> to search one, /rag build <folder> to (re)index"
		} else {
			m.notice = fmt.Sprintf("%s · %d passages · /rag off to stop", m.ragStatus(), m.ragChunks)
		}
		return nil
	case arg == "off" || arg == "none":
		m.ragDir = ""
		m.ragChunks = 0
		m.notice = "retrieval off"
		return nil
	}

	dir := strings.Trim(arg, `"`)
	rebuild := false
	if strings.HasPrefix(dir, "build ") {
		rebuild = true
		dir = strings.TrimSpace(strings.TrimPrefix(dir, "build "))
		dir = strings.Trim(dir, `"`)
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		m.err = fmt.Errorf("%q is not a folder", dir)
		return nil
	}

	store := m.indexStore()
	if !rebuild && store.Has(dir) {
		if idx, err := store.Load(dir); err == nil {
			m.ragDir = dir
			m.ragChunks = len(idx.Chunks)
			m.notice = fmt.Sprintf("📚 %s — %d passages indexed", filepath.Base(dir), len(idx.Chunks))
			return nil
		}
	}
	m.notice = "📚 indexing " + dir + "…"
	return m.indexCmd(dir)
}

// ragStatus summarises the retrieval setting for the status line.
func (m *Model) ragStatus() string {
	if m.ragDir == "" {
		return ""
	}
	name := filepath.Base(m.ragDir)
	if m.ragChunks > 0 {
		return fmt.Sprintf("📚 %s (%d)", name, m.ragChunks)
	}
	return "📚 " + name
}

// ── Export ──────────────────────────────────────────────────────────────────

type exportedMsg struct {
	path string
	err  error
}

// exportCmd writes the current session to ~/.vajra/exports.
func (m *Model) exportCmd(format string) tea.Cmd {
	sess := m.sess
	return func() tea.Msg {
		data, ext, err := sess.Export(format)
		if err != nil {
			return exportedMsg{err: err}
		}
		dir := config.ExportsDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return exportedMsg{err: err}
		}
		p := filepath.Join(dir, sess.Filename(ext))
		if err := os.WriteFile(p, data, 0o600); err != nil {
			return exportedMsg{err: err}
		}
		return exportedMsg{path: p}
	}
}

// swarmTurns converts the agent bus into the small, storage-friendly shape the
// session keeps. Storing it means an exported session still shows what the
// agents said to each other.
func swarmTurns(bus []agent.BusMessage) []session.SwarmTurn {
	out := make([]session.SwarmTurn, 0, len(bus))
	for _, m := range bus {
		out = append(out, session.SwarmTurn{From: m.From, To: m.To, Kind: m.Kind, Text: m.Text})
	}
	return out
}

// lastSwarmBus rebuilds the agent-bus view from a session loaded off disk, so
// Ctrl+B still works in a replayed conversation.
func lastSwarmBus(msgs []session.Message) []agent.BusMessage {
	for i := len(msgs) - 1; i >= 0; i-- {
		if len(msgs[i].Swarm) == 0 {
			continue
		}
		out := make([]agent.BusMessage, 0, len(msgs[i].Swarm))
		for _, t := range msgs[i].Swarm {
			out = append(out, agent.BusMessage{
				From: t.From, To: t.To, Kind: t.Kind, Text: t.Text,
			})
		}
		return out
	}
	return nil
}
