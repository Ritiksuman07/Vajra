package agent

import "testing"

func TestGetRole(t *testing.T) {
	if _, ok := Get("coder"); !ok {
		t.Error("expected the coder role")
	}
	if _, ok := Get("CODER"); !ok {
		t.Error("role lookup should be case-insensitive")
	}
	if _, ok := Get("nope"); ok {
		t.Error("unexpected role")
	}
}

func TestAllRolesHavePrompts(t *testing.T) {
	roles := All()
	if len(roles) < 5 {
		t.Fatalf("expected at least 5 roles, got %d", len(roles))
	}
	for _, r := range roles {
		if r.ID == "" || r.Name == "" || r.System == "" {
			t.Errorf("role %q is incomplete", r.ID)
		}
	}
}

func TestParseTasks(t *testing.T) {
	raw := "Sure! Here is the plan:\n```json\n" +
		`[{"role":"researcher","task":"gather facts"},{"role":"coder","task":"write code"},{"role":"bogus","task":"x"}]` +
		"\n```"
	tasks := parseTasks(raw)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2 (unknown roles are dropped)", len(tasks))
	}
	if tasks[0].Role != "researcher" {
		t.Errorf("role = %q", tasks[0].Role)
	}
	if tasks[0].ID == "" {
		t.Error("task should get an id")
	}
}

func TestParseTasksInvalid(t *testing.T) {
	if tasks := parseTasks("no json here"); tasks != nil {
		t.Errorf("expected nil, got %v", tasks)
	}
}

func TestEnsureDiversityFillsToMin(t *testing.T) {
	// A planner that returns a single task must still yield a swarm.
	opts := SwarmOptions{}.withDefaults()
	tasks := ensureDiversity([]SwarmTask{{Role: "coder", Task: "build it"}}, opts)
	if len(tasks) < opts.MinAgents {
		t.Fatalf("got %d agents, want >= %d", len(tasks), opts.MinAgents)
	}
	seen := map[string]bool{}
	for _, tk := range tasks {
		if seen[tk.Role] {
			t.Errorf("duplicate role %q", tk.Role)
		}
		seen[tk.Role] = true
		if tk.Task == "" {
			t.Errorf("role %q has an empty task", tk.Role)
		}
	}
}

func TestEnsureDiversityCapsAtMax(t *testing.T) {
	opts := SwarmOptions{MinAgents: 2, MaxAgents: 2}.withDefaults()
	tasks := ensureDiversity([]SwarmTask{
		{Role: "coder", Task: "a"},
		{Role: "researcher", Task: "b"},
		{Role: "executor", Task: "c"},
	}, opts)
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tasks))
	}
}

func TestParseHandoffs(t *testing.T) {
	text := "Here is my bit.\n" +
		"@coder: please refactor the parser to handle empty input\n" +
		"@researcher: find the spec section on retries\n" +
		"@coder: duplicate should be ignored\n" +
		"@nobody: unknown role\n"
	hs := parseHandoffs("executor", text)
	if len(hs) != 2 {
		t.Fatalf("handoffs = %d, want 2 (%v)", len(hs), hs)
	}
	if hs[0].To != "coder" || hs[0].From != "executor" {
		t.Errorf("unexpected first handoff: %+v", hs[0])
	}
	if hs[1].To != "researcher" {
		t.Errorf("unexpected second handoff: %+v", hs[1])
	}
}

func TestParseHandoffsIgnoresSelf(t *testing.T) {
	if hs := parseHandoffs("coder", "@coder: talk to myself"); len(hs) != 0 {
		t.Errorf("expected no self-handoffs, got %v", hs)
	}
}
