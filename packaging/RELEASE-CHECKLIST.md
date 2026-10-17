# Release checklist

Use this before publishing a release.

## 1. Pre-flight

- [ ] `go vet ./...` is clean
- [ ] `go test ./... -race` passes
- [ ] `gofmt -s -l .` prints nothing
- [ ] `CHANGELOG.md` has an entry for the new version
- [ ] `README.md` and docs are up to date
- [ ] `version` in `main.go` matches the tag (or is overridden by `-ldflags`)
- [ ] No secrets, keys or local data are staged (`git status`)

## 2. Build

```bash
make build-all                 # cross-compile every target
pwsh scripts/build.ps1         # windows exe + installer (if ISCC present)
bash scripts/build.sh          # linux/macOS tarballs (+ AppImage)
```

- [ ] `dist/vajra.exe` exists
- [ ] `dist/releases/VajraSetup-<version>-win-x64.exe` exists
- [ ] `dist/vajra-linux-amd64.tar.gz` exists

## 3. Verify locally

- [ ] `vajra doctor` reports Ollama reachable
- [ ] `vajra run "hello"` streams a reply
- [ ] `vajra web` opens the GUI and a message round-trips
- [ ] `vajra serve` exposes `/api/health`

## 4. Publish

```bash
git tag v1.2.0
git push origin v1.2.0
```

- [ ] The `Release` workflow turns green
- [ ] The GitHub release has every artifact + `.sha256` files
- [ ] Release notes came from `packaging/RELEASE-DRAFT.md`

## 5. Post-release

- [ ] Announce on your channels
- [ ] Bump `version` in `main.go` for the next cycle
- [ ] Close the milestone
