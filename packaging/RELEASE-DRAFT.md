# Vajra v1.2.0 — Swarm mode: watch your agents talk to each other 🐝

Vajra is a **local-first AI agent workspace** in a single Go binary. It runs
entirely on your computer through [Ollama](https://ollama.com): a terminal UI,
a web GUI and a REST API, with multi-agent orchestration and a four-layer
verification pipeline. **No telemetry. No account. No cloud.**

## ✨ Highlights

- **One binary, zero runtime.** No Python, no Node, no Docker required.
- **🐝 Swarm mode (new)** — 3–4 specialists plan, work **in parallel**, review
  each other and hand work off with `@role: requests`. You can read every
  message they exchange, in the TUI (`Ctrl+B`) and in the web UI.
- **Terminal UI** with token streaming, session history, model/agent pickers.
- **Web GUI** (`vajra web`) — self-contained, offline, dark & light themes.
- **4-layer verification** — fast checks, semantic similarity, LLM-as-a-judge, human escalation.
- **Optional cloud** — `vajra connect openai` (or Anthropic, OpenRouter, Groq, Together) if *you* choose.

## 📦 Install

**Windows** — download `VajraSetup-1.2.0-win-x64.exe` and run it.

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/Ritiksuman07/Vajra/main/scripts/install.sh | bash
```

**From source**

```bash
git clone https://github.com/Ritiksuman07/Vajra
cd Vajra
make build
```

## 🚀 Quick start

```bash
ollama pull llama3.1:8b     # one-time model download
vajra                       # terminal UI
vajra web                   # web GUI in your browser
vajra run "explain goroutines"   # one-shot, streams to stdout
vajra doctor                # check your setup
```

## 🐝 The agent swarm

| Agent | Role |
|-------|------|
| 🧭 **Chief of Staff** | Plans, decomposes and delegates, synthesises the answer |
| 💻 **Coder** | Writes and explains code |
| 🔬 **Researcher** | Analyses and summarises |
| ⚙️ **Executor** | Produces exact commands and file contents |
| 🔍 **Reviewer** | Reviews for correctness and security |

```bash
vajra run --swarm "add tests to a small Go project and review them"
```

The Chief plans, the specialists work in parallel, the reviewer critiques, and
agents can ask each other for follow-ups. Every message is shown with its
sender and recipient, so you can see the team think:

```
🐝 swarm plan · 3 agents
── round 1 — 3 agents working in parallel ──
  🔬 Researcher → 👥 everyone · reply
  💻 Coder → 👥 everyone · reply
  ✉ ⚙️ Executor → 💻 Coder   asked for help: also add tests for the parser
  💻 Coder → ⚙️ Executor · reply
── round 2 — the reviewer critiques the team's work ──
  🔍 Reviewer → 👥 everyone · critique
  🧭 Chief of Staff → 👤 you · synthesis
```

## ✅ Verification pipeline

Every answer can be checked locally before you trust it:

1. **Fast checks** — structure, PII, citation presence (~µs)
2. **Semantic similarity** — local embeddings vs. your sources (~50 ms)
3. **LLM-as-a-judge** — a rubric across correctness, completeness, faithfulness, coherence
4. **Human escalation** — low-confidence answers are flagged, not hidden

## 🔐 Privacy

Vajra sends **nothing** anywhere by default. Prompts go to `localhost:11434`.
The only outbound requests are ones you explicitly configure with
`vajra connect`.

## 📄 License

MIT © Ritik Suman and contributors.

**Full release notes:** see [CHANGELOG.md](CHANGELOG.md)
