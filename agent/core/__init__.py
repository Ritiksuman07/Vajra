# Agent Pod - Core Agent Implementation

This module implements the core agent loop:
reason → tool → verify → respond

## Agent Loop

1. **Receive task** from coordinator/user
2. **Reason** using local LLM (Ollama/LM Studio/llama.cpp)
3. **Execute** via MCP tools (file, shell, git, code exec)
4. **Verify** using 4-layer anti-hallucination pipeline
5. **Respond** with verified result

## Verification Layers

| Layer | Method | Threshold |
|-------|--------|-----------|
| 1. Fast Checks | Schema, citations, PII, length | 100% pass |
| 2. Semantic Similarity | bge-small embeddings, cosine | ≥ 0.75 |
| 3. LLM-as-a-Judge | Llama 3.1 8B, 4-dimension rubric | all dims ≥ 7.0 |
| 4. Human Escalation | CLI/TUI prompt | confidence ≥ 0.6 |

## Multi-Agent Coordination

Agents communicate via A2A protocol:
- **Chief of Staff**: Delegates tasks to specialists
- **Coder**: Writes/reads code, runs tests
- **Researcher**: Searches, synthesizes, analyzes
- **Executor**: Runs commands, file ops
- **Reviewer**: Code review, security audit

## Configuration

Set via environment variables:
```bash
export vajra_BOT_MODEL=llama3.1:8b
export vajra_BOT_OLLAMA_HOST=http://localhost:11434
export vajra_BOT_WORKSPACE=/path/to/workspace
```
