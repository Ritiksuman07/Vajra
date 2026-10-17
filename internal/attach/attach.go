// Package attach loads local files into a conversation as context.
//
// Nothing here talks to the network: attachments are read from disk (CLI and
// TUI) or handed over by the browser (web UI), and are then folded into the
// prompt. Oversized files are truncated rather than silently dropped, and the
// reason a file was skipped is always reported.
package attach

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Limits bound how much text can be attached to one prompt.
type Limits struct {
	MaxFile  int // per file, in bytes (0 = DefaultMaxFile)
	MaxTotal int // across all files, in bytes (0 = DefaultMaxTotal)
	MaxFiles int // number of files (0 = MaxFiles)
}

// Defaults for attachment limits. They are deliberately modest: the point is
// to give the model context, not to fill its window with one file.
const (
	DefaultMaxFile  = 256 << 10 // 256 KiB
	DefaultMaxTotal = 512 << 10 // 512 KiB
	MaxFiles        = 16
)

func (l Limits) normalise() Limits {
	if l.MaxFile <= 0 {
		l.MaxFile = DefaultMaxFile
	}
	if l.MaxTotal <= 0 {
		l.MaxTotal = DefaultMaxTotal
	}
	if l.MaxFiles <= 0 {
		l.MaxFiles = MaxFiles
	}
	return l
}

// File is one attachment.
type File struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Bytes     int    `json:"bytes"`
	Content   string `json:"content,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	// Skipped records why a file was not attached (binary, too large,
	// unreadable). An empty value means it was attached.
	Skipped string `json:"skipped,omitempty"`
}

// Attached reports whether the file's content was included.
func (f File) Attached() bool { return f.Skipped == "" && f.Content != "" }

// Load reads files from disk, applying the limits. It never fails because of a
// single bad file: unreadable or binary files come back with Skipped set so
// the caller can tell the user exactly what happened.
func Load(paths []string, lim Limits) []File {
	lim = lim.normalise()
	out := make([]File, 0, len(paths))
	total := 0

	for i, p := range paths {
		f := File{Name: filepath.Base(p), Path: p}
		if i >= lim.MaxFiles {
			f.Skipped = fmt.Sprintf("more than %d files", lim.MaxFiles)
			out = append(out, f)
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			f.Skipped = "cannot read: " + err.Error()
			out = append(out, f)
			continue
		}
		if info.IsDir() {
			f.Skipped = "is a directory"
			out = append(out, f)
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			f.Skipped = "cannot read: " + err.Error()
			out = append(out, f)
			continue
		}
		f.Bytes = len(data)
		if LooksBinary(data) {
			f.Skipped = "looks binary"
			out = append(out, f)
			continue
		}
		if len(data) > lim.MaxFile {
			data = data[:lim.MaxFile]
			f.Truncated = true
		}
		if total+len(data) > lim.MaxTotal {
			room := lim.MaxTotal - total
			if room <= 0 {
				f.Skipped = "attachment budget exhausted"
				out = append(out, f)
				continue
			}
			data = data[:room]
			f.Truncated = true
		}
		total += len(data)
		f.Content = strings.ToValidUTF8(string(data), "")
		out = append(out, f)
	}
	return out
}

// FromData builds an attachment from content the caller already has (used by
// the web UI, where the browser reads the file).
func FromData(name, content string, lim Limits) File {
	lim = lim.normalise()
	f := File{Name: filepath.Base(name), Bytes: len(content)}
	if len(content) > lim.MaxFile {
		f.Content = Head(content, lim.MaxFile)
		f.Truncated = true
		return f
	}
	f.Content = content
	return f
}

// Head returns at most n bytes, cut on a rune boundary.
func Head(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// LooksBinary uses the usual heuristic: a NUL byte, or invalid UTF-8, in the
// first block of the file. It is exported because the RAG indexer applies the
// same rule when walking a folder.
func LooksBinary(data []byte) bool {
	const sniff = 8192
	if len(data) > sniff {
		data = data[:sniff]
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(data)
}

// Summary describes the outcome of loading attachments in one line.
func Summary(files []File) string {
	if len(files) == 0 {
		return ""
	}
	ok, skipped, bytes := 0, 0, 0
	for _, f := range files {
		if f.Attached() {
			ok++
			bytes += len(f.Content)
		} else {
			skipped++
		}
	}
	s := fmt.Sprintf("%d attached (%s)", ok, HumanSize(bytes))
	if skipped > 0 {
		s += fmt.Sprintf(", %d skipped", skipped)
	}
	return s
}

// SkippedList returns the names and reasons of files that were not attached.
func SkippedList(files []File) []string {
	var out []string
	for _, f := range files {
		if f.Skipped != "" {
			out = append(out, fmt.Sprintf("%s (%s)", f.Name, f.Skipped))
		}
	}
	return out
}

// HumanSize renders a byte count for people.
func HumanSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := unit, 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// Grounding renders the attached files as sources for the verification
// pipeline. Layer 2 asks "does this answer resemble something the model was
// actually given?", so it needs the context text itself, not a description of
// it. Each source is capped so a large attachment does not turn verification
// into a long embedding job.
func Grounding(files []File, maxEach int) []string {
	if maxEach <= 0 {
		maxEach = 4000
	}
	var out []string
	for _, f := range files {
		if f.Attached() {
			out = append(out, f.Name+"\n"+Head(f.Content, maxEach))
		}
	}
	return out
}

// Block renders the attachments as a Markdown block to prepend to a prompt.
// It includes file names and fences so the model can tell where one file ends
// and the next begins, and so it can cite them by name.
func Block(files []File) string {
	var att []File
	for _, f := range files {
		if f.Attached() {
			att = append(att, f)
		}
	}
	if len(att) == 0 {
		return ""
	}
	sort.SliceStable(att, func(i, j int) bool { return att[i].Name < att[j].Name })

	var b strings.Builder
	fmt.Fprintf(&b, "The user attached %d file(s). Treat their contents as given context.\n", len(att))
	for _, f := range att {
		fmt.Fprintf(&b, "\n## %s\n", f.Name)
		if f.Truncated {
			b.WriteString("(truncated — the rest was not sent)\n")
		}
		fence := "```"
		for strings.Contains(f.Content, fence) {
			fence += "`"
		}
		fmt.Fprintf(&b, "%s%s\n%s\n%s\n", fence, Lang(f.Name), strings.TrimRight(f.Content, "\n"), fence)
	}
	return b.String()
}

// Lang guesses a Markdown fence language from a file extension, so code is
// highlighted and the model sees the file type.
func Lang(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".kt":
		return "kotlin"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp", ".hxx":
		return "cpp"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".ps1", ".psm1":
		return "powershell"
	case ".sql":
		return "sql"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".xml":
		return "xml"
	case ".md", ".markdown":
		return "markdown"
	case ".csv":
		return "csv"
	case ".txt", ".log":
		return "text"
	case ".env", ".ini", ".cfg":
		return "ini"
	case ".dockerfile":
		return "dockerfile"
	}
	if strings.EqualFold(filepath.Base(name), "dockerfile") {
		return "dockerfile"
	}
	if strings.EqualFold(filepath.Base(name), "makefile") {
		return "makefile"
	}
	return "text"
}
