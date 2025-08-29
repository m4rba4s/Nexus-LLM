# GOLLM Developer API Reference

Quick reference for key APIs, types, and patterns used in GOLLM development.

## 🏗️ Core Architecture

### Layer Dependencies
```
CLI Layer → Service Layer → Provider Layer → Transport Layer → Core Layer
```

### Key Packages
- `internal/core` - Core types, interfaces, and utilities
- `internal/config` - Configuration management
- `internal/providers` - LLM provider implementations
- `internal/transport` - HTTP/WebSocket clients
- `internal/cli` - Command-line interface

## 🔧 Core Types & Interfaces

### Provider Interface
```go
type Provider interface {
    Name() string
    CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
    GetModels(ctx context.Context) ([]Model, error)
    ValidateConfig() error
}
```

### CompletionRequest
```go
type CompletionRequest struct {
    // Required
    Model    string    `json:"model" validate:"required"`
    Messages []Message `json:"messages" validate:"required,min=1"`
    
    // Optional parameters
    MaxTokens        *int     `json:"max_tokens,omitempty"`
    Temperature      *float64 `json:"temperature,omitempty"`
    TopP             *float64 `json:"top_p,omitempty"`
    FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
    PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
    
    // Metadata
    RequestID string            `json:"request_id,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
    CreatedAt time.Time         `json:"created_at"`
}

// Usage
req := &core.CompletionRequest{
    Model: "gpt-3.5-turbo",
    Messages: []core.Message{
        {Role: "user", Content: "Hello, world!"},
    },
    MaxTokens: &[]int{150}[0],
}
```

### Message Types
```go
type Message struct {
    Role       string                 `json:"role" validate:"required,oneof=system user assistant tool"`
    Content    string                 `json:"content,omitempty"`
    Name       string                 `json:"name,omitempty"`
    ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
    ToolCallID string                 `json:"tool_call_id,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Common message creation patterns
systemMsg := core.Message{Role: "system", Content: "You are a helpful assistant"}
userMsg := core.Message{Role: "user", Content: "What is Go?"}
assistantMsg := core.Message{Role: "assistant", Content: "Go is a programming language..."}
```

## ⚙️ Configuration System

### Config Structure
```go
type Config struct {
    DefaultProvider string                    `mapstructure:"default_provider"`
    Providers       map[string]ProviderConfig `mapstructure:"providers"`
    Settings        GlobalSettings            `mapstructure:"settings"`
    Features        FeatureFlags              `mapstructure:"features"`
    Logging         LoggingConfig             `mapstructure:"logging"`
    Security        SecurityConfig            `mapstructure:"security"`
    Cache           CacheConfig               `mapstructure:"cache"`
    Plugins         PluginConfig              `mapstructure:"plugins"`
    MCP             MCPConfig                 `mapstructure:"mcp"`
}
```

### Loading Configuration
```go
// Default loading
config, err := config.Load()

// With options
config, err := config.LoadWithOptions(config.LoadOptions{
    ConfigFile:     "custom-config.yaml",
    SkipValidation: false,
    SkipEnvVars:    false,
})

// From specific file
config, err := config.LoadFromFile("/path/to/config.yaml")
```

### Provider Configuration
```go
type ProviderConfig struct {
    Type            string            `mapstructure:"type" validate:"required,oneof=openai anthropic ollama custom"`
    APIKey          SecureString      `mapstructure:"api_key"`
    BaseURL         string            `mapstructure:"base_url" validate:"omitempty,url"`
    Organization    string            `mapstructure:"organization"`
    MaxRetries      int               `mapstructure:"max_retries" validate:"min=0,max=10"`
    Timeout         time.Duration     `mapstructure:"timeout" validate:"min=1s,max=300s"`
    RateLimit       string            `mapstructure:"rate_limit"`
    CustomHeaders   map[string]string `mapstructure:"custom_headers"`
    TLSVerify       bool              `mapstructure:"tls_verify"`
}

// Getting provider config
providerConfig, providerName, err := config.GetProvider("openai")
```

### SecureString Usage
```go
// Creating secure strings
apiKey := config.NewSecureString("sk-secret-key")

// Getting the actual value (use sparingly)
actualKey := apiKey.Value()

// String representation is always redacted
fmt.Println(apiKey.String()) // Output: ***REDACTED***

// Clearing from memory
apiKey.Clear()
```

## 🚀 Provider Implementations

### OpenAI Provider
```go
package openai

type Provider struct {
    config Config
    client *transport.HTTPClient
}

func New(config Config) *Provider {
    return &Provider{
        config: config,
        client: transport.NewHTTPClient(config.BaseURL),
    }
}

func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
    // Implementation
}
```

### Anthropic Provider
```go
package anthropic

type Provider struct {
    config Config
    client *transport.HTTPClient
}

func New(config Config) *Provider {
    return &Provider{
        config: config,
        client: transport.NewHTTPClient(config.BaseURL),
    }
}
```

### Provider Factory Pattern
```go
// Register providers
func init() {
    core.RegisterProviderFactory("openai", func(config core.ProviderConfig) (core.Provider, error) {
        return openai.New(openai.Config{
            APIKey:  config.APIKey,
            BaseURL: config.BaseURL,
            // ... map other fields
        }), nil
    })
}

// Create provider from config
provider, err := core.CreateProviderFromConfig("openai", providerConfig)
```

## 🌐 HTTP Transport

### HTTP Client Usage
```go
client := transport.NewHTTPClient("https://api.openai.com")

// Configure client
client.SetTimeout(30 * time.Second)
client.SetMaxRetries(3)
client.SetHeaders(map[string]string{
    "Authorization": "Bearer " + apiKey,
    "Content-Type":  "application/json",
})

// Make requests
resp, err := client.Post(ctx, "/v1/chat/completions", requestBody)
```

### Streaming Support
```go
// For streaming responses
stream, err := client.PostStream(ctx, "/v1/chat/completions", requestBody)
if err != nil {
    return err
}
defer stream.Close()

for chunk := range stream.Chunks() {
    if chunk.Error != nil {
        return chunk.Error
    }
    // Process chunk.Data
}
```

## 🧪 Testing Patterns

### Table-Driven Tests
```go
func TestProvider_CreateCompletion(t *testing.T) {
    tests := []struct {
        name           string
        request        core.CompletionRequest
        mockResponse   string
        expectedError  string
        expectedResult *core.CompletionResponse
    }{
        {
            name: "successful completion",
            request: core.CompletionRequest{
                Model: "gpt-3.5-turbo",
                Messages: []core.Message{{Role: "user", Content: "Hello"}},
            },
            mockResponse: `{"choices":[{"message":{"content":"Hi there!"}}]}`,
            expectedResult: &core.CompletionResponse{
                Choices: []core.Choice{{Message: core.Message{Content: "Hi there!"}}},
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Mock Setup
```go
func TestWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockProvider := NewMockProvider(ctrl)
    mockProvider.EXPECT().
        CreateCompletion(gomock.Any(), gomock.Any()).
        Return(&core.CompletionResponse{}, nil)
    
    // Test with mock
}
```

### Test Utilities
```go
// Helper functions
func SetupTestServer(t *testing.T) *httptest.Server
func CreateTestContext(t *testing.T) context.Context
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration)
func CreateTempConfig(t *testing.T) *config.Config

// Test fixtures
var ValidCompletionRequest = core.CompletionRequest{
    Model: "gpt-3.5-turbo",
    Messages: []core.Message{{Role: "user", Content: "test"}},
}

func LoadTestData(t *testing.T, filename string) []byte {
    data, err := os.ReadFile(filepath.Join("testdata", filename))
    require.NoError(t, err)
    return data
}
```

## ⚠️ Error Handling

### Error Types
```go
// Validation errors
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Rule    string      `json:"rule"`
    Message string      `json:"message"`
}

// API errors
type APIError struct {
    StatusCode int    `json:"status_code"`
    Code       string `json:"code"`
    Message    string `json:"message"`
    Provider   string `json:"provider"`
}

// Creating errors
err := core.NewValidationError("model", "", "required", "model cannot be empty")
```

### Error Wrapping Pattern
```go
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    resp, err := p.client.Post(ctx, "/completions", req)
    if err != nil {
        return nil, fmt.Errorf("failed to call API for model %s: %w", req.Model, err)
    }
    
    return resp, nil
}
```

### Error Checking
```go
if err != nil {
    var validationErr *core.ValidationError
    if errors.As(err, &validationErr) {
        // Handle validation error
        fmt.Printf("Field %s failed validation: %s", validationErr.Field, validationErr.Message)
    }
    
    var apiErr *core.APIError
    if errors.As(err, &apiErr) {
        // Handle API error
        if apiErr.StatusCode == 429 {
            // Handle rate limiting
        }
    }
}
```

## 🔒 Security Patterns

### Input Validation
```go
func ValidateModel(model string) error {
    if len(model) == 0 {
        return core.NewValidationError("model", model, "required", "cannot be empty")
    }
    if len(model) > 100 {
        return core.NewValidationError("model", model, "max_length", "exceeds maximum length of 100")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`).MatchString(model) {
        return core.NewValidationError("model", model, "format", "contains invalid characters")
    }
    return nil
}
```

### Credential Handling
```go
// Never log credentials
func (p *Provider) logRequest(req *core.CompletionRequest) {
    // Create a copy without sensitive data
    sanitized := *req
    sanitized.Metadata = nil // Remove potentially sensitive metadata
    log.Info("Making request", "model", sanitized.Model, "message_count", len(sanitized.Messages))
}

// Clear sensitive data
defer func() {
    if apiKey != nil {
        apiKey.Clear()
    }
}()
```

## 🏃‍♂️ Performance Patterns

### Memory Pools
```go
var messagePool = sync.Pool{
    New: func() interface{} {
        return &core.Message{}
    },
}

func GetMessage() *core.Message {
    return messagePool.Get().(*core.Message)
}

func PutMessage(m *core.Message) {
    m.Reset() // Clear fields
    messagePool.Put(m)
}
```

### Connection Reuse
```go
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

### Concurrency Control
```go
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    const maxWorkers = 10
    sem := make(chan struct{}, maxWorkers)
    
    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
                s.processItem(ctx, item)
            case <-ctx.Done():
                return
            }
        }(item)
    }
    
    wg.Wait()
    return nil
}
```

## 📊 Benchmarking

### Benchmark Functions
```go
func BenchmarkProvider_CreateCompletion(b *testing.B) {
    provider := setupBenchmarkProvider(b)
    req := &core.CompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []core.Message{{Role: "user", Content: "test"}},
    }
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := provider.CreateCompletion(context.Background(), req)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## 🌍 Environment & Configuration

### Environment Variables
```
GOLLM_DEFAULT_PROVIDER=openai
GOLLM_PROVIDERS_OPENAI_API_KEY=sk-your-key
GOLLM_PROVIDERS_ANTHROPIC_API_KEY=your-key
GOLLM_SETTINGS_MAX_TOKENS=4096
GOLLM_LOGGING_LEVEL=debug
```

### Configuration Files
```yaml
# ~/.gollm/config.yaml
default_provider: openai

providers:
  openai:
    type: openai
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
  
settings:
  max_tokens: 2048
  temperature: 0.7
  timeout: 30s

features:
  streaming: true
  caching: false
```

## 🔧 Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run specific package
go test -v ./internal/config

# Run with race detection
go test -race ./...

# Generate coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building
```bash
# Build for current platform
go build ./cmd/gollm

# Cross-compilation
GOOS=linux GOARCH=amd64 go build ./cmd/gollm
GOOS=darwin GOARCH=amd64 go build ./cmd/gollm
GOOS=windows GOARCH=amd64 go build ./cmd/gollm
```

### Linting
```bash
# Run linter
golangci-lint run

# Fix auto-fixable issues
golangci-lint run --fix
```

---

**💡 Pro Tips:**
- Always use context for cancellation
- Wrap errors with meaningful context
- Apply defaults before validation
- Use SecureString for sensitive data
- Implement proper cleanup with defer
- Test both success and error paths
- Use table-driven tests for comprehensive coverage