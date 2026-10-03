# Vajra — Local-First AI Agent Team

[![Stars](https://img.shields.io/github/stars/Ritiksuman07/Vajra?style=social)](https://github.com/Ritiksuman07/Vajra)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.10+](https://img.shields.io/badge/python-3.10+-blue.svg)](https://python.org/)
[![Go](https://img.shields.io/badge/go-1.22+-teal.svg)](https://go.dev/)

> **Local-first, zero-cloud AI agents. Multi-agent orchestration with 4-layer anti-hallucination verification.**

---

## What's Vajra?

**Vajra** is a local-first, open-source AI agent platform that runs entirely on your machine. It bundles **Go** (service) + **Python** (agent pods) into a single offline-capable system.

- **No data leaves your computer** by default
- **4-layer anti-hallucination** verification (fast, semantic, judge, human)
- **Multi-agent team** — Chief of Staff + Coder + Researcher + Executor + Reviewer
- **MCP Protocol** — standardized tool access (files, shell, code, git, knowledge graph)
- **Ollama / LM Studio / llama.cpp** — run with any local LLM (no API keys needed)
- **Optional `/connect`** — add OpenAI/OpenRouter/Anthropic if you want

---

## Quick Start (30 seconds)

```bash
# 1. Install Ollama + pull models
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.1:8b
ollama pull deepseek-coder-v2

# 2. Clone + build
curl -fsSL https://vajra.ai/install | bash

# 3. Run
vajra run "write hello.py"
```

---

## Features

- ✅ Local-First — zero telemetry, zero cloud dependencies by default
- ✅ Multi-Agent — team of specialists working together
- ✅ MCP Integration — file-system, shell, git, code-exec, knowledge-graph
- ✅ Anti-Hallucination Pipeline — 4 verification layers, all local
- ✅ TUI + CLI + Web — works in terminal, browser, or headless
- ✅ Docker Compose — full stack orchestration
- ✅ Windows / Linux / macOS — NSIS + .deb + .dmg packaging
- ✅ Open Source — MIT license, full source available

---

## Architecture

```text
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

## Key Commands

```bash
vajra                      # Start interactive TUI
vajra run "create file"     # Execute task headless
vajra agents                # Manage agent pods
vajra connect               # Add OpenAI/OpenRouter/Anthropic
vajra service start         # Start background service
vajra service status        # Check health
```

---

## Verification Pipeline

Every response goes through **4-layer verification**:

| Layer | Method | Coverage | Latency |
|-------|--------|----------|---------|
| 1. Fast Checks | Schema, citations, PII, length | 100% | <5ms |
| 2. Semantic Similarity | Embedding cosine ≥ 0.75 | 100% | <50ms |
| 3. LLM-as-a-Judge | Llama 3.1 8B rubric (correctness, completeness, faithfulness, coherence) | Stratified | <2s |
| 4. Human Escalation | CLI/TUI approval gate | On failure | 5min / 2hr SLA |

---

## Keywords / Topics

`local-llm` • `agent-framework` • `mcp-protocol` • `multi-agent` • `anti-hallucination` • `open-source-ai` • `offline-ai` • `ollama` • `docker` • `python` • `go` • `tui` • `cli` • `web-ui` • `self-hosted` • `zero-cloud` • `privacy-first` • `autonomous-agents`

---

## License

MIT — Use, modify, distribute freely.
