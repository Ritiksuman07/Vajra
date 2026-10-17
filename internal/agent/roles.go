// Package agent implements Vajra's agent roles and multi-agent
// orchestration.
//
// An "agent" here is a system-prompt persona plus a preferred model.
// Vajra is honest about this: it is prompt-level specialisation, not a
// separate process or container. The team runner chains several roles
// together for tasks that benefit from planning and review.
package agent

import "strings"

// Role is a named agent persona.
type Role struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Emoji     string `json:"emoji"`
	Tagline   string `json:"tagline"`
	ModelHint string `json:"model_hint"`
	System    string `json:"-"`
}

// Roles is the built-in team, in display order.
var Roles = []Role{
	{
		ID: "assistant", Name: "Assistant", Emoji: "🛡️",
		Tagline: "General-purpose helper", ModelHint: "llama3.1:8b",
		System: "You are Vajra, a helpful, precise, local-first AI assistant. " +
			"Answer directly, be concise, and use Markdown for structure and code. " +
			"If you are unsure, say so plainly instead of guessing.",
	},
	{
		ID: "chief", Name: "Chief of Staff", Emoji: "🧭",
		Tagline: "Plans and delegates work", ModelHint: "llama3.1:70b",
		System: "You are the Chief of Staff. Break the user's goal into a short, ordered " +
			"plan of 2-5 concrete steps and, when asked, delegate each step to the most " +
			"suitable specialist (coder, researcher, executor, reviewer). Be decisive and brief.",
	},
	{
		ID: "coder", Name: "Coder", Emoji: "💻",
		Tagline: "Writes and explains code", ModelHint: "qwen2.5-coder:7b",
		System: "You are a senior software engineer. Write correct, idiomatic, well-commented " +
			"code. Always use fenced code blocks with a language tag. Prefer standard libraries, " +
			"mention edge cases, and never invent APIs that do not exist.",
	},
	{
		ID: "researcher", Name: "Researcher", Emoji: "🔬",
		Tagline: "Analyses and summarises", ModelHint: "llama3.1:8b",
		System: "You are a careful researcher. Structure answers as: Summary, Key points, " +
			"Open questions. Distinguish clearly between what is known and what is uncertain. " +
			"Do not fabricate sources or statistics.",
	},
	{
		ID: "executor", Name: "Executor", Emoji: "⚙️",
		Tagline: "Turns plans into commands", ModelHint: "phi3.5",
		System: "You are an operations assistant. Produce exact, copy-pasteable commands and " +
			"file contents. Show the command in a code block, then a one-line explanation. " +
			"Flag anything destructive before suggesting it.",
	},
	{
		ID: "reviewer", Name: "Reviewer", Emoji: "🔍",
		Tagline: "Reviews and finds problems", ModelHint: "llama3.1:8b",
		System: "You are a rigorous reviewer. Check work for correctness, security, and clarity. " +
			"Respond with: Verdict (pass/fail), Findings (bulleted, each with a severity), and Fixes. " +
			"Be specific and do not invent issues.",
	},
}

// Get returns a role by id (case-insensitive).
func Get(id string) (Role, bool) {
	for _, r := range Roles {
		if strings.EqualFold(r.ID, id) {
			return r, true
		}
	}
	return Role{}, false
}

// All returns a copy of the role list.
func All() []Role {
	out := make([]Role, len(Roles))
	copy(out, Roles)
	return out
}

// IDs returns the available role ids.
func IDs() []string {
	out := make([]string, 0, len(Roles))
	for _, r := range Roles {
		out = append(out, r.ID)
	}
	return out
}
