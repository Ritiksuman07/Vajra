package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ── Swarm model ──────────────────────────────────────────────────────────────
//
// A "swarm" is a set of specialist agents that work on the same goal, talk to
// each other over a shared message bus, and hand work off between themselves.
//
//	user ──▶ chief (plans)
//	          ├─▶ researcher ─┐
//	          ├─▶ coder ──────┤  work in parallel, each posts a reply to the bus
//	          └─▶ executor ───┘
//	                │
//	          reviewer ──▶ critique (posts to the bus)
//	                │
//	          handoffs (@role: …) ──▶ agents ask each other for follow-ups
//	                │
//	          chief ──▶ synthesis ──▶ user
//
// Every message between agents is emitted as an event, so both the terminal UI
// and the web UI can show the agents talking to one another.

// SwarmTask is one unit of delegated work.
type SwarmTask struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Task string `json:"task"`
}

// BusMessage is one message on the shared agent bus.
type BusMessage struct {
	ID    string `json:"id"`
	From  string `json:"from"` // role id, or "chief"/"user"
	To    string `json:"to"`   // role id, or "all"
	Kind  string `json:"kind"` // plan|task|reply|critique|handoff|synthesis|error
	Text  string `json:"text"`
	Round int    `json:"round"`
	At    int64  `json:"at"`
}

// SwarmEvent is streamed to callers while a swarm runs.
type SwarmEvent struct {
	Type   string      `json:"type"`            // plan|agent_start|token|agent_done|bus|round|done|error
	Role   string      `json:"role,omitempty"`  // agent a token belongs to
	From   string      `json:"from,omitempty"`  //
	To     string      `json:"to,omitempty"`    //
	Kind   string      `json:"kind,omitempty"`  //
	Text   string      `json:"text,omitempty"`  //
	Round  int         `json:"round,omitempty"` //
	Msg    *BusMessage `json:"msg,omitempty"`   //
	Plan   []SwarmTask `json:"plan,omitempty"`  //
	Agents []string    `json:"agents,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// SwarmOptions tunes a swarm run.
type SwarmOptions struct {
	MinAgents  int  // minimum distinct agents (default 3)
	MaxAgents  int  // maximum distinct agents (default 4)
	Parallel   bool // run the work round concurrently (default true)
	Concurrent int  // how many agents run at once (default 3)
	Handoffs   int  // max handoff depth between agents (default 1)
	// DisableReview skips the reviewer critique round. Review is on by
	// default; set this to true to turn it off.
	DisableReview bool
	ReviewRole    string // role used for the critique (default "reviewer")
}

func (o SwarmOptions) withDefaults() SwarmOptions {
	if o.MinAgents <= 0 {
		o.MinAgents = 3
	}
	if o.MaxAgents <= 0 {
		o.MaxAgents = 4
	}
	if o.MaxAgents < o.MinAgents {
		o.MaxAgents = o.MinAgents
	}
	if o.Concurrent <= 0 {
		o.Concurrent = 3
	}
	if o.ReviewRole == "" {
		o.ReviewRole = "reviewer"
	}
	return o
}

// SwarmResult is the outcome of a swarm run.
type SwarmResult struct {
	Plan     []SwarmTask
	Bus      []BusMessage
	Critique string
	Final    string
}

// Emitter receives swarm events. It must be safe for concurrent use; the
// runner serialises calls so implementations do not need their own lock.
type Emitter func(SwarmEvent)

// ── Planning ─────────────────────────────────────────────────────────────────

var (
	jsonArrayRe = regexp.MustCompile(`(?s)\[.*\]`)
	mentionRe   = regexp.MustCompile(`@([a-zA-Z][a-zA-Z0-9_-]*)\s*[:>\-]\s*(.+)`)
)

// swarmRotation is the order specialists are introduced when the planner does
// not name enough of them.
var swarmRotation = []string{"researcher", "coder", "executor", "reviewer"}

// defaultTaskFor is the fallback brief for each specialist.
var defaultTaskFor = map[string]string{
	"researcher": "Research the facts, constraints and prior art needed to achieve the goal. Summarise concisely and flag anything uncertain.",
	"coder":      "Produce the concrete implementation — code, file structure and configuration — needed to achieve the goal.",
	"executor":   "Provide the exact, copy-pasteable commands and steps to run, test and verify the result of the goal.",
	"reviewer":   "Review the work for correctness, security, edge cases and gaps, and list specific fixes.",
	"chief":      "Assemble a single coherent deliverable that achieves the goal.",
}

// PlanSwarm asks the Chief to decompose a goal, then guarantees a minimum
// number of distinct specialists so the swarm is never a lone agent.
func (r *Runner) PlanSwarm(ctx context.Context, goal string, opts SwarmOptions) ([]SwarmTask, error) {
	opts = opts.withDefaults()

	prompt := fmt.Sprintf(`You are the Chief of Staff coordinating a small team of specialist agents.
Goal: %s

Choose between %d and %d specialists from: researcher, coder, executor, reviewer.
Give every specialist a concrete, non-overlapping task.
Reply with ONLY a JSON array, each element being {"role":"<role>","task":"<task>"}.`,
		goal, opts.MinAgents, opts.MaxAgents)

	resp, err := r.RunModel(ctx, "chief", r.router.Config().Model, nil, prompt, nil)
	tasks := []SwarmTask{}
	if err == nil && resp != nil {
		tasks = parseTasks(resp.Content)
	}
	return ensureDiversity(tasks, opts), nil
}

// parseTasks extracts a JSON array of tasks from model output, tolerating
// prose and code fences around it.
func parseTasks(s string) []SwarmTask {
	m := jsonArrayRe.FindString(s)
	if m == "" {
		return nil
	}
	var raw []struct {
		Role string `json:"role"`
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(m), &raw); err != nil {
		return nil
	}
	out := make([]SwarmTask, 0, len(raw))
	for _, t := range raw {
		t.Role = strings.ToLower(strings.TrimSpace(t.Role))
		t.Task = strings.TrimSpace(t.Task)
		if t.Task == "" {
			continue
		}
		if _, ok := Get(t.Role); !ok {
			continue
		}
		out = append(out, SwarmTask{ID: fmt.Sprintf("t%d", len(out)+1), Role: t.Role, Task: t.Task})
	}
	return out
}

// ensureDiversity keeps at most one task per role, then tops the plan up with
// fallback specialists until it reaches MinAgents distinct agents.
func ensureDiversity(tasks []SwarmTask, opts SwarmOptions) []SwarmTask {
	seen := map[string]bool{}
	out := make([]SwarmTask, 0, opts.MaxAgents)
	for _, t := range tasks {
		if seen[t.Role] || len(out) >= opts.MaxAgents {
			continue
		}
		seen[t.Role] = true
		t.ID = fmt.Sprintf("t%d", len(out)+1)
		out = append(out, t)
	}
	for _, role := range swarmRotation {
		if len(out) >= opts.MinAgents || len(out) >= opts.MaxAgents {
			break
		}
		if seen[role] {
			continue
		}
		seen[role] = true
		out = append(out, SwarmTask{
			ID:   fmt.Sprintf("t%d", len(out)+1),
			Role: role,
			Task: defaultTaskFor[role],
		})
	}
	return out
}

// ── Handoffs ─────────────────────────────────────────────────────────────────

// handoff is an explicit request from one agent to another, written as
// "@coder: please refactor the parser".
type handoff struct {
	From string
	To   string
	Task string
}

// parseHandoffs finds @role: message directives addressed to another agent.
func parseHandoffs(from, text string) []handoff {
	var out []handoff
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		for _, m := range mentionRe.FindAllStringSubmatch(line, -1) {
			target := strings.ToLower(m[1])
			if target == from || target == "user" || target == "all" || seen[target] {
				continue
			}
			if _, ok := Get(target); !ok {
				continue
			}
			task := strings.TrimSpace(m[2])
			if task == "" {
				continue
			}
			seen[target] = true
			out = append(out, handoff{From: from, To: target, Task: task})
		}
	}
	return out
}

// ── Runner ───────────────────────────────────────────────────────────────────

type swarm struct {
	r      *Runner
	goal   string
	opts   SwarmOptions
	emit   Emitter
	mu     sync.Mutex
	bus    []BusMessage
	seq    int
	result *SwarmResult
}

func (s *swarm) fire(ev SwarmEvent) {
	if s.emit == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emit(ev)
}

// post appends a message to the bus and announces it.
func (s *swarm) post(from, to, kind string, round int, text string) *BusMessage {
	s.mu.Lock()
	s.seq++
	msg := BusMessage{
		ID:    fmt.Sprintf("m%d", s.seq),
		From:  from,
		To:    to,
		Kind:  kind,
		Text:  text,
		Round: round,
		At:    time.Now().Unix(),
	}
	s.bus = append(s.bus, msg)
	s.mu.Unlock()

	s.fire(SwarmEvent{
		Type: "bus", From: from, To: to, Kind: kind, Text: text, Round: round, Msg: &msg,
	})
	return &msg
}

// transcript renders the bus as plain text so agents can read each other.
func (s *swarm) transcript(limit int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for _, m := range s.bus {
		if m.Kind == "plan" {
			continue
		}
		fmt.Fprintf(&b, "[%s → %s] %s\n", m.From, m.To, truncate(m.Text, limit))
	}
	return strings.TrimSpace(b.String())
}

func (s *swarm) snapshot() []BusMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]BusMessage, len(s.bus))
	copy(out, s.bus)
	return out
}

// runAgent executes one specialist, streaming its output token by token.
func (s *swarm) runAgent(ctx context.Context, role, task string, round int, busKind string, to string) (string, error) {
	s.fire(SwarmEvent{Type: "agent_start", Role: role, To: to, Kind: busKind, Text: task, Round: round})

	var sb strings.Builder
	var mu sync.Mutex
	resp, err := s.r.RunModel(ctx, role, s.r.router.Config().Model, nil, task, func(tok string) error {
		mu.Lock()
		sb.WriteString(tok)
		mu.Unlock()
		s.fire(SwarmEvent{Type: "token", Role: role, Text: tok, Round: round})
		return nil
	})

	text := ""
	if resp != nil {
		text = resp.Content
	}
	if strings.TrimSpace(text) == "" {
		text = sb.String()
	}
	if err != nil {
		s.fire(SwarmEvent{Type: "agent_done", Role: role, Text: text, Round: round, Error: err.Error()})
		return text, err
	}

	s.fire(SwarmEvent{Type: "agent_done", Role: role, Text: text, Round: round})
	s.post(role, to, busKind, round, text)
	return text, nil
}

// RunSwarm plans a goal, runs the specialists, lets them message each other,
// has the reviewer critique the result, and finishes with a Chief synthesis.
func (r *Runner) RunSwarm(ctx context.Context, goal string, opts SwarmOptions, emit Emitter) (*SwarmResult, error) {
	opts = opts.withDefaults()
	s := &swarm{
		r:      r,
		goal:   goal,
		opts:   opts,
		emit:   emit,
		result: &SwarmResult{},
	}

	// ── Round 0: plan ────────────────────────────────────────────────────────
	plan, err := r.PlanSwarm(ctx, goal, opts)
	if err != nil || len(plan) == 0 {
		plan = ensureDiversity(nil, opts)
	}
	s.result.Plan = plan

	agents := make([]string, 0, len(plan))
	for _, t := range plan {
		agents = append(agents, t.Role)
	}
	s.fire(SwarmEvent{Type: "plan", Plan: plan, Agents: agents, Text: formatPlan(plan)})
	s.post("chief", "all", "plan", 0, "Goal: "+goal+"\n"+formatPlan(plan))

	// ── Round 1: parallel work ───────────────────────────────────────────────
	s.fire(SwarmEvent{Type: "round", Round: 1, Text: fmt.Sprintf("round 1 — %d agents working in parallel", len(plan))})

	sem := make(chan struct{}, opts.Concurrent)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for _, t := range plan {
		wg.Add(1)
		go func(t SwarmTask) {
			defer wg.Done()
			if opts.Parallel {
				sem <- struct{}{}
				defer func() { <-sem }()
			}
			prompt := fmt.Sprintf(`You are the %s on a small agent team.
Goal: %s
Your assignment: %s

Work so far on the bus:
%s

Do your part. Be concrete and concise. Use Markdown.
If you need another specialist to do something, ask them on one line as "@<role>: <request>" (roles: researcher, coder, executor, reviewer).`,
				t.Role, goal, t.Task, orNone(s.transcript(1200)))

			if _, err := s.runAgent(ctx, t.Role, prompt, 1, "reply", "all"); err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
		}(t)
	}
	wg.Wait()

	// ── Round 1.5: peer review ───────────────────────────────────────────────
	if !opts.DisableReview {
		s.fire(SwarmEvent{Type: "round", Round: 2, Text: "round 2 — the reviewer critiques the team's work"})
		review := fmt.Sprintf(`You are the reviewer on a small agent team.
Goal: %s

Here is what your teammates produced:
%s

Critique it: call out errors, gaps, security issues and anything unverified.
Finish with a short list of concrete fixes. Address teammates directly with
"@<role>: <request>" when you want them to change something.`,
			goal, orNone(s.transcript(3000)))

		if text, err := s.runAgent(ctx, opts.ReviewRole, review, 2, "critique", "all"); err == nil {
			s.result.Critique = text
		}
	}

	// ── Round 3+: handoffs ───────────────────────────────────────────────────
	if opts.Handoffs > 0 {
		done := map[string]bool{}
		for round := 2; round <= 1+opts.Handoffs; round++ {
			var pending []handoff
			for _, m := range s.snapshot() {
				if m.Kind != "reply" && m.Kind != "critique" {
					continue
				}
				for _, h := range parseHandoffs(m.From, m.Text) {
					key := h.From + "→" + h.To + ":" + h.Task
					if done[key] || len(pending) >= 3 {
						continue
					}
					done[key] = true
					pending = append(pending, h)
				}
			}
			if len(pending) == 0 {
				break
			}
			s.fire(SwarmEvent{Type: "round", Round: round + 1, Text: fmt.Sprintf("round %d — %d agent-to-agent handoff(s)", round+1, len(pending))})
			for _, h := range pending {
				s.post(h.From, h.To, "handoff", round+1, h.Task)
				prompt := fmt.Sprintf(`A teammate asked you for help.
Goal: %s
Request from %s: %s

Context on the bus:
%s

Respond directly and concretely.`, goal, h.From, h.Task, orNone(s.transcript(1500)))
				_, _ = s.runAgent(ctx, h.To, prompt, round+1, "reply", h.From)
			}
		}
	}

	// ── Final: synthesis ─────────────────────────────────────────────────────
	s.fire(SwarmEvent{Type: "round", Round: 9, Text: "chief is synthesising the final answer"})
	final := fmt.Sprintf(`You are the Chief of Staff. The team has finished.
Goal: %s

Team output:
%s

Reviewer critique:
%s

Write the final answer for the user: concise, well-structured Markdown,
incorporating the fixes the reviewer asked for.`, goal, orNone(s.transcript(6000)), orNone(s.result.Critique))

	s.fire(SwarmEvent{Type: "agent_start", Role: "chief", To: "user", Kind: "synthesis",
		Text: "assembling the final answer", Round: 9})

	resp, err := s.r.RunModel(ctx, "chief", s.r.router.Config().Model, nil, final, func(tok string) error {
		s.fire(SwarmEvent{Type: "token", Role: "chief", Text: tok, Round: 9})
		return nil
	})
	if resp != nil {
		s.result.Final = resp.Content
	}
	if err != nil {
		s.fire(SwarmEvent{Type: "agent_done", Role: "chief", Round: 9, Error: err.Error()})
		s.fire(SwarmEvent{Type: "error", Role: "chief", Error: err.Error()})
		return s.result, err
	}
	s.fire(SwarmEvent{Type: "agent_done", Role: "chief", Text: s.result.Final, Round: 9})
	s.post("chief", "user", "synthesis", 9, s.result.Final)
	s.result.Bus = s.snapshot()
	s.fire(SwarmEvent{Type: "done", Text: s.result.Final, Role: "chief"})

	if firstErr != nil {
		return s.result, firstErr
	}
	return s.result, nil
}

// RunTeam is a backwards-compatible alias for RunSwarm.
func (r *Runner) RunTeam(ctx context.Context, goal string, emit Emitter) (string, error) {
	res, err := r.RunSwarm(ctx, goal, SwarmOptions{}, emit)
	if res == nil {
		return "", err
	}
	return res.Final, err
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func formatPlan(tasks []SwarmTask) string {
	var b strings.Builder
	for i, t := range tasks {
		role := t.Role
		if rr, ok := Get(t.Role); ok {
			role = rr.Emoji + " " + rr.Name
		}
		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, role, t.Task)
	}
	return strings.TrimSpace(b.String())
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(nothing yet)"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
