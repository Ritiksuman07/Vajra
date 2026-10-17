# Roadmap & task log

This file tracks what has shipped and what is planned. Completed items are
kept so contributors can see the direction of travel.

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
- [x] REST API with SSE (`/api/chat/stream`, `/api/team`)
- [x] CLI: `run` `serve` `web` `models` `pull` `agents` `verify` `connect`
      `providers` `config` `doctor`

**Agents & trust**

- [x] Six agent roles with tuned prompts
- [x] Multi-agent team mode (plan → delegate → synthesise)
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

- [ ] Conversation context window management (token budgeting / trimming)
- [ ] File attachments (drag a file into the web UI, `@file` in the TUI)
- [ ] Retrieval-augmented chat over a local folder (embeddings + cosine search)
- [ ] Tool calling (shell, files) behind an explicit allow-list + sandbox
- [ ] Session export to Markdown / JSON from both UIs
- [ ] Homebrew tap + `winget` / `scoop` manifests
- [ ] Signed binaries (Windows code signing, macOS notarisation)
- [ ] Plugin API for custom agents
- [ ] i18n for the web UI
- [ ] Benchmark suite comparing verification modes

## 💡 Ideas (not committed)

- Local knowledge graph for long-term memory
- Voice input via a local Whisper server
- Diff/patch preview before applying edits
- Multi-user local server with per-session isolation

Have an idea? Open a discussion or an issue.
