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
| `--swarm` | Run a team of agents that plan, work in parallel, review and talk to each other (`--team` is an alias) |
| `--verify` | Run the verification pipeline on the answer |
| `--file <path>` | Attach a local file (repeatable, or comma-separated) |
| `--rag <folder>` | Answer using passages from an indexed folder |

```bash
vajra run "summarise this file" --agent researcher
cat main.go | vajra run --agent reviewer "review this"
vajra run --swarm "design a CLI for a bookmark manager"
vajra run --file notes.md --file src/main.go "what changed here?"
vajra run --rag ~/notes "how does the retry policy work?"
```

`--file` reads the file locally and prepends it as context, with the question
last so it survives any trimming. Files that are too large or binary are
skipped and reported — never silently dropped.

`--rag` needs the folder indexed first (`vajra index`). The passages used are
printed to stderr with their `file:line` sources.

With `--swarm` you get the plan, each agent's output labelled
`🔬 Researcher → 👥 everyone · reply`, the `✉` handoffs agents make to each
other, and the final synthesis streamed live:

```
🐝 swarm plan
1. 🔬 Researcher — gather the constraints
2. 💻 Coder — produce the implementation
3. ⚙️ Executor — give the exact commands
4. 🔍 Reviewer — check for gaps

── round 1 — 3 agents working in parallel ──
⟳ 🔬 Researcher started…
⟳ 💻 Coder started…
⟳ ⚙️ Executor started…

💻 Coder → 👥 everyone · reply
…code…

✉ ⚙️ Executor → 💻 Coder  asked for help: also add tests for the parser

🧭 Chief of Staff → 👤 you · synthesis
…final answer…
```

### `vajra web [--port <n>] [--host <addr>]`
Serve the web UI and open it in your browser.

### `vajra serve [--port <n>] [--host <addr>]`
Serve the REST API and web UI without opening a browser.
Default port: `4096`. Default host: `127.0.0.1`.

Vajra binds to loopback only: the server can read the folders you attach or
index, so it does not expose them to your network. `--host 0.0.0.0` opts in to
that (you get a warning), which is only sensible on a trusted network.

### `vajra index <folder>`
Embed a folder for local retrieval. Walks the folder, skips binaries,
dot-directories, `node_modules`, `dist`, `build`, and similar, then stores the
vectors in `~/.vajra/index`. Requires an embedding model:

```bash
ollama pull nomic-embed-text
vajra index ~/notes
```

Re-running replaces the index.

### `vajra rag <folder> "<query>"`
Search an already-indexed folder and print the best passages with
`file:line` citations and similarity scores.

```bash
vajra rag ~/notes "what did I decide about retries?"
```

### `vajra export [<id>] [--format md|json] [--out <path>]`
Export a session. With no id it exports the most recent one; an id may be a
prefix. Markdown exports include attachments, retrieved sources and the full
swarm transcript.

```bash
vajra export                       # latest, Markdown, current directory
vajra export 9f2c1a --format json --out talk.json
```

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
| `VAJRA_MAX_CONTEXT` | Context window budget in tokens (default `8192`) |
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
| `Ctrl+T` | Toggle swarm mode |
| `Ctrl+B` | Open the agent bus (read what the agents said to each other) |
| `Ctrl+E` | Export this session as Markdown |
| `Ctrl+N` | New session |
| `Ctrl+U` | Clear the view |
| `↑` / `↓`, `PgUp` / `PgDn` | Scroll |
| `Ctrl+Q` / `Ctrl+C` | Quit |

## TUI slash commands

`/model [name]` · `/agent [id]` · `/models` · `/agents` · `/sessions` · `/bus`
`/verify` · `/swarm` (alias `/team`) · `/pull <model>` · `/new` · `/clear` · `/help` · `/exit`

Local context:

| Command | Action |
|---------|--------|
| `/attach <path>` | Attach a file to your prompts (`/file` is an alias) |
| `/detach` | Clear the attachments |
| `/rag <folder>` | Search a folder; indexes it the first time |
| `/rag build <folder>` | Force a re-index |
| `/rag off` | Stop retrieving |
| `/export [md\|json]` | Write the session to `~/.vajra/exports/` |

You can also just mention a file in a prompt — `explain @internal/rag/rag.go`
attaches it if the path exists. The header shows how much of the context window
the conversation is using (`ctx 2.1k/8.2k`).
