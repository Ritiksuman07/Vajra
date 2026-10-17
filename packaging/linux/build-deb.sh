#!/usr/bin/env bash
# Builds a .deb package. Requires dpkg-deb and a linux/amd64 binary.
# Usage: bash packaging/linux/build-deb.sh [version]
set -euo pipefail

VERSION="${1:-1.3.0}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

BIN="dist/vajra-linux-amd64"
[ -f "$BIN" ] || BIN="dist/vajra"
[ -f "$BIN" ] || { echo "build the linux binary first: make build-all"; exit 1; }

PKG="build/deb/vajra_${VERSION}_amd64"
rm -rf build/deb
mkdir -p "$PKG/DEBIAN" "$PKG/usr/bin" "$PKG/usr/share/applications"

cp "$BIN" "$PKG/usr/bin/vajra"
chmod 0755 "$PKG/usr/bin/vajra"
cp packaging/linux/vajra.desktop "$PKG/usr/share/applications/vajra.desktop"
sed "s/^Version:.*/Version: ${VERSION}/" packaging/linux/DEBIAN/control > "$PKG/DEBIAN/control"

dpkg-deb --build --root-owner-group "$PKG" "dist/vajra_${VERSION}_amd64.deb"
echo "✓ dist/vajra_${VERSION}_amd64.deb"
