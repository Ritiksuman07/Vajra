#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  Vajra — Linux / macOS build script
#  Usage:  bash scripts/build.sh [version]
#  Produces: dist/vajra-<os>-<arch>, optional .tar.gz
# ─────────────────────────────────────────────────────────────
set -euo pipefail

VERSION="${1:-1.1.0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "🛡️  Building Vajra $VERSION"

mkdir -p dist

build() {
  local os="$1" arch="$2" ext="${3:-}"
  echo "  → $os/$arch"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
    -o "dist/vajra-$os-$arch$ext" .
}

build linux  amd64
build linux  arm64
build darwin arm64
build darwin amd64

# Package each binary with the license and readme.
for bin in dist/vajra-*; do
  [ -f "$bin" ] || continue
  case "$bin" in *.tar.gz|*.sha256) continue ;; esac
  name="$(basename "$bin")"
  tar -czf "$bin.tar.gz" -C dist "$name" -C "$ROOT" README.md LICENSE CHANGELOG.md
  (cd dist && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
  echo "  ✓ $bin.tar.gz"
done

# AppImage when appimagetool is present.
if command -v appimagetool >/dev/null 2>&1 && [ -f dist/vajra-linux-amd64 ]; then
  echo "  → building AppImage"
  bash packaging/linux/build-appimage.sh "$VERSION" || echo "  · AppImage build skipped"
fi

echo "✔ Done. Artifacts in dist/"
