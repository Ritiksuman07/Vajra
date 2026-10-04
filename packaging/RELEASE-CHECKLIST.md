# Release Checklist for Vajra v1.0.0

Use this checklist when preparing a new release:

## Pre-release

- [ ] Ensure `CHANGELOG.md` is updated with this release's changes
- [ ] Verify version number in `pyproject.toml` matches tag
- [ ] Merge all approved PRs to `main`
- [ ] Run full test suite (`make test`)
- [ ] Confirm CI status checks pass on `main`
- [ ] Audit `.gitignore` — no secrets/data leaks included
- [ ] Check for license headers in all source files

## Build

- [ ] Build Go binaries locally:
  ```bash
  make build
  ```
- [ ] Package for all platforms:
  ```bash
  bash scripts/build-linux.sh
  pwsh scripts/build-windows.ps1
  ```
- [ ] Confirm artifacts:
  - `dist/vajra-{VERSION}-linux-amd64.zip`  
  - `dist/vajra-{VERSION}-linux-arm64.zip`  
  - `dist/vajra-setup-{VERSION}-win-x64.exe`  
  - `dist/vajra-{VERSION}-macos-universal.dmg` (optional)

## Release

- [ ] Go to [Releases → New Release](https://github.com/Ritiksuman07/Vajra/releases/new)
- [ ] Tag version: `v{VERSION}`
- [ ] Title: `Vajra v{VERSION}`
- [ ] Paste content from [`packaging/RELEASE-DRAFT.md`](packaging/RELEASE-DRAFT.md)
- [ ] Attach binaries (drag-and-drop from `dist/`)
- [ ] Include checksums file (`sha256sum dist/* > Checksums.txt`)
- [ ] Publish release (not draft)

## Post-release

- [ ] Verify GitHub Actions `publish-python.yml` ran successfully
- [ ] Confirm PyPI package visible at https://pypi.org/project/vajra/
- [ ] Confirm GitHub Packages registry updated
- [ ] Tweet announcement with link to release notes (optional)
- [ ] Update Discord/Slack communities (if applicable)
- [ ] Close milestone associated with this release

---

> Maintainer tip: Automate this checklist using GitHub Actions and the `release-drafter/release-drafter` tool for future releases.
