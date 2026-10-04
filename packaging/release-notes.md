# Vajra Release Notes

## [1.0.0] — Initial Release

### What's New

**🚀 Single-Click Installation**
- **Windows**: Inno Setup installer (`VajraSetup-1.0.0-win-x64.exe`) — bundled Ollama, embedded Python, optional model download, automatic service install
- **Linux**: AppImage (portable, any distro), DEB, RPM, plus one-line curl install
- **macOS**: DMG in works

**🔧 Multi-Agent Team**
- Chief of Staff + Coder + Researcher + Executor + Reviewer agents
- A2A protocol for agent coordination
- Automatic task routing by type

**🛡 Anti-Hallucination Verification**
- 4-layer pipeline: Fast Checks → Semantic Similarity → LLM-as-a-Judge → Human Escalation
- All verification runs locally (zero cloud calls)
- Confidence scoring + mandatory inline citations

**🔌 MCP Protocol Integration**
- Built-in servers: file-system, shell, git, code-exec (sandboxed), knowledge-graph (Neo4j)
- Standardized tool access pattern

**🏠 Local-First Design**
- Zero telemetry, zero data egress by default
- Runs with Ollama / LM Studio / llama.cpp — no API keys needed
- `/connect` command to optionally add OpenAI / OpenRouter / Anthropic

**🖥 Interfaces**
- Interactive TUI (main interface)
- CLI for headless automation (`vajra run "..."`)
- Web UI dashboard (`python -m ui.web.app`)

**📦 Docker Support**
- `docker compose up -d` — full stack with 5 agents + Ollama + Neo4j

**🔋 Build & Packaging**
- GitHub Actions CI/CD: automated builds for Windows/Linux/macOS on tag push
- SHA256 checksums + GPG signing
- Versioned releases via go module

### Tech Stack

| Layer | Technology |
|-------|------------|
| Service | Go 1.22+, SQLite, Docker SDK |
| Agents | Python 3.11+, MCP Protocol |
| Models | Ollama / LM Studio / llama.cpp / vLLM |
| Storage | SQLite + Neo4j / Kuzu |
| Verification | sentence-transformers + local LLM-as-a-judge |

### Upgrade Notes

- Config file moved from `~/.config/grok-bot/` → `~/.config/vajra/` (if upgrading from a previous version)
- Database schema v1 (migrates automatically)
- `vajra` replaces all `grok-bot` commands (if upgrading from a previous version)

### Known Issues

- Windows SmartScreen warning on first run (unsigned binary) — click "More info" → "Run anyway"
- NSIS fallback installer requires manual `makensis` compile (Inno Setup recommended)

### Credits

Built with ❤️ by the Vajra team. Inspired by the agent community (AutoGPT, OpenCode, and open-source multi-agent frameworks).

### License

MIT — free for personal and commercial use.
