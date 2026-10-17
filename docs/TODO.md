# Roadmap & task log

This file tracks what has shipped and what is planned. Completed items are
kept so contributors can see the direction of travel.

## ✅ Shipped — v1.2.0 (swarm)

- [x] Swarm orchestrator: plan → **parallel** work round → peer review →
      agent-to-agent handoffs → synthesis
- [x] Guaranteed roster of 3–4 distinct agents (never a lone agent)
- [x] Shared agent message bus with typed events (`plan` `task` `reply`
      `critique` `handoff` `synthesis`)
- [x] Agents can address each other with `@role: request`; handoffs are
      parsed, dispatched and answered (bounded to 1 round / 3 requests)
- [x] TUI: live per-agent transcript with `from → to` labels, `✉` handoff
      lines and a scrollable agent-bus screen (`Ctrl+B`, `/bus`)
- [x] Web UI: plan card, per-agent streaming bubbles, handoff notes,
      highlighted final synthesis
- [x] `POST /api/swarm` (SSE) with typed events; `/api/team` kept as an alias
- [x] CLI `--swarm` with clean buffered blocks per agent + live synthesis
- [x] Tests: task parsing, diversity guarantee, handoff parsing, full swarm
      run, SSE endpoint

## ✅ Shipped — local context (attachments, retrieval, budgeting, export)

**File attachments**

- [x] `internal/attach`: read local files, skip binaries and oversized files,
      report every skip instead of dropping it silently
- [x] TUI: `/attach <path>`, `/detach`, and `@path/to/file` inside a prompt
- [x] Web UI: 📎 picker and drag-and-drop; the browser reads the file and sends
      the text, so the server never opens a path on your behalf
- [x] CLI: repeatable `--file` (or comma-separated), with a skip report
- [x] API: `files: [{name, content}]` on `/api/chat` and `/api/swarm`

**Retrieval over a local folder**

- [x] `internal/rag`: walk → chunk (line-aware, overlapping) → embed → JSON
      index under `~/.vajra/index`; cosine search; `path:line` citations
- [x] Ignores build output, dependencies and dot-directories; bounded files,
      chunks and size; 4-way embedding pool
- [x] CLI: `vajra index <folder>` and `vajra rag <folder> "query"`
- [x] TUI: `/rag <folder>` (indexes on first use), `/rag build`, `/rag off` —
      retrieval runs per prompt and the sources are listed under the answer
- [x] Web UI: 📚 folder panel with live indexing progress; sources shown under
      each answer
- [x] API: `GET /api/rag`, `POST /api/rag/index` (SSE progress), `DELETE /api/rag`
- [x] Retrieved passages are also handed to the verification pipeline

**Context window budgeting**

- [x] `internal/llm/budget.go`: token estimation, headroom, whole-turn
      trimming that keeps the system prompt and the final question
- [x] Applied in one place (`agent.Runner.build`), so TUI, web and CLI all
      trim identically; `Router.Chat` passes the real `num_ctx`
- [x] `VAJRA_MAX_CONTEXT` (default 8192); `vajra doctor` prints it
- [x] Surfaced in the TUI header (`ctx 2.1k/8.2k`) and the web UI
      (`budget` SSE event, plus a notice when older turns were trimmed)

**Session export**

- [x] `session.Message` records attachments, retrieved sources and the swarm
      transcript, so a stored session can be re-read exactly as it ran
- [x] `vajra export [<id>] [--format md|json] [--out <path>]`
- [x] TUI: `/export` and `Ctrl+E` → `~/.vajra/exports/`
- [x] Web UI: ⬇ Export, and the agent bus is still readable when a session is
      re-opened

## ✅ Shipped — v1.1.0 rewrite


**Core**

- [x] Single static Go binary (no Python, Node or Docker runtime)
- [x] Clean `internal/` layering: config → llm → agent/verify → session → server/tui
- [x] Fixed Ollama `/api/tags` parsing (envelope, not bare array)
- [x] True token streaming (Ollama NDJSON + OpenAI-compatible SSE)
- [x] `go vet` clean; float formatting bug fixed
- [x] Module renamed to `github.com/Ritiksuman07/Vajra`

**Interfaces**

- [x] Terminal UI: streaming, viewport, spinner, pickers, themes, help
- [x] Web GUI: offline, streaming, sessions, provider manager, dark/light
- [x] REST API with SSE (`/api/chat/stream`, `/api/swarm`)
- [x] CLI: `run` `serve` `web` `models` `pull` `agents` `verify` `connect`
      `providers` `config` `doctor`

**Agents & trust**

- [x] Six agent roles with tuned prompts
- [x] Multi-agent swarm mode (plan → parallel work → review → synthesis)
- [x] Four-layer verification pipeline (fast, semantic, judge, escalate)
- [x] Session persistence as readable JSON

**Quality**

- [x] Test suite: config, llm, session, agent, verify, server
- [x] CI: build/vet/test on Linux, Windows, macOS + cross-compile smoke test
- [x] Release workflow: cross-compiled artifacts + Windows installer
- [x] Dependabot, PR template, lint config

**Distribution & reach**

- [x] Inno Setup + NSIS installers, AppImage script, `.deb` metadata
- [x] `scripts/install.sh` one-liner
- [x] README, docs, changelog, citations, license
- [x] SEO: badges, topics, keywords, comparison table

## 🔜 Next

- [ ] Tool calling (shell, files) behind an explicit allow-list + sandbox
- [ ] Homebrew tap + `winget` / `scoop` manifests
- [ ] Signed binaries (Windows code signing, macOS notarisation)
- [ ] Plugin API for custom agents
- [ ] i18n for the web UI
- [ ] Benchmark suite comparing verification modes
- [ ] Incremental re-indexing (only changed files) for large folders

## 🧰 Maintenance

- [x] `VajraSetup-1.2.0-win-x64.exe.sha256` added to the published v1.2.0
      release, so every artifact it ships now has a checksum. Uploaded as a
      release asset rather than by re-running the workflow: a dispatch today
      checks out a later tree and would republish the wrong release notes. From
      v1.3.0 on, `release.yml` produces the installer checksum itself.
- [ ] Lint the workflows in CI with `actionlint`. `release.yml` is never
      exercised by a pull request, so a broken release workflow is only found on
      release day — which is how the two bugs above survived.

## 💡 Ideas (not committed)

- Local knowledge graph for long-term memory
- Voice input via a local Whisper server
- Diff/patch preview before applying edits
- Multi-user local server with per-session isolation

Have an idea? Open a discussion or an issue.
