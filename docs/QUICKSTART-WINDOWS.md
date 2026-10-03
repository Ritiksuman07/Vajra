# Vajra — Quick Start (Windows)

## Prerequisites
1. **Ollama** (or LM Studio/llama.cpp) installed
2. **Docker** (recommended) or **Go 1.22+** for service
3. **Port 4096** available for service

---

## Quick Install

### Option 1: One-liner (requires curl)
```powershell
# Windows PowerShell
iwr -useb https://raw.githubusercontent.com/yourorg/vajra/main/scripts/install.ps1 | iex
```

### Option 2: Manual
```powershell
# Clone
git clone https://github.com/yourorg/vajra
cd vajra

# Build Go service
make build-go

# Install Python agents
pip install -e .

# Start service
.\dist\vajra-service.exe
```

---

## First Run

### 1. Start Ollama
```powershell
ollama serve
```

### 2. Pull models
```powershell
ollama pull llama3.1:8b
ollama pull deepseek-coder-v2
```

### 3. Start Vajra service
```powershell
.\dist\vajra-service.exe &   # or double-click
```

### 4. Open TUI
```powershell
.\dist\vajra.exe
# or
vajra
```

### 5. Run a test task
```powershell
.\dist\vajra.exe run "create hello.txt with 'Hello, World!'"
```

---

## Docker (Recommended)

```powershell
docker compose up -d
```

---

## Windows NSIS Installer

```powershell
# Requires: NSIS (https://nsis.sourceforge.io/Download)
makensis packaging\windows\vajra.nsi
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| VAJRA_SERVICE_PORT | 4096 | REST API port |
| VAJRA_DATA_DIR | %LOCALAPPDATA%\Vajra | Data storage |
| VAJRA_CONFIG_DIR | %LOCALAPPDATA%\Vajra\config | Config files |
| VAJRA_MODEL | llama3.1:8b | Default model |
| VAJRA_OLLAMA_HOST | http://localhost:11434 | Ollama server |

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `service not running` | Run `.\dist\vajra-service.exe` manually |
| `model not found` | Run `ollama pull llama3.1:8b` |
| `port 4096 in use` | Set `VAJRA_SERVICE_PORT=5000` |
| `permission denied` | Run PowerShell as Administrator |