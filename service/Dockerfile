# Build stage for Go service
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY service/ service/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /vajra-service ./cmd/vajra

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite curl
COPY --from=builder /vajra-service /usr/local/bin/vajra-service

# Create default config dir
RUN mkdir -p /config /data /logs

# Environment
ENV VAJRA_SERVICE_PORT=4096
ENV VAJRA_DATA_DIR=/data
ENV VAJRA_CONFIG_DIR=/config

# Expose service port
EXPOSE 4096

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -f http://localhost:4096/health || exit 1

ENTRYPOINT ["vajra-service"]
