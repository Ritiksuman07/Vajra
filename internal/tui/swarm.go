package tui

import (
	"fmt"
	"strings"

	"github.com/Ritiksuman07/Vajra/internal/agent"
)

// swarmItem is one block in the live swarm transcript.
type swarmItem struct {
	Kind  string // plan | round | agent | handoff
	Role  string // for agent blocks
	From  string // for handoff blocks
	To    string // destination role ("all", "user", or a role id)
	Label string // the bus kind: reply | critique | synthesis | handoff
	Text  strings.Builder
}

// applySwarm folds a swarm event into the model's display state. It is the
// single place where the terminal UI learns who is talking to whom.
func (m *Model) applySwarm(ev agent.SwarmEvent) {
	switch ev.Type {
	case "plan":
		it := &swarmItem{Kind: "plan", Label: "Swarm plan"}
		it.Text.WriteString(ev.Text)
		m.swarmFeed = append(m.swarmFeed, it)
		m.swarmAgents = ev.Agents

	case "round":
		// A late round event for the synthesis overlaps the chief block; keep
		// the divider but drop the duplicate for the final round.
		it := &swarmItem{Kind: "round"}
		it.Text.WriteString(ev.Text)
		m.swarmFeed = append(m.swarmFeed, it)

	case "agent_start":
		it := &swarmItem{Kind: "agent", Role: ev.Role, To: ev.To, Label: ev.Kind}
		m.swarmFeed = append(m.swarmFeed, it)
		m.swarmOpen[ev.Role] = len(m.swarmFeed) - 1

	case "token":
		if i, ok := m.swarmOpen[ev.Role]; ok && i < len(m.swarmFeed) {
			m.swarmFeed[i].Text.WriteString(ev.Text)
		}
		m.acc.WriteString(ev.Text)

	case "agent_done":
		if i, ok := m.swarmOpen[ev.Role]; ok && i < len(m.swarmFeed) {
			if strings.TrimSpace(ev.Text) != "" {
				m.swarmFeed[i].Text.Reset()
				m.swarmFeed[i].Text.WriteString(ev.Text)
			}
		}
		delete(m.swarmOpen, ev.Role)
		if ev.Error != "" {
			m.err = fmt.Errorf("%s: %s", ev.Role, ev.Error)
		}

	case "bus":
		m.swarmBus = append(m.swarmBus, *msgCopy(ev.Msg))
		// Replies and critiques are already shown as agent blocks; the bus
		// screen keeps the full log. Only handoffs deserve an extra line.
		if ev.Kind == "handoff" {
			it := &swarmItem{Kind: "handoff", From: ev.From, To: ev.To, Label: ev.Kind}
			it.Text.WriteString(ev.Text)
			m.swarmFeed = append(m.swarmFeed, it)
		}

	case "error":
		if ev.Error != "" {
			m.err = fmt.Errorf("%s", ev.Error)
		}
	}
}

func msgCopy(msg *agent.BusMessage) *agent.BusMessage {
	if msg == nil {
		return &agent.BusMessage{}
	}
	cp := *msg
	return &cp
}

// roleParts returns the emoji and display name for a bus participant.
func roleParts(id string) (string, string) {
	switch id {
	case "user":
		return "👤", "You"
	case "all":
		return "👥", "everyone"
	case "":
		return "•", "?"
	}
	if r, ok := agent.Get(id); ok {
		return r.Emoji, r.Name
	}
	return "•", id
}

// ── Rendering ────────────────────────────────────────────────────────────────

// renderSwarm draws the live agent-to-agent transcript.
func (m *Model) renderSwarm(width int) string {
	var b strings.Builder
	b.WriteString(stTitle.Render("  🐝 Swarm") +
		stFaint.Render(fmt.Sprintf("  %d agents", len(m.swarmAgents))) + "\n")

	for _, it := range m.swarmFeed {
		switch it.Kind {
		case "plan":
			b.WriteString(stMuted.Render("  plan") + "\n")
			b.WriteString(renderMarkdownish(indent(it.Text.String(), "    "), width))
			b.WriteString("\n")
		case "round":
			b.WriteString(stFaint.Render("  ── "+it.Text.String()+" ──") + "\n\n")
		case "handoff":
			fromE, fromN := roleParts(it.From)
			toE, toN := roleParts(it.To)
			b.WriteString(stWarn.Render(fmt.Sprintf("  ✉ %s %s → %s %s", fromE, fromN, toE, toN)) +
				stMuted.Render("  asked for help: "+wrapInline(it.Text.String(), width-30)) + "\n\n")
		case "agent":
			emoji, name := roleParts(it.Role)
			var arrow string
			switch it.To {
			case "user":
				arrow = "→ you"
			case "all", "":
				arrow = "→ everyone"
			default:
				toE, toN := roleParts(it.To)
				arrow = fmt.Sprintf("→ %s %s", toE, toN)
			}
			b.WriteString(stTitle.Render(fmt.Sprintf("  %s %s ", emoji, name)) +
				stFaint.Render(arrow+" · "+it.Label))
			if _, open := m.swarmOpen[it.Role]; open && m.swarmRunning {
				b.WriteString("  " + m.spinner.View())
			}
			b.WriteString("\n")
			body := it.Text.String()
			if strings.TrimSpace(body) == "" {
				body = "working…"
			}
			b.WriteString(renderMarkdownish(body, width))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderBusBody is the scrollable content of the agent-bus screen.
func (m *Model) renderBusBody() string {
	width := m.vp.Width - 4
	if width < 24 {
		width = 24
	}
	if len(m.swarmBus) == 0 {
		return "\n" + stMuted.Render("  No swarm traffic yet.") + "\n\n" +
			stFaint.Render("  Turn on swarm mode with Ctrl+T, then send a message.") + "\n"
	}
	var b strings.Builder
	b.WriteString("\n")
	for _, msg := range m.swarmBus {
		fromE, fromN := roleParts(msg.From)
		toE, toN := roleParts(msg.To)
		b.WriteString(fmt.Sprintf("  %s %s %s %s %s %s\n",
			fromE, stKey.Render(fromN), stFaint.Render("→"), toE,
			stMuted.Render(toN), stWarn.Render("["+msg.Kind+"]")))
		b.WriteString(wrap("    "+msg.Text, width, stText))
		b.WriteString("\n\n")
	}
	return b.String()
}

// viewSwarmBus renders a dedicated screen listing every message agents
// exchanged, so you can read the conversation in full.
func (m *Model) viewSwarmBus() string {
	var b strings.Builder
	b.WriteString(stHeader.Width(m.width).Render(" 🐝  vajra  ·  Agent bus"))
	b.WriteString("\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n")
	b.WriteString(stFaint.Render("  ↑/↓ PgUp/PgDn scroll · Esc back · Ctrl+B toggles"))
	return b.String()
}

// indent prefixes every line of s with prefix.
func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
