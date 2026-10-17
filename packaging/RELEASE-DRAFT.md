<!--
  This file IS the body of the next GitHub release: release.yml passes it to
  softprops/action-gh-release as body_path and publishes it verbatim.

  It was reset to this v1.3.0 skeleton after v1.2.0 shipped, because leaving the
  previous release's notes here meant that tagging the next version would
  silently republish them. Fill in the placeholders below before tagging, and
  keep CHANGELOG.md in step with this file. (The v1.2.0 notes are preserved in
  git history and in the published release.)
-->

# Vajra v1.3.0 — <theme>

<One short paragraph on what this release is about.>

Vajra is a **local-first AI agent workspace** in a single Go binary. It runs
entirely on your computer through [Ollama](https://ollama.com): a terminal UI,
a web GUI and a REST API, with multi-agent orchestration and a four-layer
verification pipeline. **No telemetry. No account. No cloud.**

## ✨ Highlights

- **One binary, zero runtime.** No Python, no Node, no Docker required.
- **🐝 Swarm mode** — 3–4 specialists plan, work **in parallel**, review each
  other and hand work off with `@role: requests`. You can read every message.
- **📎 Local context** — attach files by drag-and-drop, `@path` or `--file`,
  ask questions about a whole folder with cited local retrieval, budget long
  chats to the model's window, and export any session as Markdown or JSON.
- **4-layer verification** — fast checks, semantic similarity, LLM-as-a-judge,
  human escalation, grounded in the sources the model was actually given.
- **Terminal UI** with token streaming, session history, model/agent pickers.
- **Web GUI** (`vajra web`) — self-contained, offline, dark & light themes.
- **Optional cloud** — `vajra connect openai` (or Anthropic, OpenRouter, Groq,
  Together) if *you* choose.
- <!-- TODO: the headline of this release -->

## 📦 Install

**Windows** — download `VajraSetup-1.3.0-win-x64.exe` and run it.

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

## 🆕 What's new in v1.3.0

<!-- TODO: replace with the real changelog for this release. -->

- <!-- TODO -->

## 🔐 Privacy

Vajra sends **nothing** anywhere by default. Prompts go to `localhost:11434`.
The only outbound requests are ones you explicitly configure with
`vajra connect`.

## 📄 License

MIT © Ritik Suman and contributors.

**Full release notes:** see [CHANGELOG.md](CHANGELOG.md)
