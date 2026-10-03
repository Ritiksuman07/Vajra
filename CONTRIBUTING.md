# Contributing to Vajra — Local AI Agents for Everyone

We're thrilled you want to contribute to **Vajra** — a local-first, offline-capable, multi-agent system built for developers who care about privacy and autonomy.

Whether you're fixing a typo, improving performance, or adding new integrations, your help matters.

---

## Table of Contents

1. [Ways to Contribute](#ways-to-contribute)
2. [Reporting Issues](#reporting-issues)
3. [Development Setup](#development-setup)
4. [Submitting Changes](#submitting-changes)
5. [Code Style](#code-style)
6. [Testing](#testing)
7. [Help Wanted](#help-wanted)
8. [License](#license)

---

## Ways to Contribute

Here are some ways you can contribute to Vajra:

### 🛠️ Engineering
- Improve agent reasoning loops
- Add support for new LLM backends (llama.cpp, HuggingFace, etc.)
- Optimize verification pipeline speed/memory
- Add MCP server plugins
- Improve installer pipelines

### 📚 Documentation
- Improve setup guides (`docs/`)
- Write tutorials (e.g., "Build your own AI team")
- Translate docs to other languages
- Create YouTube/GIF demos

### 🎨 Design/UI
- Improve web dashboard (`ui/web/`)
- Create better README visuals
- Build terminal UI themes for TUI (`cli/tui/`)

### 🧠 AI Research
- Improve anti-hallucination verification
- Add new trust scoring models
- Research local embedding models

---

## Reporting Issues

Found a bug or have a feature request?

1. Search existing issues at [github.com/Ritiksuman07/Vajra/issues](https://github.com/Ritiksuman07/Vajra/issues)
2. If no similar issue exists, open a new one with:
   - Clear title
   - Detailed description
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior

**Security vulnerabilities**: Please report privately via **security@vajra.ai**.

---

## Development Setup

```bash
# Clone
git clone https://github.com/Ritiksuman07/Vajra.git
cd Vajra

# Install Python deps
pip install -e ".[dev]"

# Install Go modules
cd service && go mod tidy && cd ../

# (Optional) Install Ollama for local models
curl -fsSL https://ollama.ai/install.sh | sh

# Run tests
bash scripts/run-tests.sh
```

---

## Submitting Changes

1. Fork the repository
2. Create a branch: `git checkout -b feat/my-feature`
3. Make changes
4. Run tests: `bash scripts/run-tests.sh`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feat/my-feature`
7. Open a Pull Request

**PR Requirements**:
- Tests pass (`make test`)
- Code formatted (`ruff check agent/`, `gofmt service/`)
- Changes documented (if user-facing)

---

## Code Style

| Language | Linter/Formatter | Command |
|----------|------------------|---------|
| Python   | ruff + black     | `ruff check . && black .` |
| Go       | gofmt            | `gofmt -w service/` |
| Shell    | shellcheck       | `shellcheck scripts/*.sh` |

---

## Testing

```bash
# Run all tests
python -m pytest tests/ -v

# Run specific test
python tests/test_agent_pod.py

# Verify installers
bash scripts/verify-installers.sh

# Run verification layer tests
python tests/test_verification_pipeline.py
```

---

## Help Wanted

Check our open issues with the `good first issue` or `help wanted` labels:
👉 [github.com/Ritiksuman07/Vajra/issues?q=is:issue+is:open+label:"good first issue"](https://github.com/Ritiksuman07/Vajra/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)

Current needs:
- [ ] Add VS Code extension for agent management
- [ ] Improve Windows installer error handling
- [ ] Add TypeScript code execution sandbox
- [ ] Write tutorial: "Building your first multi-agent workflow"

---

## License

By contributing to Vajra, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

Thank you for helping make AI more accessible, private, and powerful! 🚀
