# GOLLM Code Generation Prompt

You are generating high-quality Go code for the GOLLM project. Follow the established architecture and quality standards while implementing the specific feature requested.

## Context Awareness

Before generating code, always consider:
- **Current architecture**: Maintain consistency with existing patterns
- **Performance implications**: Every allocation and operation matters
- **Security requirements**: Validate inputs, handle errors properly
- **Testability**: Design for easy testing with clear interfaces
- **Maintainability**: Write self-documenting, readable code

## Generation Rules

### 1. File Structure
```go
// Package comment explaining purpose and usage examples
package packagename

import (
    // Standard library first
    "context"
    "fmt"
    "net/http"
    
    // External dependencies second
    "github.com/spf13/cobra"
    
    // Internal imports last
    "github.com/yourusername/gollm/internal/core"
)

// Types and constants
type MyStruct struct {
    // Fields with validation tags
    Field string `json:"field" validate:"required"`
}

// Interface definitions
type MyInterface interface {
    Method(ctx context.Context, param string) error
}

// Implementation
func (m *MyStruct) Method(ctx context.Context, param string) error {
    // Implementation with proper error handling
}
```

### 2. Function Generation Template
```go
// [FunctionName] [brief description of what it does].
//
// [Detailed description including parameters, return values, and behavior]
//
// Example:
//    result, err := FunctionName(ctx, "example")
//    if err != nil {
//        return fmt.Errorf("operation failed: %w", err)
//    }
//
// Returns an error if [specific error conditions].
func FunctionName(ctx context.Context, param string) (*Result, error) {
    // Input validation
    if param == "" {
        return nil, fmt.Errorf("param cannot be empty")
    }
    
    // Main logic with error handling
    result, err := doSomething(ctx, param)
    if err != nil {
        return nil, fmt.Errorf("failed to do something with param %s: %w", param, err)
    }
    
    return result, nil
}
```

### 3. Struct Generation Template
```go
// [StructName] represents [what it represents].
//
// [Detailed description of the struct's purpose and usage]
type StructName struct {
    // Required fields first with validation tags
    ID       string    `json:"id" validate:"required"`
    Name     string    `json:"name" validate:"required,min=1,max=100"`
    
    // Optional fields
    Optional *string   `json:"optional,omitempty"`
    
    // Timestamps and metadata last
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    // Embedding for composition
    Metadata
}

// String implements fmt.Stringer for debugging.
func (s *StructName) String() string {
    return fmt.Sprintf("StructName{ID: %s, Name: %s}", s.ID, s.Name)
}

// Validate validates the struct fields.
func (s *StructName) Validate() error {
    if s.ID == "" {
        return fmt.Errorf("ID is required")
    }
    if s.Name == "" {
        return fmt.Errorf("Name is required")
    }
    return nil
}
```

### 4. Interface Generation Template
```go
// [InterfaceName] defines the contract for [what it does].
//
// Implementations should [specific requirements or behaviors expected].
type InterfaceName interface {
    // Method does [what it does] and returns [what it returns].
    Method(ctx context.Context, param Type) (*Result, error)
    
    // SecondMethod [description].
    SecondMethod(ctx context.Context) error
}
```

### 5. Error Handling Pattern
```go
// Custom error types for business logic
var (
    ErrNotFound     = errors.New("resource not found")
    ErrInvalidInput = errors.New("invalid input provided")
    ErrUnauthorized = errors.New("unauthorized access")
)

// Typed error with details
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s (value: %v): %s", e.Field, e.Value, e.Message)
}

// Error wrapping in functions
func processData(data string) error {
    if data == "" {
        return &ValidationError{
            Field:   "data",
            Value:   data,
            Message: "cannot be empty",
        }
    }
    
    result, err := externalCall(data)
    if err != nil {
        return fmt.Errorf("failed to process data %q: %w", data, err)
    }
    
    return nil
}
```

### 6. Concurrency Patterns
```go
// Worker pool pattern
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    const maxWorkers = 10
    
    jobs := make(chan Item, len(items))
    results := make(chan error, len(items))
    
    // Start workers
    for i := 0; i < maxWorkers; i++ {
        go func() {
            for item := range jobs {
                results <- s.processItem(ctx, item)
            }
        }()
    }
    
    // Send jobs
    go func() {
        defer close(jobs)
        for _, item := range items {
            select {
            case jobs <- item:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Collect results
    var firstError error
    for i := 0; i < len(items); i++ {
        select {
        case err := <-results:
            if err != nil && firstError == nil {
                firstError = err
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    return firstError
}
```

### 7. HTTP Client Pattern
```go
// HTTPClient wraps http.Client with additional functionality.
type HTTPClient struct {
    client  *http.Client
    baseURL string
    headers map[string]string
}

// NewHTTPClient creates a new HTTP client with sensible defaults.
func NewHTTPClient(baseURL string) *HTTPClient {
    return &HTTPClient{
        client: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        baseURL: baseURL,
        headers: make(map[string]string),
    }
}

// Post performs a POST request with JSON body.
func (c *HTTPClient) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
    jsonData, err := json.Marshal(body)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request body: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    for key, value := range c.headers {
        req.Header.Set(key, value)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    
    return resp, nil
}
```

### 8. Configuration Pattern
```go
// Config represents application configuration.
type Config struct {
    // Server configuration
    Port int    `mapstructure:"port" validate:"min=1,max=65535"`
    Host string `mapstructure:"host" validate:"required"`
    
    // Provider configurations
    Providers map[string]ProviderConfig `mapstructure:"providers"`
    
    // Feature flags
    Features FeatureConfig `mapstructure:"features"`
}

// LoadConfig loads configuration from file and environment.
func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("$HOME/.gollm")
    viper.AddConfigPath("/etc/gollm")
    
    // Set defaults
    viper.SetDefault("port", 8080)
    viper.SetDefault("host", "localhost")
    
    // Environment variables
    viper.AutomaticEnv()
    viper.SetEnvPrefix("GOLLM")
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    if err := validator.New().Struct(&config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return &config, nil
}
```

## Testing Generation Rules

### 1. Unit Test Template
```go
func TestFunction_Scenario(t *testing.T) {
    tests := []struct {
        name        string
        input       InputType
        setupMock   func(*MockInterface)
        expected    ExpectedType
        expectError string
    }{
        {
            name:  "successful case",
            input: InputType{Field: "value"},
            setupMock: func(m *MockInterface) {
                m.EXPECT().Method(gomock.Any(), gomock.Any()).Return(nil)
            },
            expected: ExpectedType{Result: "success"},
        },
        {
            name:        "error case",
            input:       InputType{Field: ""},
            expectError: "field cannot be empty",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            
            mockInterface := NewMockInterface(ctrl)
            if tt.setupMock != nil {
                tt.setupMock(mockInterface)
            }
            
            service := NewService(mockInterface)
            
            // Execute
            result, err := service.Function(context.Background(), tt.input)
            
            // Assert
            if tt.expectError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectError)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2. Benchmark Template
```go
func BenchmarkFunction(b *testing.B) {
    // Setup
    service := NewService()
    input := InputType{Field: "test"}
    ctx := context.Background()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := service.Function(ctx, input)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Quality Checklist

When generating code, ensure:

- [ ] **Error handling**: All errors are properly wrapped and handled
- [ ] **Input validation**: All user inputs are validated
- [ ] **Context usage**: Context is used for cancellation and timeouts
- [ ] **Memory efficiency**: No unnecessary allocations in hot paths
- [ ] **Concurrency safety**: Proper synchronization for shared state
- [ ] **Documentation**: Public APIs have comprehensive documentation
- [ ] **Testing**: Test cases cover success and error scenarios
- [ ] **Security**: No credential exposure or injection vulnerabilities
- [ ] **Performance**: Efficient algorithms and data structures
- [ ] **Maintainability**: Clear, readable code following Go idioms

## Common Patterns to Apply

1. **Accept interfaces, return structs**
2. **Use dependency injection for testability**
3. **Implement proper cleanup with defer**
4. **Use typed errors for business logic**
5. **Validate inputs at boundaries**
6. **Handle context cancellation**
7. **Use connection pooling for external services**
8. **Implement proper logging without exposing secrets**
9. **Use sync.Pool for frequent allocations**
10. **Follow single responsibility principle**

Remember: Generate code that exemplifies Go best practices while meeting the specific requirements of the GOLLM project. Every line should contribute to creating a fast, secure, and maintainable CLI tool.