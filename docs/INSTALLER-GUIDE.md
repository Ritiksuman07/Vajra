# Vajra — README for the Installer (README-WINDOWS.md)
# Included in the installer package; guides the user through one-click install

# Vajra: One-Click Installer Guide

## Windows

Download `VajraSetup-1.0.0-win-x64.exe` and double-click it:

1. **Welcome** — click Next
2. **Components** — choose:
   - ☑ Vajra (service + CLI) — 15 MB
   - ☑ Ollama (local LLM runtime) — 15 MB (recommended)
   - ☐ Download default model (`llama3.1:8b`, ~4.7 GB) — **uncheck to save time**
   - ☑ Desktop shortcut
3. **Finish** — Vajra starts in a new terminal window

After install:

```bash
# Verify install
vajra

# Run your first task
vajra run "write a Python hello world to C:\workspace\hello.py"

# Start the service manually (if not auto-started)
vajra service start

# Check agents
vajra agents list

# Connect cloud providers (optional)
vajra connect --add openai --key sk-...
```

### Troubleshooting

| Problem | Fix |
|---------|-----|
| "Service not running" | Run `vajra service start` first |
| "Model not found" | Open Ollama (`C:\Users\<user>\.ollama\ollama serve`) and run `ollama pull llama3.1:8b` |
| "Port 4096 in use" | Edit `C:\Users\<user>\AppData\Local\Vajra\data\settings\config.yaml`, change `port: 4096` to another port |
| SmartScreen warning | Click "More info" → "Run anyway" — it's unsigned until code signing |

## Linux

**AppImage** (no install, any distro):
```bash
chmod +x ./vajra-1.0.0-x86_64.AppImage
./vajra-1.0.0-x86_64.AppImage
```

**Deb/RPM** (system install):
```bash
sudo apt install ./vajra-1.0.0_amd64.deb   # Debian/Ubuntu
# or
sudo rpm -i ./vajra-1.0.0-1.x86_64.rpm     # Fedora/RHEL
sudo systemctl enable --now vajra
vajra
```

**One-liner** (all distros):
```bash
curl -fsSL https://vajra.ai/install | bash
```

---

## Environment Variables

| Variable | Windows | Linux | Default |
|----------|---------|-------|---------|
| VAJRA_SERVICE_PORT | Registry | env | 4096 |
| VAJRA_DATA_DIR | `%LOCALAPPDATA%\Vajra\data` | `~/.local/share/vajra` | — |
| VAJRA_CONFIG_DIR | `%LOCALAPPDATA%\Vajra\config` | `~/.config/vajra` | — |
| VAJRA_MODEL | env | env | `llama3.1:8b` |
| VAJRA_OLLAMA_HOST | env | env | `http://localhost:11434` |

---

## Uninstall

**Windows**: Control Panel → Programs → Vajra → Uninstall

**Linux**:
```bash
sudo apt remove vajra && sudo apt autoremove   # deb
sudo rpm -e vajra                               # rpm
rm -rf ~/.local/share/vajra ~/.config/vajra     # appimage
```
