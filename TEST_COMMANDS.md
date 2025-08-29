# GOLLM LLM Operator Testing - Quick Reference

This document provides quick commands to run all available LLM operator tests for GOLLM CLI.

## Prerequisites

1. Ensure you're in the `gollm-cli` directory
2. Have Go installed (1.21+)
3. Configuration file `config.yaml` should exist with provider settings

## Test Files Overview

| File | Purpose | Duration | Best For |
|------|---------|----------|----------|
| `test_providers.go` | Basic connectivity | 1-2 min | Initial setup validation |
| `test_operators_simple.go` | Comprehensive testing | 2-3 min | Development & debugging |
| `test_operators_quick.go` | Fast integration | 30-60 sec | CI/CD pipelines |
| `test_operators_benchmark.go` | Performance analysis | 5-10 min | Production readiness |

## Quick Commands

### 1. Basic Provider Testing
```bash
# Test basic provider connectivity and responses
go run test_providers.go

# Expected output:
# 🧪 GOLLM Provider Testing Tool
# ✅ Mock provider ready
# ❌ OpenAI: API key invalid
# ✅ Gemini: Response received
```

### 2. Comprehensive Operator Testing (Recommended)
```bash
# Run full operator test suite with detailed analysis
go run test_operators_simple.go

# Expected output:
# 🧪 GOLLM Operator Testing Tool
# 📋 Basic Completion: ✅ mock (100ms), ❌ openai (401 error)
# 📊 Overall: 18/36 passed (50.0% success rate)
# 🏆 Provider ranking with performance metrics
# 🔍 Error analysis and recommendations
```

### 3. Quick Integration Testing
```bash
# Fast validation for CI/CD
go run test_operators_quick.go

# Expected output:
# ⚡ GOLLM Quick Operator Integration Test
# ✅ Passed: 12, ❌ Failed: 18, ⏭️ Skipped: 0
# Success Rate: 40.0%
# 💡 Quick recommendations
```

### 4. Advanced Benchmarking
```bash
# Full performance benchmark (takes longer)
go run test_operators_benchmark.go

# With options:
go run test_operators_benchmark.go --quick    # Reduced iterations
go run test_operators_benchmark.go --stress   # Include stress testing
go run test_operators_benchmark.go --export   # Save results to JSON
```

### 5. Capability Testing (Advanced)
```bash
# Test specific LLM capabilities with intelligent scoring
go run test_llm_operators.go

# Tests math, logic, text processing, code generation, etc.
```

## Test Scenarios Covered

### All Tests Include:
- ✅ **Basic Completion**: Text generation with simple prompts
- ✅ **Error Handling**: Invalid request validation
- ✅ **Model Discovery**: Available model listing
- ✅ **Configuration**: Provider setup validation

### Extended Tests Include:
- ✅ **Streaming**: Real-time response handling
- ✅ **Concurrency**: Multi-threaded request processing
- ✅ **Context Cancellation**: Timeout and cancellation handling
- ✅ **Token Limits**: Response length management
- ✅ **Cost Analysis**: Token usage and pricing

### Advanced Tests Include:
- ✅ **Performance Benchmarking**: Latency, throughput, P95/P99 metrics
- ✅ **Stress Testing**: Breaking point detection
- ✅ **Memory Profiling**: Resource usage analysis
- ✅ **Load Testing**: Concurrent user simulation

## Interpreting Results

### Success Indicators
```
✅ PASS - Test completed successfully
🥇 🥈 🥉 - Performance ranking (gold, silver, bronze)
⚡ - Fastest response time
📊 - Average performance metrics
```

### Warning Indicators
```
⚠️ SKIP - Feature not supported (normal)
⏭️ - Test skipped due to configuration
```

### Failure Indicators
```
❌ FAIL - Test failed
🔑 Authentication Issues - Invalid API keys
⏱️ Timeout Issues - Request timeouts
🌐 Network Issues - Connection problems
```

## Common Use Cases

### Development Workflow
```bash
# 1. Quick validation during development
go run test_operators_quick.go

# 2. Detailed analysis when needed
go run test_operators_simple.go

# 3. Full benchmark before release
go run test_operators_benchmark.go --quick
```

### CI/CD Integration
```bash
# Add to your CI pipeline
go run test_operators_quick.go
if [ $? -eq 0 ]; then
    echo "✅ LLM operators working correctly"
else
    echo "❌ LLM operator tests failed"
    exit 1
fi
```

### Production Readiness
```bash
# Full validation suite
go run test_operators_simple.go > test_results.log
go run test_operators_benchmark.go --export > benchmark_results.json
```

### Debugging Issues
```bash
# Run with verbose output
go run test_operators_simple.go > debug_output.log 2>&1

# Focus on specific provider
# (Edit config.yaml to enable only one provider)
```

## Configuration Tips

### Mock Provider (Always Works)
```yaml
providers:
  mock:
    type: mock
    api_key: mock-api-key
    timeout: 1s
    default_model: mock-gpt-3.5-turbo
```

### Real Provider Example
```yaml
providers:
  gemini:
    type: gemini
    api_key: YOUR_REAL_API_KEY_HERE
    base_url: https://generativelanguage.googleapis.com/v1beta
    default_model: gemini-2.0-flash
```

## Troubleshooting

### All Tests Failing
1. Check `config.yaml` exists and has valid provider configurations
2. Verify network connectivity
3. Ensure API keys are valid and have quota

### Streaming Tests Timing Out
1. This is normal for some providers
2. Indicates streaming implementation may need work
3. Not a critical failure for basic functionality

### Mock Provider Failing
1. Should never happen - indicates core implementation issue
2. Check Go modules are properly installed
3. Verify no conflicting dependencies

## Performance Baselines

### Expected Performance (Mock Provider)
- Basic Completion: ~100ms
- Streaming: Variable (implementation dependent)
- Model Discovery: ~10ms
- Error Handling: <1ms
- Concurrency: 200ms for 3 concurrent requests

### Expected Performance (Real Providers)
- Basic Completion: 300-2000ms (varies by provider)
- Streaming: 1-5 seconds for multi-chunk responses
- Model Discovery: 100-1000ms
- Geographic latency affects all metrics

## Next Steps

After running tests:

1. **Fix Authentication**: Update API keys for failing providers
2. **Monitor Performance**: Track metrics over time
3. **Optimize Configuration**: Adjust timeouts and retry settings
4. **Implement Fallbacks**: Use fastest/most reliable providers
5. **Cost Analysis**: Monitor token usage and costs

## Support

For issues or enhancements:
1. Check `OPERATOR_TESTING_IMPLEMENTATION.md` for detailed technical docs
2. Review test output logs for specific error messages
3. Verify provider-specific configuration requirements
4. Consider provider-specific limitations and quotas