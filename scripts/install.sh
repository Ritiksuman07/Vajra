#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  Vajra — installer script (Linux / macOS)
#  Usage:
#    curl -fsSL https://raw.githubusercontent.com/Ritiksuman07/Vajra/main/scripts/install.sh | bash
# ─────────────────────────────────────────────────────────────
set -euo pipefail

REPO="Ritiksuman07/Vajra"
BIN="${VAJRA_BIN_DIR:-$HOME/.local/bin}"
VERSION="${VAJRA_VERSION:-latest}"

cyan() { printf '\033[36m%s\033[0m\n' "$1"; }
green() { printf '\033[32m%s\033[0m\n' "$1"; }
red() { printf '\033[31m%s\033[0m\n' "$1"; }

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) red "Unsupported OS: $os"; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) red "Unsupported architecture: $arch"; exit 1 ;;
esac

cyan "🛡️  Installing Vajra ($os/$arch)"

if [ "$VERSION" = "latest" ]; then
  url="https://github.com/$REPO/releases/latest/download/vajra-$os-$arch.tar.gz"
else
  url="https://github.com/$REPO/releases/download/$VERSION/vajra-$os-$arch.tar.gz"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cyan "→ downloading $url"
if ! curl -fsSL "$url" -o "$tmp/vajra.tar.gz"; then
  red "Download failed. Build from source instead:"
  echo "  git clone https://github.com/$REPO && cd Vajra && make build"
  exit 1
fi

tar -xzf "$tmp/vajra.tar.gz" -C "$tmp"
mkdir -p "$BIN"
install -m 0755 "$tmp/vajra-$os-$arch" "$BIN/vajra"

green "✓ installed to $BIN/vajra"
case ":$PATH:" in
  *":$BIN:"*) ;;
  *) echo "  add to PATH:  export PATH=\"$BIN:\$PATH\"" ;;
esac

if command -v ollama >/dev/null 2>&1; then
  green "✓ Ollama detected"
else
  echo "  Ollama not found — install it: https://ollama.com/download"
fi

echo
green "Get started:"
echo "  ollama pull llama3.1:8b"
echo "  vajra"
