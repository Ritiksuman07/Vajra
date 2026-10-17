package tui

import (
	"strings"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/agent"
)

// TestApplySwarmBuildsTranscript walks a realistic swarm event stream through
// the TUI state and checks that the transcript and the agent bus both come out
// readable.
func TestApplySwarmBuildsTranscript(t *testing.T) {
	m := &Model{swarmOpen: map[string]int{}}

	m.applySwarm(agent.SwarmEvent{
		Type:   "plan",
		Agents: []string{"researcher", "coder", "executor"},
		Text:   "1. researcher — gather facts\n2. coder — write code",
	})
	m.applySwarm(agent.SwarmEvent{Type: "round", Round: 1, Text: "round 1 — 3 agents working in parallel"})

	// Two agents stream in parallel; tokens must be routed by role.
	m.applySwarm(agent.SwarmEvent{Type: "agent_start", Role: "researcher", To: "all", Kind: "reply"})
	m.applySwarm(agent.SwarmEvent{Type: "agent_start", Role: "coder", To: "all", Kind: "reply"})
	m.applySwarm(agent.SwarmEvent{Type: "token", Role: "coder", Text: "```go"})
	m.applySwarm(agent.SwarmEvent{Type: "token", Role: "researcher", Text: "Summary"})
	m.applySwarm(agent.SwarmEvent{Type: "token", Role: "coder", Text: "x"})
	m.applySwarm(agent.SwarmEvent{Type: "agent_done", Role: "researcher", Text: "Summary done"})
	m.applySwarm(agent.SwarmEvent{Type: "agent_done", Role: "coder", Text: "code done"})

	// The executor asks the coder for help.
	msg := &agent.BusMessage{ID: "m1", From: "executor", To: "coder", Kind: "handoff", Text: "also add tests"}
	m.applySwarm(agent.SwarmEvent{Type: "bus", From: "executor", To: "coder", Kind: "handoff", Text: msg.Text, Msg: msg})
	m.applySwarm(agent.SwarmEvent{Type: "agent_start", Role: "coder", To: "executor", Kind: "reply"})
	m.applySwarm(agent.SwarmEvent{Type: "token", Role: "coder", Text: "tests added"})
	m.applySwarm(agent.SwarmEvent{Type: "agent_done", Role: "coder", Text: "tests added"})

	if len(m.swarmOpen) != 0 {
		t.Errorf("expected no open agents, got %v", m.swarmOpen)
	}
	if len(m.swarmBus) != 1 {
		t.Fatalf("bus messages = %d, want 1", len(m.swarmBus))
	}

	// feed: plan, round, agent(researcher), agent(coder), handoff, agent(coder)
	if len(m.swarmFeed) != 6 {
		t.Fatalf("feed items = %d, want 6", len(m.swarmFeed))
	}
	if got := m.swarmFeed[2].Text.String(); got != "Summary done" {
		t.Errorf("researcher block = %q", got)
	}
	if got := m.swarmFeed[3].Text.String(); got != "code done" {
		t.Errorf("coder block = %q", got)
	}
	if got := m.swarmFeed[5].Text.String(); got != "tests added" {
		t.Errorf("handoff reply block = %q", got)
	}

	out := m.renderSwarm(72)
	for _, want := range []string{"Swarm", "plan", "Researcher", "Coder", "Executor", "add tests", "tests added"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered swarm missing %q\n%s", want, out)
		}
	}

	bus := m.renderBusBody()
	if !strings.Contains(bus, "handoff") || !strings.Contains(bus, "also add tests") {
		t.Errorf("bus view incomplete:\n%s", bus)
	}
}

// TestSwarmStatusNamesAgents checks the status bar lists who is working.
func TestSwarmStatusNamesAgents(t *testing.T) {
	m := &Model{swarmOpen: map[string]int{}}
	m.applySwarm(agent.SwarmEvent{Type: "agent_start", Role: "researcher", To: "all", Kind: "reply"})
	s := m.swarmStatus()
	if !strings.Contains(s, "Researcher") {
		t.Errorf("status = %q, want the researcher named", s)
	}
	m.applySwarm(agent.SwarmEvent{Type: "agent_done", Role: "researcher", Text: "done"})
	if s := m.swarmStatus(); !strings.Contains(s, "swarm working") {
		t.Errorf("status after finish = %q", s)
	}
}
