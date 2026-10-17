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

## Swarm mode

Swarm mode runs a small team of specialists on one goal, **in parallel**, with
a shared message bus so they can talk to each other. Flow:

```
                    ┌────────────────┐
goal ──────────────▶│  Chief plans   │──▶ researcher ┐
                    └────────────────┘    coder      ├─ work in parallel
                                          executor   ┘  (each sees the bus)
                                                 │
                                        ┌────────▼────────┐
                                        │ Reviewer        │  posts a critique
                                        └────────┬────────┘
                                                 │
                                @role: requests ─┤  agents ask each other
                                (handoffs)       │  for follow-up work
                                                 │
                                        ┌────────▼────────┐
                                        │ Chief synthesis │──▶ final answer
                                        └─────────────────┘
```

Enable it:

```bash
vajra run --swarm "add tests to a small Go project and review them"
```

In the TUI: `Ctrl+T` toggles it, `Ctrl+B` (or `/bus`) opens the agent bus so
you can read every message the agents exchanged. In the web UI: the 🐝 toggle
next to the composer.

### Who talks to whom

Every message on the bus is shown with its sender and recipient:

```
🔬 Researcher → 👥 everyone · reply
⚙️ Executor → 👥 everyone · reply
✉ 🔍 Reviewer → 🔬 Researcher   asked for help: check the 5-7-5 syllables
🔬 Researcher → 🔍 Reviewer · reply
🧭 Chief of Staff → 👤 you · synthesis
```

Agents hand work to each other by writing `@role: request` on its own line.
The orchestrator parses that, posts it to the bus, and runs the follow-up —
bounded to one round and three requests so a chatty model cannot spiral.

### Guardrails

- **Never a lone agent.** The planner is asked for 3–4 specialists; if a small
  model returns one task, Vajra tops the roster up from a default rotation
  (`researcher`, `coder`, `executor`, `reviewer`). Swarm mode always runs at
  least three distinct agents.
- **Bounded cost.** One parallel work round, one critique round, at most three
  handoffs, then one synthesis.
- **Honest failure.** If an agent errors, the event stream carries the error;
  the run is never silently truncated.
- **The transcript is kept.** Every bus message is stored with the answer, so a
  re-opened session still shows the conversation (`Ctrl+B`, the 🐝 panel in the
  web UI, or `vajra export`).

## Local context for a swarm

A swarm is given the same local context as a single agent: attached files
(📎 / `--file` / `@path`) are prepended to the goal, and if a folder is indexed
(`/rag`, `--rag`, 📚) the closest passages are included with `file:line`
citations. The agents see the retrieved passages as part of the goal, so the
researcher can quote them and the reviewer can check the others against them.

Because the whole goal is echoed as the plan message on the bus, an exported
transcript shows exactly what the team was given — including the retrieved
context. Nothing is summarised away.

## How routing works

`llm.Router` maps a model name to a backend:

1. an exact match against a provider's configured default model, then
2. a name pattern — `claude-*` → Anthropic, `gpt-*`/`o*` → OpenAI,
   `vendor/model` → OpenRouter, then
3. **local Ollama** as the default.

Unknown models always go to Ollama, which keeps Vajra local-first even if
you add a cloud key.
