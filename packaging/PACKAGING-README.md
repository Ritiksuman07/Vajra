# Vajra Packaging Guide

## Overview

Vajra ships as single-click installers for all platforms:

- **Windows** (primary): Inno Setup `VajraSetup-1.0.0-win-x64.exe`
- **Windows** (fallback): NSIS `vajra-1.0.0-windows-x64.exe`
- **Linux** (primary): AppImage `vajra-1.0.0-x86_64.AppImage`
- **Linux** (secondary): DEB + RPM packages
- **macOS**: DMG (template ready)

All built automatically via GitHub Actions on release tags.

---

## Prerequisites

### Windows
- Go 1.22+ (https://go.dev/dl/)
- Inno Setup 6.x (https://jrsoftware.org/isdl.php)
- NSIS (https://nsis.sourceforge.io/Download)
- Python 3.11+ (for agent runtime)
- Ollama Windows binary (for bundling)

### Linux
- Go 1.22+
- appimagetool (https://docs.appimage.org/)
- fpm (https://github.com/jordansissel/fpm)
- Docker (optional, for pod isolation)

---

## Build Instructions

### Windows Build

```powershell
# 1. Build Go binaries
go build -o dist/vajra.exe ./cli
go build -o dist/vajra-service.exe ./service/cmd/vajra/

# 2. Bundle dependencies
. .github/workflows/scripts\bundle-windows.ps1

# 3. Create Inno Setup installer
ISCC packaging\windows\vajra.iss

# Output: dist/releases\VajraSetup-1.0.0-win-x64.exe
```

**Manual Inno Setup compile** (if `ISCC` not in PATH):
```cmd
"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" packaging\windows\vajra.iss
```

### Linux Build

```bash
# 1. Build Go binaries
go build -o dist/vajra ./cli
go build -o dist/vajra-service ./service/cmd/vajra/

# 2. Build Linux packages
bash scripts/build-linux.sh

# Outputs:
#   dist/vajra-1.0.0-x86_64.AppImage
#   dist/vajra-1.0.0_amd64.deb
#   dist/vajra-1.0.0-1.x86_64.rpm
```

### Verify Artifacts

```bash
bash scripts/verify-installers.sh
```

---

## Inno Setup Details

`packaging/windows/vajra.iss` defines:

| Setting | Value |
|---------|-------|
| Install dir | `%LOCALAPPDATA%\Vajra` |
| Service | auto-start (optional) |
| Desktop icon | yes |
| Ollama bundled | yes (silent) |
| Model download | opt-in (checkbox) |

**Build:**
```cmd
ISCC packaging/windows/vajra.iss
```

**Test** (run the .iss in "Setup Compilation Wizard" or use `iscc /cc` for compile only):
```cmd
iscc /cc packaging/windows/vajra.iss
```

---

## AppImage Details

Structure (`dist/vajra.AppDir`):
```
vajra.AppDir/
├── AppRun                    # Entry point
├── usr/
│   ├── bin/                  # vajra, vajra-service
│   └── lib/vajra/            # agent, mcp, ui, config
└── vajra.desktop
```

**Build:**
```bash
appimagetool vajra.AppDir vajra-1.0.0-x86_64.AppImage
```

---

## DEB/RPM Details

Built with `fpm` from the AppDir:
- Installs to `/opt/vajra/`
- systemd service unit (auto-start)
- Desktop entry + icons
- postinst/postrm hooks

---

## GitHub Actions

On tag push (`v*`), `.github/workflows/release.yaml` runs:

1. **build-service** — compile Go binaries for win/linux/mac
2. **windows-installer** — Inno Setup compilation
3. **linux-package** — AppImage + DEB + RPM
4. **create-release** — GitHub release with all artifacts + SHA256 checksums

---

## Release Checklist

- [ ] Update `appVersion` in `.iss` and all scripts
- [ ] Test installer on fresh Windows VM
- [ ] Test AppImage on Ubuntu + Fedora
- [ ] Generate and attach SHA256 checksums
- [ ] Sign binary with code-signing cert (Windows)
- [ ] Update README badges
- [ ] Create release notes (templates: `packaging/release-notes.md`)
- [ ] Publish to PyPI + npm if applicable

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| "Cannot find module" during Go build | Run `go mod download` first |
| AppImage doesn't launch | Check `AppRun` is executable; ensure all deps bundled |
| Inno Setup compile fails | Update `[Setup] AppVersion`; check `[Files]` paths |
| SmartScreen warning | Sign binary (EV cert) or add to Microsoft Whitelist |

---

## Contact

Questions? Issues? Open a GitHub issue at https://github.com/Ritiksuman07/Vajra/issues
