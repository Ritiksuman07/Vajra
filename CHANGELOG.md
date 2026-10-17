# Changelog

All notable changes to Vajra are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
