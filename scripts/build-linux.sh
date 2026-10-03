#!/bin/bash
# Vajra Linux AppImage + DEB/RPM Builder
# Usage: ./scripts/build-linux.sh
# Requires: Go 1.22+, Python 3.11+, appimagetool, fpm, docker (optional)

set -euo pipefail

VERSION="1.0.0"
ROOT=$(dirname "$(dirname "$(realpath "$0")")")
DIST="${ROOT}/dist"
APPNAME="vajra"

echo "=== Vajra Linux Build ==="
echo "Version: $VERSION"
echo "Root: $ROOT"

# ---- 1. Build Go binaries ----
echo "[1/6] Building Go service..."
cd "${ROOT}/service"
go build -ldflags "-X main.version=${VERSION}" -o "${DIST}/vajra-service" ./cmd/vajra/

echo "[2/6] Building Go CLI..."
cd "${ROOT}/cli"
go build -ldflags "-X main.version=${VERSION}" -o "${DIST}/vajra" .

# ---- 2. Prepare Python runtime ----
echo "[3/6] Preparing Python runtime..."
cd "${ROOT}"
# Use system python or create virtual env
mkdir -p "${DIST}/python"
python3 -m venv "${DIST}/python/venv"
"${DIST}/python/venv/bin/pip" install -q -e "${ROOT}/agent" 2>/dev/null || true

# ---- 4. Create AppDir structure ----
echo "[4/6] Creating AppDir..."
APPDIR="${DIST}/vajra.AppDir"
rm -rf "${APPDIR}"
mkdir -p "${APPDIR}/usr/bin"
mkdir -p "${APPDIR}/usr/lib/vajra"
mkdir -p "${APPDIR}/usr/share/applications"
mkdir -p "${APPDIR}/usr/share/icons/hicolor/256x256/apps"

# Copy binaries
cp "${DIST}/vajra" "${APPDIR}/usr/bin/"
cp "${DIST}/vajra-service" "${APPDIR}/usr/bin/"

# Copy agent code
cp -r "${ROOT}/agent" "${APPDIR}/usr/lib/vajra/"
cp -r "${ROOT}/mcp" "${APPDIR}/usr/lib/vajra/"
cp -r "${ROOT}/ui" "${APPDIR}/usr/lib/vajra/"

# Copy config
mkdir -p "${APPDIR}/usr/lib/vajra/config"
cp "${ROOT}/config/default.yaml" "${APPDIR}/usr/lib/vajra/config/"

# ---- 5. Create AppRun entrypoint ----
cat > "${APPDIR}/AppRun" << 'EOF'
#!/bin/bash
HERE=$(dirname "$(readlink -f "$0")")
export VAJRA_HOME="${HOME}/.config/vajra"
export VAJRA_DATA="${VAJRA_HOME}/data"
export VAJRA_CONFIG="${VAJRA_HOME}/config.yaml"
export VAJRA_PYTHON="${HERE}/usr/lib/vajra/agent"
export PYTHONPATH="${HERE}/usr/lib/vajra:${PYTHONPATH}"

mkdir -p "${VAJRA_HOME}" "${VAJRA_DATA}/workspace" "${VAJRA_DATA}/models" "${VAJRA_DATA}/logs"

# Start service if not running
if ! pgrep -f "vajra-service" >/dev/null; then
    "${HERE}/usr/bin/vajra-service" &
fi

# Launch CLI/TUI
exec "${HERE}/usr/bin/vajra" "$@"
EOF
chmod +x "${APPDIR}/AppRun"

# ---- 6. Desktop file ----
cat > "${APPDIR}/vajra.desktop" << EOF
[Desktop Entry]
Type=Application
Name=Vajra
Comment=Local-first AI agent team
Exec=vajra
Icon=vajra
Terminal=true
Categories=Development;Utility;
StartupNotify=true
EOF

# Icon (placeholder)
# cp icon.png "${APPDIR}/vajra.png"  # Add your icon

# ---- 7. Build AppImage ----
echo "[5/6] Building AppImage..."
if command -v appimagetool &> /dev/null; then
    appimagetool "${APPDIR}" "${DIST}/vajra-${VERSION}-x86_64.AppImage"
    echo "AppImage: ${DIST}/vajra-${VERSION}-x86_64.AppImage"
else
    echo "appimagetool not found - skipping AppImage"
fi

# ---- 8. Build DEB/RPM with fpm ----
echo "[6/6] Building DEB/RPM..."
if command -v fpm &> /dev/null; then
    fpm -s dir -t deb \
        -n "${APPNAME}" -v "${VERSION}" \
        --description "Local-first AI agent team" \
        --url "https://github.com/Ritiksuman07/Vajra" \
        --license "MIT" \
        --maintainer "Vajra Contributors" \
        --depends "libc6 (>= 2.31)" \
        --after-install "${ROOT}/packaging/linux/postinst.sh" \
        --after-remove "${ROOT}/packaging/linux/postrm.sh" \
        --prefix=/opt/vajra \
        "${APPDIR}/usr/bin/vajra=/usr/bin/vajra" \
        "${APPDIR}/usr/bin/vajra-service=/usr/bin/vajra-service" \
        "${APPDIR}/usr/lib/vajra/=/opt/vajra/" \
        -p "${DIST}/vajra-${VERSION}_amd64.deb"

    fpm -s dir -t rpm \
        -n "${APPNAME}" -v "${VERSION}" \
        --description "Local-first AI agent team" \
        --url "https://github.com/Ritiksuman07/Vajra" \
        --license "MIT" \
        --maintainer "Vajra Contributors" \
        --depends "glibc >= 2.31" \
        --prefix=/opt/vajra \
        "${APPDIR}/usr/bin/vajra=/usr/bin/vajra" \
        "${APPDIR}/usr/bin/vajra-service=/usr/bin/vajra-service" \
        "${APPDIR}/usr/lib/vajra/=/opt/vajra/" \
        -p "${DIST}/vajra-${VERSION}-1.x86_64.rpm"

    echo "DEB: ${DIST}/vajra-${VERSION}_amd64.deb"
    echo "RPM: ${DIST}/vajra-${VERSION}-1.x86_64.rpm"
else
    echo "fpm not found - skipping DEB/RPM"
fi

echo ""
echo "=== Linux Build Complete ==="
ls -lh "${DIST}"/vajra* 2>/dev/null || true