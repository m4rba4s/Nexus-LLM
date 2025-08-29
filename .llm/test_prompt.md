# GOLLM Testing Prompt

You are writing comprehensive tests for the GOLLM project, a high-performance LLM CLI tool built in Go. Your testing approach must ensure bulletproof reliability, security, and performance.

## Testing Philosophy

### Core Principles
- **Test-Driven Development**: Write tests before implementation
- **Comprehensive Coverage**: Minimum 85% code coverage, 100% for critical paths
- **Fast Feedback**: Tests must run quickly (< 5 seconds for unit tests)
- **Reliable**: Tests should never be flaky or dependent on external services
- **Readable**: Tests serve as documentation and examples

### Testing Pyramid
```
    E2E Tests (Few)
      ↑ Slow, Expensive
   Integration Tests (Some)
      ↑ Medium Speed
    Unit Tests (Many)
      ↑ Fast, Cheap
```

## Test Organization

### Directory Structure
```
/
├── internal/
│   ├── cli/
│   │   ├── commands.go
│   │   └── commands_test.go
│   ├── providers/
│   │   ├── openai/
│   │   │   ├── client.go
│   │   │   ├── client_test.go
│   │   │   └── testdata/
│   │   │       ├── responses/
│   │   │       └── fixtures/
│   └── testutils/
│       ├── mocks/
│       ├── fixtures/
│       └── helpers.go
├── test/
│   ├── integration/
│   ├── e2e/
│   └── benchmarks/
```

### Naming Conventions
```go
// Test function naming: TestSubject_Method_Condition
func TestProvider_CreateCompletion_Success(t *testing.T)
func TestProvider_CreateCompletion_InvalidModel(t *testing.T)
func TestProvider_CreateCompletion_NetworkError(t *testing.T)

// Benchmark naming: BenchmarkSubject_Method
func BenchmarkProvider_CreateCompletion(b *testing.B)
func BenchmarkJSON_Marshal_LargePayload(b *testing.B)
```

## Unit Testing Patterns

### 1. Table-Driven Tests
```go
func TestValidateModel(t *testing.T) {
    tests := []struct {
        name        string
        model       string
        expectError bool
        errorType   error
    }{
        {
            name:        "valid model name",
            model:       "gpt-3.5-turbo",
            expectError: false,
        },
        {
            name:        "empty model name",
            model:       "",
            expectError: true,
            errorType:   ErrInvalidModel,
        },
        {
            name:        "model name too long",
            model:       strings.Repeat("a", 101),
            expectError: true,
            errorType:   ErrInvalidModel,
        },
        {
            name:        "model with invalid characters",
            model:       "gpt-3.5@turbo",
            expectError: true,
            errorType:   ErrInvalidModel,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateModel(tt.model)
            
            if tt.expectError {
                assert.Error(t, err)
                if tt.errorType != nil {
                    assert.True(t, errors.Is(err, tt.errorType))
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. Mock-Based Testing
```go
func TestService_ProcessRequest(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockProvider := mocks.NewMockProvider(ctrl)
    mockConfig := mocks.NewMockConfig(ctrl)

    service := NewService(mockProvider, mockConfig)

    // Test successful processing
    t.Run("successful completion", func(t *testing.T) {
        req := &CompletionRequest{
            Model: "gpt-3.5-turbo",
            Messages: []Message{{Role: "user", Content: "Hello"}},
        }

        expectedResp := &CompletionResponse{
            Choices: []Choice{{
                Message: Message{Role: "assistant", Content: "Hi there!"},
            }},
        }

        mockProvider.EXPECT().
            CreateCompletion(gomock.Any(), gomock.Eq(req)).
            Return(expectedResp, nil)

        result, err := service.ProcessRequest(context.Background(), req)

        assert.NoError(t, err)
        assert.Equal(t, expectedResp, result)
    })

    // Test error handling
    t.Run("provider error", func(t *testing.T) {
        req := &CompletionRequest{Model: "invalid"}

        expectedErr := errors.New("provider error")
        mockProvider.EXPECT().
            CreateCompletion(gomock.Any(), gomock.Any()).
            Return(nil, expectedErr)

        result, err := service.ProcessRequest(context.Background(), req)

        assert.Error(t, err)
        assert.Nil(t, result)
        assert.Contains(t, err.Error(), "provider error")
    })
}
```

### 3. Error Testing Patterns
```go
func TestHTTPClient_Post_ErrorHandling(t *testing.T) {
    tests := []struct {
        name          string
        setupServer   func() *httptest.Server
        expectError   string
        errorType     error
    }{
        {
            name: "network timeout",
            setupServer: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    time.Sleep(2 * time.Second) // Longer than client timeout
                }))
            },
            expectError: "context deadline exceeded",
            errorType:   context.DeadlineExceeded,
        },
        {
            name: "server error",
            setupServer: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusInternalServerError)
                    w.Write([]byte(`{"error": "internal server error"}`))
                }))
            },
            expectError: "HTTP 500",
        },
        {
            name: "invalid JSON response",
            setupServer: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.Write([]byte(`{invalid json`))
                }))
            },
            expectError: "failed to parse response",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := tt.setupServer()
            defer server.Close()

            client := NewHTTPClient(server.URL)
            ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
            defer cancel()

            _, err := client.Post(ctx, "/test", map[string]string{"key": "value"})

            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.expectError)

            if tt.errorType != nil {
                assert.True(t, errors.Is(err, tt.errorType))
            }
        })
    }
}
```

## Integration Testing

### 1. Provider Integration Tests
```go
func TestOpenAIProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    provider := openai.New(openai.Config{
        APIKey:  apiKey,
        BaseURL: "https://api.openai.com/v1",
    })

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    t.Run("create completion", func(t *testing.T) {
        req := &CompletionRequest{
            Model: "gpt-3.5-turbo",
            Messages: []Message{
                {Role: "user", Content: "Say hello in one word"},
            },
            MaxTokens:   10,
            Temperature: 0.7,
        }

        resp, err := provider.CreateCompletion(ctx, req)

        assert.NoError(t, err)
        assert.NotNil(t, resp)
        assert.NotEmpty(t, resp.Choices)
        assert.NotEmpty(t, resp.Choices[0].Message.Content)
    })

    t.Run("stream completion", func(t *testing.T) {
        req := &CompletionRequest{
            Model: "gpt-3.5-turbo",
            Messages: []Message{
                {Role: "user", Content: "Count from 1 to 5"},
            },
            Stream: true,
        }

        stream, err := provider.StreamCompletion(ctx, req)
        assert.NoError(t, err)

        var chunks []CompletionChunk
        for chunk := range stream {
            if chunk.Error != nil {
                t.Fatalf("stream error: %v", chunk.Error)
            }
            chunks = append(chunks, chunk)
        }

        assert.NotEmpty(t, chunks)
    })
}
```

### 2. CLI Integration Tests
```go
func TestCLI_Integration(t *testing.T) {
    // Setup temporary config
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yaml")
    
    config := `
providers:
  test:
    type: mock
    api_key: test-key
default_provider: test
`
    err := os.WriteFile(configFile, []byte(config), 0644)
    require.NoError(t, err)

    tests := []struct {
        name     string
        args     []string
        stdin    string
        expected string
        exitCode int
    }{
        {
            name:     "simple completion",
            args:     []string{"--config", configFile, "complete", "Hello world"},
            expected: "Mock response: Hello world",
            exitCode: 0,
        },
        {
            name:     "invalid model",
            args:     []string{"--config", configFile, "complete", "--model", "invalid", "test"},
            expected: "Error: invalid model",
            exitCode: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command("go", append([]string{"run", "./cmd/gollm"}, tt.args...)...)
            
            if tt.stdin != "" {
                cmd.Stdin = strings.NewReader(tt.stdin)
            }

            output, err := cmd.CombinedOutput()
            
            if tt.exitCode == 0 {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
            }
            
            assert.Contains(t, string(output), tt.expected)
        })
    }
}
```

## Performance Testing

### 1. Benchmark Tests
```go
func BenchmarkProvider_CreateCompletion(b *testing.B) {
    provider := setupMockProvider()
    ctx := context.Background()
    req := &CompletionRequest{
        Model:    "gpt-3.5-turbo",
        Messages: []Message{{Role: "user", Content: "Hello"}},
    }

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := provider.CreateCompletion(ctx, req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkJSONMarshal_LargePayload(b *testing.B) {
    largeRequest := &CompletionRequest{
        Model: "gpt-4",
        Messages: make([]Message, 100),
    }
    
    // Fill with test data
    for i := range largeRequest.Messages {
        largeRequest.Messages[i] = Message{
            Role:    "user",
            Content: strings.Repeat("test content ", 100),
        }
    }

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := json.Marshal(largeRequest)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 2. Memory Benchmark Tests
```go
func BenchmarkMemoryPool_GetPut(b *testing.B) {
    pool := &sync.Pool{
        New: func() interface{} {
            return make([]byte, 4096)
        },
    }

    b.ResetTimer()
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            buf := pool.Get().([]byte)
            // Simulate usage
            _ = buf[:1024]
            pool.Put(buf)
        }
    })
}
```

## Security Testing

### 1. Input Validation Tests
```go
func TestSecurity_InputValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        field    string
        expected bool
    }{
        {
            name:     "SQL injection attempt",
            input:    "'; DROP TABLE users; --",
            field:    "model",
            expected: false,
        },
        {
            name:     "XSS attempt",
            input:    "<script>alert('xss')</script>",
            field:    "content",
            expected: false,
        },
        {
            name:     "Path traversal attempt",
            input:    "../../../etc/passwd",
            field:    "filename",
            expected: false,
        },
        {
            name:     "Extremely long input",
            input:    strings.Repeat("a", 1000000),
            field:    "content",
            expected: false,
        },
        {
            name:     "Valid input",
            input:    "gpt-3.5-turbo",
            field:    "model",
            expected: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            valid := ValidateInput(tt.field, tt.input)
            assert.Equal(t, tt.expected, valid)
        })
    }
}
```

### 2. API Key Security Tests
```go
func TestSecurity_APIKeyHandling(t *testing.T) {
    t.Run("API key not logged", func(t *testing.T) {
        var logOutput bytes.Buffer
        logger := log.New(&logOutput, "", 0)
        
        config := Config{
            APIKey: "sk-secret-key-12345",
        }
        
        // Simulate logging configuration
        logger.Printf("Config: %+v", config)
        
        logContent := logOutput.String()
        assert.NotContains(t, logContent, "sk-secret-key-12345")
        assert.Contains(t, logContent, "***REDACTED***")
    })

    t.Run("API key cleared from memory", func(t *testing.T) {
        key := NewSecureString("sk-secret-key")
        originalData := make([]byte, len(key.data))
        copy(originalData, key.data)
        
        key.Clear()
        
        // Verify memory is cleared
        for i, b := range key.data {
            assert.Zero(t, b, "byte at index %d should be zero", i)
        }
    })
}
```

### 3. Rate Limiting Tests
```go
func TestSecurity_RateLimiting(t *testing.T) {
    limiter := NewRateLimiter(5, time.Second) // 5 requests per second
    
    // Test within limits
    for i := 0; i < 5; i++ {
        allowed := limiter.Allow()
        assert.True(t, allowed, "request %d should be allowed", i+1)
    }
    
    // Test exceeding limits
    allowed := limiter.Allow()
    assert.False(t, allowed, "6th request should be blocked")
    
    // Test reset after time window
    time.Sleep(1100 * time.Millisecond)
    allowed = limiter.Allow()
    assert.True(t, allowed, "request should be allowed after reset")
}
```

## Concurrency Testing

### 1. Race Condition Tests
```go
func TestConcurrency_RaceConditions(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping race condition test")
    }

    cache := NewCache()
    const numGoroutines = 100
    const numOperations = 1000

    var wg sync.WaitGroup
    wg.Add(numGoroutines)

    // Start multiple goroutines performing concurrent operations
    for i := 0; i < numGoroutines; i++ {
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < numOperations; j++ {
                key := fmt.Sprintf("key-%d-%d", id, j)
                value := fmt.Sprintf("value-%d-%d", id, j)
                
                // Concurrent read/write operations
                cache.Set(key, value)
                retrieved, exists := cache.Get(key)
                
                if exists {
                    assert.Equal(t, value, retrieved)
                }
                
                cache.Delete(key)
            }
        }(i)
    }

    wg.Wait()
}
```

### 2. Context Cancellation Tests
```go
func TestConcurrency_ContextCancellation(t *testing.T) {
    provider := NewMockProvider()
    service := NewService(provider)

    t.Run("cancellation during processing", func(t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())
        
        // Start long-running operation
        done := make(chan error, 1)
        go func() {
            _, err := service.ProcessLongRunning(ctx, &Request{})
            done <- err
        }()
        
        // Cancel after short delay
        time.Sleep(100 * time.Millisecond)
        cancel()
        
        // Verify cancellation is handled
        select {
        case err := <-done:
            assert.Error(t, err)
            assert.True(t, errors.Is(err, context.Canceled))
        case <-time.After(5 * time.Second):
            t.Fatal("operation should have been cancelled")
        }
    })

    t.Run("timeout handling", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
        defer cancel()
        
        // This should timeout
        _, err := service.ProcessSlow(ctx, &Request{})
        
        assert.Error(t, err)
        assert.True(t, errors.Is(err, context.DeadlineExceeded))
    })
}
```

## Property-Based Testing

### 1. Fuzzing Tests
```go
func FuzzValidateModel(f *testing.F) {
    // Seed with known values
    f.Add("gpt-3.5-turbo")
    f.Add("claude-2")
    f.Add("")
    f.Add("invalid@model")

    f.Fuzz(func(t *testing.T, model string) {
        // The function should never panic
        defer func() {
            if r := recover(); r != nil {
                t.Errorf("ValidateModel panicked with input %q: %v", model, r)
            }
        }()

        err := ValidateModel(model)
        
        // Basic invariants
        if model == "" {
            assert.Error(t, err, "empty model should be invalid")
        }
        
        if len(model) > 100 {
            assert.Error(t, err, "too long model should be invalid")
        }
        
        // Valid models should not return error
        if regexp.MustCompile(`^[a-zA-Z0-9\-_.]+$`).MatchString(model) && 
           len(model) > 0 && len(model) <= 100 {
            assert.NoError(t, err, "valid model should not return error")
        }
    })
}
```

## Test Utilities and Helpers

### 1. Test Setup Helpers
```go
// testutils/helpers.go
package testutils

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/require"
)

// SetupTestServer creates a test HTTP server with common middleware.
func SetupTestServer(t *testing.T, handler http.Handler) *httptest.Server {
    t.Helper()
    
    server := httptest.NewServer(handler)
    t.Cleanup(server.Close)
    
    return server
}

// CreateTestContext creates a context with reasonable timeout for tests.
func CreateTestContext(t *testing.T) (context.Context, context.CancelFunc) {
    t.Helper()
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    
    return ctx, cancel
}

// AssertEventuallyTrue waits for a condition to become true.
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
    t.Helper()
    
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    timer := time.NewTimer(timeout)
    defer timer.Stop()
    
    for {
        select {
        case <-ticker.C:
            if condition() {
                return
            }
        case <-timer.C:
            t.Fatalf("condition never became true: %s", msg)
        }
    }
}

// CreateTempConfig creates a temporary configuration file for testing.
func CreateTempConfig(t *testing.T, content string) string {
    t.Helper()
    
    tmpDir := t.TempDir()
    configFile := filepath.Join(tmpDir, "config.yaml")
    
    err := os.WriteFile(configFile, []byte(content), 0644)
    require.NoError(t, err)
    
    return configFile
}
```

### 2. Test Fixtures
```go
// testutils/fixtures.go
package testutils

// Common test fixtures
var (
    ValidCompletionRequest = &CompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []Message{
            {Role: "user", Content: "Hello, world!"},
        },
        MaxTokens:   150,
        Temperature: 0.7,
    }
    
    ValidCompletionResponse = &CompletionResponse{
        ID:      "chatcmpl-123",
        Object:  "chat.completion",
        Created: 1677652288,
        Model:   "gpt-3.5-turbo",
        Choices: []Choice{
            {
                Index: 0,
                Message: Message{
                    Role:    "assistant",
                    Content: "Hello! How can I help you today?",
                },
                FinishReason: "stop",
            },
        },
        Usage: Usage{
            PromptTokens:     12,
            CompletionTokens: 9,
            TotalTokens:      21,
        },
    }
)

// LoadTestData loads test data from files.
func LoadTestData(t *testing.T, filename string) []byte {
    t.Helper()
    
    data, err := os.ReadFile(filepath.Join("testdata", filename))
    require.NoError(t, err, "failed to load test data: %s", filename)
    
    return data
}
```

## Coverage Requirements

### 1. Coverage Analysis
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Check coverage threshold
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | awk '{if ($1 < 85) exit 1}'
```

### 2. Critical Path Coverage
```go
// Use build tags to ensure critical paths are tested
//go:build !integration

func TestCriticalPath_AllBranches(t *testing.T) {
    // Test all code paths in critical functions
    // This test must achieve 100% line and branch coverage
}
```

## E2E Testing

### 1. CLI End-to-End Tests
```go
func TestE2E_CLIWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Build the binary
    binary := buildBinary(t)
    defer os.Remove(binary)

    // Test complete workflow
    t.Run("complete workflow", func(t *testing.T) {
        // Setup mock server
        server := setupMockLLMServer(t)
        defer server.Close()

        // Create config pointing to mock server
        config := fmt.Sprintf(`
providers:
  test:
    type: openai
    base_url: %s
    api_key: test-key
default_provider: test
`, server.URL)

        configFile := createTempFile(t, "config.yaml", config)

        // Test various CLI commands
        tests := []struct {
            name string
            args []string
            want string
        }{
            {
                name: "simple completion",
                args: []string{"--config", configFile, "complete", "Hello"},
                want: "Hello! How can I help you?",
            },
            {
                name: "list models",
                args: []string{"--config", configFile, "models"},
                want: "gpt-3.5-turbo",
            },
        }

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                cmd := exec.Command(binary, tt.args...)
                output, err := cmd.CombinedOutput()
                
                assert.NoError(t, err)
                assert.Contains(t, string(output), tt.want)
            })
        }
    })
}
```

## Testing Best Practices

### 1. Test Organization
- **Arrange, Act, Assert** pattern
- **Given, When, Then** for BDD-style tests
- **One assertion per test** when possible
- **Descriptive test names** that explain the scenario

### 2. Test Independence
- Tests should not depend on other tests
- Each test should set up its own data
- Clean up after each test
- Use `t.Parallel()` for independent tests

### 3. Error Testing
- Test both success and failure paths
- Verify error messages and types
- Test edge cases and boundary conditions
- Use typed errors for business logic validation

### 4. Performance Testing
- Benchmark critical paths
- Monitor memory allocations
- Test under load
- Verify resource cleanup

Remember: Tests are your safety net and documentation. Write them as if your production system depends on them—because it does.