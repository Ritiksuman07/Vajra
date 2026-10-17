package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Ritiksuman07/Vajra/internal/llm"
)

// ── Palette ───────────────────────────────────────────
var (
	colAccent  = lipgloss.Color("99") // violet
	colAccent2 = lipgloss.Color("45") // cyan
	colText    = lipgloss.Color("252")
	colMuted   = lipgloss.Color("243")
	colFaint   = lipgloss.Color("240")
	colOK      = lipgloss.Color("42")
	colWarn    = lipgloss.Color("214")
	colErr     = lipgloss.Color("203")
	colUser    = lipgloss.Color("111")
)

var (
	stHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("57")).Padding(0, 1)
	stFooterBar = lipgloss.NewStyle().Foreground(colMuted)
	stUser      = lipgloss.NewStyle().Foreground(colUser).Bold(true)
	stText      = lipgloss.NewStyle().Foreground(colText)
	stMuted     = lipgloss.NewStyle().Foreground(colMuted)
	stFaint     = lipgloss.NewStyle().Foreground(colFaint)
	stOK        = lipgloss.NewStyle().Foreground(colOK)
	stWarn      = lipgloss.NewStyle().Foreground(colWarn)
	stErr       = lipgloss.NewStyle().Foreground(colErr)
	stCode      = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	stCodeBG    = lipgloss.NewStyle().Foreground(lipgloss.Color("150")).Background(lipgloss.Color("236"))
	stTitle     = lipgloss.NewStyle().Bold(true).Foreground(colAccent2)
	stKey       = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	stSelected  = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("231")).Padding(0, 1)
	stBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colFaint).Padding(0, 2)
)

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return "\n  " + m.spinner.View() + " starting vajra…"
	}
	switch m.screen {
	case scrHelp:
		return m.viewHelp()
	case scrModels:
		return m.viewModels()
	case scrAgents:
		return m.viewAgents()
	case scrSessions:
		return m.viewSessions()
	case scrSwarm:
		return m.viewSwarmBus()
	}
	return m.viewChat()
}

// ── Chat ──────────────────────────────────────────────

func (m *Model) viewChat() string {
	var b strings.Builder

	// Header.
	status := "○ offline"
	if m.health {
		status = "● ready"
	}
	head := fmt.Sprintf(" 🛡️  vajra %s  %s  %s  verify:%s swarm:%s ctx:%s ",
		m.version, m.curModel, status, onOff(m.verifyOn), onOff(m.swarmOn), m.ctxMeter())
	if extras := m.contextBadges(); extras != "" {
		head = strings.TrimRight(head, " ") + "  " + extras + " "
	}
	b.WriteString(stHeader.Width(m.width).Render(head))
	b.WriteString("\n")

	// Body.
	b.WriteString(m.vp.View())
	b.WriteString("\n")

	// Footer.
	inputLine := m.input.View()
	b.WriteString(inputLine)
	b.WriteString("\n")

	bar := ""
	if m.streaming {
		if m.swarmRunning {
			bar = " " + m.spinner.View() + " " + m.swarmStatus()
		} else {
			bar = " " + m.spinner.View() + " generating…"
		}
	} else if m.err != nil {
		bar = " ⚠ " + m.err.Error()
	} else if m.notice != "" {
		bar = " " + m.notice
	} else if m.attached != nil || m.ragDir != "" {
		var bits []string
		if len(m.attached) > 0 {
			bits = append(bits, fmt.Sprintf("📎 %d file(s) → %s", len(m.attached), strings.Join(m.attached, ", ")))
		}
		if s := m.ragStatus(); s != "" {
			bits = append(bits, s+" per prompt")
		}
		bar = " " + strings.Join(bits, " · ")
	} else {
		bar = " Ctrl+H help · Ctrl+M models · Ctrl+A agents · Ctrl+B bus · Ctrl+E export · Ctrl+T swarm · Ctrl+Q quit"
	}
	style := stFaint
	if m.err != nil {
		style = stErr
	}
	if w := m.width - 1; w > 0 {
		style = style.Width(w)
	}
	b.WriteString(style.Render(bar))
	return b.String()
}

func (m *Model) renderMessages() string {
	contentW := m.vp.Width - 2
	if contentW < 20 {
		contentW = 20
	}

	if len(m.messages) == 0 {
		box := stBox.Width(minInt(contentW, 70)).Render(
			stTitle.Render("Welcome to Vajra 🛡️") + "\n\n" +
				stText.Render("Local-first AI agents. Nothing leaves your machine.") + "\n\n" +
				stMuted.Render("Type a message and press Enter.") + "\n" +
				stMuted.Render("Switch agents with Ctrl+A · models with Ctrl+M.") + "\n" +
				stMuted.Render("Toggle verification with Ctrl+V."))
		return "\n" + box + "\n"
	}

	var b strings.Builder
	for i, msg := range m.messages {
		last := i == len(m.messages)-1
		// While a swarm runs, the transcript below replaces the empty
		// assistant placeholder.
		if last && m.swarmRunning && msg.Role == "assistant" && strings.TrimSpace(msg.Content) == "" {
			continue
		}
		switch msg.Role {
		case "user":
			b.WriteString(stUser.Render("  You "))
			b.WriteString("\n")
			b.WriteString(wrap(msg.Content, contentW, stText))
			b.WriteString("\n")
			if len(msg.Attachments) > 0 {
				b.WriteString(stFaint.Render("  📎 " + strings.Join(msg.Attachments, ", ")))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		case "system":
			b.WriteString(stWarn.Render("  ⚠ " + wrapInline(msg.Content, contentW-4)))
			b.WriteString("\n\n")
		case "assistant":
			label := msg.Agent
			if label == "" {
				label = "assistant"
			}
			b.WriteString(stTitle.Render("  ◆ Vajra"))
			b.WriteString(stFaint.Render("  " + label))
			b.WriteString("\n")
			body := msg.Content
			if last && m.streaming {
				body += "▋"
			}
			if strings.TrimSpace(body) == "" {
				body = m.spinner.View() + " thinking…"
			}
			b.WriteString(renderMarkdownish(body, contentW))
			b.WriteString("\n")
			if msg.Verification != nil {
				v := msg.Verification
				if v.NeedsReview {
					b.WriteString(stWarn.Render(fmt.Sprintf("  ⚠ needs review · trust %.0f%%", v.Score)))
				} else {
					b.WriteString(stOK.Render(fmt.Sprintf("  ✓ verified · trust %.0f%%", v.Score)))
				}
				b.WriteString("\n")
			} else if msg.Tokens > 0 {
				b.WriteString(stFaint.Render(fmt.Sprintf("  %d tokens", msg.Tokens)))
				b.WriteString("\n")
			}
			// The live swarm transcript is rendered above, next to the prompt
			// that triggered it; a replayed session gets a pointer to the bus
			// instead of duplicating the whole thing.
			liveShown := i == m.swarmAfter+1 && len(m.swarmFeed) > 0
			if len(msg.Swarm) > 0 && !liveShown {
				b.WriteString(stFaint.Render(fmt.Sprintf("  🐝 agent bus — %d messages · Ctrl+B to read", len(msg.Swarm))))
				b.WriteString("\n")
			}
			if len(msg.Sources) > 0 {
				b.WriteString(stFaint.Render("  📚 sources: " + wrapInline(strings.Join(msg.Sources, ", "), contentW-16)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		// Weave the swarm transcript in right after the prompt that
		// triggered it.
		if i == m.swarmAfter && len(m.swarmFeed) > 0 {
			b.WriteString(m.renderSwarm(contentW))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// swarmStatus summarises which agents are currently working, for the
// status bar.
func (m *Model) swarmStatus() string {
	if len(m.swarmOpen) == 0 {
		return "swarm working…"
	}
	names := make([]string, 0, len(m.swarmOpen))
	for i := range m.swarmFeed {
		it := m.swarmFeed[i]
		if it.Kind != "agent" {
			continue
		}
		if _, open := m.swarmOpen[it.Role]; !open {
			continue
		}
		_, name := roleParts(it.Role)
		names = append(names, name)
	}
	if len(names) == 0 {
		return "swarm working…"
	}
	return "swarm · " + strings.Join(names, " + ") + " working…"
}

// renderMarkdownish renders a small subset of Markdown: code fences,
// headings and bullets, with terminal styling.
func renderMarkdownish(s string, width int) string {
	var b strings.Builder
	inCode := false
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			inCode = !inCode
			continue
		case inCode:
			b.WriteString("  " + stCodeBG.Render(" "+line+" "))
			b.WriteString("\n")
		case strings.HasPrefix(trimmed, "### "):
			b.WriteString(stTitle.Render("   " + strings.TrimPrefix(trimmed, "### ")))
			b.WriteString("\n")
		case strings.HasPrefix(trimmed, "## "):
			b.WriteString(stTitle.Render("   " + strings.TrimPrefix(trimmed, "## ")))
			b.WriteString("\n")
		case strings.HasPrefix(trimmed, "# "):
			b.WriteString(stTitle.Render("   " + strings.TrimPrefix(trimmed, "# ")))
			b.WriteString("\n")
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			b.WriteString("   • " + wrapInline(inline(trimmed[2:]), width-4))
			b.WriteString("\n")
		default:
			b.WriteString(wrap("   "+line, width, stText))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func inline(s string) string {
	return strings.ReplaceAll(s, "**", "")
}

func wrapInline(s string, width int) string {
	return wrap(s, width, stText)
}

// wrap soft-wraps text to width and applies a style.
func wrap(text string, width int, style lipgloss.Style) string {
	if width < 10 {
		width = 10
	}
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if len(line)+1+len(w) > width {
				out = append(out, line)
				line = w
			} else {
				line += " " + w
			}
		}
		out = append(out, line)
	}
	return style.Render(strings.Join(out, "\n"))
}

// ── Lists ─────────────────────────────────────────────

func (m *Model) listView(title string, rows []string, hint string) string {
	var b strings.Builder
	b.WriteString(stHeader.Width(m.width).Render(" 🛡️  vajra  ·  " + title))
	b.WriteString("\n\n")
	for i, r := range rows {
		if rows[i] == "" {
			continue
		}
		if i == m.cursor {
			b.WriteString("  " + stSelected.Render("▸ "+r))
		} else {
			b.WriteString("    " + r)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(stFaint.Render("  " + hint))
	return b.String()
}

func (m *Model) viewModels() string {
	rows := make([]string, 0, len(m.models))
	for _, mm := range m.models {
		mark := " "
		if mm.Name == m.curModel {
			mark = "●"
		}
		rows = append(rows, fmt.Sprintf("%s %-44s %s", mark, mm.Name, stFaint.Render(mm.Backend)))
	}
	if len(rows) == 0 {
		rows = append(rows, stMuted.Render("No models found — is Ollama running?  Try: vajra pull llama3.1:8b"))
	}
	return m.listView("Models", rows, "↑/↓ move · Enter select · Esc back")
}

func (m *Model) viewAgents() string {
	rows := make([]string, 0, len(m.agents))
	for _, a := range m.agents {
		mark := " "
		if a.ID == m.curAgent {
			mark = "●"
		}
		rows = append(rows, fmt.Sprintf("%s %s %-14s %s", mark, a.Emoji, a.ID, stFaint.Render(a.Tagline)))
	}
	return m.listView("Agents", rows, "↑/↓ move · Enter select · Esc back")
}

func (m *Model) viewSessions() string {
	rows := make([]string, 0, len(m.sessions))
	for _, s := range m.sessions {
		title := s.Title
		if len(title) > 40 {
			title = title[:40] + "…"
		}
		rows = append(rows, fmt.Sprintf("%-42s %s", title, stFaint.Render(fmt.Sprintf("%d msgs", s.Count))))
	}
	if len(rows) == 0 {
		rows = append(rows, stMuted.Render("No saved sessions yet."))
	}
	return m.listView("Sessions", rows, "↑/↓ move · Enter open · Esc back")
}

func (m *Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(stHeader.Width(m.width).Render(" 🛡️  vajra  ·  Help"))
	b.WriteString("\n\n")
	b.WriteString(stTitle.Render("  Commands") + "\n")
	for _, c := range [][2]string{
		{"/model [name]", "switch model (no name opens the picker)"},
		{"/agent [id]", "switch agent persona"},
		{"/models /agents /sessions", "open the corresponding browser"},
		{"/verify", "toggle the verification pipeline"},
		{"/swarm (or /team)", "toggle the multi-agent swarm"},
		{"/bus", "read the agent-to-agent message bus"},
		{"/attach <path>", "attach a local file to the next prompt"},
		{"/detach", "clear attached files"},
		{"/rag <folder>", "search a local folder (indexes it first time)"},
		{"/rag build <folder>", "re-index a folder"},
		{"/rag off", "stop retrieving from a folder"},
		{"/export [md|json]", "write this session to ~/.vajra/exports"},
		{"/pull <model>", "download a model from Ollama"},
		{"/new", "start a fresh session"},
		{"/clear", "clear the current view"},
		{"/exit", "quit Vajra"},
	} {
		b.WriteString(fmt.Sprintf("  %-24s %s\n", stKey.Render(c[0]), stMuted.Render(c[1])))
	}
	b.WriteString("\n")
	b.WriteString(stTitle.Render("  Local context") + "\n")
	for _, c := range [][2]string{
		{"@path/to/file", "attach that file by mentioning it in a prompt"},
		{"/rag <folder>", "passages are retrieved per prompt and cited"},
		{"headroom", "older turns are trimmed to fit the context window"},
	} {
		b.WriteString(fmt.Sprintf("  %-24s %s\n", stKey.Render(c[0]), stMuted.Render(c[1])))
	}
	b.WriteString("\n")
	b.WriteString(stTitle.Render("  Swarm mode") + "\n")
	for _, c := range [][2]string{
		{"Ctrl+T, then ask", "chief plans, specialists work in parallel"},
		{"🔬 💻 ⚙️ 🔍 blocks", "each agent streams its own answer"},
		{"✉ handoffs", "agents ask each other for follow-ups"},
		{"Ctrl+B", "read every message they exchanged"},
	} {
		b.WriteString(fmt.Sprintf("  %-24s %s\n", stKey.Render(c[0]), stMuted.Render(c[1])))
	}
	b.WriteString("\n")
	b.WriteString(stTitle.Render("  Keys") + "\n")
	for _, c := range [][2]string{
		{"Ctrl+H / F1", "this help"},
		{"Ctrl+M", "models"},
		{"Ctrl+A", "agents"},
		{"Ctrl+L", "sessions"},
		{"Ctrl+B", "agent bus"},
		{"Ctrl+V", "toggle verify"},
		{"Ctrl+T", "toggle swarm mode"},
		{"Ctrl+E", "export this session (Markdown)"},
		{"Ctrl+N", "new session"},
		{"Ctrl+U", "clear view"},
		{"↑/↓ PgUp/PgDn", "scroll"},
		{"Ctrl+Q / Ctrl+C", "quit"},
	} {
		b.WriteString(fmt.Sprintf("  %-24s %s\n", stKey.Render(c[0]), stMuted.Render(c[1])))
	}
	b.WriteString("\n")
	b.WriteString(stFaint.Render("  Enter or Esc to go back"))
	return b.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ctxMeter shows how much of the model's context window the visible
// conversation is using, so a long chat is never a surprise.
func (m *Model) ctxMeter() string {
	max := m.cfg.MaxContext
	used := 0
	for _, msg := range m.messages {
		used += llm.EstimateMessageTokens(llm.Message{Content: msg.Content})
	}
	if max <= 0 {
		return humanTokens(used)
	}
	return fmt.Sprintf("%s/%s", humanTokens(used), humanTokens(max))
}

// contextBadges shows the local-context extras that are switched on.
func (m *Model) contextBadges() string {
	var bits []string
	if len(m.attached) > 0 {
		bits = append(bits, fmt.Sprintf("📎%d", len(m.attached)))
	}
	if m.ragDir != "" {
		bits = append(bits, fmt.Sprintf("📚%d", m.ragChunks))
	}
	return strings.Join(bits, " ")
}

func humanTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}
