# Agents

Vajra is honest about what an "agent" is here: a **system prompt and a
preferred model**, executed by the same binary. There is no separate
container or process per agent. That keeps Vajra fast, debuggable and
trivially auditable — you can read every persona in
[`internal/agent/roles.go`](../internal/agent/roles.go).

## The team

| ID | Name | Tagline | Suggested model |
|----|------|---------|-----------------|
| `assistant` | 🛡️ Assistant | General-purpose helper | `llama3.1:8b` |
| `chief` | 🧭 Chief of Staff | Plans and delegates work | `llama3.1:70b` |
| `coder` | 💻 Coder | Writes and explains code | `qwen2.5-coder:7b` |
| `researcher` | 🔬 Researcher | Analyses and summarises | `llama3.1:8b` |
| `executor` | ⚙️ Executor | Turns plans into commands | `phi3.5` |
| `reviewer` | 🔍 Reviewer | Reviews and finds problems | `llama3.1:8b` |

Model hints are suggestions; every agent works with any model you have.

## Single-agent mode

```bash
vajra run --agent coder "write a Go function that reverses a linked list"
```

In the TUI: `Ctrl+A` to pick an agent, or `/agent coder`.

## Team mode

Team mode chains several roles for tasks that benefit from planning and
review. Flow:

```
        ┌──────────────┐
goal ──▶│ Chief plans  │──▶ [step 1] [step 2] [step 3]
        └──────────────┘
                 │
        ┌────────▼─────────┐
        │ each step routed │  coder / researcher / executor / reviewer
        │ to its role      │  (each sees the work so far)
        └────────┬─────────┘
                 │
        ┌────────▼─────────┐
        │ Chief synthesises│──▶ final answer
        └──────────────────┘
```

Enable it:

```bash
vajra run --team "add tests to a small Go project and review them"
```

In the TUI: `Ctrl+T`, or `/team`.

The planner is asked for strict JSON and falls back to a single step if the
model does not cooperate, so team mode never hard-fails on a small model.

## How routing works

`llm.Router` maps a model name to a backend:

1. an exact match against a provider's configured default model, then
2. a name pattern — `claude-*` → Anthropic, `gpt-*`/`o*` → OpenAI,
   `vendor/model` → OpenRouter, then
3. **local Ollama** as the default.

Unknown models always go to Ollama, which keeps Vajra local-first even if
you add a cloud key.
