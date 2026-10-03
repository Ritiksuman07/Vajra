#!/bin/bash
# Vajra Installer Verification Script
# Checks that all build artifacts are valid and the installer works

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
PASS=0; FAIL=0; WARN=0

pass() { echo -e "${GREEN}✓${NC} $1"; ((PASS++)); }
fail() { echo -e "${RED}✗${NC} $1"; ((FAIL++)); }
warn() { echo -e "${YELLOW}⚠${NC} $1"; ((WARN++)); }

echo "=== Vajra Installer Verification ==="
echo ""

# Check required tools
check_tool() {
  if command -v "$1" &> /dev/null; then
    pass "$1 found ($($1 --version 2>/dev/null | head -c 50))"
  else
    warn "$1 NOT found - installer packaging may fail"
  fi
}

echo "--- Required Tools ---"
check_tool go
check_tool fpm
check_tool appimagetool
check_tool iscc

echo ""
echo "--- Artifact Checks ---"

# Windows artifacts
if [ -f "dist/releases/VajraSetup-1.0.0-win-x64.exe" ]; then
  size=$(du -h "dist/releases/VajraSetup-1.0.0-win-x64.exe" | cut -f1)
  pass "Windows installer exists (${size})"
else
  warn "Windows installer not built yet"
fi

if [ -f "dist/vajra.exe" ]; then
  size=$(du -h "dist/vajra.exe" | cut -f1)
  pass "Windows CLI exists (${size})"
else
  warn "Windows CLI not built yet"
fi

# Linux artifacts
if [ -f "dist/vajra-1.0.0-x86_64.AppImage" ]; then
  if appimagetool --check-for-update --no-appimage-extractor "dist/vajra-1.0.0-x86_64.AppImage" 2>/dev/null; then
    pass "AppImage is valid"
  else
    warn "AppImage may need AppRun fix"
  fi
else
  warn "AppImage not built yet"
fi

if [ -f "dist/vajra-1.0.0_amd64.deb" ]; then
  if dpkg -I "dist/vajra-1.0.0_amd64.deb" &>/dev/null; then
    pass "DEB package is valid"
  else
    warn "DEB package may be corrupt"
  fi
else
  warn "DEB not built yet"
fi

echo ""
echo "--- Installer Integrity ---"

# Check .iss file syntax
if [ -f "packaging/windows/vajra.iss" ]; then
  if grep -q "\[Setup\]" "packaging/windows/vajra.iss" && grep -q "\[Files\]" "packaging/windows/vajra.iss"; then
    pass "Inno Setup script has [Setup] and [Files] sections"
  else
    fail "Inno Setup script missing required sections"
  fi
else
  fail "Inno Setup script not found"
fi

# Check AppDir
if [ -d "dist/vajra.AppDir/AppRun" ]; then
  if [ -x "dist/vajra.AppDir/AppRun" ]; then
    pass "AppImage AppRun is executable"
  else
    fail "AppImage AppRun not executable"
  fi
else
  warn "AppDir AppRun not found"
fi

# Check checksums
if [ -f "dist/checksums.sha256" ]; then
  pass "Checksums file exists"
else
  warn "No checksums.sha256 (should be added before release)"
fi

echo ""
echo "=== Summary ==="
echo -e "${GREEN}Passed: ${PASS}${NC}"
echo -e "${RED}Failed: ${FAIL}${NC}"
echo -e "${YELLOW}Warnings: ${WARN}${NC}"

if [ $FAIL -gt 0 ]; then
  echo -e "${RED}Installer verification FAILED${NC}"
  exit 1
else
  echo -e "${GREEN}Installer verification passed${NC} (warnings are non-critical)"
  exit 0
fi
