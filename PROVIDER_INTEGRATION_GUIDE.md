# GOLLM CLI - Provider Integration Guide

## 🎯 Overview

This guide provides comprehensive instructions for integrating new LLM providers into GOLLM CLI. Follow these patterns to ensure consistent, reliable, and performant provider implementations.

## 🏗️ Provider Architecture

### Core Interface
```go
// internal/core/provider.go
type Provider interface {
    // Identification
    Name() string
    
    // Core functionality
    CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    ValidateConfig() error
    
    // Health check
    Ping(ctx context.Context) error
}

// Optional capabilities
type Streamer interface {
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
}

type ModelLister interface {
    GetModels(ctx context.Context) ([]Model, error)
}

type TokenCounter interface {
    CountTokens(text string) (int, error)
    EstimateTokens(req *CompletionRequest) (int, error)
}

type CostCalculator interface {
    CalculateCost(usage Usage) (float64, error)
    EstimateCost(req *CompletionRequest) (float64, error)
}

type FunctionCaller interface {
    SupportsFunction() bool
    CallFunction(ctx context.Context, req *FunctionCallRequest) (*FunctionCallResponse, error)
}
```

### Provider Template
```go
// internal/providers/template/provider.go
package template

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/yourusername/gollm/internal/core"
)

const (
    ProviderName = "template"
    DefaultBaseURL = "https://api.provider.com/v1"
    DefaultTimeout = 30 * time.Second
)

// Config holds provider-specific configuration
type Config struct {
    APIKey        string            `json:"api_key" validate:"required"`
    BaseURL       string            `json:"base_url,omitempty"`
    Model         string            `json:"model,omitempty"`
    MaxRetries    int               `json:"max_retries,omitempty"`
    Timeout       time.Duration     `json:"timeout,omitempty"`
    Headers       map[string]string `json:"headers,omitempty"`
    UserAgent     string            `json:"user_agent,omitempty"`
    // Provider-specific options
    Extra         map[string]interface{} `json:"extra,omitempty"`
}

// Provider implements the template provider
type Provider struct {
    mu      sync.RWMutex
    config  Config
    client  *http.Client
    
    // Caching
    modelsCache     []core.Model
    modelsCacheTime time.Time
    modelsCacheTTL  time.Duration
    
    // Metrics
    requestCount  int64
    errorCount    int64
    lastError     error
    lastErrorTime time.Time
}

// New creates a new provider instance
func New(config Config) (*Provider, error) {
    // Apply defaults
    if config.BaseURL == "" {
        config.BaseURL = DefaultBaseURL
    }
    if config.Timeout == 0 {
        config.Timeout = DefaultTimeout
    }
    if config.MaxRetries == 0 {
        config.MaxRetries = 3
    }
    
    // Validate configuration
    if err := validateConfig(config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    // Create HTTP client
    client := &http.Client{
        Timeout: config.Timeout,
        Transport: &http.Transport{
            MaxIdleConns:       100,
            IdleConnTimeout:    90 * time.Second,
            DisableCompression: false,
        },
    }
    
    return &Provider{
        config:         config,
        client:         client,
        modelsCacheTTL: 10 * time.Minute,
    }, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
    return ProviderName
}

// CreateCompletion implements the core completion functionality
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
    // Input validation
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Convert to provider format
    providerReq, err := p.convertRequest(req)
    if err != nil {
        return nil, fmt.Errorf("failed to convert request: %w", err)
    }
    
    // Make API call with retries
    var response *ProviderResponse
    var apiErr error
    
    for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
        response, apiErr = p.makeAPICall(ctx, providerReq)
        if apiErr == nil {
            break
        }
        
        // Check if error is retryable
        if !isRetryableError(apiErr) {
            break
        }
        
        // Exponential backoff
        if attempt < p.config.MaxRetries {
            backoff := time.Duration(attempt+1) * time.Second
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(backoff):
                continue
            }
        }
    }
    
    if apiErr != nil {
        p.recordError(apiErr)
        return nil, fmt.Errorf("API request failed: %w", apiErr)
    }
    
    // Convert response
    return p.convertResponse(response, req)
}

// ValidateConfig validates the provider configuration
func (p *Provider) ValidateConfig() error {
    return validateConfig(p.config)
}

// Ping checks if the provider is accessible
func (p *Provider) Ping(ctx context.Context) error {
    // Implementation depends on provider API
    // Usually a lightweight endpoint like /models or /health
    return nil
}
```

## 🔧 Configuration Patterns

### Standard Configuration
```yaml
providers:
  new_provider:
    type: new_provider
    api_key: "${NEW_PROVIDER_API_KEY}"
    base_url: "https://api.newprovider.com/v1"
    max_retries: 3
    timeout: "30s"
    tls_verify: true
    default_model: "provider-model-name"
    
    # Standard headers
    custom_headers:
      User-Agent: "GOLLM/1.0.0"
      X-Client-Version: "1.0.0"
    
    # Provider-specific configuration
    extra:
      supports_streaming: true
      supports_functions: false
      rate_limit_rpm: 60
      rate_limit_rph: 1000
      
      # Pricing information
      cost_per_1k_input: 0.001
      cost_per_1k_output: 0.003
      
      # Model capabilities
      max_context_length: 4096
      supports_system_message: true
      supports_stop_sequences: true
```

### Environment Variable Mapping
```yaml
# Configuration with environment variable substitution
providers:
  secure_provider:
    type: secure_provider
    api_key: "${SECURE_API_KEY:default-key-for-testing}"
    organization: "${SECURE_ORG_ID:}"
    project: "${SECURE_PROJECT_ID:}"
    base_url: "${SECURE_BASE_URL:https://api.secure.com}"
    
    # Conditional configuration
    tls_verify: "${TLS_VERIFY:true}"
    debug_mode: "${DEBUG_MODE:false}"
    log_requests: "${LOG_REQUESTS:false}"
```

## 🚀 Implementation Best Practices

### 1. Error Handling Strategy
```go
// Define provider-specific error types
type ProviderError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Type    string `json:"type"`
    Param   string `json:"param,omitempty"`
}

func (e *ProviderError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Classify errors for proper handling
func classifyError(err error) core.ErrorType {
    if err == nil {
        return core.ErrorTypeNone
    }
    
    // Network errors
    if isNetworkError(err) {
        return core.ErrorTypeNetwork
    }
    
    // Authentication errors
    if isAuthError(err) {
        return core.ErrorTypeAuth
    }
    
    // Rate limiting
    if isRateLimitError(err) {
        return core.ErrorTypeRateLimit
    }
    
    // Invalid request
    if isValidationError(err) {
        return core.ErrorTypeValidation
    }
    
    return core.ErrorTypeUnknown
}

// Implement retry logic
func isRetryableError(err error) bool {
    switch classifyError(err) {
    case core.ErrorTypeNetwork, core.ErrorTypeRateLimit:
        return true
    default:
        return false
    }
}
```

### 2. Request/Response Conversion
```go
// Convert from standard format to provider format
func (p *Provider) convertRequest(req *core.CompletionRequest) (*ProviderRequest, error) {
    providerReq := &ProviderRequest{
        Model:    req.Model,
        Messages: make([]ProviderMessage, len(req.Messages)),
    }
    
    // Convert messages
    for i, msg := range req.Messages {
        providerReq.Messages[i] = ProviderMessage{
            Role:    msg.Role,
            Content: msg.Content,
        }
    }
    
    // Set optional parameters with defaults
    if req.MaxTokens != nil {
        providerReq.MaxTokens = *req.MaxTokens
    }
    if req.Temperature != nil {
        providerReq.Temperature = *req.Temperature
    } else {
        providerReq.Temperature = 0.7 // Provider default
    }
    
    // Provider-specific transformations
    return p.applyProviderSpecificTransforms(providerReq)
}

// Convert from provider format to standard format
func (p *Provider) convertResponse(resp *ProviderResponse, originalReq *core.CompletionRequest) (*core.CompletionResponse, error) {
    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no choices in response")
    }
    
    choices := make([]core.Choice, len(resp.Choices))
    for i, choice := range resp.Choices {
        choices[i] = core.Choice{
            Index: choice.Index,
            Message: core.Message{
                Role:    choice.Message.Role,
                Content: choice.Message.Content,
            },
            FinishReason: choice.FinishReason,
        }
    }
    
    return &core.CompletionResponse{
        ID:      resp.ID,
        Object:  "chat.completion",
        Model:   resp.Model,
        Choices: choices,
        Usage: core.Usage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
            TotalCost:        p.calculateCost(resp.Usage),
        },
    }, nil
}
```

### 3. Streaming Implementation
```go
// StreamCompletion implements streaming responses
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
    // Convert request and enable streaming
    providerReq, err := p.convertRequest(req)
    if err != nil {
        return nil, fmt.Errorf("failed to convert request: %w", err)
    }
    providerReq.Stream = true
    
    // Make streaming request
    stream, err := p.createStream(ctx, providerReq)
    if err != nil {
        return nil, fmt.Errorf("failed to create stream: %w", err)
    }
    
    // Create output channel
    chunks := make(chan core.StreamChunk, 10)
    
    // Start streaming goroutine
    go func() {
        defer close(chunks)
        defer stream.Close()
        
        for {
            select {
            case <-ctx.Done():
                chunks <- core.StreamChunk{Error: ctx.Err()}
                return
            default:
                chunk, err := stream.ReadChunk()
                if err != nil {
                    if err == io.EOF {
                        chunks <- core.StreamChunk{Done: true}
                        return
                    }
                    chunks <- core.StreamChunk{Error: err}
                    return
                }
                
                // Convert and send chunk
                standardChunk := p.convertStreamChunk(chunk)
                chunks <- standardChunk
                
                if standardChunk.Done {
                    return
                }
            }
        }
    }()
    
    return chunks, nil
}
```

## 🧪 Testing Framework

### Unit Tests
```go
// provider_test.go
package template_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/yourusername/gollm/internal/core"
    "github.com/yourusername/gollm/internal/providers/template"
)

func TestProvider_CreateCompletion(t *testing.T) {
    tests := []struct {
        name           string
        request        *core.CompletionRequest
        serverResponse string
        serverStatus   int
        wantError      bool
        wantResponse   *core.CompletionResponse
    }{
        {
            name: "successful completion",
            request: &core.CompletionRequest{
                Model: "test-model",
                Messages: []core.Message{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: intPtr(100),
            },
            serverResponse: `{
                "id": "test-123",
                "choices": [
                    {
                        "index": 0,
                        "message": {"role": "assistant", "content": "Hello there!"},
                        "finish_reason": "stop"
                    }
                ],
                "usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
            }`,
            serverStatus: 200,
            wantError:    false,
        },
        {
            name: "authentication error",
            request: &core.CompletionRequest{
                Model: "test-model",
                Messages: []core.Message{
                    {Role: "user", Content: "Hello"},
                },
            },
            serverResponse: `{"error": {"code": "invalid_api_key", "message": "Invalid API key"}}`,
            serverStatus:   401,
            wantError:      true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create test server
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.serverStatus)
                w.Write([]byte(tt.serverResponse))
            }))
            defer server.Close()

            // Create provider with test server URL
            config := template.Config{
                APIKey:  "test-key",
                BaseURL: server.URL,
                Timeout: time.Second,
            }
            
            provider, err := template.New(config)
            if err != nil {
                t.Fatalf("Failed to create provider: %v", err)
            }

            // Test completion
            ctx := context.Background()
            resp, err := provider.CreateCompletion(ctx, tt.request)

            if tt.wantError {
                if err == nil {
                    t.Error("Expected error but got none")
                }
                return
            }

            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }

            if resp == nil {
                t.Fatal("Expected response but got nil")
            }

            // Validate response structure
            if len(resp.Choices) == 0 {
                t.Error("Expected at least one choice")
            }
        })
    }
}

func intPtr(i int) *int {
    return &i
}
```

### Integration Tests
```go
// integration_test.go
func TestProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    apiKey := os.Getenv("PROVIDER_API_KEY")
    if apiKey == "" {
        t.Skip("PROVIDER_API_KEY not set")
    }

    config := template.Config{
        APIKey: apiKey,
    }

    provider, err := template.New(config)
    if err != nil {
        t.Fatalf("Failed to create provider: %v", err)
    }

    t.Run("ping", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        if err := provider.Ping(ctx); err != nil {
            t.Errorf("Ping failed: %v", err)
        }
    })

    t.Run("list models", func(t *testing.T) {
        if lister, ok := provider.(core.ModelLister); ok {
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()

            models, err := lister.GetModels(ctx)
            if err != nil {
                t.Errorf("GetModels failed: %v", err)
            }
            if len(models) == 0 {
                t.Error("Expected at least one model")
            }
        }
    })

    t.Run("completion", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        req := &core.CompletionRequest{
            Messages: []core.Message{
                {Role: "user", Content: "Say hello"},
            },
            MaxTokens: intPtr(10),
        }

        resp, err := provider.CreateCompletion(ctx, req)
        if err != nil {
            t.Errorf("CreateCompletion failed: %v", err)
        }
        if len(resp.Choices) == 0 {
            t.Error("Expected at least one choice")
        }
    })
}
```

## 📊 Performance Optimization

### Connection Pooling
```go
// Optimize HTTP client for high throughput
func createOptimizedHTTPClient(timeout time.Duration) *http.Client {
    transport := &http.Transport{
        // Connection pooling
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   20,
        IdleConnTimeout:       90 * time.Second,
        
        // Timeouts
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 30 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        
        // Performance
        DisableCompression: false,
        ForceAttemptHTTP2:  true,
        
        // DNS
        DNSCacheTimeout: 10 * time.Minute,
    }

    return &http.Client{
        Timeout:   timeout,
        Transport: transport,
    }
}
```

### Caching Strategy
```go
// Implement intelligent caching
type Cache struct {
    mu       sync.RWMutex
    data     map[string]CacheItem
    maxSize  int
    ttl      time.Duration
}

type CacheItem struct {
    Value     interface{}
    ExpiresAt time.Time
    AccessCount int64
    LastAccess  time.Time
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    item, exists := c.data[key]
    if !exists || time.Now().After(item.ExpiresAt) {
        return nil, false
    }
    
    // Update access statistics
    item.AccessCount++
    item.LastAccess = time.Now()
    c.data[key] = item
    
    return item.Value, true
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if necessary
    if len(c.data) >= c.maxSize {
        c.evictLRU()
    }
    
    c.data[key] = CacheItem{
        Value:     value,
        ExpiresAt: time.Now().Add(c.ttl),
        AccessCount: 1,
        LastAccess:  time.Now(),
    }
}
```

## 🔒 Security Considerations

### API Key Protection
```go
// Secure API key handling
type SecureString struct {
    value []byte
    mu    sync.RWMutex
}

func NewSecureString(value string) *SecureString {
    s := &SecureString{
        value: make([]byte, len(value)),
    }
    copy(s.value, value)
    return s
}

func (s *SecureString) Value() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return string(s.value)
}

func (s *SecureString) Clear() {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.value {
        s.value[i] = 0
    }
}

func (s *SecureString) String() string {
    return "[REDACTED]"
}

// Use in provider config
type Config struct {
    APIKey *SecureString `json:"api_key"`
    // ... other fields
}
```

### Request Sanitization
```go
// Sanitize requests before logging
func sanitizeForLogging(req *core.CompletionRequest) *core.CompletionRequest {
    sanitized := *req
    
    // Remove or mask sensitive content
    for i, msg := range sanitized.Messages {
        sanitized.Messages[i].Content = maskSensitiveData(msg.Content)
    }
    
    return &sanitized
}

func maskSensitiveData(content string) string {
    // Mask common sensitive patterns
    patterns := []struct {
        regex *regexp.Regexp
        replacement string
    }{
        {regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`), "[EMAIL]"},
        {regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`), "[CARD]"},
        {regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), "[SSN]"},
        {regexp.MustCompile(`(?i)api[_-]?key[:\s=]+[a-zA-Z0-9]+`), "api_key=[REDACTED]"},
    }
    
    result := content
    for _, pattern := range patterns {
        result = pattern.regex.ReplaceAllString(result, pattern.replacement)
    }
    
    return result
}
```

## 🎯 Provider Registration

### Auto-Registration Pattern
```go
// internal/providers/registry.go
func init() {
    // Auto-register all providers
    RegisterProvider("openai", openai.New)
    RegisterProvider("anthropic", anthropic.New)
    RegisterProvider("deepseek", deepseek.New)
    RegisterProvider("gemini", gemini.New)
    RegisterProvider("openrouter", openrouter.New)
    RegisterProvider("mock", mock.New)
}

// Register a new provider
func RegisterProvider(name string, factory ProviderFactory) {
    providerRegistry[name] = factory
}

// Create provider by name
func CreateProvider(name string, config config.ProviderConfig) (core.Provider, error) {
    factory, exists := providerRegistry[name]
    if !exists {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }
    
    return factory(config)
}

// List all registered providers
func ListProviders() []string {
    var names []string
    for name := range providerRegistry {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

## 📋 Checklist for New Providers

### Implementation Checklist
- [ ] Basic Provider interface implemented
- [ ] Configuration structure defined
- [ ] Request/response conversion implemented  
- [ ] Error handling with proper classification
- [ ] Retry logic for transient failures
- [ ] Unit tests with mocked responses
- [ ] Integration tests with real API
- [ ] Performance benchmarks
- [ ] Security review completed
- [ ] Documentation updated

### Optional Features Checklist
- [ ] Streaming support (Streamer interface)
- [ ] Model listing (ModelLister interface)
- [ ] Token counting (TokenCounter interface)
- [ ] Cost calculation (CostCalculator interface)
- [ ] Function calling (FunctionCaller interface)
- [ ] Response caching
- [ ] Request rate limiting
- [ ] Metrics collection
- [ ] Custom headers support
- [ ] Proxy support

### Quality Assurance
- [ ] Error messages are descriptive and actionable
- [ ] No sensitive data in logs
- [ ] Graceful handling of API changes
- [ ] Memory efficient implementation
- [ ] Thread-safe operations
- [ ] Context cancellation support
- [ ] Proper resource cleanup
- [ ] Configuration validation
- [ ] Backwards compatibility maintained
- [ ] Performance meets benchmarks

---

**Following this guide ensures consistent, reliable, and maintainable provider implementations that integrate seamlessly with GOLLM CLI's architecture.**