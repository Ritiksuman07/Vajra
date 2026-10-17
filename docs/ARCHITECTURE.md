# Architecture

Vajra is a single Go program, layered so each package has one job.

```
                         ┌───────────────────────────────┐
                         │            main.go            │
                         │   CLI: run · serve · web · …  │
                         └───────────────┬───────────────┘
             ┌───────────────────────────┼───────────────────────────┐
             ▼                           ▼                           ▼
     ┌───────────────┐          ┌────────────────┐          ┌────────────────┐
     │ internal/tui  │          │ internal/server│          │  internal/agent│
     │  Bubble Tea   │          │  REST + SSE    │          │ roles + swarm   │
     └───────┬───────┘          │  + embedded UI │          └───────┬────────┘
             │                  └───────┬────────┘                  │
             └──────────────┬───────────┴───────────────┬──────────┘
                            ▼                           ▼
                    ┌───────────────┐           ┌────────────────┐
                    │  internal/llm │◀──────────│ internal/verify│
                    │  router + …   │           │ 4 layers       │
                    └───────┬───────┘           └────────────────┘
                            │
             ┌──────────────┴──────────────┐
             ▼                             ▼
     ┌───────────────┐            ┌────────────────┐
     │ Ollama (local)│            │ Cloud (opt-in) │
     │ /api/chat etc │            │ OpenAI, …      │
     └───────────────┘            └────────────────┘

     internal/config    internal/session
     ~/.vajra/…         ~/.vajra/sessions/*.json
```

## Packages

| Package | Responsibility |
|---------|----------------|
| `internal/config` | Load/save `~/.vajra/config.json`, env overrides |
| `internal/llm` | `Backend` interface; Ollama client (streaming, embed, pull) and cloud providers; `Router` model resolution |
| `internal/agent` | Agent roles and the swarm orchestrator (plan → parallel work → review → handoffs → synthesis) |
| `internal/verify` | The four-layer verification pipeline |
| `internal/session` | JSON session persistence |
| `internal/server` | REST API, SSE streaming, embedded web UI |
| `internal/tui` | Interactive terminal interface |

## Data flow (TUI)

1. `main` loads config and constructs an `llm.Router`.
2. `tui.Run` starts Bubble Tea with an `agent.Runner`.
3. On Enter, the runner builds `[system, …history, user]` and calls the router.
4. The Ollama client streams newline-delimited JSON; each token becomes a
   `tokenMsg` fed into the Bubble Tea loop.
5. On completion the session is saved and, if enabled, the verification
   pipeline runs and attaches a report.

## Data flow (web)

The browser talks to the same server that the TUI logic is built on:

- `POST /api/chat/stream` → Server-Sent Events with `token`, `verification`, `done`
- `POST /api/swarm` → SSE with `plan`, `round`, `agent_start`, `token`, `agent_done`, `bus`, `done`
  (`POST /api/team` is an alias)

## Why one binary

The previous iteration split work across Python pods, Docker and an MCP
layer. That added a runtime, a container engine and a build matrix without
changing what the user gets. A single static binary is easier to install,
audit, ship and test — so Vajra does that instead, and is honest about it.

## Concurrency

- One generation at a time per TUI session; the goroutine reports through a
  buffered channel consumed by the Bubble Tea loop.
- Swarm mode runs its specialists through a bounded worker pool (default 3 at
  once) against one shared, mutex-guarded message bus. Event emission is
  serialised, and every streamed token carries the role it belongs to, so the
  TUI and the web UI can route it back to the right agent's bubble.
- The HTTP server has per-request context timeouts and graceful shutdown.
- Verification runs after the answer, so it never blocks first-token latency.
