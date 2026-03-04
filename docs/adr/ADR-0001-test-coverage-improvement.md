# ADR-0001: Test Coverage Improvement Strategy

## Status
**Proposed** - 2025-01-09

## Context

The NexusLLM CLI project currently has approximately 25% test coverage, significantly below the target of 85% for enterprise-grade software. This gap represents a technical debt that impacts:

- **Reliability**: Untesteчd code paths may contain hidden bugs
- **Maintainability**: Refactoring without tests is risky
- **Performance**: No benchmark baseline for optimization
- **Security**: Potential vulnerabilities in untested code
- **Developer Confidence**: Fear of breaking existing functionality

### Current Testing State
- **Unit Tests**: 16 test files out of 65 Go files
- **Integration Tests**: Basic structure exists but incomplete
- **Benchmark Tests**: Missing for critical paths
- **Security Tests**: Limited to audit_test.go
- **E2E Tests**: Framework present but underutilized

### Critical Untested Areas
1. Provider implementations (streaming, error handling)
2. TUI components and user interactions
3. Configuration management and validation
4. Security validators and input sanitization
5. Concurrent operations and race conditions

## Decision

Implement a phased test coverage improvement strategy targeting 85% coverage within 4 weeks:

### Phase 1: Foundation (Week 1)
```go
// Focus: Core business logic
- Provider interface implementations
- Core types and transformations
- Configuration loading and validation
- Error handling paths
```

### Phase 2: Integration (Week 2)
```go
// Focus: Component interactions
- Provider-to-core integration
- CLI command execution paths
- Configuration cascade (flags → env → file)
- Stream processing pipelines
```

### Phase 3: Performance (Week 3)
```go
// Focus: Benchmarks and optimization
- Benchmark suite for critical paths
- Memory allocation profiling
- Concurrent operation testing
- Race detection validation
```

### Phase 4: Security & E2E (Week 4)
```go
// Focus: Security and full workflows
- Input validation comprehensive testing
- Security vulnerability scanning
- End-to-end user workflows
- Chaos testing for resilience
```

## Implementation Strategy

### 1. Test Structure Template
```go
package package_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeature_MethodName(t *testing.T) {
    t.Parallel() // Enable parallel execution
    
    tests := []struct {
        name      string
        setup     func() *Feature
        input     Input
        want      Output
        wantErr   error
        timeout   time.Duration
    }{
        {
            name: "successful_operation",
            setup: func() *Feature {
                return NewFeature(Config{})
            },
            input:   Input{Value: "test"},
            want:    Output{Result: "success"},
            wantErr: nil,
            timeout: 5 * time.Second,
        },
        {
            name: "error_condition",
            setup: func() *Feature {
                return NewFeature(Config{Invalid: true})
            },
            input:   Input{},
            wantErr: ErrInvalidInput,
            timeout: 1 * time.Second,
        },
    }
    
    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
            ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
            defer cancel()
            
            feature := tt.setup()
            got, err := feature.Method(ctx, tt.input)
            
            if tt.wantErr != nil {
                require.Error(t, err)
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 2. Benchmark Template
```go
func BenchmarkFeature_Method(b *testing.B) {
    scenarios := []struct {
        name  string
        size  int
        setup func() *Feature
    }{
        {"small", 10, setupSmall},
        {"medium", 100, setupMedium},
        {"large", 1000, setupLarge},
    }
    
    for _, s := range scenarios {
        b.Run(s.name, func(b *testing.B) {
            feature := s.setup()
            input := generateInput(s.size)
            
            b.ResetTimer()
            b.ReportAllocs()
            
            for i := 0; i < b.N; i++ {
                _, _ = feature.Method(context.Background(), input)
            }
        })
    }
}
```

### 3. Fuzzing Template
```go
func FuzzInputValidation(f *testing.F) {
    // Seed corpus
    f.Add([]byte("valid input"))
    f.Add([]byte(""))
    f.Add([]byte(strings.Repeat("a", 10000)))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        validator := NewValidator()
        err := validator.Validate(data)
        
        // Should never panic
        if err != nil {
            // Validate error is well-formed
            assert.NotEmpty(t, err.Error())
        }
    })
}
```

### 4. Mock Generation
```go
//go:generate mockgen -source=provider.go -destination=mocks/provider_mock.go -package=mocks

type MockProvider struct {
    ctrl *gomock.Controller
}

func (m *MockProvider) CreateCompletion(ctx context.Context, req Request) (*Response, error) {
    // Mock implementation
}
```

## Testing Priorities

### Critical Path Coverage (Priority 1)
1. `internal/core/provider.go` - Provider interface
2. `internal/providers/*/client.go` - Provider implementations
3. `internal/cli/commands/*.go` - CLI commands
4. `internal/security/validators.go` - Input validation
5. `internal/config/config.go` - Configuration management

### Integration Points (Priority 2)
1. Provider registry and selection
2. Streaming response handling
3. Error propagation and recovery
4. Configuration precedence
5. TUI interaction flows

### Performance Critical (Priority 3)
1. Response streaming performance
2. Memory allocation in hot paths
3. Concurrent request handling
4. Configuration parsing speed
5. Startup time optimization

## Tools and Infrastructure

### Required Tools
```bash
# Testing tools
go install github.com/golang/mock/mockgen@latest
go install gotest.tools/gotestsum@latest
go install github.com/axw/gocov/gocov@latest
go install github.com/AlekSi/gocov-xml@latest

# Security tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### CI/CD Integration
```yaml
# .github/workflows/ci.yml enhancement
test:
  runs-on: ubuntu-latest
  steps:
    - name: Run tests with coverage
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        
    - name: Check coverage threshold
      run: |
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage < 85" | bc -l) )); then
          echo "Coverage $coverage% is below 85% threshold"
          exit 1
        fi
        
    - name: Run benchmarks
      run: go test -bench=. -benchmem ./...
      
    - name: Security scan
      run: gosec -fmt sarif -out gosec.sarif ./...
```

## Consequences

### Positive
- **Increased Reliability**: Bugs caught before production
- **Faster Development**: Confident refactoring
- **Better Performance**: Baseline benchmarks for optimization
- **Enhanced Security**: Vulnerability detection
- **Documentation**: Tests serve as usage examples
- **Team Confidence**: Reduced fear of changes

### Negative
- **Initial Time Investment**: ~160 hours of development
- **Maintenance Overhead**: Tests need updates with code changes
- **CI/CD Time**: Longer build times with comprehensive tests
- **Learning Curve**: Team needs to adopt testing best practices

### Mitigation
- Parallelize test execution to reduce CI time
- Use test generators for boilerplate reduction
- Implement test helpers for common patterns
- Create testing guidelines and templates
- Regular test refactoring sessions

## Alternatives Considered

### Alternative 1: Minimal Testing (50% coverage)
- **Pros**: Faster to implement, less maintenance
- **Cons**: Insufficient for enterprise requirements
- **Rejected**: Does not meet quality standards

### Alternative 2: 100% Coverage
- **Pros**: Maximum confidence
- **Cons**: Diminishing returns, excessive time investment
- **Rejected**: Not cost-effective for all code paths

### Alternative 3: Property-Based Testing Only
- **Pros**: Discovers edge cases automatically
- **Cons**: Complex setup, not suitable for all scenarios
- **Rejected**: Should complement, not replace unit tests

## Metrics for Success

### Quantitative Metrics
- Test coverage ≥ 85%
- All benchmarks established
- Zero critical security findings
- Test execution time < 2 minutes
- Zero race conditions detected

### Qualitative Metrics
- Developer confidence in making changes
- Reduced production incidents
- Faster feature development
- Improved code review quality
- Better onboarding for new developers

## Review Schedule

- **Week 1 Review**: Foundation tests complete
- **Week 2 Review**: Integration tests complete
- **Week 3 Review**: Benchmarks established
- **Week 4 Review**: Security and E2E complete
- **Month 2**: Maintenance and optimization
- **Quarterly**: Coverage and performance review

## References

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Benchmarking in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Fuzzing in Go](https://go.dev/doc/tutorial/fuzz)
- [Test Coverage Guidelines](https://testing.googleblog.com/2020/08/code-coverage-best-practices.html)

---

**Author**: GOLLM Dev Operator  
**Date**: 2025-01-09  
**Version**: 1.0.0  
**Next Review**: 2025-02-09
