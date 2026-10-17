# Contributing to Vajra

Thanks for wanting to help! Vajra is a local-first AI agent workspace written
in Go. Contributions of every size are welcome — code, docs, design, bug
reports and ideas.

## Ways to contribute

**Engineering**
- Improve streaming, UI responsiveness and terminal rendering
- Add model backends (llama.cpp, vLLM, Hugging Face endpoints)
- Improve the verification pipeline (better semantic checks, calibration)
- Add tool calling behind a sandbox + allow-list
- Sharpen packaging, installers and cross-platform builds

**Documentation & design**
- Improve the [docs](docs/) and write tutorials
- Polish the web UI (`internal/server/web/`) or the TUI (`internal/tui/`)
- Create demo GIFs and README visuals
- Translate docs

**Research**
- Better trust scoring and judge calibration
- Cheaper local embeddings / retrieval

## Reporting issues

1. Search [existing issues](https://github.com/Ritiksuman07/Vajra/issues) first.
2. If new, use a template and include:
   - what you expected vs. what happened,
   - exact commands,
   - the output of `vajra doctor`,
   - OS, Go version and model name.

Security issues: follow [SECURITY.md](SECURITY.md) — **do not** open a public issue.

## Development setup

Requires Go 1.24+ and, for real answers, [Ollama](https://ollama.com).

```bash
git clone https://github.com/Ritiksuman07/Vajra.git
cd Vajra

go mod download
make build            # → dist/vajra
make test             # go test ./... -race
make vet              # go vet ./...
make run              # build + launch the TUI
```

Project layout (see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)):

```
main.go                 CLI entrypoint
internal/config         configuration
internal/llm            backends + router
internal/agent          roles + team runner
internal/verify         verification pipeline
internal/session        session storage
internal/server         REST API + embedded web UI
internal/tui            terminal UI
```

## Submitting changes

1. Fork and branch: `git checkout -b feat/my-feature`
2. Keep commits focused and messages clear (`feat:`, `fix:`, `docs:`…)
3. Run the checks:
   ```bash
   gofmt -s -l .        # must print nothing
   go vet ./...
   go test ./... -race
   ```
4. Add tests for behaviour changes.
5. Open a PR and fill in the template.

**Ground rules**
- No telemetry, analytics or update pings.
- No new always-on network calls; anything outbound must be opt-in.
- Keep the binary self-contained (assets embedded, no CDN).

## Code style

| Area | Tool |
|------|------|
| Go | `gofmt -s`, `go vet`, [golangci-lint](.golangci.yml) |
| Shell | `shellcheck` |
| Docs | clear, concise, UK/US English consistent |

Prefer small, well-named packages and functions; document exported symbols;
return errors rather than panicking.

## Testing

```bash
go test ./... -race -count=1          # everything
go test ./internal/verify/ -v         # one package
go test ./internal/llm/ -run Ollama   # one group
```

HTTP behaviour is tested with `httptest`, so tests never need a real Ollama.

## Good first issues

Look for the [`good first issue`](https://github.com/Ritiksuman07/Vajra/labels/good%20first%20issue)
and [`help wanted`](https://github.com/Ritiksuman07/Vajra/labels/help%20wanted)
labels. The [roadmap](docs/TODO.md) lists what's planned.

## License

By contributing you agree your work is licensed under the [MIT License](LICENSE).
