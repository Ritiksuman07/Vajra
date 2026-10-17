package rag

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeEmbedder is a deterministic bag-of-words hashing embedder. It is not a
// good embedding model, but it is enough to prove that retrieval ranks the
// passage that shares words with the query first — and it keeps the tests
// hermetic (no Ollama required).
//
// It holds no mutable state on purpose: Build calls Embed from several worker
// goroutines at once, so an embedder with a counter or a cache would need its
// own lock (and a race in a test helper is still a race).
type fakeEmbedder struct {
	err error
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	const dim = 128
	v := make([]float64, dim)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		w = strings.Trim(w, ".,:;()[]{}\"'`#*-_")
		if w == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(w))
		v[h.Sum32()%dim]++
	}
	return v, nil
}

func TestSplitRecordsLineNumbers(t *testing.T) {
	text := strings.Join([]string{
		"line one", "line two", "line three", "line four", "line five",
		"line six", "line seven", "line eight",
	}, "\n")
	pieces := Split(text, 30, 0)
	if len(pieces) < 2 {
		t.Fatalf("expected several pieces, got %d", len(pieces))
	}
	if pieces[0].Line != 1 {
		t.Errorf("first piece starts at line %d, want 1", pieces[0].Line)
	}
	// Every piece must start on or after the previous one, and pieces must
	// not be empty.
	last := 0
	for i, p := range pieces {
		if p.Line < last {
			t.Errorf("piece %d starts at line %d, before the previous %d", i, p.Line, last)
		}
		last = p.Line
		if strings.TrimSpace(p.Text) == "" {
			t.Errorf("piece %d is empty", i)
		}
	}
}

func TestSplitOverlapCarriesContext(t *testing.T) {
	text := strings.Repeat("alpha beta gamma delta ", 40)
	pieces := Split(text, 100, 40)
	if len(pieces) < 2 {
		t.Fatalf("expected multiple pieces, got %d", len(pieces))
	}
	// The overlap means consecutive pieces share a suffix/prefix.
	prev := pieces[0].Text
	next := pieces[1].Text
	tail := prev[len(prev)-20:]
	if !strings.Contains(next, tail) {
		t.Errorf("expected overlap; tail %q not found in %q", tail, next)
	}
}

func TestSplitSingleShortText(t *testing.T) {
	pieces := Split("hello world\n", 1200, 200)
	if len(pieces) != 1 {
		t.Fatalf("pieces = %d, want 1", len(pieces))
	}
	if pieces[0].Text != "hello world" {
		t.Errorf("text = %q", pieces[0].Text)
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestBuildIndexesTextAndSkipsNoise(t *testing.T) {
	root := writeTree(t, map[string]string{
		"notes.md":            "The retry policy uses exponential backoff with jitter.",
		"src/code.go":         "package main\n\nfunc main() { println(\"hi\") }\n",
		".git/config":         "should not be indexed",
		"node_modules/dep.js": "should not be indexed either",
		".hidden":             "hidden files are skipped",
	})

	st := NewStore(t.TempDir())
	idx, err := Build(context.Background(), &fakeEmbedder{}, root, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if idx.Files != 2 {
		t.Errorf("indexed %d files, want 2", idx.Files)
	}
	for _, c := range idx.Chunks {
		if strings.Contains(c.Path, ".git") || strings.Contains(c.Path, "node_modules") || c.Path == ".hidden" {
			t.Errorf("noise file was indexed: %s", c.Path)
		}
		if len(c.Vec) == 0 {
			t.Errorf("chunk from %s has no vector", c.Path)
		}
	}
	if idx.Dim != 128 {
		t.Errorf("dim = %d, want 128", idx.Dim)
	}

	if err := st.Save(idx); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !st.Has(root) {
		t.Error("Has() = false after Save")
	}
	back, err := st.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(back.Chunks) != len(idx.Chunks) {
		t.Errorf("roundtrip lost chunks: %d != %d", len(back.Chunks), len(idx.Chunks))
	}
}

func TestBuildSkipsBinaryAndEmptyFolders(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte{0, 1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), &fakeEmbedder{}, root, Options{}); err == nil {
		t.Error("a folder with no text files must be reported, not silently indexed")
	}
}

func TestBuildRejectsNonFolder(t *testing.T) {
	p := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), &fakeEmbedder{}, p, Options{}); err == nil {
		t.Error("expected an error for a non-folder")
	}
}

func TestSearchRanksRelevantPassageFirst(t *testing.T) {
	root := writeTree(t, map[string]string{
		"retries.md": "The retry policy uses exponential backoff with jitter between attempts.",
		"colors.md":  "The banner uses violet and cyan accents on a dark background.",
		"math.md":    "Pythagoras relates the sides of a right triangle.",
	})
	em := &fakeEmbedder{}
	st := NewStore(t.TempDir())
	idx, err := Build(context.Background(), em, root, Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := st.Save(idx); err != nil {
		t.Fatal(err)
	}

	hits, err := Search(context.Background(), em, idx, "what does the retry backoff policy do?", SearchOptions{K: 3})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("no hits")
	}
	if hits[0].Path != "retries.md" {
		t.Errorf("top hit = %q, want retries.md (scores: %v)", hits[0].Path, scores(hits))
	}
	for i := 1; i < len(hits); i++ {
		if hits[i].Score > hits[i-1].Score {
			t.Errorf("hits are not sorted by score: %v", scores(hits))
		}
	}
}

func scores(hits []Hit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, fmt.Sprintf("%s=%.2f", h.Path, h.Score))
	}
	return out
}

func TestSearchCapsPerFileAndMinScore(t *testing.T) {
	// One file with lots of matching text, one with a single match.
	big := strings.Repeat("retry backoff policy details and more words here. ", 40)
	root := writeTree(t, map[string]string{
		"big.md":  big,
		"one.md":  "retry backoff policy",
		"none.md": "completely unrelated text about gardening",
	})
	em := &fakeEmbedder{}
	idx, err := Build(context.Background(), em, root, Options{ChunkChars: 200, Overlap: 0})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hits, err := Search(context.Background(), em, idx, "retry backoff policy", SearchOptions{K: 6, PerFile: 1, MinScore: 0.2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	perFile := map[string]int{}
	for _, h := range hits {
		perFile[h.Path]++
	}
	for p, n := range perFile {
		if n > 1 {
			t.Errorf("file %s returned %d passages, want at most 1", p, n)
		}
	}
	for _, h := range hits {
		if h.Score < 0.2 {
			t.Errorf("hit below MinScore returned: %+v", h)
		}
	}
}

func TestSearchEmptyIndexIsNotAnError(t *testing.T) {
	hits, err := Search(context.Background(), &fakeEmbedder{}, &Index{}, "anything", SearchOptions{})
	if err != nil {
		t.Fatalf("Search on empty index: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("hits = %v, want none", hits)
	}
}

func TestContextRendersCitationsAndBudget(t *testing.T) {
	hits := []Hit{
		{Chunk: Chunk{Path: "retries.md", Line: 12, Text: "exponential backoff with jitter"}, Score: 0.81},
		{Chunk: Chunk{Path: "src/code.go", Line: 40, Text: strings.Repeat("code ", 500)}, Score: 0.4},
	}
	block := Context(hits, 0)
	if !strings.Contains(block, "[1] retries.md:12") || !strings.Contains(block, "[2] src/code.go:40") {
		t.Errorf("citations missing:\n%s", block)
	}
	if !strings.Contains(block, "similarity 0.81") {
		t.Errorf("scores missing:\n%s", block)
	}

	small := Context(hits, 20)
	if len(small) >= len(block) {
		t.Errorf("budgeted context should be smaller (%d vs %d bytes)", len(small), len(block))
	}
	if !strings.Contains(small, "retries.md:12") {
		t.Errorf("the first passage should survive a tight budget:\n%s", small)
	}
}

func TestContextEmpty(t *testing.T) {
	if got := Context(nil, 0); got != "" {
		t.Errorf("Context(nil) = %q", got)
	}
}

func TestCosine(t *testing.T) {
	if got := Cosine([]float64{1, 0}, []float64{1, 0}); got != 1 {
		t.Errorf("cosine of identical vectors = %v, want 1", got)
	}
	if got := Cosine([]float64{1, 0}, []float64{0, 1}); got != 0 {
		t.Errorf("cosine of orthogonal vectors = %v, want 0", got)
	}
	if got := Cosine([]float64{1, 0}, []float64{1, 0, 0}); got != 0 {
		t.Errorf("cosine of mismatched dimensions = %v, want 0", got)
	}
	if got := Cosine(nil, nil); got != 0 {
		t.Errorf("cosine of empty vectors = %v, want 0", got)
	}
}

func TestStorePathIsStableAndDistinct(t *testing.T) {
	st := NewStore(t.TempDir())
	a, b := t.TempDir(), t.TempDir()

	// The index file name must be a pure function of the folder: two calls, and
	// a fresh Store, have to agree — otherwise an index saved in one run would
	// not be found in the next, which is the whole point of the store. The
	// directory part does follow the store, so compare file names.
	first := st.Path(a)
	if second := st.Path(a); second != first {
		t.Errorf("Path must be stable for the same folder: %q vs %q", first, second)
	}
	if again := filepath.Base(NewStore(t.TempDir()).Path(a)); again != filepath.Base(first) {
		t.Errorf("index file name must not depend on the store dir: %q vs %q", again, filepath.Base(first))
	}
	if st.Path(b) == first {
		t.Error("different folders must not share an index file")
	}
	if !strings.HasSuffix(first, ".json") {
		t.Errorf("index path = %q", first)
	}
	if err := st.Remove(a); err != nil {
		t.Errorf("Remove on a missing index should be a no-op, got %v", err)
	}
}

func TestBuildReportsEmbedFailureOnce(t *testing.T) {
	root := writeTree(t, map[string]string{"a.md": "some text to embed"})
	em := &fakeEmbedder{err: fmt.Errorf("model not installed")}
	if _, err := Build(context.Background(), em, root, Options{}); err == nil {
		t.Error("expected Build to fail when the embedding model is unavailable")
	}
}

func TestGroundingLabelsPassages(t *testing.T) {
	hits := []Hit{
		{Chunk: Chunk{Path: "retries.md", Line: 12, Text: "exponential backoff"}, Score: 0.8},
		{Chunk: Chunk{Path: "src/code.go", Line: 40, Text: strings.Repeat("code ", 5000)}, Score: 0.5},
	}
	g := Grounding(hits, 400)
	if len(g) != 2 {
		t.Fatalf("grounding = %d entries, want 2", len(g))
	}
	if !strings.Contains(g[0], "retries.md:12") || !strings.Contains(g[0], "exponential backoff") {
		t.Errorf("grounding should label the passage: %q", g[0])
	}
	if len(g[1]) > 500 {
		t.Errorf("passage not capped: %d bytes", len(g[1]))
	}
	if got := Grounding(nil, 0); len(got) != 0 {
		t.Errorf("Grounding(nil) = %v", got)
	}
}
