# 🛡️ Vajra — Local-First AI Agent Team

[![License](https://img.shields.io/github/license/Ritiksuman07/Vajra)](https://github.com/Ritiksuman07/Vajra/blob/main/LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/Ritiksuman07/Vajra/pulls)
[![Python](https://img.shields.io/badge/python-3.10+-blue.svg)](https://www.python.org/)
[![Go](https://img.shields.io/badge/go-1.22+-teal.svg)](https://go.dev/)
[![Build Status](https://github.com/Ritiksuman07/Vajra/actions/workflows/release.yaml/badge.svg)](https://github.com/Ritiksuman07/Vajra/actions/workflows/release.yaml)
[![Open Source](https://img.shields.io/badge/Open%20Source-💖-red.svg)](https://github.com/Ritiksuman07/Vajra)

> **Local-first, offline-capable AI agent platform with multi-agent coordination and 4-layer anti-hallucination verification.**

## 🌟 Why Vajra?

**Vajra** is a **privacy-first**, **zero-telemetry**, **offline-capable** AI agent framework — designed for developers who refuse to trust the cloud.

It bundles:
- **Go-based service** for orchestration + state
- **Python agent pods** running in isolated Docker containers
- **MCP (Model Context Protocol)** integration
- **4-layer anti-hallucination verification** (fast checks, semantic similarity, LLM-as-judge, human escalation)
- **Multi-agent team**: Chief of Staff + Coder + Researcher + Executor + Reviewer
- **Single-click installers** for Windows (Inno Setup) + Linux (AppImage/DEB/RPM)
- **Local LLM backends**: Ollama, LM Studio, llama.cpp
- **Optional cloud**: `/connect` to OpenAI, OpenRouter, Anthropic (if needed)

---

## Features

| Feature | Status | Description |
|---------|--------|-------------|
| Local LLMs | ✅ First | Ollama, LM Studio, llama.cpp |
| MCP Support | ✅ Full | File-system, shell, git, code-exec, knowledge-graph |
| Multi-Agent | ✅ Full | Chief + Coder + Researcher + Executor + Reviewer |
| Anti-Hallucination | ✅ 4-Layer | Fast checks, semantic similarity, LLM judge, human escalation |
| Offline Mode | ✅ Full | Zero data leaves your machine |
| Single-Click Install | ✅ Windows | Inno Setup installer (.exe) |
| Single-Click Install | ✅ Linux | AppImage + DEB + RPM |
| Web Dashboard | ✅ Basic | Flask UI at `/web` |
| Terminal UI | ✅ TUI | Full-screen interactive terminal |

---

## Quick Start

### Download (Pre-built)

**Windows:**
```powershell
# Download installer from releases
# Double-click: VajraSetup-1.0.0-win-x64.exe
```

**Linux:**
```bash
# Download AppImage
chmod +x vajra-1.0.0-x86_64.AppImage
./vajra-1.0.0-x86_64.AppImage run "Hello World"
```

**macOS:**
```bash
brew install ollama && ollama serve
curl -fsSL https://vajra.ai/install | sh
```

### Install from Source

```bash
git clone https://github.com/Ritiksuman07/Vajra.git
cd Vajra
pip install -e ".[dev]"   # Python agent core
go build ./service/cmd/vajra/  # Go service
make install
```

---

## Usage

### Run interactively (TUI):
```bash
vajra
```

### Run headless:
```bash
vajra run "write a Python REST API with OpenAPI docs"
```

### Connect cloud providers (optional):
```bash
vajra connect --add openai --key sk-...
vajra connect --list
```

### Manage agent pods:
```bash
vajra agents list
vajra agents create --role coder --model deepseek-coder
vajra agents stop <pod-id>
```

---

## Architecture

```
┌──────────────────────────────────────────────┐
│                 Vajra Service (Go)           │
│  ├── REST API (localhost:4096)              │
│  ├── Pod lifecycle (Docker SDK)             │
│  ├── SQLite (sessions, auth, audit)         │
│  └── Provider registry (/connect)          │
├──────────────────────────────────────────────┤
│                 Agent Pods (Python)          │
│  ├── Chief of Staff (planning)               │
│  ├── Coder (generate/refactor/test)         │
│  ├── Researcher (search/analyze)            │
│  ├── Executor (run/file/ops)                │
│  └── Reviewer (audit/security)              │
├──────────────────────────────────────────────┤
│                 MCP Servers                  │
│  ├── file-system  (read/write/delete)      │
│  ├── shell        (exec)                    │
│  ├── git          (status/diff/commit)      │
│  ├── code-exec    (sandbox)                 │
│  └── knowledge    (Neo4j/Kuzu graph)        │
├──────────────────────────────────────────────┤
│                 Local LLM Backends           │
│  ├── Ollama (primary)                       │
│  ├── LM Studio (GUI management)              │
│  ├── llama.cpp (embedded, no server)        │
│  └── vLLM (high-throughput)                 │
└──────────────────────────────────────────────┘
```

---

## Supported Models

| Provider | Models | Notes |
|----------|--------|-------|
| **Ollama**  | llama3.1:8b, llama3.1:70b, deepseek-coder, codellama | Default backend |
| **LM Studio** | Any GGUF | GUI management |
| **llama.cpp** | Any GGUF | Embedded (no server) |
| **OpenAI**   | gpt-4o, gpt-4-turbo, gpt-3.5-turbo | Via `/connect` |
| **OpenRouter** | meta-llama-3.1, mistral-8x7b, etc. | Via `/connect` |
| **Anthropic** | claude-3-5-sonnet, claude-3-opus | Via `/connect` |

---

## Documentation

- [Setup Guide](docs/SETUP.md) — Full installation from source
- [Agent Guide](docs/agents.md) — How agents work, roles, and verification pipeline
- [Installer Guide](docs/INSTALLER-GUIDE.md) — Packaging details for Windows/Linux
- [API Reference](docs/API.md) — REST endpoints
- [MCP Servers](docs/MCP.md) — Available tools

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and how to submit changes.

Check our [open issues](https://github.com/Ritiksuman07/Vajra/issues) for `good first issue` or `help wanted` labels.

---

## License

MIT © [Ritik Suman](https://github.com/Ritiksuman07) and [Vajra Contributors](https://github.com/Ritiksuman07/Vajra/contributors)

---

## Keywords / Topics

`ai-agent` `local-llm` `offline-ai` `open-source-ai` `autonomous-agents` `multi-agent` `mcp` `model-context-protocol` `anti-hallucination` `privacy-first` `self-hosted` `ollama` `appimage` `inno-setup` `go-lang` `python` `gpt-oss` `openrouter` `anthropic-claude` `openai-compatible`

---

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Ritiksuman07/Vajra&type=date)](https://star-history.com/?utm=yes&repo=Ritiksuman07/Vajra)

If you find Vajra useful, please consider giving us a ⭐!
