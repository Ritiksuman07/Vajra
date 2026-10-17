# CLI reference

```
vajra [command] [flags]
```

Running `vajra` with no arguments launches the terminal UI.

## Commands

### `vajra`
Launch the interactive TUI.

### `vajra run "<prompt>" [flags]`
Send one prompt and stream the answer to stdout. Reads from stdin when
piped.

| Flag | Description |
|------|-------------|
| `--model <name>` | Override the model |
| `--agent <id>` | Agent role (`assistant`, `chief`, `coder`, `researcher`, `executor`, `reviewer`) |
| `--team` | Plan + delegate across the agent team |
| `--verify` | Run the verification pipeline on the answer |

```bash
vajra run "summarise this file" --agent researcher
cat main.go | vajra run --agent reviewer "review this"
vajra run --team "design a CLI for a bookmark manager"
```

### `vajra web [--port <n>]`
Serve the web UI and open it in your browser.

### `vajra serve [--port <n>]`
Serve the REST API and web UI without opening a browser.
Default port: `4096`.

### `vajra models`
List models from Ollama and any configured cloud providers.

### `vajra pull <model>`
Download a model through Ollama, with a progress bar.

### `vajra agents`
List the built-in agent roles.

### `vajra verify "<text>"`
Run the four-layer verification pipeline over some text.

### `vajra connect <provider> [key] [--url <u>] [--model <m>]`
Add a provider. Supported: `openai`, `anthropic`, `openrouter`, `groq`,
`together`, `lmstudio`, `custom`.

```bash
vajra connect openai sk-...
vajra connect lmstudio
vajra connect custom --url http://localhost:8000/v1
```

### `vajra providers`
List configured providers (keys are masked).

### `vajra config [--set key=value]`
Print the configuration, or change a value.

```bash
vajra config --set model=llama3.1:8b
vajra config --set verify=true
vajra config --set ollama_host=http://localhost:11434
```

### `vajra doctor`
Check platform, config, Ollama reachability, embedding model and routing.

### `vajra version` · `vajra help`

## Environment variables

| Variable | Effect |
|----------|--------|
| `VAJRA_MODEL` | Default model |
| `VAJRA_AGENT` | Default agent role |
| `VAJRA_PORT` | Default server port |
| `VAJRA_VERIFY` | `1`/`true` to enable verification |
| `VAJRA_CONFIG_DIR` | Override the config directory |
| `OLLAMA_HOST` / `VAJRA_OLLAMA_HOST` | Ollama base URL |

## TUI keys

| Key | Action |
|-----|--------|
| `Enter` | Send |
| `Ctrl+H` / `F1` | Help |
| `Ctrl+M` | Model picker |
| `Ctrl+A` | Agent picker |
| `Ctrl+L` | Session browser |
| `Ctrl+V` | Toggle verification |
| `Ctrl+T` | Toggle team mode |
| `Ctrl+N` | New session |
| `Ctrl+U` | Clear the view |
| `↑` / `↓`, `PgUp` / `PgDn` | Scroll |
| `Ctrl+Q` / `Ctrl+C` | Quit |

## TUI slash commands

`/model [name]` · `/agent [id]` · `/models` · `/agents` · `/sessions`
`/verify` · `/team` · `/pull <model>` · `/new` · `/clear` · `/help` · `/exit`
