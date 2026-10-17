// Package rag implements retrieval-augmented generation over a local folder.
//
// Vajra walks the folder, splits text files into overlapping chunks, embeds
// each chunk with a local embedding model (Ollama by default) and stores the
// result as JSON under ~/.vajra/index. A question is then answered with the
// most similar passages prepended as context, each labelled with its file and
// line so the model can cite them — and so you can check them.
//
// Everything stays on the machine: there is no vector database and no remote
// service. Cosine similarity over an in-memory slice is enough for the folder
// sizes Vajra targets, and it is easy to inspect.
package rag

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ritiksuman07/Vajra/internal/attach"
)

// Embedder is satisfied by llm.Router.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// Chunk is one embedded passage.
type Chunk struct {
	Path string    `json:"path"` // path relative to the indexed folder
	Line int       `json:"line"` // 1-based line the chunk starts on
	Text string    `json:"text"`
	Vec  []float64 `json:"vec"`
}

// Index is a folder's embedded chunks.
type Index struct {
	Dir     string  `json:"dir"`
	Model   string  `json:"model"`
	Dim     int     `json:"dim"`
	Files   int     `json:"files"`
	Created int64   `json:"created"`
	Chunks  []Chunk `json:"chunks"`
}

// Options tune indexing. Zero values are replaced with defaults.
type Options struct {
	ChunkChars  int // characters per chunk (default 1200)
	Overlap     int // overlapping characters (default 200)
	MaxFileSize int // skip files larger than this (default 512 KiB)
	MaxFiles    int // stop after this many files (default 2000)
	MaxChunks   int // stop after this many chunks (default 4000)
	Workers     int // parallel embedding requests (default 4)
	Model       string

	// Progress is called as files are read, then as chunks are embedded.
	Progress func(stage string, done, total int, current string)
}

func (o Options) normalise() Options {
	if o.ChunkChars <= 0 {
		o.ChunkChars = 1200
	}
	if o.Overlap < 0 {
		o.Overlap = 0
	}
	if o.Overlap >= o.ChunkChars {
		o.Overlap = o.ChunkChars / 4
	}
	if o.MaxFileSize <= 0 {
		o.MaxFileSize = 512 << 10
	}
	if o.MaxFiles <= 0 {
		o.MaxFiles = 2000
	}
	if o.MaxChunks <= 0 {
		o.MaxChunks = 4000
	}
	if o.Workers <= 0 {
		o.Workers = 4
	}
	return o
}

// ignoredDirs are never walked: build output, dependencies and metadata.
var ignoredDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".idea": true, ".vscode": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "bin": true, "obj": true, "__pycache__": true,
	".venv": true, "venv": true, "env": true, ".mypy_cache": true,
	".pytest_cache": true, ".next": true, ".cache": true, "coverage": true,
	".vajra": true,
}

// ── Chunking ────────────────────────────────────────────────────────────────

// Piece is one chunk of text plus the line it starts on.
type Piece struct {
	Line int
	Text string
}

// Split cuts text into overlapping pieces of roughly `size` characters,
// recording the 1-based line each piece starts on.
//
// It prefers to break on a newline, then on a space, so pieces tend to end on
// a natural boundary; very long lines (minified code, a wall of prose) are cut
// mid-line rather than being allowed to blow past the size limit.
func Split(text string, size, overlap int) []Piece {
	if size <= 0 {
		size = 1200
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}

	var out []Piece
	line := 1
	pos := 0
	for pos < len(text) {
		end := pos + size
		if end >= len(text) {
			end = len(text)
		} else {
			window := text[pos:end]
			if nl := strings.LastIndexByte(window, '\n'); nl > size/3 {
				end = pos + nl
			} else if sp := strings.LastIndexByte(window, ' '); sp > size/3 {
				end = pos + sp
			}
		}

		piece := text[pos:end]
		if strings.TrimSpace(piece) != "" {
			out = append(out, Piece{Line: line, Text: strings.TrimRight(piece, "\n")})
		}
		line += strings.Count(piece, "\n")

		if end >= len(text) {
			break
		}
		step := end - pos - overlap
		if step < 1 {
			step = end - pos
		}
		if step < 1 {
			step = 1
		}
		next := pos + step
		// The overlap re-reads the tail of this piece, so step the line
		// counter back over the newlines we are about to see again.
		line -= strings.Count(text[next:end], "\n")
		if line < 1 {
			line = 1
		}
		pos = next
	}
	return out
}

// ── Building ────────────────────────────────────────────────────────────────

// Build indexes a folder. It returns the index; callers normally pass it to
// Store.Save.
func Build(ctx context.Context, em Embedder, dir string, opts Options) (*Index, error) {
	opts = opts.normalise()

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", dir)
	}

	type pending struct {
		path string
		rel  string
		line int
		text string
	}
	var queued []pending
	files := 0

	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := d.Name()
		if d.IsDir() {
			if path != abs && (ignoredDirs[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !d.Type().IsRegular() {
			return nil
		}
		if len(queued) >= opts.MaxChunks {
			return filepath.SkipAll
		}
		fi, err := d.Info()
		if err != nil || fi.Size() == 0 || fi.Size() > int64(opts.MaxFileSize) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || attach.LooksBinary(data) {
			return nil
		}
		files++
		if opts.Progress != nil {
			opts.Progress("reading", files, 0, path)
		}
		rel, _ := filepath.Rel(abs, path)
		rel = filepath.ToSlash(rel)
		for _, c := range Split(string(data), opts.ChunkChars, opts.Overlap) {
			queued = append(queued, pending{path: path, rel: rel, line: c.Line, text: c.Text})
			if len(queued) >= opts.MaxChunks {
				break
			}
		}
		if files >= opts.MaxFiles {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(queued) == 0 {
		return nil, fmt.Errorf("no readable text files found in %s", dir)
	}

	// Embed with a small worker pool; Ollama handles a few concurrent calls
	// well and this is the slow part of indexing.
	vecs := make([][]float64, len(queued))
	errs := make([]error, len(queued))
	total := len(queued)
	var done int
	var mu sync.Mutex

	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := opts.Workers
	if workers > total {
		workers = total
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				v, err := em.Embed(ctx, queued[i].text)
				vecs[i], errs[i] = v, err
				mu.Lock()
				done++
				if opts.Progress != nil {
					opts.Progress("embedding", done, total, queued[i].rel)
				}
				mu.Unlock()
			}
		}()
	}
	for i := range queued {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	idx := &Index{Dir: abs, Model: opts.Model, Created: time.Now().Unix()}
	for i, q := range queued {
		if errs[i] != nil {
			// One failing chunk (e.g. an overly long passage) must not abort
			// the whole index.
			continue
		}
		if len(vecs[i]) == 0 {
			continue
		}
		idx.Dim = len(vecs[i])
		idx.Chunks = append(idx.Chunks, Chunk{Path: q.rel, Line: q.line, Text: q.text, Vec: vecs[i]})
	}
	if len(idx.Chunks) == 0 {
		return nil, fmt.Errorf("could not embed any chunks (is the embedding model installed?)")
	}
	idx.Files = files
	return idx, nil
}

// ── Store ───────────────────────────────────────────────────────────────────

// Store keeps indexes on disk.
type Store struct{ Root string }

// NewStore roots an index store at dir.
func NewStore(dir string) *Store { return &Store{Root: dir} }

// Path returns the index file for a folder.
func (st *Store) Path(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	sum := sha1.Sum([]byte(strings.ToLower(filepath.ToSlash(abs))))
	return filepath.Join(st.Root, hex.EncodeToString(sum[:])[:12]+".json")
}

// Has reports whether a folder has been indexed.
func (st *Store) Has(dir string) bool {
	_, err := os.Stat(st.Path(dir))
	return err == nil
}

// Save writes an index atomically.
func (st *Store) Save(idx *Index) error {
	if err := os.MkdirAll(st.Root, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	p := st.Path(idx.Dir)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Load reads a folder's index.
func (st *Store) Load(dir string) (*Index, error) {
	data, err := os.ReadFile(st.Path(dir))
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// Remove deletes a folder's index.
func (st *Store) Remove(dir string) error {
	err := os.Remove(st.Path(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ── Search ──────────────────────────────────────────────────────────────────

// Hit is a scored passage.
type Hit struct {
	Chunk
	Score float64 `json:"score"`
}

// SearchOptions tune retrieval.
type SearchOptions struct {
	K        int     // passages to return (default 5)
	MinScore float64 // drop weaker matches (default 0.25)
	PerFile  int     // at most this many passages from one file (default 2)
}

func (o SearchOptions) normalise() SearchOptions {
	if o.K <= 0 {
		o.K = 5
	}
	if o.MinScore <= 0 {
		o.MinScore = 0.25
	}
	if o.PerFile <= 0 {
		o.PerFile = 2
	}
	return o
}

// Search embeds the query and returns the most similar passages.
func Search(ctx context.Context, em Embedder, idx *Index, query string, opts SearchOptions) ([]Hit, error) {
	opts = opts.normalise()
	if idx == nil || len(idx.Chunks) == 0 {
		return nil, nil
	}
	qv, err := em.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	hits := make([]Hit, 0, len(idx.Chunks))
	for _, c := range idx.Chunks {
		hits = append(hits, Hit{Chunk: c, Score: Cosine(qv, c.Vec)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	out := make([]Hit, 0, opts.K)
	perFile := map[string]int{}
	for _, h := range hits {
		if len(out) >= opts.K {
			break
		}
		if h.Score < opts.MinScore {
			break
		}
		if perFile[h.Path] >= opts.PerFile {
			continue
		}
		perFile[h.Path]++
		out = append(out, h)
	}
	return out, nil
}

// Cosine returns the cosine similarity of two vectors (0 when unusable).
func Cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Context renders hits as a prompt block with file:line citations, trimmed to
// maxTokens (0 means no limit).
func Context(hits []Hit, maxTokens int) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Retrieved from the user's local files. Cite sources as [file:line].\n")
	used := 0
	for i, h := range hits {
		header := fmt.Sprintf("\n[%d] %s:%d (similarity %.2f)\n", i+1, h.Path, h.Line, h.Score)
		body := h.Text
		if maxTokens > 0 {
			// Give each passage an equal share of what is left.
			share := (maxTokens - used - len(hits) + i) / (len(hits) - i)
			if share < 40 {
				share = 40
			}
			body = trimToTokens(body, share)
		}
		fmt.Fprintf(&b, "%s%s\n", header, body)
		used += len(header)/4 + len(body)/4
	}
	return strings.TrimRight(b.String(), "\n")
}

func trimToTokens(s string, tokens int) string {
	max := tokens * 4
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut] + "\n…[truncated]"
}

// Grounding renders retrieved passages as sources for the verification
// pipeline, so "grounded" means grounded in the passages the answer was
// actually given.
func Grounding(hits []Hit, maxEach int) []string {
	if maxEach <= 0 {
		maxEach = 4000
	}
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, fmt.Sprintf("%s:%d\n%s", h.Path, h.Line, trimToTokens(h.Text, maxEach/4)))
	}
	return out
}

// Sources lists the distinct files a set of hits came from.
func Sources(hits []Hit) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hits {
		label := fmt.Sprintf("%s:%d", h.Path, h.Line)
		if seen[h.Path] {
			label = h.Path
		}
		if !seen[h.Path] {
			seen[h.Path] = true
		}
		if !contains(out, label) {
			out = append(out, label)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
