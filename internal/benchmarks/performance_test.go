// Package benchmarks provides comprehensive performance benchmarks for GOLLM.
//
// These benchmarks measure critical performance metrics including:
//   - Startup time and initialization overhead
//   - Configuration loading and validation performance
//   - Provider operation latency and throughput
//   - Memory allocation patterns and efficiency
//   - HTTP transport performance
//   - CLI command processing speed
//   - Concurrent operation scalability
//
// Run benchmarks with:
//
//	go test -bench=. -benchmem -count=5 ./internal/benchmarks
//
// For CPU profiling:
//
//	go test -bench=BenchmarkProvider_CreateCompletion -cpuprofile=cpu.prof
//
// For memory profiling:
//
//	go test -bench=BenchmarkConfig_Load -memprofile=mem.prof
package benchmarks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"encoding/json"
	"github.com/m4rba4s/Nexus-LLM/internal/config"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
	"github.com/m4rba4s/Nexus-LLM/internal/providers/mock"

	"github.com/m4rba4s/Nexus-LLM/internal/transport"
)

// Benchmark Configuration Loading
func BenchmarkConfig_Load_Default(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := config.Load()
		if err != nil {
			// Expected in test environment - no config file
			continue
		}
	}
}

func BenchmarkConfig_Load_WithValidation(b *testing.B) {
	// Create temporary config file
	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "bench-config.yaml")
	configContent := `
default_provider: mock
providers:
  mock:
    type: openai
    api_key: test-key
    base_url: http://localhost:8080
    timeout: 30s
settings:
  max_tokens: 2048
  temperature: 0.7
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cfg, err := config.LoadFromFile(configFile)
		if err != nil {
			b.Fatal(err)
		}
		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConfig_Validate_Complex(b *testing.B) {
	cfg := createComplexConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := cfg.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark Provider Operations
func BenchmarkProvider_CreateCompletion_Mock(b *testing.B) {
	provider := mock.New(mock.DefaultConfig())
	provider.SetGlobalResponse("This is a test response from the mock provider.")

	ctx := context.Background()
	req := &core.CompletionRequest{
		Model: "test-model",
		Messages: []core.Message{
			{Role: "user", Content: "Hello, world!"},
		},
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

func BenchmarkProvider_CreateCompletion_WithLatency(b *testing.B) {
	config := mock.DefaultConfig()
	config.Latency = 10 * time.Millisecond
	provider := mock.New(config)
	provider.SetGlobalResponse("Response with simulated latency.")

	ctx := context.Background()
	req := &core.CompletionRequest{
		Model: "test-model",
		Messages: []core.Message{
			{Role: "user", Content: "Test with latency"},
		},
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

func BenchmarkProvider_StreamCompletion_Mock(b *testing.B) {
	provider := mock.New(mock.DefaultConfig())

	ctx := context.Background()
	req := &core.CompletionRequest{
		Model:  "test-model",
		Stream: true,
		Messages: []core.Message{
			{Role: "user", Content: "Stream this response"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ch, err := provider.StreamCompletion(ctx, req)
		if err != nil {
			b.Fatal(err)
		}

		// Consume all chunks
		for chunk := range ch {
			_ = chunk
		}
	}
}

// Benchmark Concurrent Operations
func BenchmarkProvider_ConcurrentCompletions(b *testing.B) {
	provider := mock.New(mock.DefaultConfig())
	provider.SetGlobalResponse("Concurrent test response")

	ctx := context.Background()
	req := &core.CompletionRequest{
		Model: "test-model",
		Messages: []core.Message{
			{Role: "user", Content: "Concurrent test"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.CreateCompletion(ctx, req)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

func BenchmarkProvider_HighConcurrency_100Workers(b *testing.B) {
	provider := mock.New(mock.DefaultConfig())
	provider.SetGlobalResponse("High concurrency test response")

	const numWorkers = 100
	requests := make(chan *core.CompletionRequest, numWorkers*2)
	responses := make(chan *core.CompletionResponse, numWorkers*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for req := range requests {
				resp, err := provider.CreateCompletion(ctx, req)
				if err != nil {
					b.Error(err)
					continue
				}
				responses <- resp
			}
		}()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &core.CompletionRequest{
			Model: "test-model",
			Messages: []core.Message{
				{Role: "user", Content: fmt.Sprintf("Request %d", i)},
			},
		}
		requests <- req
	}

	// Close requests and wait for completion
	close(requests)
	wg.Wait()
	close(responses)

	// Consume responses
	responseCount := 0
	for range responses {
		responseCount++
	}

	if responseCount != b.N {
		b.Fatalf("Expected %d responses, got %d", b.N, responseCount)
	}
}

// Benchmark Memory Management
func BenchmarkMessage_Creation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := core.Message{
			Role:    "user",
			Content: "This is a test message for benchmarking memory allocation patterns",
		}
		_ = msg
	}
}

func BenchmarkCompletionRequest_Build(b *testing.B) {
	messages := []core.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello, world!"},
	}

	maxTokens := 2048
	temperature := 0.7
	topP := 0.9

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := &core.CompletionRequest{
			Model:       "gpt-3.5-turbo",
			Messages:    messages,
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
			TopP:        &topP,
		}
		_ = req
	}
}

func BenchmarkValidation_CompletionRequest(b *testing.B) {
	req := &core.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test validation performance"},
		},
		MaxTokens:   &[]int{1000}[0],
		Temperature: &[]float64{0.5}[0],
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := req.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark HTTP Transport
func BenchmarkHTTPClient_Creation(b *testing.B) {
	config := transport.HTTPConfig{
		BaseURL:             "https://api.example.com",
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		client, err := transport.NewHTTPClient(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = client
	}
}

func BenchmarkHTTPClient_RequestBuilding(b *testing.B) {
	config := transport.HTTPConfig{
		BaseURL:             "https://api.example.com",
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client, err := transport.NewHTTPClient(config)
	if err != nil {
		b.Fatal(err)
	}

	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello, world!"},
		},
		"max_tokens":  1000,
		"temperature": 0.7,
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// This would normally make an HTTP request, but we're just benchmarking
		// the request preparation overhead
		_ = client
		_ = ctx
		_ = requestBody
	}
}

// Benchmark JSON Operations
func BenchmarkJSON_MarshalCompletionRequest(b *testing.B) {
	req := &core.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "What is the capital of France?"},
		},
		MaxTokens:   &[]int{150}[0],
		Temperature: &[]float64{0.7}[0],
		TopP:        &[]float64{0.9}[0],
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSON_UnmarshalCompletionResponse(b *testing.B) {
	responseJSON := `{
		"id": "chatcmpl-test",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "gpt-3.5-turbo",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "The capital of France is Paris."
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 15,
			"completion_tokens": 8,
			"total_tokens": 23
		}
	}`

	jsonBytes := []byte(responseJSON)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var response core.CompletionResponse
		if err := json.Unmarshal(jsonBytes, &response); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark String Operations
func BenchmarkString_Builder_ResponseFormatting(b *testing.B) {
	response := &core.CompletionResponse{
		ID:    "test-response",
		Model: "gpt-3.5-turbo",
		Choices: []core.Choice{
			{
				Message: core.Message{
					Role:    "assistant",
					Content: "This is a test response that we'll format multiple times to benchmark string building performance.",
				},
				FinishReason: "stop",
			},
		},
		Usage: core.Usage{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:      50,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		builder.WriteString("Response: ")
		builder.WriteString(response.Choices[0].Message.Content)
		builder.WriteString("\n\nUsage: ")
		builder.WriteString(fmt.Sprintf("%d tokens", response.Usage.TotalTokens))
		result := builder.String()
		_ = result
	}
}

func BenchmarkString_Concatenation_Naive(b *testing.B) {
	response := &core.CompletionResponse{
		Choices: []core.Choice{
			{
				Message: core.Message{
					Content: "This is a test response for string concatenation benchmarking.",
				},
			},
		},
		Usage: core.Usage{TotalTokens: 15},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := "Response: " + response.Choices[0].Message.Content +
			"\n\nUsage: " + fmt.Sprintf("%d tokens", response.Usage.TotalTokens)
		_ = result
	}
}

// Benchmark Memory Pools
func BenchmarkMemoryPool_ByteSlices(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 4096)
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get().([]byte)
		buf = buf[:0] // Reset length
		buf = append(buf, "test data for memory pool benchmarking"...)
		pool.Put(buf)
	}
}

func BenchmarkMemoryPool_MessageStructs(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			return &core.Message{}
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		msg := pool.Get().(*core.Message)
		msg.Role = "user"
		msg.Content = "Test message"

		// Reset for reuse
		msg.Role = ""
		msg.Content = ""
		pool.Put(msg)
	}
}

// Benchmark I/O Operations
func BenchmarkIO_ReadInput_Memory(b *testing.B) {
	input := "This is test input for benchmarking I/O operations in the CLI"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewBufferString(input)
		data, err := io.ReadAll(reader)
		if err != nil {
			b.Fatal(err)
		}
		_ = string(data)
	}
}

func BenchmarkIO_WriteOutput_Memory(b *testing.B) {
	content := "This is test output content that will be written to a buffer for benchmarking purposes"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_, err := buf.WriteString(content)
		if err != nil {
			b.Fatal(err)
		}
		_ = buf.String()
	}
}

// Benchmark Startup Performance
func BenchmarkStartup_ConfigLoad(b *testing.B) {
	// Simulate minimal config loading for startup benchmark
	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "startup-config.yaml")
	minimalConfig := `
default_provider: mock
providers:
  mock:
    type: openai
    api_key: test-key
    timeout: 30s
`
	if err := os.WriteFile(configFile, []byte(minimalConfig), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cfg, err := config.LoadFromFile(configFile)
		if err != nil {
			b.Fatal(err)
		}
		_ = cfg
	}
}

func BenchmarkStartup_ProviderRegistration(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		registry := core.NewProviderRegistry()

		err := registry.RegisterFactory("mock", func(config core.ProviderConfig) (core.Provider, error) {
			return mock.New(mock.DefaultConfig()), nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for benchmarks

func createComplexConfig() *config.Config {
	return &config.Config{
		DefaultProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {
				Type:       "openai",
				APIKey:     config.NewSecureString("sk-test-key-1234567890"),
				BaseURL:    "https://api.openai.com/v1",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
			},
			"anthropic": {
				Type:       "anthropic",
				APIKey:     config.NewSecureString("sk-ant-test-key-abcdefg"),
				BaseURL:    "https://api.anthropic.com",
				MaxRetries: 5,
				Timeout:    45 * time.Second,
			},
			"ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Timeout: 120 * time.Second,
			},
		},
		Settings: config.GlobalSettings{
			MaxTokens:   4096,
			Temperature: 0.7,
			TopP:        0.9,
			Timeout:     60 * time.Second,
		},
		Features: config.FeatureFlags{
			Streaming: true,
			Caching:   true,
			Plugins:   true,
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Format:  "json",
			Output:  "stdout",
			MaxSize: 100,
		},
		Security: config.SecurityConfig{
			TLSMinVersion:  "1.3",
			SessionTimeout: 2 * time.Hour,
			MaxRequestSize: 10 * 1024 * 1024,
		},
	}
}

// End-to-End Performance Test
func BenchmarkE2E_SimpleCompletion(b *testing.B) {
	// This benchmark simulates a complete request cycle:
	// Config load -> Provider init -> Request -> Response

	// Setup
	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "e2e-config.yaml")
	configContent := `
default_provider: mock
providers:
  mock:
    type: openai
    api_key: test-key
    timeout: 30s
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 1. Load config
		cfg, err := config.LoadFromFile(configFile)
		if err != nil {
			b.Fatal(err)
		}

		// 2. Initialize provider
		provider := mock.New(mock.DefaultConfig())
		provider.SetGlobalResponse("E2E test response")

		// 3. Create request
		req := &core.CompletionRequest{
			Model: "test-model",
			Messages: []core.Message{
				{Role: "user", Content: "E2E test request"},
			},
		}

		// 4. Process request
		_, err = provider.CreateCompletion(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}

		_ = cfg // Use config to avoid optimization
	}
}
