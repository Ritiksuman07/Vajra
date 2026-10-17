# Packaging

Vajra ships as a single static binary. Everything here wraps that binary
for distribution.

```
packaging/
├── windows/
│   ├── vajra.iss          # Inno Setup (primary)
│   └── vajra.nsi          # NSIS (fallback)
├── linux/
│   ├── build-appimage.sh  # AppImage
│   ├── build-deb.sh       # .deb (needs dpkg-deb)
│   ├── vajra.desktop      # desktop entry
│   └── DEBIAN/control     # deb metadata
├── RELEASE-DRAFT.md       # release notes template
└── RELEASE-CHECKLIST.md   # maintainer checklist
```

## Windows

```powershell
# 1. build the binary
go build -o dist\vajra.exe .

# 2. compile the installer (Inno Setup 6)
ISCC packaging\windows\vajra.iss
# or with a version override
ISCC /DVersion=1.2.0 packaging\windows\vajra.iss
```

Output: `dist\releases\VajraSetup-<version>-win-x64.exe`

One-shot helper: `pwsh scripts\build.ps1 -Version 1.2.0`

> Inno Setup is not bundled. Install it with `choco install innosetup`
> or from <https://jrsoftware.org/is/>. If it is missing, `build.ps1`
> still produces `dist\vajra.exe`.

## Linux

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/vajra-linux-amd64 .
bash packaging/linux/build-appimage.sh 1.2.0   # needs appimagetool
bash packaging/linux/build-deb.sh 1.2.0       # needs dpkg-deb
```

`scripts/build.sh` cross-compiles Linux and macOS binaries and tars them
with the license and readme.

## macOS

Unsigned builds are produced by `scripts/build.sh`. Gatekeeper will warn on
first launch — right-click → **Open**, or `xattr -dr com.apple.quarantine vajra-darwin-arm64`.

## Automated releases

Pushing a tag like `v1.2.0` triggers `.github/workflows/release.yml`, which
cross-compiles every target, builds the Windows installer, and publishes a
GitHub release with checksums.
