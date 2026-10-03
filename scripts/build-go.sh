#!/bin/bash
# Build Vajra Go Service Binary
# Requires Go 1.22+ installed

set -e

echo "Building Vajra service binary..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "Go not installed. Installing via curl..."
    curl -fsSL https://go.dev/dl/go1.22.5.linux-amd64.tar.gz | tar -C /usr/local -xzf - 2>/dev/null || echo "Please install Go manually: https://go.dev/dl/"
    export PATH=$PATH:/usr/local/go/bin
fi

echo "Go version: $(go version)"

# Build
cd "$(dirname "$0")/service/cmd/vajra"
go mod tidy 2>/dev/null || true
go build -ldflags "-X main.version=$(cat ../../VERSION 2>/dev/null || echo 1.0.0)" -o ../../../dist/vajra-service .

echo "Binary built: $(pwd)/../../../dist/vajra-service"
chmod +x "../../../dist/vajra-service"
ls -lh "../../../dist/vajra-service"
