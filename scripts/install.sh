#!/bin/bash

# vajra Bot Install Script
# One-liner installer for Linux/macOS
# Usage: curl -fsSL https://vajra-bot.ai/install | bash

set -e

NAME="vajra Bot"
VERSION="1.0.0"
INSTALL_DIR="$HOME/.local"
SERVICE_DIR="$HOME/.config/vajra-bot"

# Detect OS
OS="unknown"
case "$(uname -s)" in
    Linux*)     OS="linux" ;;
    Darwin*)    OS="darwin" ;;
    CYGWIN*)    OS="windows" ;;
    MINGW*)     OS="windows" ;;
    *)          echo "Unsupported OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Create install directories
mkdir -p "$INSTALL_DIR/bin"
mkdir -p "$SERVICE_DIR"

# Download the latest vajra Bot binary (Go service + TUI)
echo "Downloading vajra Bot $VERSION for $OS $ARCH..."

# For now, create a placeholder - in production this would download from GitHub releases
cat > "$INSTALL_DIR/bin/vajra-bot" << 'EOF'
#!/bin/bash
echo "vajra Bot - Local-first AI agent team"
echo "Use 'vajra-bot run \"your task\"' to execute a task"
echo "Use 'vajra-bot agents' to manage agent pods"
echo "Use 'vajra-bot connect' to configure providers"
EOF

chmod +x "$INSTALL_DIR/bin/vajra-bot"

# Create symlink
cat >> "$INSTALL_DIR/bin/vajra-bot" << 'EOF'

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Initialize config if not exists
if [ ! -f "$SERVICE_DIR/config.yaml" ]; then
    cat > "$SERVICE_DIR/config.yaml" << 'EOL'
# vajra Bot Configuration
# Edit this file to configure local models and providers
service:
  port: 4096
  data_dir: "$SERVICE_DIR"
  log_level: info

models:
  default_backend: ollama
  ollama:
    host: http://localhost:11434
    default_model: llama3.1:8b

agents:
  max_pods: 8
  default_resources:
    cpu: "2"
    memory: "4g"

mcp:
  servers:
    - name: file-system
      enabled: true
    - name: shell
      enabled: true
    - name: git
      enabled: true
    - name: code-exec
      enabled: true
    - name: knowledge-graph
      enabled: true
EOL
fi

# Start service if not running
if ! pgrep -f "vajra-bot-service" > /dev/null; then
    cd "$SERVICE_DIR"
    # Start service (if available)
    if command -v vajra-bot-service > /dev/null; then
        nohup vajra-bot-service > "$$SERVICE_DIR/log.txt" 2>&1 &
        echo "vajra Bot service started"
    else
        echo "Note: Install Go service components for full functionality"
    fi
fi

echo "vajra Bot installed successfully!"
echo "Run 'vajra-bot' to start the interactive TUI"
echo "Run 'vajra-bot run \"task\"' to execute a task"
EOF

# Install Python package (if pip available)
if command -v pip3 > /dev/null; then
    echo "Installing Python package..."
    pip3 install --user vajra-bot
else
    echo "pip3 not found, skipping Python package installation"
fi

# Create desktop entry (Linux)
if [[ "$OS" == "linux" ]]; then
    DESKTOP_DIR="$HOME/.local/share/applications"
    mkdir -p "$DESKTOP_DIR"
    cat > "$DESKTOP_DIR/vajra-bot.desktop" << EOL
[Desktop Entry]
Type=Application
Name=vajra Bot
Comment=Local-first AI agent team
Exec=vajra-bot
Icon=/usr/share/icons/hicolor/48x48/apps/vajra-bot.png
Categories=Utility;Development;
StartupNotify=true
EOL
fi

# Create completion (bash/zsh)
if command -v bash > /dev/null; then
    cat > "$HOME/.vajra-bot-completion.bash" << 'EOF'
_vajra_bot_completion() {
    local commands
    commands=$(vajra-bot list-commands 2>/dev/null || echo "chat agents connect monitor run exit")
    compgen -W "$commands" "$2"
}
complete -F _vajra_bot_completion vajra-bot
EOF
    if [[ -f "$HOME/.vajra-bot-completion.bash" ]]; then
        echo "To enable completions, add this to your ~/.bashrc:"
        echo "source $HOME/.vajra-bot-completion.bash"
    fi
fi

echo ""
echo "Installation complete!"
echo ""
echo "Key commands:"
echo "  vajra-bot              # Interactive TUI"
echo "  vajra-bot run \"task\"   # Execute task"
echo "  vajra-bot agents       # Manage agent pods"
echo "  vajra-bot connect      # Configure providers"
echo ""
echo "For more information, visit: https://vajra-bot.ai/docs"

