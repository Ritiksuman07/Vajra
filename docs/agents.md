# Vajra — Agent Documentation

## Project Overview

**Vajra** is a local-first, offline-capable AI agent platform that runs entirely on-device. It consists of a Go service, Python agent pods, TUI/CLI interfaces, and multi-platform installers. No cloud dependencies by default.

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

## Key Files

### Service Layer (Go)
- `service/cmd/vajra/main.go` — Entry point for background service
- `service/internal/api/router.go` — REST API endpoints (sessions, pods, models, providers)
- `service/internal/db/sqlite.go` — SQLite initialization (sessions, audit logs)
- `service/internal/pods/manager.go` — Docker pod lifecycle management
- `service/internal/config/loader.go` — YAML config loading
- `service/pkg/models/ollama_client.go` — Ollama API integration
- `service/pkg/models/provider.go` — Provider abstraction (local + cloud)

### Agent Layer (Python)
- `agent/core/agent.py` — AgentPod class (core reasoning loop with verification)
- `agent/core/verification.py` — 4-layer anti-hallucination pipeline
- `agent/core/orchestrator.py` — Multi-agent coordination (team spawning)
- `agent/core/a2a_protocol.py` — Agent-to-Agent messaging protocol
- `agent/core/__init__.py` — Module init

### MCP Servers (JSON-RPC 2.0 over stdio)
- `agent/tools/file_server/mcp_server.py` — File system operations (workspace-scoped)
- `agent/tools/shell_server/mcp_server.py` — Shell command execution (sandboxed)
- `agent/tools/code_exec/sandbox.py` — Code execution sandbox (Python/JS/TS)

### CLI & TUI
- `cli/commands/root.go` — Root command (`vajra`)
- `cli/commands/run.go` — Run tasks headlessly (`vajra run "task"`)
- `cli/commands/connect.go` — Manage providers (`vajra connect --add openai`)
- `cli/commands/models.go` — List models (`vajra models`)
- `cli/commands/agents.go` — Manage agent pods (`vajra agents list`)
- `cli/commands/service.go` — Service control (`vajra service start/status`)
- `cli/tui/cli_main.py` — Interactive TUI (chat, monitor, provider views)
- `cli/tui/app.py` — Flask web UI backend

### Packaging
- `packaging/windows/vajra.iss` — Inno Setup installer (primary Windows)
- `packaging/windows/vajra.nsi` — NSIS fallback installer
- `packaging/linux/postinst.sh` — Debian post-install hook
- `packaging/linux/debian/DEBIAN/control` — DEB metadata
- `scripts/build-windows.ps1` — Windows build pipeline
- `scripts/build-linux.sh` — Linux build pipeline (AppImage + DEB + RPM)
- `scripts/verify-installers.sh` — Installer verification script
- `docs/PACKAGING-SUMMARY.md` — Packaging overview

### Infrastructure
- `docker-compose.yaml` — Full stack with 5 agents + Ollama + Neo4j
- `service/Dockerfile` — Service container
- `agent/Dockerfile` — Agent pod container
- `.github/workflows/release.yaml` — CI/CD for multi-platform releases

---

## Agent Roles

### Chief of Staff (`chief`)
- **Model**: llama3.1:70b (recommended)
- **Role**: Planner, orchestrator, task decomposition
- **Tools**: Full suite + A2A protocol for delegation
- **Memory**: Full conversation history + session context

### Coder (`coder`)
- **Model**: codellama / deepseek-coder-v2
- **Role**: Code generation, refactoring, bug fixing
- **Tools**: file.read/write, code.exec, git, shell
- **Environment**: Read-only filesystem except workspace

### Researcher (`researcher`)
- **Model**: llama3.1:8b
- **Role**: Information gathering, analysis, synthesis
- **Tools**: web.search, file.read, knowledge.graph.query
- **Output**: Summaries with citations and confidence scores

### Executor (`executor`)
- **Model**: phi-3.5-mini
- **Role**: File operations, command execution, API calls
- **Tools**: shell.exec, file.read/write/delete, git
- **Constraints**: Network disabled by default, strict workspace isolation

### Reviewer (`reviewer`)
- **Model**: llama3.1:8b
- **Role**: Code review, security audit, style checking
- **Tools**: git.diff, code.exec (linting), knowledge.graph.query
- **Output**: Pass/fail with detailed findings + fixes

---

## Verification Pipeline

Every agent response passes through a 4-layer verification pipeline:

1. **Layer 1: Fast Checks** (<5ms, 100% coverage)
   - Schema validation
   - Citation presence (`[cite:doc:chunk:start-end]`)
   - PII detection and redaction
   - Length limits

2. **Layer 2: Semantic Similarity** (<50ms)
   - Embed response + retrieved context
   - Cosine similarity ≥ 0.75 required
   - Claim-level similarity scoring

3. **Layer 3: LLM-as-a-Judge** (<2s)
   - 4-dimension rubric (correctness, completeness, faithfulness, coherence)
   - Confidence scoring via calibrated composite
   - Trust scores with Bayesian updates

4. **Layer 4: Human Escalation**
   - CLI prompt / TUI dialog
   - SLA: Acknowledge within 5min, resolve within 2hr
   - Feedback loop retrains judges + updates trust scores

---

## Configuration

### Config File (`config/default.yaml`)
```yaml
service:
  port: 4096
  data_dir: ~/.local/share/vajra
  log_level: info

models:
  default_backend: ollama
  ollama:
    host: http://localhost:11434
    default_model: llama3.1:8b

agents:
  max_pods: 8
  default_resources:
    cpu: "2"
    memory: "4g"

verification:
  enabled: true
  layers: [fast, semantic, llm_judge, human]
  local_judge_model: llama3.1:8b

mcp:
  servers:
    - name: file-system
      enabled: true
    - name: shell
      enabled: true
    - name: git
      enabled: true
    - name: code-exec
      enabled: true
    - name: knowledge-graph
      enabled: true
```

---

## Quick Start

### Windows (Single-Click)
1. Build: `ISCC packaging\windows\vajra.iss` → `dist\releases\VajraSetup-1.0.0-win-x64.exe`
2. Install: Double-click installer
3. Run: `vajra run "write hello.py"`

### Linux (Single-Click)
1. Build: `bash scripts/build-linux.sh`
2. Install: `./vajra-1.0.0-x86_64.AppImage`
3. Run: `./vajra-1.0.0-x86_64.AppImage run "write hello.py"`

### Docker
```bash
docker compose up -d
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VAJRA_SERVICE_PORT` | 4096 | Service API port |
| `VAJRA_DATA_DIR` | platform-specific | Data storage dir |
| `VAJRA_CONFIG_DIR` | platform-specific | Config dir |
| `VAJRA_MODEL` | llama3.1:8b | Default model |
| `VAJRA_OLLAMA_HOST` | localhost:11434 | Ollama API endpoint |
| `VAJRA_WORKSPACE` | /workspace | Agent workspace |

---

## Testing

```bash
# Unit tests
python -m pytest tests/ -v

# Agent verification
python tests/test_agent_pod.py

# Multi-agent verification
python tests/test_multi_agent.py
```

---

## Build Notes

- All references renamed from `grok-bot` to `vajra` throughout
- All paths updated to vajra naming
- Config dirs: `~/.config/vajra` (Linux/macOS), `%LOCALAPPDATA%\Vajra` (Windows)
- See `packaging/CHANGES.md` for full build history
