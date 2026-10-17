# FAQ

**Is Vajra really offline?**
Yes. By default every prompt goes to `http://localhost:11434` (Ollama).
The only outbound connections are the ones you create with
`vajra connect`. There is no telemetry, no analytics and no update check.

**Do I need an API key?**
No. Vajra is built around local models. Cloud keys are optional.

**Which models work?**
Anything Ollama can serve, plus OpenAI/OpenRouter/Groq/Together/Anthropic
if you add a key, and LM Studio (OpenAI-compatible, local). See
[SETUP.md](SETUP.md#1-install-a-model-server-ollama).

**What are the minimum specs?**
A 7B quantised model runs in ~8 GB of RAM. `phi3.5` runs comfortably in
~4 GB. Vajra itself is a ~13 MB binary.

**Does Vajra send my code anywhere?**
No. Nothing leaves the machine unless you configure a cloud provider.

**How is "anti-hallucination" implemented?**
Four checks, all local: fast structural/PII checks, semantic similarity
against any sources you provide, an LLM-as-a-judge rubric, and a
human-escalation flag for low-confidence output. It is a set of heuristics,
not a guarantee — [docs/agents.md](agents.md) explains the limits.

**Why is there no Docker/Python anymore?**
It added a runtime and a build matrix without changing the user experience.
One static binary installs faster, tests faster and is easier to audit. See
[ARCHITECTURE.md](ARCHITECTURE.md#why-one-binary).

**Can I use it as a library or in scripts?**
Yes — `vajra serve` exposes a REST + SSE API. See [API.md](API.md).

**Where is my data stored?**
`~/.vajra/`. Config in `config.json`, sessions as JSON files in
`sessions/`. Delete the directory to reset.

**Does it work on a Raspberry Pi?**
Linux arm64 builds are produced for every release. A Pi 5 can run a small
quantised model through Ollama.

**How do I update?**
Download the latest release, or `git pull && make build`. Sessions and
config are preserved.

**Something is broken.**
Run `vajra doctor` first — it checks the platform, config, Ollama, the
embedding model and routing. Then open an issue with that output.
