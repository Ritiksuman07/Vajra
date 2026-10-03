# Vajra v1.0.0 — Local-First AI Agent Team

## Highlights

- ✅ **Local First**: Ollama, LM Studio, and llama.cpp integration — zero data leaves your machine
- ✅ **Multi-Agent**: Chief of Staff + Coder + Researcher + Executor + Reviewer working in parallel
- ✅ **Anti-Hallucination**: 4-layer verification pipeline (fast checks → semantic similarity → LLM judge → human escalation)
- ✅ **MCP Native**: Full Model Context Protocol support with file-system, shell, git, and code-exec servers
- ✅ **Single-Click Installers**:
  - Windows: Inno Setup `.exe`
  - Linux: AppImage + DEB + RPM
- ✅ **Offline Mode Fully Supported**: No internet required after model download
- ✅ **Privacy By Default**: Zero telemetry. Zero phone-home. Zero cloud dependencies unless opted-in.

## Breaking Changes

None in this initial release.

## Migration Notes

N/A — this is the first public release.

## Full Changelog
See [CHANGELOG.md](CHANGELOG.md) or visit the [GitHub release page](https://github.com/Ritiksuman07/Vajra/releases).

## How to Upgrade

If you’re already using Vajra:

```bash
# Download the latest release
# https://github.com/Ritiksuman07/Vajra/releases/latest

# Or, if installed from source:
cd vajra
git pull origin main
pip install -e ".[dev]" 
go build ./service/cmd/vajra/
```

## Verified Environments

| OS          | Arch    | Notes                         |
|-------------|---------|-------------------------------|
| Windows 10/11 | x64    | Installer tested              |
| Ubuntu 22.04+ | x64    | AppImage + DEB tested         |
| Ubuntu 22.04+ | ARM64  | AppImage tested               |
| macOS 13+     | ARM64  | Runs via Rosetta              |

## Download Links

- **Windows**: [VajraSetup-1.0.0-win-x64.exe](https://github.com/Ritiksuman07/Vajra/releases/download/v1.0.0/VajraSetup-1.0.0-win-x64.exe)
- **Linux AppImage**: [vajra-1.0.0-x86_64.AppImage](https://github.com/Ritiksuman07/Vajra/releases/download/v1.0.0/vajra-1.0.0-x86_64.AppImage)
- **Linux DEB**: [vajra-1.0.0_amd64.deb](https://github.com/Ritiksuman07/Vajra/releases/download/v1.0.0/vajra-1.0.0_amd64.deb)
- **Linux RPM**: [vajra-1.0.0-1.x86_64.rpm](https://github.com/Ritiksuman07/Vajra/releases/download/v1.0.0/vajra-1.0.0-1.x86_64.rpm)

Checksums are available in `checksums.sha256` on the same releases page.
