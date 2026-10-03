# PowerShell build script for Windows packaging
# Usage: pwsh scripts\build-windows.ps1
# Requires: Go 1.22+, Python 3.11+, Ollama (optional), Inno Setup (for .iss compile)

param(
    [switch]$NoModels,       # Skip model pull
    [switch]$InstallerOnly,  # Only create installer (assumes binaries exist)
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
$Dist = Join-Path $Root "dist"
$BinDir = Join-Path $Dist "bin"
$PythonDir = Join-Path $Dist "python"
$OllamaDir = Join-Path $Dist "ollama"

Write-Host "=== Vajra Windows Build ===" -ForegroundColor Cyan
Write-Host "Root: $Root"
Write-Host "Dist: $Dist"
Write-Host ""

# ---- 1. Build Go service ----
Write-Host "[1/5] Building Go service..." -ForegroundColor Yellow
Push-Location (Join-Path $Root "service")
go build -ldflags "-X main.version=$Version" -o (Join-Path $Dist "bin\vajra-service.exe") ./cmd/vajra/
Pop-Location

# ---- 2. Build Go CLI ----
Write-Host "[2/5] Building Go CLI..." -ForegroundColor Yellow
Push-Location (Join-Path $Root "cli")
go build -ldflags "-X main.version=$Version" -o (Join-Path $Dist "bin\vajra.exe") .
Pop-Location

# ---- 3. Prepare embedded Python runtime ----
Write-Host "[3/5] Preparing embedded Python runtime..." -ForegroundColor Yellow
if (-not (Test-Path $PythonDir)) {
    New-Item -ItemType Directory -Path $PythonDir -Force | Out-Null
}
# Copy python embed (python-3.x.x-embed-amd64.zip)
$PyZip = "C:\Python314\python-3.14-embed-amd64.zip"  # Adjust to your Python path
if (Test-Path $PyZip) {
    Expand-Archive -Path $PyZip -DestinationPath $PythonDir -Force
} else {
    Write-Host "  Python embed not found at $PyZip - copying from installed Python..." -ForegroundColor Yellow
    robocopy "C:\Python314" $PythonDir *.dll *.pyd *.py /E /NFL /NDL /NJH /NJS 2>$null
}

# ---- 4. Prepare Ollama ----
Write-Host "[4/5] Preparing Ollama..." -ForegroundColor Yellow
if (-not (Test-Path $OllamaDir)) {
    New-Item -ItemType Directory -Path $OllamaDir -Force | Out-Null
}
# Copy Ollama binary (downloaded via ollama install)
$OllamaExe = "$env:USERPROFILE\.ollama\ollama.exe"
if (Test-Path $OllamaExe) {
    Copy-Item $OllamaExe "$OllamaDir\ollama.exe" -Force
} else {
    Write-Host "  Ollama not found at $OllamaExe - downloading..." -ForegroundColor Yellow
    # Download Ollama for Windows
    Invoke-WebRequest -Uri "https://ollama.ai/download/windows" -OutFile "$OllamaDir\ollama.exe" -ErrorAction SilentlyContinue
}

# ---- 5. Copy config & scripts ----
Write-Host "[5/5] Copying config & scripts..." -ForegroundColor Yellow
Copy-Item (Join-Path $Root "config\default.yaml") (Join-Path $Dist "config\vajra.yaml") -Force -ErrorAction SilentlyContinue
Copy-Item (Join-Path $Root "scripts\install.ps1") (Join-Path $Dist "install.ps1") -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "=== Build Complete ===" -ForegroundColor Green
Write-Host "Dist contents:"
Get-ChildItem $Dist | ForEach-Object { Write-Host "  $($_.Name) ($($_.Length) bytes)" }
Write-Host ""
Write-Host "To create installer, run: ISCC packaging\windows\vajra.iss" -ForegroundColor Cyan
