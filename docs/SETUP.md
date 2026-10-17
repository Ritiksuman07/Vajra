# Setup

Vajra is a single binary. It talks to a local model server — by default
[Ollama](https://ollama.com). Setup is three steps.

## 1. Install a model server (Ollama)

| Platform | Command |
|----------|---------|
| Windows | Download from <https://ollama.com/download/windows> |
| macOS | `brew install ollama` or download from <https://ollama.com/download> |
| Linux | `curl -fsSL https://ollama.com/install.sh \| sh` |

Start it and pull a model:

```bash
ollama serve            # usually already running as a service
ollama pull llama3.1:8b
```

Recommended local models:

| Model | Size | Best for |
|-------|------|----------|
| `llama3.1:8b` | ~4.7 GB | General chat, good default |
| `qwen2.5-coder:7b` | ~4.7 GB | Code |
| `phi3.5` | ~2.2 GB | Fast, low-RAM machines |
| `nomic-embed-text` | ~274 MB | Verification (semantic layer) |

## 2. Install Vajra

**Windows** — run `VajraSetup-<version>-win-x64.exe`.

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/Ritiksuman07/Vajra/main/scripts/install.sh | bash
```

**From source** (needs [Go 1.24+](https://go.dev/dl/))

```bash
git clone https://github.com/Ritiksuman07/Vajra
cd Vajra
make build          # → dist/vajra
# optionally
make install        # → $GOPATH/bin/vajra
```

## 3. Run

```bash
vajra doctor        # verify everything is wired up
vajra               # terminal UI
```

Or try the web UI:

```bash
vajra web
```

## Configuration

Config lives at `~/.vajra/config.json`
(`%USERPROFILE%\.vajra\config.json` on Windows).

```bash
vajra config                                  # show
vajra config --set model=llama3.1:8b          # change model
vajra config --set verify=true                # enable verification
vajra config --set ollama_host=http://other:11434
```

Sessions are stored as readable JSON in `~/.vajra/sessions/`.

## Optional: cloud providers

Vajra is local-first, but you can add a cloud key if you want to.

```bash
vajra connect openai sk-...
vajra connect anthropic sk-ant-...
vajra connect openrouter sk-or-...
vajra connect groq gsk_...
vajra connect lmstudio            # local, OpenAI-compatible on :1234
```

Then select a cloud model:

```bash
vajra run --model gpt-4o "hello"
```

Routing is automatic — `claude-*` goes to Anthropic, `gpt-*` to OpenAI,
`vendor/model` to OpenRouter, everything else to Ollama.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `doctor` says Ollama not reachable | Run `ollama serve`; check `ollama_host` |
| `no models — start Ollama` | `ollama pull llama3.1:8b` |
| Semantic verification skipped | `ollama pull nomic-embed-text` |
| Very slow answers | Use a smaller model (`phi3.5`) or reduce context |
| Windows SmartScreen warning | Unsigned build — *More info* → *Run anyway* |

See [FAQ.md](FAQ.md) for more.
