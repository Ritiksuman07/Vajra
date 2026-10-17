package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadReadsTextFile(t *testing.T) {
	p := writeTemp(t, "main.go", "package main\n\nfunc main() {}\n")
	files := Load([]string{p}, Limits{})
	if len(files) != 1 {
		t.Fatalf("got %d files", len(files))
	}
	f := files[0]
	if !f.Attached() {
		t.Fatalf("file not attached: %+v", f)
	}
	if f.Name != "main.go" {
		t.Errorf("name = %q", f.Name)
	}
	if !strings.Contains(f.Content, "func main") {
		t.Errorf("content = %q", f.Content)
	}
}

func TestLoadSkipsBinary(t *testing.T) {
	p := filepath.Join(t.TempDir(), "blob.bin")
	if err := os.WriteFile(p, []byte{0x00, 0x01, 0x02, 0xff, 0xfe}, 0o600); err != nil {
		t.Fatal(err)
	}
	files := Load([]string{p}, Limits{})
	if files[0].Attached() {
		t.Error("binary file should not be attached")
	}
	if files[0].Skipped != "looks binary" {
		t.Errorf("skipped = %q", files[0].Skipped)
	}
}

func TestLoadSkipsMissingAndDirs(t *testing.T) {
	dir := t.TempDir()
	files := Load([]string{filepath.Join(dir, "nope.txt"), dir}, Limits{})
	if len(files) != 2 {
		t.Fatalf("got %d files", len(files))
	}
	for _, f := range files {
		if f.Attached() {
			t.Errorf("%s should have been skipped", f.Name)
		}
		if f.Skipped == "" {
			t.Errorf("%s has no reason recorded", f.Name)
		}
	}
}

func TestLoadTruncatesBigFiles(t *testing.T) {
	p := writeTemp(t, "big.txt", strings.Repeat("a", 5000))
	files := Load([]string{p}, Limits{MaxFile: 1000})
	f := files[0]
	if !f.Truncated {
		t.Error("expected the file to be marked truncated")
	}
	if len(f.Content) != 1000 {
		t.Errorf("content = %d bytes, want 1000", len(f.Content))
	}
}

func TestLoadRespectsTotalBudget(t *testing.T) {
	a := writeTemp(t, "a.txt", strings.Repeat("a", 400))
	b := writeTemp(t, "b.txt", strings.Repeat("b", 400))
	files := Load([]string{a, b}, Limits{MaxFile: 400, MaxTotal: 500})
	if !files[0].Attached() {
		t.Fatal("first file should be attached")
	}
	if got := len(files[0].Content) + len(files[1].Content); got > 500 {
		t.Errorf("total attached %d bytes > 500", got)
	}
	if !files[1].Truncated {
		t.Error("second file should be truncated to fit the total budget")
	}
}

func TestFromDataTruncates(t *testing.T) {
	f := FromData("chat.md", strings.Repeat("x", 300), Limits{MaxFile: 100})
	if !f.Truncated || len(f.Content) != 100 {
		t.Errorf("truncated=%v len=%d", f.Truncated, len(f.Content))
	}
}

func TestBlockRendersFences(t *testing.T) {
	block := Block([]File{
		{Name: "main.go", Content: "package main"},
		{Name: "notes.md", Content: "hello\n```\nnested fence\n```"},
		{Name: "image.png", Skipped: "looks binary"},
	})
	if !strings.Contains(block, "## main.go") || !strings.Contains(block, "```go") {
		t.Errorf("go file not fenced with a language:\n%s", block)
	}
	if !strings.Contains(block, "## notes.md") {
		t.Errorf("markdown file missing:\n%s", block)
	}
	if strings.Contains(block, "image.png") {
		t.Errorf("skipped files must not appear in the block:\n%s", block)
	}
	// A file containing fences must get a longer fence so it cannot break out.
	if !strings.Contains(block, "````markdown") {
		t.Errorf("nested fences should widen the outer fence:\n%s", block)
	}
}

func TestBlockEmptyWhenNothingAttached(t *testing.T) {
	if got := Block([]File{{Name: "x", Skipped: "looks binary"}}); got != "" {
		t.Errorf("block = %q, want empty", got)
	}
}

func TestSummaryAndSkippedList(t *testing.T) {
	files := []File{
		{Name: "good.txt", Content: "hi"},
		{Name: "bad.bin", Skipped: "looks binary"},
	}
	s := Summary(files)
	if !strings.Contains(s, "1 attached") || !strings.Contains(s, "1 skipped") {
		t.Errorf("summary = %q", s)
	}
	sk := SkippedList(files)
	if len(sk) != 1 || !strings.Contains(sk[0], "bad.bin") {
		t.Errorf("skipped list = %v", sk)
	}
}

func TestLang(t *testing.T) {
	cases := map[string]string{
		"a.go": "go", "b.py": "python", "c.md": "markdown",
		"Dockerfile": "dockerfile", "Makefile": "makefile", "weird.zzz": "text",
	}
	for name, want := range cases {
		if got := Lang(name); got != want {
			t.Errorf("Lang(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestGroundingCarriesNamesAndText(t *testing.T) {
	p := writeTemp(t, "notes.md", "the retry cap is 30 seconds")
	files := Load([]string{p}, Limits{})
	g := Grounding(files, 0)
	if len(g) != 1 {
		t.Fatalf("grounding = %d entries, want 1", len(g))
	}
	if !strings.Contains(g[0], "notes.md") || !strings.Contains(g[0], "30 seconds") {
		t.Errorf("grounding should carry the name and the text: %q", g[0])
	}
	if got := Grounding(nil, 0); got != nil {
		t.Errorf("Grounding(nil) = %v", got)
	}
}

func TestGroundingCapsEachSource(t *testing.T) {
	f := FromData("big.txt", strings.Repeat("x", 20000), Limits{})
	g := Grounding([]File{f}, 100)
	if len(g) != 1 {
		t.Fatalf("grounding = %d entries", len(g))
	}
	if len(g[0]) > 200 {
		t.Errorf("source not capped: %d bytes", len(g[0]))
	}
}

func TestGroundingSkipsUnattachedFiles(t *testing.T) {
	f := FromData("blob.bin", "x\x00y", Limits{})
	f.Skipped = "looks binary"
	if got := Grounding([]File{f}, 0); len(got) != 0 {
		t.Errorf("a skipped file must not ground verification: %v", got)
	}
}
