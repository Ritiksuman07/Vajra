# Vajra Agent Pod Container
# Each agent pod runs in its own isolated container with no network access by default.

FROM python:3.11-slim

LABEL maintainer="Vajra Contributors"
LABEL description="Vajra Agent Pod — local-first AI agent with MCP tools"

# System dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user for security
RUN useradd -m -u 1000 vajra
WORKDIR /app

# Install Python dependencies
COPY agent/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt 2>/dev/null || pip install pydantic loguru

# Copy agent code
COPY agent/ /app/agent/

# Create workspace and config directories
RUN mkdir -p /workspace /config /logs
RUN chown -R vajra:vajra /app /workspace /config /logs

USER vajra

# Default environment
ENV VAJRA_MODEL=llama3.1:8b
ENV VAJRA_OLLAMA_HOST=http://localhost:11434
ENV VAJRA_WORKSPACE=/workspace

# Start agent pod (entrypoint handles --role, --model flags)
ENTRYPOINT ["python", "/app/agent/core/agent.py"]
