# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

GOLLM is a high-performance command-line interface for Large Language Models (LLMs) built in Go. It provides a unified interface to interact with multiple LLM providers while delivering exceptional performance, security, and ease of use.

## Common Development Commands

### Building

```bash
# Build binary for current platform
make build

# Build with debug information
make build-debug

# Build with race detection enabled
make build-race

# Build for all supported platforms
make build-all

# Build specific platform (example for linux/amd64)
GOOS=linux GOARCH=amd64 make build-platform PLATFORM=linux/amd64
```

### Testing

```bash
# Run unit tests
make test

# Run tests in short mode (faster)
make test-short

# Run integration tests (requires API keys)
make test-integration

# Run end-to-end tests
make test-e2e

# Run all tests
make test-all

# Run benchmarks
make benchmark

# Generate and view test coverage
make coverage
# Check if coverage meets 75% threshold
make coverage-check

# Run a single test
go test -v -run TestFunctionName ./internal/core/
```

### Code Quality

```bash
# Format code (uses gofmt and goimports)
make fmt

# Run linter (golangci-lint)
make lint

# Run linter with auto-fix
make lint-fix

# Run go vet
make vet

# Run staticcheck
make staticcheck

# Quick development cycle (format, lint, test)
make dev

# Quick test cycle (format and short tests)
make quick
```

### Security

```bash
# Run all security checks
make security

# Run security scanner (gosec)
make security-scan

# Check for known vulnerabilities
make security-vulns

# Audit dependencies for vulnerabilities
make security-deps
```

### Installation

```bash
# Install binary to GOPATH/bin
make install

# Install binary to /usr/local/bin (requires sudo)
make install-local

# Setup development environment (installs required tools)
make setup

# Download and verify dependencies
make deps
```

### Docker

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Release Preparation

```bash
# Prepare release (builds for all platforms, runs tests, security checks)
make release-prepare

# Generate checksums for release files
make release-checksums
```

## Architecture Overview

### Package Structure

The codebase follows a clean architecture pattern with clear separation of concerns:

```
gollm-cli/
├── cmd/gollm/          # Main entry point
│   └── main.go         # Application bootstrap and signal handling
├── internal/           # Private application code
│   ├── cli/           # Command-line interface layer
│   │   ├── commands/  # Individual CLI commands
│   │   │   ├── chat.go          # Chat completion command
│   │   │   ├── complete.go      # Code completion command
│   │   │   ├── interactive.go   # Interactive mode
│   │   │   ├── interactive_enhanced.go  # Enhanced interactive mode
│   │   │   ├── tui.go           # Terminal UI mode
│   │   │   ├── config.go        # Configuration management
│   │   │   ├── profile.go       # Profile management
│   │   │   ├── models.go        # Model listing/info
│   │   │   ├── benchmark.go     # Performance benchmarking
│   │   │   └── version.go       # Version information
│   │   ├── root.go    # Root command setup and global flags
│   │   └── logo.go    # ASCII art logo generation
│   ├── config/        # Configuration management
│   │   ├── config.go  # Configuration loading and validation
│   │   └── profiles.go # Pre-configured profiles
│   ├── core/          # Core business logic
│   │   ├── types.go   # Core types and interfaces
│   │   ├── provider.go # Provider interface and registry
│   │   └── integration.go # Integration patterns
│   ├── providers/     # LLM provider implementations
│   │   ├── openai/    # OpenAI provider
│   │   ├── anthropic/ # Anthropic provider
│   │   ├── deepseek/  # DeepSeek provider
│   │   ├── gemini/    # Google Gemini provider
│   │   ├── openrouter/ # OpenRouter provider
│   │   ├── ollama/    # Local Ollama provider
│   │   └── mock/      # Mock provider for testing
│   ├── display/       # Output formatting and display
│   │   ├── display.go # Display utilities
│   │   ├── formatter.go # Response formatting
│   │   └── syntax.go  # Syntax highlighting
│   ├── security/      # Security features
│   │   ├── validators.go # Input validation
│   │   └── audit_test.go # Security audit tests
│   ├── tui/           # Terminal UI components
│   │   ├── ai_service.go # AI service integration
│   │   ├── cyberpunk.go  # Cyberpunk theme
│   │   ├── minimal.go    # Minimal theme
│   │   └── simple.go     # Simple theme
│   └── version/       # Version information
└── test/              # Test suites
    ├── integration/   # Integration tests
    └── e2e/          # End-to-end tests
```

### Key Design Patterns

1. **Provider Registry Pattern**: All LLM providers implement a common `Provider` interface defined in `internal/core/provider.go`. The registry pattern allows dynamic provider registration and selection.

2. **Command Pattern**: Each CLI command is isolated in its own file under `internal/cli/commands/`, following the Cobra command pattern with consistent initialization and execution.

3. **Configuration Hierarchy**: Configuration follows a precedence order:
   - Command-line flags (highest priority)
   - Environment variables (GOLLM_* prefix)
   - Configuration file (~/.gollm/config.yaml)
   - Default values (lowest priority)

4. **Streaming Response Handling**: All providers support both streaming and non-streaming modes, with the streaming implementation handling real-time token generation display.

5. **Security-First Design**: 
   - Input validation layer (`internal/security/validators.go`)
   - Credential management with automatic memory clearing
   - Rate limiting and circuit breaker patterns
   - Comprehensive audit logging

### Provider Interface

All providers must implement the core `Provider` interface:

```go
type Provider interface {
    CreateCompletion(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)
    CreateCompletionStream(ctx context.Context, request CompletionRequest) (<-chan StreamChunk, error)
    ListModels(ctx context.Context) ([]Model, error)
    GetModel(ctx context.Context, modelID string) (*Model, error)
    ValidateCredentials(ctx context.Context) error
}
```

### Adding a New Provider

1. Create a new package under `internal/providers/yourprovider/`
2. Implement the `Provider` interface
3. Register the provider in the provider registry
4. Add configuration support in `internal/config/config.go`
5. Add tests following the existing pattern

### Adding a New Command

1. Create a new file in `internal/cli/commands/`
2. Define the command using Cobra structure
3. Add the command to the root command in `internal/cli/root.go`
4. Add corresponding tests

## Testing Strategy

- **Unit Tests**: Cover individual functions and methods (target: >75% coverage)
- **Integration Tests**: Test provider integrations with real APIs (requires API keys)
- **E2E Tests**: Test complete command workflows
- **Benchmark Tests**: Performance validation for critical paths
- **Security Tests**: Comprehensive security audit suite

## Performance Targets

The project has strict performance requirements:
- Startup time: <100ms
- Memory usage: <10MB per operation
- Concurrent request handling: 1000+ requests
- Binary size: ~15MB (optimized)

## CI/CD Pipeline

GitHub Actions workflows handle:
- **CI** (`ci.yml`): Tests, linting, security scanning, coverage checks on every push/PR
- **Release** (`release.yml`): Automated multi-platform builds, Docker images, and package creation for tagged releases

## Configuration

The application uses Viper for configuration management with support for:
- YAML configuration files
- Environment variables (GOLLM_* prefix)
- Command-line flag overrides
- Profile-based configurations for different use cases

Example configuration location: `~/.gollm/config.yaml`

## Dependencies

Key dependencies managed via Go modules:
- **cobra**: CLI framework
- **viper**: Configuration management
- **bubbletea**: Terminal UI framework
- **lipgloss**: Terminal styling
- **chroma**: Syntax highlighting
- **progressbar**: Progress indicators

Run `go mod tidy` after adding new dependencies to keep go.mod clean.
