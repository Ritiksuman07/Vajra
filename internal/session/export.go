package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Export renders a session as Markdown or JSON.
//
// Markdown is the human-readable form: it keeps the questions, the answers,
// any verification result, the attachments and retrieved sources that produced
// an answer, and — for swarm runs — the full agent-to-agent transcript. That
// last part matters: the point of swarm mode is that you can read what the
// agents said to each other, so exporting a run must not throw that away.
func (s *Session) Export(format string) ([]byte, string, error) {
	switch strings.ToLower(format) {
	case "", "md", "markdown":
		return []byte(s.Markdown()), "md", nil
	case "json":
		b, err := s.JSON()
		return b, "json", err
	default:
		return nil, "", fmt.Errorf("unknown export format %q (use md or json)", format)
	}
}

// JSON renders the session exactly as it is stored.
func (s *Session) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Filename suggests a file name for an export with the given extension.
func (s *Session) Filename(ext string) string {
	if ext == "" {
		ext = "md"
	}
	title := strings.ToLower(s.Title)
	title = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '_':
			return '-'
		default:
			return -1
		}
	}, title)
	title = strings.Trim(strings.Join(strings.Fields(title), "-"), "-")
	if len(title) > 40 {
		title = strings.TrimRight(title[:40], "-")
	}
	if title == "" {
		title = "session"
	}
	return fmt.Sprintf("vajra-%s-%s.%s", s.ID, title, ext)
}

// Markdown renders the session as a Markdown document.
func (s *Session) Markdown() string {
	var b strings.Builder

	title := s.Title
	if title == "" {
		title = "Session"
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	fmt.Fprintf(&b, "| | |\n|---|---|\n")
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "| **%s** | %s |\n", k, v)
		}
	}
	row("Session", "`"+s.ID+"`")
	row("Model", "`"+s.Model+"`")
	row("Default agent", "`"+s.Agent+"`")
	row("Created", stamp(s.Created))
	row("Updated", stamp(s.Updated))
	row("Turns", fmt.Sprintf("%d", countTurns(s.Messages)))
	b.WriteString("\n")

	if len(s.Messages) == 0 {
		b.WriteString("_No messages._\n")
		return b.String()
	}

	b.WriteString("---\n")
	n := 0
	for _, m := range s.Messages {
		if m.Role == "system" {
			continue
		}
		n++
		b.WriteString("\n")
		b.WriteString(header(m, n))
		b.WriteString("\n\n")
		b.WriteString(strings.TrimRight(m.Content, "\n"))
		b.WriteString("\n")

		if len(m.Attachments) > 0 {
			fmt.Fprintf(&b, "\n**Attached:** %s\n", strings.Join(codeList(m.Attachments), ", "))
		}
		if len(m.Sources) > 0 {
			fmt.Fprintf(&b, "\n**Sources:** %s\n", strings.Join(codeList(m.Sources), ", "))
		}
		if len(m.Swarm) > 0 {
			b.WriteString("\n")
			b.WriteString(swarmSection(m.Swarm))
		}
	}
	return b.String()
}

func header(m Message, n int) string {
	switch m.Role {
	case "user":
		return fmt.Sprintf("## %d · You", n)
	case "assistant":
		who := "Vajra"
		if m.Agent != "" && m.Agent != "assistant" {
			who = "Vajra · " + m.Agent
		}
		meta := []string{}
		if m.Model != "" {
			meta = append(meta, "`"+m.Model+"`")
		}
		if m.Tokens > 0 {
			meta = append(meta, fmt.Sprintf("%d tokens", m.Tokens))
		}
		if m.Verification != nil {
			meta = append(meta, verifyBadge(m.Verification))
		}
		if len(meta) > 0 {
			who += " — " + strings.Join(meta, " · ")
		}
		return fmt.Sprintf("## %d · %s", n, who)
	default:
		return fmt.Sprintf("## %d · %s", n, m.Role)
	}
}

func verifyBadge(v *Verification) string {
	if v == nil {
		return ""
	}
	label := fmt.Sprintf("%.0f%%", v.Score)
	switch {
	case v.NeedsReview:
		return "⚠ needs review · " + label
	case v.Verified:
		return "✓ verified · " + label
	default:
		return "unverified · " + label
	}
}

// swarmSection renders the agent bus for one assistant message.
func swarmSection(turns []SwarmTurn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<details>\n<summary>🐝 Agent bus — %d message(s)</summary>\n\n", len(turns))
	for i, t := range turns {
		fmt.Fprintf(&b, "**%d. %s → %s** · `%s`\n\n", i+1, t.From, t.To, t.Kind)
		fence := "```"
		for strings.Contains(t.Text, fence) {
			fence += "`"
		}
		fmt.Fprintf(&b, "%s\n%s\n%s\n\n", fence, strings.TrimRight(t.Text, "\n"), fence)
	}
	b.WriteString("</details>\n")
	return b.String()
}

// Blurb is a one-line description of a session, used by `vajra export`.
func (s *Session) Blurb() string {
	return fmt.Sprintf("%s  (%d turns, %s)", s.Title, countTurns(s.Messages), stamp(s.Updated))
}

func countTurns(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == "user" || m.Role == "assistant" {
			n++
		}
	}
	return n
}

func stamp(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).Local().Format("2006-01-02 15:04")
}

func codeList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		out = append(out, "`"+s+"`")
	}
	return out
}
