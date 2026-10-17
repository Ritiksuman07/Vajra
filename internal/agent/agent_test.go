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

func TestParseSteps(t *testing.T) {
	raw := "Sure! Here is the plan:\n```json\n" +
		`[{"role":"researcher","task":"gather facts"},{"role":"coder","task":"write code"},{"role":"bogus","task":"x"}]` +
		"\n```"
	steps := parseSteps(raw)
	if len(steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(steps))
	}
	if steps[0].Role != "researcher" {
		t.Errorf("role = %q", steps[0].Role)
	}
	// Unknown roles fall back to chief rather than being dropped.
	if steps[2].Role != "chief" {
		t.Errorf("unknown role should fall back to chief, got %q", steps[2].Role)
	}
}

func TestParseStepsInvalid(t *testing.T) {
	if steps := parseSteps("no json here"); steps != nil {
		t.Errorf("expected nil, got %v", steps)
	}
}
