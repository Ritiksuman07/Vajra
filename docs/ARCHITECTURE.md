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

     internal/config    internal/session     internal/attach
     ~/.vajra/…         ~/.vajra/sessions/…   files → prompt
                        ~/.vajra/exports/     internal/rag → ~/.vajra/index/*.json
```

## Packages

| Package | Responsibility |
|---------|----------------|
| `internal/config` | Load/save `~/.vajra/config.json`, env overrides |
| `internal/llm` | `Backend` interface; Ollama client (streaming, embed, pull) and cloud providers; `Router` model resolution; token estimation and context budgeting |
| `internal/agent` | Agent roles and the swarm orchestrator (plan → parallel work → review → handoffs → synthesis) |
| `internal/attach` | Read local files into a prompt; skip binaries, cap size, report every skip |
| `internal/rag` | Index a folder (chunk → embed → JSON) and retrieve passages by cosine similarity, with `path:line` citations |
| `internal/verify` | The four-layer verification pipeline, grounded in the attachments and retrieved passages |
| `internal/session` | JSON session persistence and Markdown/JSON export |
| `internal/server` | REST API, SSE streaming, embedded web UI |
| `internal/tui` | Interactive terminal interface |

## Data flow (TUI)

1. `main` loads config and constructs an `llm.Router`.
2. `tui.Run` starts Bubble Tea with an `agent.Runner`.
3. On Enter, pending attachments (`/attach`, `@path`) are read locally and any
   configured folder is searched; the prompt becomes
   `[attachments] [passages] question`.
4. The runner budgets that prompt against the context window, trimming the
   oldest whole turns first, then calls the router.
5. The Ollama client streams newline-delimited JSON; each token becomes a
   `tokenMsg` fed into the Bubble Tea loop.
6. On completion the session is saved (prompt, attachments, sources and — for
   swarm runs — the agent transcript) and, if enabled, the verification
   pipeline runs with the local context as grounding.

## Data flow (web)

The browser talks to the same server that the TUI logic is built on:

- `POST /api/chat/stream` → Server-Sent Events with `context`, `budget`, `token`, `verification`, `done`
- `POST /api/swarm` → SSE with `plan`, `round`, `agent_start`, `token`, `agent_done`, `bus`, `context`, `done`
  (`POST /api/team` is an alias)
- `POST /api/rag/index` → SSE with `progress`, `done` — indexing without a
  hanging request
- `GET /api/sessions/{id}/export?format=md|json` → a downloaded transcript

## Security boundary

The server can read local files on behalf of the browser — that is the whole
point of attachments and retrieval — so it binds to `127.0.0.1` by default and
warns if you use `--host` to expose it. Attachments are read *by the browser*
and sent as text, so a web request can never name an arbitrary path; RAG
folders are named explicitly by the user and only ever read, never written.

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
