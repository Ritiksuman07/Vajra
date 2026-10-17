#!/usr/bin/env bash
# Builds a Vajra AppImage. Requires appimagetool on PATH.
# Usage: bash packaging/linux/build-appimage.sh [version]
set -euo pipefail

VERSION="${1:-1.3.0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

APP="AppDir"
rm -rf "$APP"
mkdir -p "$APP/usr/bin" "$APP/usr/share/applications" "$APP/usr/share/icons/hicolor/256x256/apps"

BIN="dist/vajra-linux-amd64"
[ -f "$BIN" ] || BIN="dist/vajra"
[ -f "$BIN" ] || { echo "build the linux binary first: make build-all"; exit 1; }

cp "$BIN" "$APP/usr/bin/vajra"
chmod +x "$APP/usr/bin/vajra"

cat > "$APP/vajra.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Vajra
Comment=Local-first AI agent workspace
Exec=vajra
Terminal=true
Categories=Development;Utility;
EOF

cat > "$APP/AppRun" <<'EOF'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/vajra" "$@"
EOF
chmod +x "$APP/AppRun"

cp "$APP/vajra.desktop" "$APP/usr/share/applications/" 2>/dev/null || true

VERSION="$VERSION" appimagetool "$APP" "dist/vajra-$VERSION-x86_64.AppImage"
echo "✓ dist/vajra-$VERSION-x86_64.AppImage"
