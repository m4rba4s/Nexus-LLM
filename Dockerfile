# Multi-stage build for GOLLM CLI
# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /src

# Create non-root user for build
RUN addgroup -g 1001 -S gollm && \
    adduser -u 1001 -S gollm -G gollm

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the application
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -X 'github.com/yourusername/gollm/internal/version.Version=${VERSION}' -X 'github.com/yourusername/gollm/internal/version.Commit=${COMMIT}' -X 'github.com/yourusername/gollm/internal/version.BuildTime=${BUILD_TIME}'" \
    -o gollm \
    ./cmd/gollm

# Verify the binary
RUN ./gollm version

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S gollm && \
    adduser -u 1001 -S gollm -G gollm

# Create necessary directories
RUN mkdir -p /home/gollm/.gollm && \
    chown -R gollm:gollm /home/gollm

# Copy the binary from builder stage
COPY --from=builder --chown=gollm:gollm /src/gollm /usr/local/bin/gollm

# Copy additional files
COPY --chown=gollm:gollm README.md LICENSE /usr/share/doc/gollm/

# Set proper permissions
RUN chmod 755 /usr/local/bin/gollm

# Switch to non-root user
USER gollm

# Set working directory
WORKDIR /home/gollm

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}"
ENV GOLLM_CONFIG_DIR="/home/gollm/.gollm"
ENV GOLLM_LOG_LEVEL="info"

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD gollm version --short || exit 1

# Metadata labels
LABEL org.opencontainers.image.title="GOLLM" \
      org.opencontainers.image.description="High-performance CLI for Large Language Models" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/yourusername/gollm" \
      org.opencontainers.image.documentation="https://docs.gollm.dev" \
      org.opencontainers.image.vendor="GOLLM Team" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.revision="${COMMIT}"

# Default command
ENTRYPOINT ["gollm"]
CMD ["--help"]

# Expose common ports (if running as server in future)
# EXPOSE 8080

# Volume for configuration and data
VOLUME ["/home/gollm/.gollm"]
