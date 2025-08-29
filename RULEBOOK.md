# GOLLM Project Rulebook

## Table of Contents
- [Project Vision](#project-vision)
- [Architecture Principles](#architecture-principles)
- [Code Quality Standards](#code-quality-standards)
- [Security Requirements](#security-requirements)
- [Performance Standards](#performance-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)
- [Development Workflow](#development-workflow)
- [Go-Specific Standards](#go-specific-standards)
- [Anti-Patterns](#anti-patterns)

## Project Vision

GOLLM is a high-performance, cross-platform LLM client built in Go that prioritizes:
- **Zero-dependency binary distribution**
- **Sub-millisecond startup time**
- **Memory efficiency under 10MB baseline**
- **Cross-platform compatibility** (Linux, macOS, Windows on amd64/arm64)
- **Extensible provider architecture**
- **MCP (Model Context Protocol) integration**
- **Enterprise-grade security**

## Architecture Principles

### 1. Layered Architecture
```
┌─────────────────┐
│   CLI Layer     │ ← User interface, command parsing
├─────────────────┤
│  Service Layer  │ ← Business logic, orchestration  
├─────────────────┤
│  Provider Layer │ ← API integrations, protocols
├─────────────────┤
│ Transport Layer │ ← HTTP, WebSocket, gRPC
├─────────────────┤
│   Core Layer    │ ← Shared types, utilities
└─────────────────┘
```

### 2. Dependency Injection
- Use interfaces for all external dependencies
- Constructor injection with dependency containers
- No global state or singletons except for configuration

### 3. Error Handling Strategy
```go
// REQUIRED: Wrap all errors with context
return fmt.Errorf("failed to create chat completion for model %s: %w", model, err)

// REQUIRED: Use typed errors for business logic
type ValidationError struct {
    Field string
    Value interface{}
    Err   error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s (value: %v): %v", e.Field, e.Value, e.Err)
}
```

### 4. Configuration Management
- Environment variables override config files
- Hierarchical configuration: CLI args > ENV > config file > defaults
- Validation at startup with clear error messages
- Hot-reload capability for non-critical settings

## Code Quality Standards

### 1. File Organization
```
internal/
├── cli/           # Command-line interface
│   ├── commands/  # Individual command implementations
│   ├── flags/     # Flag definitions and parsing
│   └── ui/        # Terminal UI components
├── config/        # Configuration management
├── core/          # Core types and interfaces
├── providers/     # LLM provider implementations
│   ├── openai/
│   ├── anthropic/
│   ├── ollama/
│   └── custom/
├── transport/     # HTTP/WebSocket/gRPC clients
├── security/      # Authentication, encryption
├── mcp/          # Model Context Protocol
└── utils/        # Shared utilities
```

### 2. Naming Conventions
- **Packages**: lowercase, single word when possible (`config`, not `configuration`)
- **Interfaces**: end with `-er` suffix when representing behavior (`Provider`, `Streamer`)
- **Structs**: PascalCase, descriptive (`ChatCompletionRequest`, not `CCR`)
- **Methods**: camelCase, action verbs (`CreateCompletion`, `ValidateConfig`)
- **Constants**: ALL_CAPS with underscores for package-level, PascalCase for exported

### 3. Function Design Rules
```go
// REQUIRED: Functions must be < 50 lines
// REQUIRED: Max 4 parameters, use structs for more
// REQUIRED: Single responsibility principle

// GOOD:
func (p *Provider) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

// BAD:
func CreateCompletion(apiKey, model, prompt, system string, maxTokens int, temperature float64, stream bool) (string, error)
```

### 4. Struct Design Rules
```go
// REQUIRED: Use composition over inheritance
// REQUIRED: Embed interfaces, not structs
// REQUIRED: Tag all JSON fields explicitly

type CompletionRequest struct {
    Model       string    `json:"model" validate:"required"`
    Messages    []Message `json:"messages" validate:"required,min=1"`
    MaxTokens   int       `json:"max_tokens,omitempty" validate:"min=1,max=8192"`
    Temperature float64   `json:"temperature,omitempty" validate:"min=0,max=2"`
    Stream      bool      `json:"stream,omitempty"`
    
    // Embedding for extensibility
    Metadata
}

type Metadata struct {
    UserID    string            `json:"user_id,omitempty"`
    RequestID string            `json:"request_id,omitempty"`
    Tags      map[string]string `json:"tags,omitempty"`
}
```

## Security Requirements

### 1. API Key Management
```go
// REQUIRED: Never log API keys
// REQUIRED: Use secure memory clearing
// REQUIRED: Implement key rotation

type SecureString struct {
    data []byte
}

func (s *SecureString) Clear() {
    for i := range s.data {
        s.data[i] = 0
    }
}

func (s *SecureString) String() string {
    return "***REDACTED***"
}
```

### 2. Input Validation
```go
// REQUIRED: Validate all user inputs
// REQUIRED: Use allowlists over denylists
// REQUIRED: Sanitize for logging

func ValidateModel(model string) error {
    if len(model) == 0 {
        return &ValidationError{Field: "model", Err: errors.New("cannot be empty")}
    }
    if len(model) > 100 {
        return &ValidationError{Field: "model", Err: errors.New("exceeds maximum length of 100")}
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`).MatchString(model) {
        return &ValidationError{Field: "model", Err: errors.New("contains invalid characters")}
    }
    return nil
}
```

### 3. Network Security
- **TLS 1.3 minimum** for all external connections
- **Certificate pinning** for known providers
- **Request timeout** maximum 5 minutes
- **Rate limiting** per provider
- **Circuit breaker** pattern for resilience

## Performance Standards

### 1. Memory Management
```go
// REQUIRED: Use object pools for frequently allocated types
var messagePool = sync.Pool{
    New: func() interface{} {
        return &Message{}
    },
}

func GetMessage() *Message {
    return messagePool.Get().(*Message)
}

func PutMessage(m *Message) {
    m.Reset() // Clear fields
    messagePool.Put(m)
}
```

### 2. Concurrency Rules
```go
// REQUIRED: Always use context for cancellation
// REQUIRED: Limit concurrent goroutines
// REQUIRED: Use buffered channels appropriately

func (p *Provider) ProcessBatch(ctx context.Context, requests []Request) ([]Response, error) {
    const maxConcurrency = 10
    sem := make(chan struct{}, maxConcurrency)
    
    var wg sync.WaitGroup
    results := make([]Response, len(requests))
    errors := make([]error, len(requests))
    
    for i, req := range requests {
        wg.Add(1)
        go func(idx int, request Request) {
            defer wg.Done()
            
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
                results[idx], errors[idx] = p.ProcessSingle(ctx, request)
            case <-ctx.Done():
                errors[idx] = ctx.Err()
            }
        }(i, req)
    }
    
    wg.Wait()
    return results, combineErrors(errors)
}
```

### 3. Network Optimization
- **HTTP/2 connection reuse**
- **Request pipelining** where supported
- **Compression** (gzip/brotli) for requests/responses
- **Connection pooling** with proper cleanup

## Testing Requirements

### 1. Test Coverage
- **Minimum 85% code coverage**
- **100% coverage** for core business logic
- **All public APIs** must have tests
- **Error paths** must be tested

### 2. Test Structure
```go
func TestProvider_CreateCompletion(t *testing.T) {
    tests := []struct {
        name           string
        request        CompletionRequest
        mockResponse   string
        expectedError  string
        expectedResult *CompletionResponse
    }{
        {
            name: "successful completion",
            request: CompletionRequest{
                Model: "gpt-3.5-turbo",
                Messages: []Message{{Role: "user", Content: "Hello"}},
            },
            mockResponse: `{"choices":[{"message":{"content":"Hi there!"}}]}`,
            expectedResult: &CompletionResponse{
                Choices: []Choice{{Message: Message{Content: "Hi there!"}}},
            },
        },
        {
            name: "invalid model name",
            request: CompletionRequest{
                Model: "",
                Messages: []Message{{Role: "user", Content: "Hello"}},
            },
            expectedError: "validation failed for field model",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 3. Integration Tests
```go
// REQUIRED: Test with real providers in CI
// REQUIRED: Mock external dependencies
// REQUIRED: Test timeout and cancellation scenarios

func TestProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    provider := setupRealProvider(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Test real API call
}
```

## Documentation Standards

### 1. Code Documentation
```go
// Package provider implements LLM provider integrations.
//
// This package provides a unified interface for interacting with various
// Large Language Model providers including OpenAI, Anthropic, and local models.
//
// Example usage:
//
//    provider := openai.New(config.OpenAIConfig{
//        APIKey: "sk-...",
//    })
//    
//    response, err := provider.CreateCompletion(ctx, CompletionRequest{
//        Model: "gpt-3.5-turbo",
//        Messages: []Message{{Role: "user", Content: "Hello"}},
//    })
package provider

// CreateCompletion creates a chat completion using the specified model.
//
// The request is validated before sending to ensure all required fields
// are present and within acceptable ranges. The context is used for
// cancellation and timeout control.
//
// Returns an error if:
//   - The request fails validation
//   - The API returns an error response
//   - The context is cancelled or times out
//
// Example:
//
//    req := CompletionRequest{
//        Model: "gpt-3.5-turbo",
//        Messages: []Message{{Role: "user", Content: "What is Go?"}},
//        MaxTokens: 150,
//    }
//    
//    resp, err := provider.CreateCompletion(ctx, req)
//    if err != nil {
//        log.Fatal(err)
//    }
//    
//    fmt.Println(resp.Choices[0].Message.Content)
func (p *Provider) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
    // Implementation
}
```

### 2. API Documentation
- **OpenAPI 3.0 specification** for all HTTP endpoints
- **Examples** for every request/response type
- **Error code documentation** with solutions
- **Rate limiting** and **authentication** details

## Development Workflow

### 1. Git Workflow
```bash
# Branch naming convention
feature/add-anthropic-provider
fix/memory-leak-in-streaming
refactor/simplify-config-loading
docs/update-api-documentation

# Commit message format
type(scope): description

feat(providers): add Anthropic Claude support
fix(streaming): resolve goroutine leak in WebSocket handling
docs(api): add examples for batch processing
test(integration): add timeout scenarios
```

### 2. Code Review Checklist
- [ ] **Performance**: No unnecessary allocations, efficient algorithms
- [ ] **Security**: Input validation, no credential exposure
- [ ] **Error handling**: Proper wrapping, meaningful messages
- [ ] **Testing**: Adequate coverage, edge cases handled
- [ ] **Documentation**: Public APIs documented, examples provided
- [ ] **Concurrency**: Race conditions checked, proper synchronization
- [ ] **Memory safety**: No leaks, proper cleanup

### 3. CI/CD Pipeline
```yaml
# .github/workflows/ci.yml
- Linting (golangci-lint)
- Testing (all Go versions)
- Security scanning (gosec)
- Vulnerability checking (nancy)
- Performance benchmarking
- Cross-compilation verification
- Integration testing
- Documentation generation
```

## Go-Specific Standards

### 1. Project Layout
```
cmd/           # Main applications
internal/      # Private application code
pkg/           # Library code (if needed)
api/           # API definitions (OpenAPI, protobuf)
web/           # Web application files
configs/       # Configuration files
deployments/   # Deployment configurations
test/          # Additional test data
docs/          # Documentation
tools/         # Supporting tools
examples/      # Example code
```

### 2. Interface Design
```go
// REQUIRED: Keep interfaces small and focused
type Provider interface {
    CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    StreamCompletion(ctx context.Context, req CompletionRequest) (<-chan CompletionChunk, error)
}

// REQUIRED: Accept interfaces, return structs
func NewClient(provider Provider, config Config) *Client {
    return &Client{provider: provider, config: config}
}
```

### 3. Error Handling
```go
// REQUIRED: Use custom error types for business logic
type APIError struct {
    StatusCode int    `json:"status_code"`
    Message    string `json:"message"`
    Code       string `json:"code"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
}

// REQUIRED: Check if error is of specific type
if apiErr := new(APIError); errors.As(err, &apiErr) {
    if apiErr.StatusCode == 429 {
        // Handle rate limiting
    }
}
```

### 4. Context Usage
```go
// REQUIRED: Always accept context as first parameter
// REQUIRED: Propagate context through call chains
// REQUIRED: Check context cancellation in loops

func (c *Client) ProcessBatch(ctx context.Context, items []Item) error {
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := c.ProcessItem(ctx, item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

## Anti-Patterns

### 1. Forbidden Practices
```go
// NEVER: Global state
var GlobalClient *http.Client

// NEVER: Unhandled errors
resp, _ := http.Get(url)

// NEVER: Empty catch-all interfaces
func Process(data interface{}) error

// NEVER: Goroutines without context
go func() {
    for {
        // This goroutine can never be stopped
        doWork()
    }
}()

// NEVER: Mutex in structs that will be copied
type BadStruct struct {
    mu sync.Mutex // Will be copied by value
    data map[string]string
}
```

### 2. Performance Anti-Patterns
```go
// NEVER: String concatenation in loops
var result string
for _, item := range items {
    result += item // Inefficient, use strings.Builder
}

// NEVER: Unnecessary JSON marshaling
func LogRequest(req Request) {
    data, _ := json.Marshal(req) // Expensive, use structured logging
    log.Info(string(data))
}

// NEVER: Creating goroutines without bounds
for _, item := range items {
    go processItem(item) // Can create thousands of goroutines
}
```

### 3. Security Anti-Patterns
```go
// NEVER: SQL injection (even though we don't use SQL directly)
query := fmt.Sprintf("SELECT * FROM models WHERE name = '%s'", userInput)

// NEVER: Logging sensitive data
log.Infof("API Key: %s", config.APIKey)

// NEVER: Ignoring TLS verification
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    },
}
```

## Quality Gates

### 1. Before Commit
- All tests pass
- Linter warnings addressed
- Code coverage maintained
- Documentation updated

### 2. Before Merge
- Code review completed
- Integration tests pass
- Performance benchmarks stable
- Security scan clean

### 3. Before Release
- Full test suite passes
- Cross-platform