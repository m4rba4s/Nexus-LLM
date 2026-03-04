# Build stage
FROM golang:1.22-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y git gcc libc-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary with zero-trust optimizations
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o bin/nexus-llm ./cmd/gollm

# Execution stage
FROM debian:bookworm-slim

# Install CA certs, dependencies for Playwright headless mode
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    radare2 \
    && rm -rf /var/lib/apt/lists/*

# Add non-root user for security (Drop Privileges)
RUN useradd -m -s /bin/bash appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/nexus-llm .

# Configure playwright cache location and permissions so appuser can install browser
ENV PLAYWRIGHT_BROWSERS_PATH=/home/appuser/.cache/ms-playwright
RUN mkdir -p /home/appuser/.cache/ms-playwright && chown -R appuser:appuser /home/appuser/.cache /app

# eBPF notes: To enable the Ring-0 sandbox, run this container with:
# --cap-add=CAP_SYS_ADMIN --cap-add=CAP_BPF
USER appuser

# Run in Telegram Daemon mode by default for prod
CMD ["./nexus-llm", "-mode=tg"]
