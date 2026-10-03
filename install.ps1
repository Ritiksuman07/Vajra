# Vajra Windows Quickstart Script
# Usage: .\install.ps1 (PowerShell, run as Admin recommended)

[CmdletBinding()]
param(
    [switch]$Model,        # Download default model (llama3.1:8b)
    [switch]$SkipModel,    # Skip model download
    [switch]$Standalone    # Just install without auto-starting
)

$ErrorActionPreference = "Continue"
$VajraHome = "$env:LOCALAPPDATA\Vajra"
$DataDir = "$VajraHome\data"

Write-Host "=== Vajra Installation ===" -ForegroundColor Cyan

# Create directories
Write-Host "[1/4] Creating directories..." -ForegroundColor Yellow
foreach ($d in @("$VajraHome", "$DataDir\workspace", "$DataDir\models", "$VajraHome\logs")) {
    if (-not (Test-Path $d)) { New-Item -ItemType Directory -Path $d -Force | Out-Null }
}

# Verify binary exists
if (-not (Test-Path "$VajraHome\vajra.exe")) {
    Write-Host "[2/4] No binary found. Please extract vajra-x64.zip to $VajraHome first." -ForegroundColor Red
    exit 1
}

# Configure PATH (optional)
Write-Host "[3/4] Updating PATH..." -ForegroundColor Yellow
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "User")
Add-Content "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Start Up\vajra.bat" "`"$(Split-Path "$VajraHome\vajra.exe" -Parent)\vajra.exe`"

# Start Ollama if installed
if (Get-Command ollama -ErrorAction SilentlyContinue) {
    ollama serve -c 2 > "$DataDir\ollama.log" 2>&1 &
    Start-Sleep -Seconds 3
    Write-Host "  Ollama running" -ForegroundColor Green
}

# Start Vajra service
Write-Host "[4/4] Starting Vajra service..." -ForegroundColor Yellow
& "$VajraHome\vajra-service.exe" --daemon
Write-Host "  Service started on port 4096" -ForegroundColor Green

# Install model
if (-not $SkipModel) {
    Write-Host "Downloading model..." -ForegroundColor Yellow
    & ollama pull llama3.1:8b
}

Write-Host ""
Write-Host "=== Vajra Installed ===" -ForegroundColor Green
Write-Host "Run 'vajra' to start the TUI"
Write-Host "Run 'vajra run \"task\"' for quick tasks"
