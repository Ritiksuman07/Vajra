package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/llm"
)

// fakeOllama stands in for a local Ollama server. It answers the plan prompt
// with a two-agent JSON plan (so the swarm must top it up to three) and
// returns role-specific text for everything else.
type fakeOllama struct {
	mu       sync.Mutex
	requests int
}

func (f *fakeOllama) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		system, user := "", ""
		for _, m := range body.Messages {
			switch m.Role {
			case "system":
				system = m.Content
			case "user":
				user = m.Content
			}
		}

		reply := "generic reply"
		switch {
		case strings.Contains(user, "Reply with ONLY a JSON array"):
			reply = `[{"role":"researcher","task":"gather facts"},{"role":"coder","task":"write code"}]`
		case strings.Contains(user, "Critique it"):
			reply = "Verdict: pass. One gap: no tests."
		case strings.Contains(system, "Chief of Staff"):
			reply = "Final synthesised answer."
		case strings.Contains(system, "software engineer"):
			reply = "```go\nfunc main() {}\n```"
		case strings.Contains(system, "operations assistant"):
			// Request a handoff to the coder to exercise agent-to-agent messaging.
			reply = "Run it with go run .\n@coder: also add tests for the parser"
		case strings.Contains(system, "careful researcher"):
			reply = "Summary: the facts."
		case strings.Contains(system, "rigorous reviewer"):
			reply = "Looks fine to me."
		}

		f.mu.Lock()
		f.requests++
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		// Stream in two pieces to exercise the token path.
		half := len(reply) / 2
		_ = enc.Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": reply[:half]}})
		_ = enc.Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": reply[half:]}})
		_ = enc.Encode(map[string]any{"done": true, "eval_count": 7})
	}
}

func newTestRunner(t *testing.T, host string) *Runner {
	t.Helper()
	cfg := config.Default()
	cfg.OllamaHost = host
	cfg.Providers[0].BaseURL = host
	return NewRunner(llm.NewRouter(cfg))
}

func TestRunSwarmProducesBusAndHandoffs(t *testing.T) {
	fake := &fakeOllama{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	runner := newTestRunner(t, srv.URL)

	var (
		mu     sync.Mutex
		events []SwarmEvent
	)
	emit := func(e SwarmEvent) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}

	res, err := runner.RunSwarm(context.Background(), "build a thing", SwarmOptions{Handoffs: 1}, emit)
	if err != nil {
		t.Fatalf("RunSwarm: %v", err)
	}

	// A one-line planner must still produce a swarm of at least three agents.
	if len(res.Plan) < 3 {
		t.Fatalf("plan has %d agents, want >= 3: %+v", len(res.Plan), res.Plan)
	}
	roles := map[string]bool{}
	for _, tk := range res.Plan {
		roles[tk.Role] = true
	}
	if !roles["executor"] {
		t.Errorf("expected the diversity top-up to add the executor: %+v", res.Plan)
	}

	kinds := map[string]int{}
	for _, m := range res.Bus {
		kinds[m.Kind]++
	}
	if kinds["plan"] != 1 {
		t.Errorf("plan messages = %d, want 1", kinds["plan"])
	}
	if kinds["reply"] < 4 {
		t.Errorf("reply messages = %d, want >= 4 (3 agents + a handoff reply)", kinds["reply"])
	}
	if kinds["critique"] != 1 {
		t.Errorf("critique messages = %d, want 1", kinds["critique"])
	}
	if kinds["handoff"] != 1 {
		t.Errorf("handoff messages = %d, want 1", kinds["handoff"])
	}
	if kinds["synthesis"] != 1 {
		t.Errorf("synthesis messages = %d, want 1", kinds["synthesis"])
	}

	// The handoff must be attributed correctly and answered.
	var handoff, answer *BusMessage
	for i := range res.Bus {
		m := &res.Bus[i]
		if m.Kind == "handoff" {
			handoff = m
		}
		if m.Kind == "reply" && m.From == "coder" && m.To == "executor" {
			answer = m
		}
	}
	if handoff == nil || handoff.From != "executor" || handoff.To != "coder" {
		t.Fatalf("unexpected handoff: %+v", handoff)
	}
	if answer == nil {
		t.Fatal("the handoff was never answered by the coder")
	}
	if !strings.Contains(answer.Text, "func main") {
		t.Errorf("handoff reply looks wrong: %q", answer.Text)
	}

	if res.Final != "Final synthesised answer." {
		t.Errorf("final = %q", res.Final)
	}
	if res.Critique == "" {
		t.Error("expected a reviewer critique")
	}

	// The event stream must describe the whole run for the UIs.
	types := map[string]bool{}
	var streamed strings.Builder
	for _, e := range events {
		types[e.Type] = true
		if e.Type == "token" {
			streamed.WriteString(e.Text)
		}
	}
	for _, want := range []string{"plan", "round", "agent_start", "token", "agent_done", "bus", "done"} {
		if !types[want] {
			t.Errorf("missing event type %q (have %v)", want, types)
		}
	}
	if !strings.Contains(streamed.String(), "func main") {
		t.Error("expected agent output to be streamed token by token")
	}
}

func TestRunSwarmEventsCarryRoles(t *testing.T) {
	fake := &fakeOllama{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	runner := newTestRunner(t, srv.URL)

	var (
		mu     sync.Mutex
		starts = map[string]bool{}
	)
	_, err := runner.RunSwarm(context.Background(), "goal", SwarmOptions{DisableReview: true, Handoffs: 0}, func(e SwarmEvent) {
		if e.Type == "agent_start" {
			mu.Lock()
			starts[e.Role] = true
			mu.Unlock()
		}
	})
	if err != nil {
		t.Fatalf("RunSwarm: %v", err)
	}
	if !starts["researcher"] || !starts["coder"] || !starts["executor"] {
		t.Errorf("expected start events for every agent, got %v", starts)
	}
}

func TestRunTeamAliasReturnsFinal(t *testing.T) {
	fake := &fakeOllama{}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	runner := newTestRunner(t, srv.URL)
	final, err := runner.RunTeam(context.Background(), "goal", nil)
	if err != nil {
		t.Fatalf("RunTeam: %v", err)
	}
	if final != "Final synthesised answer." {
		t.Errorf("final = %q", final)
	}
}
