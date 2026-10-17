# -------------------------------------------------------------
#  Vajra - Windows build script
#  Usage:  powershell -ExecutionPolicy Bypass -File scripts\build.ps1 [-Version 1.1.0]
# -------------------------------------------------------------
param(
    [string]$Version = "1.1.0"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

Write-Host "Building Vajra $Version" -ForegroundColor Magenta

New-Item -ItemType Directory -Force -Path "dist\releases" | Out-Null

Write-Host "-> compiling windows/amd64" -ForegroundColor Cyan
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o "dist\vajra.exe" .
if ($LASTEXITCODE -ne 0) { throw "go build failed" }
if (-not (Test-Path "dist\vajra.exe")) { throw "binary missing" }
Write-Host "  ok dist\vajra.exe" -ForegroundColor Green

# Optional: compile the Inno Setup installer when ISCC is available.
$iscc = $null
$cmd = Get-Command ISCC.exe -ErrorAction SilentlyContinue
if ($cmd) { $iscc = $cmd.Source }
if (-not $iscc) {
    foreach ($p in @(
        "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe",
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe"
    )) { if (Test-Path $p) { $iscc = $p; break } }
}

if ($iscc) {
    Write-Host "-> compiling installer (Inno Setup)" -ForegroundColor Cyan
    & $iscc "/DVersion=$Version" "packaging\windows\vajra.iss"
    if ($LASTEXITCODE -ne 0) { throw "ISCC failed" }
    Get-ChildItem "dist\releases\VajraSetup-*.exe" |
        ForEach-Object { Write-Host "  ok $($_.FullName)" -ForegroundColor Green }
} else {
    Write-Host ". Inno Setup (ISCC) not found - skipping installer" -ForegroundColor Yellow
    Write-Host "  install it: choco install innosetup" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Done. Artifacts in dist\" -ForegroundColor Magenta
Pop-Location
