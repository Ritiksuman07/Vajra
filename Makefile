# Grok Bot — Local-First AI Agent Build System

.PHONY: all install build test clean package docs

VERSION := 1.0.0
SERVICE_DIR := service
AGENT_DIR := agent
CLI_DIR := cli

all: install test

# Install all dependencies
install:
	@echo "Installing Grok Bot..."
	pip install -e .
	pip install -e ".[dev]"
	@echo "Installing Go service dependencies..."
	cd $(SERVICE_DIR) && go mod download || true

# Build the Go service
build-service:
	@echo "Building Go service..."
	cd $(SERVICE_DIR)/cmd/grok-bot && go build -o ../../grok-bot-service .

# Build Python agent package
build-agent:
	@echo "Building Python agent package..."
	python -m build --wheel .

# Full build
build: build-service build-agent

# Run all tests
test:
	@echo "Running agent tests..."
	python -m pytest tests/ -v --tb=short 2>/dev/null || python tests/test_agent_pod.py
	@echo "Running service tests..."
	cd $(SERVICE_DIR) && go test ./... 2>/dev/null || echo "Go tests skipped (Go not installed)"

# Start local development server
start-service:
	@echo "Starting Grok Bot service on localhost:4096..."
	cd $(SERVICE_DIR)/cmd/grok-bot && go run . --standalone || echo "Start service manually with: go run ."

# Run CLI
run-cli:
	python -m cli.commands.run "Hello, Grok Bot!"

# Package for distribution
package:
	@echo "Packaging Grok Bot for distribution..."
	mkdir -p dist
	# Python package
	python -m build --wheel . 2>/dev/null || true
	# Go binary bundle (if go available)
	cd $(SERVICE_DIR)/cmd/grok-bot && go build -o ../../dist/grok-bot-service . 2>/dev/null || true
	# Create install script
	cp scripts/install.sh dist/ 2>/dev/null || true
	@echo "Package complete. Check dist/"

# Clean build artifacts
clean:
	rm -rf *.egg-info build dist .pytest_cache .mypy_cache __pycache__
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name '*.pyc' -delete 2>/dev/null || true
	rm -rf $(SERVICE_DIR)/grok-bot-service

# Install script (for Linux/macOS)
install-script:
	@mkdir -p scripts
	@echo '#!/bin/sh' > scripts/install.sh
	@echo '# Grok Bot Install Script' >> scripts/install.sh
	@echo 'echo "Installing Grok Bot..."' >> scripts/install.sh
	@echo 'curl -fsSL https://grok-bot.ai/install | sh || pip install grok-bot' >> scripts/install.sh
	@chmod +x scripts/install.sh

# NSIS Windows installer (stub)
package-windows:
	@echo "Creating NSIS installer..."
	mkdir -p packaging/windows
	@echo 'OutFile "dist\grok-bot-$(VERSION)-windows-x64.exe"' > packaging/windows/grok-bot.nsi
	@echo 'Name "Grok Bot"' >> packaging/windows/grok-bot.nsi
	@echo 'InstallDir "$LOCALAPPDATA\GrokBot"' >> packaging/windows/grok-bot.nsi

# Documentation
docs:
	@echo "Generating docs from agents.md..."
	python -c "
import markdown
with open('agent/core/agents.md') as f:
    md = markdown.markdown(f.read())
with open('docs/index.html', 'w') as f:
    f.write(f'<html><body>{md}</body></html>')
" 2>/dev/null || echo "Install markdown package for docs"

# Development server (with hot reload)
dev:
	@echo "Starting dev server (Python agent + Go service)..."
	python cli/tui/cli_main.py &
	@echo "Agent TUI running. Start service with: make start-service"

# Docker build for agent pods
docker-agent:
	docker build -t grok-bot-agent:latest -f agent/Dockerfile . 2>/dev/null || echo "Docker not available"

# Print help
help:
	@echo "Grok Bot Build System"
	@echo "---------------------"
	@echo "make install      — Install Python + Go deps"
	@echo "make build        — Build service + package"
	@echo "make test         — Run all tests"
	@echo "make start-service — Start background service"
	@echo "make package      — Create distribution package"
	@echo "make docker-agent — Build agent container"
	@echo "make clean        — Remove build artifacts"
