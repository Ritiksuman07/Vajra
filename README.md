<div align="center">

# 🛡️ Vajra

### The local-first AI agent workspace — your machine, your models, your data.

A single Go binary that gives you a **terminal UI**, a **web GUI** and a **REST API**
on top of local models via [Ollama](https://ollama.com) — with **multi-agent
orchestration** and a **4-layer anti-hallucination pipeline**.

**No telemetry. No account. No cloud. Works fully offline.**

[![Stars](https://img.shields.io/github/stars/Ritiksuman07/Vajra?style=flat-square&logo=github&color=7c5cff)](https://github.com/Ritiksuman07/Vajra/stargazers)
[![Forks](https://img.shields.io/github/forks/Ritiksuman07/Vajra?style=flat-square&logo=github&color=22d3ee)](https://github.com/Ritiksuman07/Vajra/forks)
[![Issues](https://img.shields.io/github/issues/Ritiksuman07/Vajra?style=flat-square&color=orange)](https://github.com/Ritiksuman07/Vajra/issues)
[![Release](https://img.shields.io/github/v/release/Ritiksuman07/Vajra?style=flat-square&color=34d399)](https://github.com/Ritiksuman07/Vajra/releases)
[![CI](https://github.com/Ritiksuman07/Vajra/actions/workflows/ci.yml/badge.svg)](https://github.com/Ritiksuman07/Vajra/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/Ritiksuman07/Vajra?style=flat-square&color=blue)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Ollama](https://img.shields.io/badge/Ollama-ready-000?style=flat-square&logo=ollama&logoColor=white)](https://ollama.com)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)
[![Downloads](https://img.shields.io/github/downloads/Ritiksuman07/Vajra/total?style=flat-square&color=magenta)](https://github.com/Ritiksuman07/Vajra/releases)

**[Quick start](#-quick-start)** · **[Features](#-features)** · **[Web GUI](#-web-gui)** · **[Agents](#-the-agent-team)** · **[Verification](#-anti-hallucination)** · **[Docs](docs/)** · **[FAQ](docs/FAQ.md)**

</div>

---

```text
╭──────────────────────────────────────────────╮
│  🛡️  vajra 1.1.0   llama3.1:8b   ● ready      │
│                 verify:on  team:off           │
╰──────────────────────────────────────────────╯

  You  write a Go function that reverses a linked list
  ◆ Vajra  coder
     Here's a clean iterative reversal — O(n) time, O(1) space:

     ```go
     func Reverse(head *Node) *Node {
         var prev *Node
         for head != nil {
             next := head.Next
             head.Next = prev
             prev, head = head, next
         }
         return prev
     }
     ```
  ✓ verified · trust 94%
```

## ✨ Why Vajra?

Most "AI agent" projects need Python, Node, Docker **and** an API key before
they do anything. Vajra is different:

- 🧩 **One static binary.** ~13 MB. No runtime, no containers, no `pip install`.
- 🔒 **Private by construction.** Prompts go to `localhost:11434`. Nothing else.
- 🛡️ **Honest about trust.** A real verification pipeline, with the limits documented.
- 🧠 **A team, not a chatbot.** Specialists that plan, execute and review.
- 🖥️ **Three interfaces.** Terminal, browser, or HTTP API — your choice.

## 🚀 Quick start

**1. Get a local model** ([Ollama](https://ollama.com/download))

```bash
ollama pull llama3.1:8b
```

**2. Install Vajra**

| Platform | Install |
|----------|---------|
| **Windows** | Download `VajraSetup-1.1.0-win-x64.exe` from [Releases](https://github.com/Ritiksuman07/Vajra/releases) |
| **Linux / macOS** | `curl -fsSL https://raw.githubusercontent.com/Ritiksuman07/Vajra/main/scripts/install.sh \| bash` |
| **Any (Go)** | `git clone https://github.com/Ritiksuman07/Vajra && cd Vajra && make build` |

**3. Run**

```bash
vajra            # terminal UI
vajra web        # web GUI in your browser
vajra doctor     # check your setup
```

## 🖥️ Three ways to use it

```bash
vajra                                  # interactive TUI
vajra web                              # web GUI  →  http://localhost:4096
vajra run "explain goroutines"         # one-shot, streams to stdout
cat main.go | vajra run --agent reviewer "review this"
vajra run --team "design a CLI for a bookmark manager"   # multi-agent
vajra serve --port 8080                # REST API + SSE
```

## 🎛️ Web GUI

`vajra web` opens a self-contained interface — **no CDN, no build step, works
offline**. Streaming answers, session history, agent & model pickers,
dark/light themes and a provider manager.

```
┌──────────────────┬───────────────────────────────────────────────┐
│ 🛡️ Vajra         │  llama3.1:8b            ollama      👥 Team     │
│ ＋ New chat       ├───────────────────────────────────────────────┤
│                  │  🧑 how do I profile a Go program?            │
│ AGENTS           │                                               │
│  🛡️ Assistant    │  🛡️ Vajra        ✓ verified · 94%             │
│  🧭 Chief        │  Use pprof — import net/http/pprof, then      │
│  💻 Coder        │  go tool pprof http://localhost:6060/debug…    │
│  🔬 Researcher   │                                               │
│  ⚙️ Executor     │                                               │
│  🔍 Reviewer     │                                               │
│                  │  ┌───────────────────────────────────────────┐ │
│ SESSIONS         │  │ Message Vajra…                  Send ↵     │ │
│  💬 Profiling    │  └───────────────────────────────────────────┘ │
└──────────────────┴───────────────────────────────────────────────┘
```

## 🧠 The agent team

| Agent | Role | Suggested model |
|-------|------|-----------------|
| 🧭 **Chief of Staff** | Plans, decomposes, delegates | `llama3.1:70b` |
| 💻 **Coder** | Writes and explains code | `qwen2.5-coder:7b` |
| 🔬 **Researcher** | Analyses and summarises | `llama3.1:8b` |
| ⚙️ **Executor** | Produces exact commands | `phi3.5` |
| 🔍 **Reviewer** | Reviews for correctness & security | `llama3.1:8b` |
| 🛡️ **Assistant** | Everyday general help | `llama3.1:8b` |

**Team mode** chains them: the Chief plans, each step is routed to the right
specialist (who sees the work so far), then the Chief synthesises a final
answer. Enable with `--team`, `Ctrl+T`, or the toggle in the web UI.

## ✅ Anti-hallucination

Every answer can be checked locally before you trust it:

| Layer | What it does | Cost |
|-------|--------------|------|
| **1 · Fast checks** | Structure, unbalanced code fences, PII, citation presence | ~µs |
| **2 · Semantic similarity** | Embeds the answer + your sources, requires cosine ≥ 0.75 | ~50 ms |
| **3 · LLM-as-a-judge** | Rubric: correctness, completeness, faithfulness, coherence | ~1 s |
| **4 · Human escalation** | Low-confidence answers are **flagged**, never hidden | instant |

Run it everywhere:

```bash
vajra run --verify "what is the capital of Australia?"
vajra verify "Paris is the capital of France."
```

In the TUI press `Ctrl+V`; in the web UI toggle **🛡️ Verify**.

> Vajra is upfront: this is a set of heuristics, not a proof of truth. It
> surfaces uncertainty so *you* can decide what to trust.

## 🤝 Optional cloud

Local-first doesn't mean local-only. If you want a frontier model, add a key
— you control exactly what leaves your machine.

```bash
vajra connect openai     sk-...
vajra connect anthropic  sk-ant-...
vajra connect openrouter sk-or-...
vajra connect groq       gsk_...
vajra connect lmstudio           # local, OpenAI-compatible
```

Routing is automatic: `claude-*` → Anthropic, `gpt-*` → OpenAI,
`vendor/model` → OpenRouter, **everything else → local Ollama**.

## 📊 How Vajra compares

| | **Vajra** | Cloud chat apps | Other local agent frameworks |
|---|:---:|:---:|:---:|
| Runs fully offline | ✅ | ❌ | ✅ |
| Single binary, no runtime | ✅ | ✅ | ❌ |
| Works without an API key | ✅ | ❌ | ✅ |
| Terminal UI | ✅ | ❌ | ⚠️ varies |
| Built-in web GUI | ✅ | ✅ | ❌ |
| Multi-agent team mode | ✅ | ❌ | ⚠️ varies |
| Verification pipeline | ✅ | ❌ | ❌ |
| Zero telemetry | ✅ | ❌ | ⚠️ varies |

## 🏗️ Architecture

Layered into focused packages — see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
main.go ─┬─ internal/tui      (Bubble Tea terminal UI)
         ├─ internal/server   (REST + SSE + embedded web UI)
         └─ internal/agent    (roles + multi-agent team)
                 │
       internal/llm  ◀── internal/verify
        (router)          (4-layer pipeline)
           │
      Ollama (local)  ·  OpenAI/Anthropic/OpenRouter (opt-in)
```

## ⚙️ Configuration

Config lives at `~/.vajra/config.json`; sessions are readable JSON in
`~/.vajra/sessions/`.

```bash
vajra config --set model=llama3.1:8b
vajra config --set verify=true
vajra config --set ollama_host=http://localhost:11434
```

| Env var | Effect |
|---------|--------|
| `VAJRA_MODEL`, `VAJRA_AGENT`, `VAJRA_PORT`, `VAJRA_VERIFY` | Defaults |
| `OLLAMA_HOST` / `VAJRA_OLLAMA_HOST` | Ollama endpoint |

## 📚 Documentation

| | |
|---|---|
| [Setup](docs/SETUP.md) | Install Ollama + Vajra, troubleshooting |
| [CLI reference](docs/CLI.md) | Every command, flag, keybinding |
| [HTTP API](docs/API.md) | REST + SSE endpoints |
| [Agents](docs/agents.md) | Roles, team mode, routing |
| [Architecture](docs/ARCHITECTURE.md) | How the pieces fit |
| [FAQ](docs/FAQ.md) | Privacy, specs, common questions |
| [Roadmap](docs/TODO.md) | What shipped, what's next |
| [Packaging](packaging/PACKAGING-README.md) | Installers & release process |

## 🗺️ Roadmap

Coming next: retrieval over local files, file attachments, tool calling with a
sandbox, session export, and winget/brew packages. See [docs/TODO.md](docs/TODO.md).

## 🤝 Contributing

Contributions are welcome! Read [CONTRIBUTING.md](CONTRIBUTING.md), then look for
[`good first issue`](https://github.com/Ritiksuman07/Vajra/labels/good%20first%20issue)
or [`help wanted`](https://github.com/Ritiksuman07/Vajra/labels/help%20wanted).

```bash
git clone https://github.com/Ritiksuman07/Vajra && cd Vajra
make test        # go test ./... -race
make vet
make run
```

## 🔐 Security

Vajra is local-first by design. See [SECURITY.md](SECURITY.md) to report a
vulnerability.

## 📄 License

MIT © [Ritik Suman](https://github.com/Ritiksuman07) and
[contributors](https://github.com/Ritiksuman07/Vajra/graphs/contributors).
See [LICENSE](LICENSE).

## ⭐ Star history

[![Star History Chart](https://api.star-history.com/svg?repos=Ritiksuman07/Vajra&type=Date)](https://star-history.com/#Ritiksuman07/Vajra&Date)

<div align="center">

**If Vajra is useful to you, a ⭐ helps others find it.**

[Report a bug](https://github.com/Ritiksuman07/Vajra/issues/new?template=bug_report.md) ·
[Request a feature](https://github.com/Ritiksuman07/Vajra/issues/new?template=feature_request.md) ·
[Read the docs](docs/)

<sub>ai agent · local llm · offline ai · open source ai · autonomous agents · multi-agent · mcp · model context protocol · anti-hallucination · privacy-first · self-hosted · ollama · golang · terminal ui · web gui · zero telemetry · local-first · no api key</sub>

</div>
