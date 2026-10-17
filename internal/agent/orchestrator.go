package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Step is one unit of delegated work.
type Step struct {
	Role string `json:"role"`
	Task string `json:"task"`
}

// Event is emitted while the team runs, so callers can render progress.
type Event struct {
	Kind string // plan, step_start, token, step_done, final, error
	Role string
	Text string
}

// Emitter receives team events.
type Emitter func(Event)

// Plan asks the Chief of Staff to decompose a goal into steps.
// It always returns at least one step.
func (r *Runner) Plan(ctx context.Context, task string) ([]Step, error) {
	instruction := fmt.Sprintf(`Decompose the goal below into 2-4 concrete steps.
Reply with ONLY a JSON array. Each element: {"role":"<role>","task":"<what to do>"}.
Valid roles: chief, coder, researcher, executor, reviewer.
Goal: %s`, task)

	resp, err := r.RunModel(ctx, "chief", r.router.Config().Model, nil, instruction, nil)
	if err != nil {
		return []Step{{Role: "chief", Task: task}}, nil
	}
	steps := parseSteps(resp.Content)
	if len(steps) == 0 {
		return []Step{{Role: "chief", Task: task}}, nil
	}
	return steps, nil
}

var jsonArrayRe = regexp.MustCompile(`(?s)\[.*\]`)

func parseSteps(s string) []Step {
	m := jsonArrayRe.FindString(s)
	if m == "" {
		return nil
	}
	var steps []Step
	if err := json.Unmarshal([]byte(m), &steps); err != nil {
		return nil
	}
	out := make([]Step, 0, len(steps))
	for _, st := range steps {
		st.Task = strings.TrimSpace(st.Task)
		if st.Task == "" {
			continue
		}
		if _, ok := Get(st.Role); !ok {
			st.Role = "chief"
		}
		out = append(out, st)
	}
	return out
}

// RunTeam plans a task and executes the steps sequentially, feeding each
// result into the next. It finishes with a synthesis from the Chief.
// Progress is reported through emit (which may be nil).
func (r *Runner) RunTeam(ctx context.Context, task string, emit Emitter) (string, error) {
	fire := func(e Event) {
		if emit != nil {
			emit(e)
		}
	}

	steps, err := r.Plan(ctx, task)
	if err != nil {
		return "", err
	}
	fire(Event{Kind: "plan", Text: formatPlan(steps)})

	var transcript strings.Builder
	for i, st := range steps {
		fire(Event{Kind: "step_start", Role: st.Role, Text: st.Task})

		prompt := st.Task
		if transcript.Len() > 0 {
			prompt = fmt.Sprintf("Goal: %s\n\nWork so far:\n%s\n\nYour step: %s",
				task, truncate(transcript.String(), 4000), st.Task)
		}

		resp, err := r.RunModel(ctx, st.Role, r.router.Config().Model, nil, prompt, func(tok string) error {
			fire(Event{Kind: "token", Role: st.Role, Text: tok})
			return nil
		})
		if err != nil {
			fire(Event{Kind: "error", Role: st.Role, Text: err.Error()})
			return transcript.String(), err
		}
		transcript.WriteString(fmt.Sprintf("\n### Step %d — %s\n%s\n", i+1, st.Role, resp.Content))
		fire(Event{Kind: "step_done", Role: st.Role, Text: resp.Content})
	}

	// Final synthesis.
	final, err := r.RunModel(ctx, "chief", r.router.Config().Model, nil,
		fmt.Sprintf("Original goal: %s\n\nWork completed:\n%s\n\nWrite the final answer for the user. Be concise and use Markdown.",
			task, truncate(transcript.String(), 6000)),
		func(tok string) error {
			fire(Event{Kind: "final", Text: tok})
			return nil
		})
	if err != nil {
		return transcript.String(), err
	}
	fire(Event{Kind: "final", Text: final.Content})
	return transcript.String(), nil
}

func formatPlan(steps []Step) string {
	var b strings.Builder
	for i, s := range steps {
		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, s.Role, s.Task)
	}
	return strings.TrimSpace(b.String())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
