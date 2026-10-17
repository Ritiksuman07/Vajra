# Changelog

All notable changes to Vajra are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

_Changes since v1.2.0. This section becomes `1.3.0` when the tag is cut._

### Changed

- Development version bumped to `1.3.0` across `main.go`, the Linux packaging
  scripts, both Windows installers and `CITATION.cff`.

## [1.2.0] — 2026-10-17

Swarm mode: the multi-agent runner is now a real team that talks to itself,
and you can read every message. This release also adds the local context
layer — attachments, retrieval, budgeting and export.

### Added

- **File attachments.** Attach local files to a prompt and Vajra folds them in
  as context, with the question kept last: `/attach <path>` or `@path/to/file`
  in the TUI, 📎 or drag-and-drop in the web UI, a repeatable `--file` on the
  CLI, and `files: [{name, content}]` on the API. Binary and oversized files
  are skipped *and reported* — nothing disappears silently.
- **Retrieval over a local folder.** `vajra index <folder>` chunks, embeds and
  stores an index in `~/.vajra/index`; `vajra rag <folder> "query"` searches
  it. In the TUI (`/rag <folder>`) and web UI (📚 Folder) every prompt is
  answered with the closest passages, cited as `file:line` with a similarity
  score. `GET/POST/DELETE /api/rag` expose it over HTTP, with indexing progress
  streamed as SSE. No vector database, no service — JSON and cosine similarity.
- **Conversation context budgeting.** Prompts are trimmed to fit the model's
  window (`VAJRA_MAX_CONTEXT`, default 8192). The oldest whole turns are
  dropped first; the system prompt and your latest question always survive. The
  TUI header shows `ctx 2.1k/8.2k`, the web UI shows a meter, and both say when
  something was trimmed. `Router.Chat` now passes the real `num_ctx`.
- **Grounded verification.** Attached files and retrieved passages are passed
  to the verification pipeline, so layer 2 compares the answer with what the
  model was actually given, and says so when there was nothing to compare.
- **Session export.** `session.Message` now records attachments, sources and
  the swarm transcript. `vajra export [<id>] [--format md|json] [--out <path>]`,
  `/export` (or `Ctrl+E`) in the TUI → `~/.vajra/exports/`, ⬇ Export in the web
  UI, and `GET /api/sessions/{id}/export` over HTTP.
- **Swarm mode** (`--swarm`, `Ctrl+T`, or the 🐝 toggle in the web UI). The
  Chief decomposes the goal, and **3–4 specialist agents work in parallel**,
  each streaming its own answer. A plan guarantee tops up the roster when a
  small model only names one or two agents, so swarm mode is never a lone
  agent.
- **Agent-to-agent messaging.** Agents watch a shared bus and can address each
  other with `@role: request`. The orchestrator parses those handoffs,
  dispatches follow-up work, and the answer comes back on the bus. Handoff
  depth is bounded (default 1 round, max 3 requests) to keep runs cheap.
- **Peer review round.** A reviewer critiques the team's combined work before
  the Chief synthesises, and the critique is fed into the final answer.
- **Agent bus in the TUI.** A live per-agent transcript labelled
  `🔬 Researcher → 👥 everyone · reply`, with `✉` handoff lines, plus a
  dedicated scrollable bus screen (`Ctrl+B` / `/bus`) listing every message
  exchanged.
- **Swarm view in the web UI.** Plan card, one streaming bubble per agent with
  avatar and `→ recipient` label, handoff notes, and a highlighted final
  synthesis.
- **`POST /api/swarm`** (SSE), streaming typed `agent.SwarmEvent`s: `plan`,
  `round`, `agent_start`, `token`, `agent_done`, `bus`, `done`, `error`.
  `POST /api/team` is kept as an alias. Swarm runs are saved as sessions,
  transcript included.
- CLI swarm renderer: parallel output is buffered per agent and printed as
  clean labelled blocks; the final synthesis streams live.
- Tests for task parsing, diversity guarantees, handoff parsing, the full
  swarm run, the SSE endpoint, chunking and ranking, attachments, budgeting
  and export.

### Changed

- `vajra run --team` is now `--swarm` (`--team` still works).
- The old sequential "team" runner was replaced by the swarm orchestrator.
- `vajra serve` / `vajra web` bind to `127.0.0.1` instead of every interface.
  `--host` opts out, with a warning: the server can read the folders you attach
  or index.
- `vajra doctor` reports the context budget and how many folders are indexed.

## [1.1.0] — 2026-10-17

A ground-up rewrite into a single, dependency-light Go binary.

### Added

- **Web GUI** (`vajra web`) — a self-contained, offline web interface with
  streaming responses, session history, agent + model pickers, theming and a
  provider manager. No CDN, no build step.
- **Better TUI** — token streaming, scrollable viewport, model/agent/session
  browsers, a command palette and a spinner.
- **Agent roles** — `assistant`, `chief`, `coder`, `researcher`, `executor`,
  `reviewer`, each with a tuned system prompt.
- **Multi-agent team mode** (`--team`) — the Chief plans, specialists execute,
  and the Chief synthesises a final answer.
- **Real verification pipeline** — four layers (fast checks, semantic
  similarity, LLM-as-a-judge, human escalation) implemented natively.
- **CLI subcommands** — `run`, `serve`, `web`, `models`, `pull`, `agents`,
  `verify`, `connect`, `providers`, `config`, `doctor`.
- **Real REST API** with SSE streaming, sessions and provider management.
- **Embedded web assets** via `go:embed`, so the binary stays self-contained.
- **Test suite** — config, llm, session, agent, verify and server packages.
- `LICENSE`, `.gitignore`, `.editorconfig`, this changelog and a CITATION file.

### Fixed

- Ollama `/api/tags` responses are now parsed correctly as a
  `{"models":[...]}` envelope (previously expected a bare array, so model
  listing silently returned nothing).
- `%d` was used to format a float in the health endpoint (`go vet` failure).
- Server was started with no request timeouts and no graceful shutdown.
- Backspace handled bytes instead of runes (broke multi-byte input).

### Changed

- Project restructured into `internal/` packages with a clean layering:
  `config → llm → agent/verify → session → server/tui`.
- Module path is now `github.com/Ritiksuman07/Vajra`.
- Configuration moved from `~/.config/vajra` to `~/.vajra/config.json`.
- Dropped Python, Docker and MCP scaffolding in favour of one static binary.

### Removed

- Python agent pods, Docker orchestration and the old `pyproject.toml`
  toolchain — the same goals are met by a single binary with no runtime.

## [1.0.0] — 2026-10-01

Initial public release.
