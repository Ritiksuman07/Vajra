# Vajra v1.1.0 — Your machine. Your models. Your data. 🛡️

Vajra is a **local-first AI agent workspace** in a single Go binary. It runs
entirely on your computer through [Ollama](https://ollama.com): a terminal UI,
a web GUI and a REST API, with multi-agent orchestration and a four-layer
verification pipeline. **No telemetry. No account. No cloud.**

## ✨ Highlights

- **One binary, zero runtime.** No Python, no Node, no Docker required.
- **Terminal UI** with token streaming, session history, model/agent pickers.
- **Web GUI** (`vajra web`) — self-contained, offline, dark & light themes.
- **Multi-agent team mode** — a Chief plans, specialists execute, the Chief synthesises.
- **4-layer verification** — fast checks, semantic similarity, LLM-as-a-judge, human escalation.
- **Optional cloud** — `vajra connect openai` (or Anthropic, OpenRouter, Groq, Together) if *you* choose.

## 📦 Install

**Windows** — download `VajraSetup-1.1.0-win-x64.exe` and run it.

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

## 🧠 The agent team

| Agent | Role |
|-------|------|
| 🧭 **Chief of Staff** | Plans, decomposes and delegates |
| 💻 **Coder** | Writes and explains code |
| 🔬 **Researcher** | Analyses and summarises |
| ⚙️ **Executor** | Produces exact commands and file contents |
| 🔍 **Reviewer** | Reviews for correctness and security |

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
