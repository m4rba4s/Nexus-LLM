# GOLLM Code Review Checklist

## Overview

This checklist ensures that all code changes meet GOLLM's high standards for performance, security, maintainability, and Go best practices. Use this as a systematic guide for both self-review and peer review.

## Quick Reference

### Critical Items (Must Pass)
- [ ] No security vulnerabilities
- [ ] No data races or concurrency issues
- [ ] Error handling implemented correctly
- [ ] Tests provide adequate coverage
- [ ] Performance impact assessed

### Before Starting Review
- [ ] PR description clearly explains the change
- [ ] CI checks are passing
- [ ] Branch is up to date with main
- [ ] No merge conflicts

## Architecture & Design

### 1. Design Principles
- [ ] **Single Responsibility**: Each function/struct has one clear purpose
- [ ] **Open/Closed Principle**: Code is open for extension, closed for modification
- [ ] **Interface Segregation**: Interfaces are small and focused
- [ ] **Dependency Inversion**: Depends on abstractions, not concretions
- [ ] **Composition over Inheritance**: Uses embedding and composition appropriately

### 2. Layered Architecture Compliance
- [ ] Changes respect the layered architecture (CLI → Service → Provider → Transport → Core)
- [ ] Dependencies flow downward only (no circular dependencies)
- [ ] Layer boundaries are well-defined with interfaces
- [ ] No layer skipping (e.g., CLI directly calling Transport)

### 3. Interface Design
```go
// ✅ Good: Small, focused interface
type Provider interface {
    CreateCompletion(ctx context.Context, req Request) (*Response, error)
}

// ❌ Bad: Too many responsibilities
type MegaInterface interface {
    CreateCompletion(ctx context.Context, req Request) (*Response, error)
    ValidateConfig() error
    LogMessage(msg string)
    SaveToFile(filename string) error
}
```
- [ ] Interfaces are small (1-3 methods)
- [ ] Named with behavior verbs (-er suffix: Provider, Streamer)
- [ ] Accept interfaces, return structs pattern followed
- [ ] No empty interfaces (`interface{}`)

## Go Best Practices

### 1. Code Organization
- [ ] **Package naming**: lowercase, single word when possible
- [ ] **Import grouping**: stdlib, external, internal (with blank lines)
- [ ] **File organization**: logical grouping of related functionality
- [ ] **Exported vs unexported**: Appropriate visibility levels

### 2. Naming Conventions
```go
// ✅ Good naming
type CompletionRequest struct {
    Model       string `json:"model"`
    Messages    []Message `json:"messages"`
    MaxTokens   int `json:"max_tokens,omitempty"`
}

func (c *Client) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

// ❌ Bad naming  
type CCR struct {
    M string `json:"model"`
    Msg []Message `json:"messages"`  
    MT int `json:"max_tokens,omitempty"`
}

func (c *Client) CC(ctx context.Context, r CCR) (*CCResp, error)
```
- [ ] **Structs**: PascalCase, descriptive names
- [ ] **Functions**: camelCase, action verbs
- [ ] **Variables**: camelCase, meaningful names
- [ ] **Constants**: ALL_CAPS for package-level, PascalCase for exported
- [ ] **Packages**: lowercase, single word preferred

### 3. Function Design
- [ ] Functions are < 50 lines
- [ ] Maximum 4 parameters (use structs for more)
- [ ] Single responsibility principle followed
- [ ] Pure functions where possible (no side effects)
- [ ] Context as first parameter when needed

### 4. Struct Design
```go
// ✅ Good struct design
type Request struct {
    // Required fields first
    Model    string    `json:"model" validate:"required"`
    Messages []Message `json:"messages" validate:"required,min=1"`
    
    // Optional fields
    MaxTokens   *int     `json:"max_tokens,omitempty"`
    Temperature *float64 `json:"temperature,omitempty"`
    
    // Metadata
    RequestID string            `json:"request_id,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}
```
- [ ] JSON tags explicitly defined
- [ ] Validation tags present where appropriate
- [ ] Logical field ordering (required first, optional, then metadata)
- [ ] Pointer fields for optional values
- [ ] Embedding used appropriately for composition

## Error Handling

### 1. Error Wrapping
```go
// ✅ Good error handling
func (p *Provider) CreateCompletion(ctx context.Context, req Request) (*Response, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    resp, err := p.client.Post(ctx, "/completions", req)
    if err != nil {
        return nil, fmt.Errorf("failed to call API for model %s: %w", req.Model, err)
    }
    
    return resp, nil
}

// ❌ Bad error handling
func (p *Provider) CreateCompletion(ctx context.Context, req Request) (*Response, error) {
    req.Validate() // Ignored error
    
    resp, err := p.client.Post(ctx, "/completions", req)
    if err != nil {
        return nil, err // No context
    }
    
    return resp, nil
}
```
- [ ] All errors are handled (no `_` assignments)
- [ ] Errors are wrapped with context using `fmt.Errorf` and `%w` verb
- [ ] Error messages are descriptive and include relevant values
- [ ] Typed errors used for business logic
- [ ] Error types can be checked with `errors.Is()` and `errors.As()`

### 2. Custom Error Types
```go
// ✅ Good custom errors
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s (value: %v): %s", e.Field, e.Value, e.Message)
}

var (
    ErrInvalidModel = errors.New("invalid model specified")
    ErrRateLimited  = errors.New("rate limit exceeded")
)
```
- [ ] Custom error types implement `error` interface
- [ ] Sentinel errors defined as package variables
- [ ] Error types provide structured information
- [ ] Error messages are user-friendly

## Security

### 1. Input Validation
```go
// ✅ Good input validation
func ValidateModel(model string) error {
    if len(model) == 0 {
        return &ValidationError{Field: "model", Err: errors.New("cannot be empty")}
    }
    if len(model) > 100 {
        return &ValidationError{Field: "model", Err: errors.New("exceeds maximum length")}
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`).MatchString(model) {
        return &ValidationError{Field: "model", Err: errors.New("contains invalid characters")}
    }
    return nil
}

// ❌ Bad input validation
func ValidateModel(model string) error {
    // No validation
    return nil
}
```
- [ ] All user inputs are validated
- [ ] Length limits enforced
- [ ] Character allowlists used (not denylists)
- [ ] No SQL injection vulnerabilities
- [ ] No path traversal vulnerabilities

### 2. Credential Management
```go
// ✅ Good credential handling
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

// ❌ Bad credential handling
type Config struct {
    APIKey string `json:"api_key"` // Will be logged
}
```
- [ ] API keys never logged or exposed
- [ ] Sensitive data cleared from memory
- [ ] Credentials stored securely
- [ ] String() methods redact sensitive data

### 3. Network Security
- [ ] TLS 1.3 minimum for external connections
- [ ] Certificate validation enabled
- [ ] Request/response size limits enforced
- [ ] Timeouts configured appropriately
- [ ] Rate limiting implemented

## Performance

### 1. Memory Management
```go
// ✅ Good memory management
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 4096)
    },
}

func processData(data []byte) error {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf[:0])
    
    // Use buf for processing
    return nil
}

// ❌ Bad memory management
func processData(data []byte) error {
    buf := make([]byte, 4096) // New allocation every call
    // Process without pooling
    return nil
}
```
- [ ] Object pools used for frequent allocations
- [ ] Slices pre-allocated with known capacity
- [ ] String concatenation uses `strings.Builder`
- [ ] Unnecessary allocations avoided in hot paths
- [ ] Memory leaks prevented

### 2. Efficient Operations
- [ ] HTTP connections reused (connection pooling)
- [ ] JSON marshaling avoided in hot paths
- [ ] Efficient data structures chosen
- [ ] Zero-copy operations where possible
- [ ] Appropriate buffer sizes for channels

### 3. Concurrency
```go
// ✅ Good concurrency
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

// ❌ Bad concurrency
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    for _, item := range items {
        go s.processItem(ctx, item) // Unbounded goroutines
    }
    return nil
}
```
- [ ] Goroutines bounded to prevent resource exhaustion
- [ ] Context used for cancellation
- [ ] WaitGroups or channels used for synchronization
- [ ] Race conditions avoided
- [ ] Deadlocks prevented

## Testing

### 1. Test Coverage
- [ ] Unit tests for all public functions
- [ ] Integration tests for external dependencies
- [ ] Error paths tested
- [ ] Edge cases covered
- [ ] Minimum 85% code coverage achieved

### 2. Test Quality
```go
// ✅ Good test structure
func TestProvider_CreateCompletion_Success(t *testing.T) {
    // Arrange
    provider := setupTestProvider(t)
    req := &CompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []Message{{Role: "user", Content: "Hello"}},
    }
    
    // Act
    resp, err := provider.CreateCompletion(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, "Hello there!", resp.Choices[0].Message.Content)
}

// ❌ Bad test structure
func TestStuff(t *testing.T) {
    // Everything mixed together, unclear what's being tested
}
```
- [ ] Table-driven tests used for multiple scenarios
- [ ] Arrange, Act, Assert pattern followed
- [ ] Test names describe the scenario
- [ ] Mocks used appropriately
- [ ] Tests are deterministic (no flaky tests)

### 3. Benchmarks
- [ ] Benchmark tests for performance-critical code
- [ ] Memory allocation tracking (`b.ReportAllocs()`)
- [ ] Performance regressions detected
- [ ] Baseline measurements established

## Documentation

### 1. Code Documentation
```go
// ✅ Good documentation
// CreateCompletion creates a chat completion using the specified model.
//
// The request is validated before sending to ensure all required fields
// are present and within acceptable ranges. The context is used for
// cancellation and timeout control.
//
// Example:
//    req := CompletionRequest{
//        Model: "gpt-3.5-turbo",
//        Messages: []Message{{Role: "user", Content: "Hello"}},
//    }
//    resp, err := provider.CreateCompletion(ctx, req)
//    if err != nil {
//        return fmt.Errorf("completion failed: %w", err)
//    }
//
// Returns an error if the request is invalid or the API call fails.
func (p *Provider) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

// ❌ Bad documentation
// CreateCompletion does stuff
func (p *Provider) CreateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
```
- [ ] All public functions have godoc comments
- [ ] Package comments explain purpose and usage
- [ ] Examples provided for complex APIs
- [ ] Error conditions documented
- [ ] Usage patterns explained

### 2. README and Docs
- [ ] README updated if functionality changes
- [ ] API documentation updated
- [ ] Configuration examples provided
- [ ] Migration guides for breaking changes

## Code Quality

### 1. Readability
- [ ] Code is self-documenting
- [ ] Variable names are descriptive
- [ ] Comments explain "why", not "what"
- [ ] Consistent formatting (gofmt)
- [ ] Logical code organization

### 2. Maintainability
- [ ] DRY principle followed (Don't Repeat Yourself)
- [ ] KISS principle followed (Keep It Simple, Stupid)
- [ ] No premature optimization
- [ ] Code is easy to modify and extend
- [ ] Technical debt minimized

### 3. Dependencies
- [ ] Minimal external dependencies
- [ ] Dependencies are well-maintained
- [ ] No circular dependencies
- [ ] Vendor directory updated if needed

## Anti-Patterns Check

### 1. Forbidden Practices
- [ ] ❌ No global variables (except configuration)
- [ ] ❌ No unhandled errors (`_` assignments)
- [ ] ❌ No empty interfaces (`interface{}`)
- [ ] ❌ No goroutines without context
- [ ] ❌ No string concatenation in loops
- [ ] ❌ No unnecessary JSON marshaling

### 2. Performance Anti-Patterns
- [ ] ❌ No unbounded goroutine creation
- [ ] ❌ No HTTP connection creation per request
- [ ] ❌ No excessive allocations in hot paths
- [ ] ❌ No blocking operations without timeouts

### 3. Security Anti-Patterns
- [ ] ❌ No credential logging
- [ ] ❌ No SQL injection vulnerabilities
- [ ] ❌ No path traversal vulnerabilities
- [ ] ❌ No insecure TLS configuration

## Checklist Summary

### Before Approving
- [ ] All critical items addressed
- [ ] No security vulnerabilities
- [ ] Performance impact acceptable
- [ ] Tests provide adequate coverage
- [ ] Documentation is complete
- [ ] Code follows Go best practices
- [ ] Architecture principles respected

### Final Verification
- [ ] CI/CD pipeline passes
- [ ] Integration tests pass
- [ ] Performance benchmarks stable
- [ ] Security scan clean
- [ ] Code review comments addressed

## Reviewer Notes Template

```markdown
## Review Summary
- **Overall Assessment**: [Approve/Request Changes/Comment]
- **Performance Impact**: [None/Positive/Negative - explain]
- **Security Concerns**: [None/List any concerns]
- **Test Coverage**: [Adequate/Needs improvement - explain]

## Required Changes
1. [List any blocking issues]

## Suggestions
1. [List non-blocking improvements]

## Questions
1. [Any clarifying questions]
```

---

**Remember**: This checklist ensures we maintain GOLLM's high standards while building a tool that developers can rely on. Every item matters for creating production-ready, enterprise-grade software.