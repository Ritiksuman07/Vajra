# Vajra — Setup Guide

## Prerequisites

### 1. Install Ollama (Local LLM)
```bash
# Linux/macOS
curl -fsSL https://ollama.ai/install.sh | sh

# macOS
brew install ollama

# Windows
# Download from https://ollama.ai/download/windows
```

### 2. Pull Models
```bash
# Chief of Staff (planning, delegation)
ollama pull llama3.1:70b

# Coder (code generation)
ollama pull deepseek-coder-v2

# Researcher (analysis, synthesis)
ollama pull llama3.1:8b

# Executor (fast tasks)
ollama pull phi-3.5-mini
```

### 3. Start Ollama Service
```bash
ollama serve &
```

### 4. Install Vajra
```bash
# Linux/macOS
curl -fsSL https://vajra.ai/install | sh

# Or build from source
git clone https://github.com/yourorg/vajra
cd vajra
make install
```

### 5. Start Vajra Service
```bash
vajra service start
# or: vajra --standalone
```

### 6. Use Vajra
```bash
# Interactive TUI
vajra

# One-shot task
vajra run "create a Python REST API with tests"

# Check agents
vajra agents list

# Connect cloud providers (optional)
vajra connect --add openai --key sk-...
```

## Docker Compose (Full Stack)

```bash
docker compose up -d
```

## Development

```bash
# Install dependencies
pip install -e ".[dev]"

# Run tests
make test

# Start service
make start-service
```

## Architecture

```
┌─────────────────────────────────────────────┐
│                 Vajra                        │
├─────────────────────────────────────────────┤
│  CLI/TUI (Go)                                │
│  └── Commands: run, connect, agents, models  │
├─────────────────────────────────────────────┤
│  Background Service (Go)                     │
│  └── REST API on localhost:4096             │
│  └── Pod lifecycle management                │
│  └── SQLite for sessions, auth, state       │
├─────────────────────────────────────────────┤
│  Agent Pods (Python in Docker)               │
│  └── Chief of Staff (llama3.1:70b)           │
│  └── Coder (deepseek-coder-v2)               │
│  └── Researcher (llama3.1:8b)                │
│  └── Executor (phi-3.5-mini)                 │
│  └── Reviewer (llama3.1:8b)                  │
├─────────────────────────────────────────────┤
│  MCP Servers (JSON-RPC 2.0)                  │
│  └── file-system, shell, code-exec, git      │
│  └── knowledge-graph (Neo4j)                 │
├─────────────────────────────────────────────┤
│  Local Models (Ollama/LM Studio/llama.cpp)   │
│  └── Zero cloud calls by default             │
│  └── /connect for optional cloud providers   │
├─────────────────────────────────────────────┤
│  Anti-Hallucination (4 layers, all local)     │
│  └── Layer 1: Fast checks                    │
│  └── Layer 2: Semantic similarity            │
│  └── Layer 3: LLM-as-a-Judge                 │
│  └── Layer 4: Human escalation               │
└─────────────────────────────────────────────┘
```

## Configuration

Edit `~/.config/vajra/config.yaml`:

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