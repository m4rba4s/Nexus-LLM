package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/providers/anthropic"
	"github.com/yourusername/gollm/internal/providers/deepseek"
	"github.com/yourusername/gollm/internal/providers/gemini"
	"github.com/yourusername/gollm/internal/providers/mock"
	"github.com/yourusername/gollm/internal/providers/openai"
	"github.com/yourusername/gollm/internal/providers/openrouter"
)

// BenchmarkConfig holds configuration for benchmark runs
type BenchmarkConfig struct {
	WarmupRuns          int           `json:"warmup_runs"`
	BenchmarkRuns       int           `json:"benchmark_runs"`
	ConcurrentUsers     []int         `json:"concurrent_users"`
	RequestTimeout      time.Duration `json:"request_timeout"`
	TotalTestTimeout    time.Duration `json:"total_test_timeout"`
	TokenLimits         []int         `json:"token_limits"`
	TemperatureValues   []float64     `json:"temperature_values"`
	EnableMemoryProfile bool          `json:"enable_memory_profile"`
	EnableStressTest    bool          `json:"enable_stress_test"`
}

// BenchmarkResult stores comprehensive metrics for a single benchmark
type BenchmarkResult struct {
	Provider        string        `json:"provider"`
	Operation       string        `json:"operation"`
	ConcurrentUsers int           `json:"concurrent_users"`
	TotalRequests   int           `json:"total_requests"`
	SuccessfulReqs  int           `json:"successful_requests"`
	FailedReqs      int           `json:"failed_requests"`
	AvgLatency      time.Duration `json:"avg_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	P50Latency      time.Duration `json:"p50_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	Throughput      float64       `json:"throughput"` // requests per second
	TokenThroughput float64       `json:"token_throughput"` // tokens per second
	TotalTokens     int           `json:"total_tokens"`
	TotalCost       float64       `json:"total_cost"`
	CostPerToken    float64       `json:"cost_per_token"`
	ErrorRate       float64       `json:"error_rate"`
	MemoryUsage     int64         `json:"memory_usage_mb"`
	StartTime       time.Time     `json:"start_time"`
	Duration        time.Duration `json:"duration"`
	Errors          []string      `json:"errors,omitempty"`
}

// OperatorBenchmarker performs comprehensive benchmarking of LLM operators
type OperatorBenchmarker struct {
	providers map[string]core.Provider
	config    BenchmarkConfig
	results   []BenchmarkResult
	mu        sync.RWMutex
}

func main() {
	fmt.Println("🚀 GOLLM Advanced Operator Benchmarking Suite")
	fmt.Println("=" + strings.Repeat("=", 55))

	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		return
	}

	// Initialize benchmarker
	benchmarker := NewOperatorBenchmarker()

	// Load providers
	if err := benchmarker.initializeProviders(); err != nil {
		log.Fatalf("❌ Failed to initialize providers: %v", err)
	}

	// Run comprehensive benchmarks
	benchmarker.runBenchmarks()

	// Generate detailed report
	benchmarker.generateReport()

	// Export results if requested
	if len(os.Args) > 1 && strings.Contains(strings.Join(os.Args, " "), "--export") {
		benchmarker.exportResults()
	}
}

func printUsage() {
	fmt.Println(`
🧪 GOLLM Advanced Operator Benchmarking Suite

USAGE:
  go run test_operators_benchmark.go [options]

OPTIONS:
  --help, -h     Show this help message
  --export       Export results to JSON file
  --quick        Run quick benchmark (reduced iterations)
  --stress       Enable stress testing mode
  --memory       Enable memory profiling

BENCHMARKS:
  • Completion Latency     - Response time analysis
  • Throughput Testing     - Requests per second capacity
  • Concurrent Load        - Multi-user simulation
  • Token Efficiency       - Cost and speed per token
  • Streaming Performance  - Real-time response metrics
  • Error Resilience      - Failure rate under load
  • Memory Usage          - Resource consumption analysis
  • Stress Testing        - Breaking point detection

OUTPUT:
  • Statistical analysis with percentiles
  • Provider performance comparison
  • Cost efficiency metrics
  • Detailed error analysis
  • Performance recommendations
`)
}

func NewOperatorBenchmarker() *OperatorBenchmarker {
	config := BenchmarkConfig{
		WarmupRuns:          3,
		BenchmarkRuns:       10,
		ConcurrentUsers:     []int{1, 5, 10, 20},
		RequestTimeout:      30 * time.Second,
		TotalTestTimeout:    5 * time.Minute,
		TokenLimits:         []int{10, 50, 100, 200},
		TemperatureValues:   []float64{0.0, 0.5, 1.0},
		EnableMemoryProfile: true,
		EnableStressTest:    contains(os.Args, "--stress"),
	}

	// Adjust for quick mode
	if contains(os.Args, "--quick") {
		config.WarmupRuns = 1
		config.BenchmarkRuns = 5
		config.ConcurrentUsers = []int{1, 5}
	}

	return &OperatorBenchmarker{
		providers: make(map[string]core.Provider),
		config:    config,
		results:   make([]BenchmarkResult, 0),
	}
}

func (ob *OperatorBenchmarker) initializeProviders() error {
	fmt.Println("\n🔧 Initializing providers for benchmarking...")

	// Always include mock provider
	mockProvider := mock.New(mock.DefaultConfig())
	ob.providers["mock"] = mockProvider
	fmt.Println("  ✅ Mock provider ready")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  ⚠️  Config not found, using mock provider only: %v\n", err)
		return nil
	}

	// Initialize real providers
	providerFactories := map[string]func(config.ProviderConfig) (core.Provider, error){
		"openai": func(cfg config.ProviderConfig) (core.Provider, error) {
			return openai.New(openai.Config{
				APIKey:     cfg.APIKey.Value(),
				BaseURL:    cfg.BaseURL,
				MaxRetries: cfg.MaxRetries,
				Timeout:    cfg.Timeout,
			})
		},
		"anthropic": func(cfg config.ProviderConfig) (core.Provider, error) {
			return anthropic.New(anthropic.Config{
				APIKey:     cfg.APIKey.Value(),
				BaseURL:    cfg.BaseURL,
				MaxRetries: cfg.MaxRetries,
				Timeout:    cfg.Timeout,
			})
		},
		"gemini": func(cfg config.ProviderConfig) (core.Provider, error) {
			return gemini.New(gemini.Config{
				APIKey:     cfg.APIKey.Value(),
				BaseURL:    cfg.BaseURL,
				MaxRetries: cfg.MaxRetries,
				Timeout:    cfg.Timeout,
				Headers:    cfg.CustomHeaders,
			})
		},
		"deepseek": func(cfg config.ProviderConfig) (core.Provider, error) {
			return deepseek.New(deepseek.Config{
				APIKey:     cfg.APIKey.Value(),
				BaseURL:    cfg.BaseURL,
				MaxRetries: cfg.MaxRetries,
				Timeout:    cfg.Timeout,
				Headers:    cfg.CustomHeaders,
			})
		},
		"openrouter": func(cfg config.ProviderConfig) (core.Provider, error) {
			return openrouter.New(openrouter.Config{
				APIKey:     cfg.APIKey.Value(),
				BaseURL:    cfg.BaseURL,
				MaxRetries: cfg.MaxRetries,
				Timeout:    cfg.Timeout,
				Headers:    cfg.CustomHeaders,
			})
		},
	}

	for name, providerConfig := range cfg.Providers {
		factory, exists := providerFactories[providerConfig.Type]
		if !exists {
			fmt.Printf("  ⚠️  Unsupported provider type: %s\n", providerConfig.Type)
			continue
		}

		provider, err := factory(providerConfig)
		if err != nil {
			fmt.Printf("  ⚠️  Failed to create %s: %v\n", name, err)
			continue
		}

		ob.providers[name] = provider
		fmt.Printf("  ✅ %s provider ready\n", name)
	}

	fmt.Printf("\n📊 Total providers for benchmarking: %d\n", len(ob.providers))
	return nil
}

func (ob *OperatorBenchmarker) runBenchmarks() {
	fmt.Println("\n🚀 Starting comprehensive benchmarks...\n")

	benchmarkSuites := []struct {
		name        string
		description string
		benchFunc   func(string, core.Provider)
	}{
		{"completion_latency", "Basic completion response time", ob.benchmarkCompletionLatency},
		{"throughput", "Maximum requests per second", ob.benchmarkThroughput},
		{"concurrent_load", "Multi-user concurrent performance", ob.benchmarkConcurrentLoad},
		{"token_efficiency", "Cost and speed per token analysis", ob.benchmarkTokenEfficiency},
		{"streaming", "Real-time streaming performance", ob.benchmarkStreaming},
		{"error_resilience", "Failure handling under stress", ob.benchmarkErrorResilience},
	}

	if ob.config.EnableStressTest {
		benchmarkSuites = append(benchmarkSuites, struct {
			name        string
			description string
			benchFunc   func(string, core.Provider)
		}{"stress_test", "Breaking point detection", ob.benchmarkStressTest})
	}

	for i, suite := range benchmarkSuites {
		fmt.Printf("📋 [%d/%d] %s - %s\n", i+1, len(benchmarkSuites),
			strings.ToUpper(strings.ReplaceAll(suite.name, "_", " ")), suite.description)
		fmt.Println(strings.Repeat("-", 50))

		for providerName, provider := range ob.providers {
			fmt.Printf("  🧪 Testing %s... ", providerName)
			start := time.Now()

			suite.benchFunc(providerName, provider)

			fmt.Printf("completed in %v\n", time.Since(start).Round(time.Millisecond))
		}
		fmt.Println()
	}

	fmt.Println("✅ All benchmarks completed!\n")
}

func (ob *OperatorBenchmarker) benchmarkCompletionLatency(providerName string, provider core.Provider) {
	ctx, cancel := context.WithTimeout(context.Background(), ob.config.RequestTimeout)
	defer cancel()

	var latencies []time.Duration
	var tokenCounts []int
	var costs []float64
	var errors []string

	// Warmup runs
	for i := 0; i < ob.config.WarmupRuns; i++ {
		ob.performSingleCompletion(ctx, provider, 50, 0.5, nil, nil, nil, nil)
	}

	// Benchmark runs
	for i := 0; i < ob.config.BenchmarkRuns; i++ {
		start := time.Now()
		resp, err := ob.performSingleCompletion(ctx, provider, 50, 0.5, nil, nil, nil, nil)
		latency := time.Since(start)

		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		latencies = append(latencies, latency)
		tokenCounts = append(tokenCounts, resp.Usage.TotalTokens)
		if resp.Usage.TotalCost != nil {
			costs = append(costs, *resp.Usage.TotalCost)
		}
	}

	ob.recordBenchmarkResult(BenchmarkResult{
		Provider:        providerName,
		Operation:       "completion_latency",
		ConcurrentUsers: 1,
		TotalRequests:   ob.config.BenchmarkRuns,
		SuccessfulReqs:  len(latencies),
		FailedReqs:      len(errors),
		AvgLatency:      calculateMean(latencies),
		MinLatency:      calculateMin(latencies),
		MaxLatency:      calculateMax(latencies),
		P50Latency:      calculatePercentile(latencies, 0.5),
		P95Latency:      calculatePercentile(latencies, 0.95),
		P99Latency:      calculatePercentile(latencies, 0.99),
		TotalTokens:     sum(tokenCounts),
		TotalCost:       sumFloat64(costs),
		ErrorRate:       float64(len(errors)) / float64(ob.config.BenchmarkRuns) * 100,
		Errors:          errors,
	})
}

func (ob *OperatorBenchmarker) benchmarkThroughput(providerName string, provider core.Provider) {
	ctx, cancel := context.WithTimeout(context.Background(), ob.config.TotalTestTimeout)
	defer cancel()

	const duration = 30 * time.Second
	var requestCount int64
	var successCount int64
	var totalTokens int64
	var totalCost float64
	var mu sync.Mutex

	start := time.Now()
	endTime := start.Add(duration)

	// Launch goroutines to make continuous requests
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ { // 5 concurrent workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(endTime) {
				resp, err := ob.performSingleCompletion(ctx, provider, 20, 0.3, nil, nil, nil, nil)
				atomic.AddInt64(&requestCount, 1)

				if err == nil {
					atomic.AddInt64(&successCount, 1)
					atomic.AddInt64(&totalTokens, int64(resp.Usage.TotalTokens))

					mu.Lock()
					if resp.Usage.TotalCost != nil {
						totalCost += *resp.Usage.TotalCost
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	actualDuration := time.Since(start)

	ob.recordBenchmarkResult(BenchmarkResult{
		Provider:        providerName,
		Operation:       "throughput",
		ConcurrentUsers: 5,
		TotalRequests:   int(requestCount),
		SuccessfulReqs:  int(successCount),
		FailedReqs:      int(requestCount - successCount),
		Throughput:      float64(successCount) / actualDuration.Seconds(),
		TokenThroughput: float64(totalTokens) / actualDuration.Seconds(),
		TotalTokens:     int(totalTokens),
		TotalCost:       totalCost,
		Duration:        actualDuration,
		ErrorRate:       float64(requestCount-successCount) / float64(requestCount) * 100,
	})
}

func (ob *OperatorBenchmarker) benchmarkConcurrentLoad(providerName string, provider core.Provider) {
	for _, concurrency := range ob.config.ConcurrentUsers {
		ob.benchmarkConcurrentLoadAtLevel(providerName, provider, concurrency)
	}
}

func (ob *OperatorBenchmarker) benchmarkConcurrentLoadAtLevel(providerName string, provider core.Provider, concurrency int) {
	ctx, cancel := context.WithTimeout(context.Background(), ob.config.TotalTestTimeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var latencies []time.Duration
	var tokenCounts []int
	var costs []float64
	var errors []string

	requestsPerUser := ob.config.BenchmarkRuns / concurrency
	if requestsPerUser < 1 {
		requestsPerUser = 1
	}

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < requestsPerUser; j++ {
				reqStart := time.Now()
				resp, err := ob.performSingleCompletion(ctx, provider, 30, 0.4, nil, nil, nil, nil)
				latency := time.Since(reqStart)

				mu.Lock()
				if err != nil {
					errors = append(errors, err.Error())
				} else {
					latencies = append(latencies, latency)
					tokenCounts = append(tokenCounts, resp.Usage.TotalTokens)
					if resp.Usage.TotalCost != nil {
						costs = append(costs, *resp.Usage.TotalCost)
					}
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	totalRequests := concurrency * requestsPerUser

	ob.recordBenchmarkResult(BenchmarkResult{
		Provider:        providerName,
		Operation:       "concurrent_load",
		ConcurrentUsers: concurrency,
		TotalRequests:   totalRequests,
		SuccessfulReqs:  len(latencies),
		FailedReqs:      len(errors),
		AvgLatency:      calculateMean(latencies),
		P50Latency:      calculatePercentile(latencies, 0.5),
		P95Latency:      calculatePercentile(latencies, 0.95),
		Throughput:      float64(len(latencies)) / duration.Seconds(),
		TotalTokens:     sum(tokenCounts),
		TotalCost:       sumFloat64(costs),
		Duration:        duration,
		ErrorRate:       float64(len(errors)) / float64(totalRequests) * 100,
		Errors:          errors,
	})
}

func (ob *OperatorBenchmarker) benchmarkTokenEfficiency(providerName string, provider core.Provider) {
	for _, tokenLimit := range ob.config.TokenLimits {
		ctx, cancel := context.WithTimeout(context.Background(), ob.config.RequestTimeout)

		var latencies []time.Duration
		var tokenCounts []int
		var costs []float64
		var errors []string

		for i := 0; i < ob.config.BenchmarkRuns; i++ {
			start := time.Now()
			resp, err := ob.performSingleCompletion(ctx, provider, tokenLimit, 0.3, nil, nil, nil, nil)
			latency := time.Since(start)

			if err != nil {
				errors = append(errors, err.Error())
				continue
			}

			latencies = append(latencies, latency)
			tokenCounts = append(tokenCounts, resp.Usage.TotalTokens)
			if resp.Usage.TotalCost != nil {
				costs = append(costs, *resp.Usage.TotalCost)
			}
		}

		cancel()

		totalTokens := sum(tokenCounts)
		totalCost := sumFloat64(costs)
		costPerToken := 0.0
		if totalTokens > 0 {
			costPerToken = totalCost / float64(totalTokens)
		}

		ob.recordBenchmarkResult(BenchmarkResult{
			Provider:        providerName,
			Operation:       fmt.Sprintf("token_efficiency_%d", tokenLimit),
			TotalRequests:   ob.config.BenchmarkRuns,
			SuccessfulReqs:  len(latencies),
			FailedReqs:      len(errors),
			AvgLatency:      calculateMean(latencies),
			TotalTokens:     totalTokens,
			TotalCost:       totalCost,
			CostPerToken:    costPerToken,
			ErrorRate:       float64(len(errors)) / float64(ob.config.BenchmarkRuns) * 100,
		})
	}
}

func (ob *OperatorBenchmarker) benchmarkStreaming(providerName string, provider core.Provider) {
	streamer, ok := provider.(core.Streamer)
	if !ok {
		ob.recordBenchmarkResult(BenchmarkResult{
			Provider:      providerName,
			Operation:     "streaming",
			TotalRequests: 1,
			FailedReqs:    1,
			ErrorRate:     100.0,
			Errors:        []string{"streaming not supported"},
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), ob.config.RequestTimeout)
	defer cancel()

	var streamLatencies []time.Duration
	var totalChunks []int
	var errors []string

	for i := 0; i < ob.config.BenchmarkRuns; i++ {
		chunks, latency, err := ob.performStreamingRequest(ctx, streamer)

		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		streamLatencies = append(streamLatencies, latency)
		totalChunks = append(totalChunks, chunks)
	}

	ob.recordBenchmarkResult(BenchmarkResult{
		Provider:       providerName,
		Operation:      "streaming",
		TotalRequests:  ob.config.BenchmarkRuns,
		SuccessfulReqs: len(streamLatencies),
		FailedReqs:     len(errors),
		AvgLatency:     calculateMean(streamLatencies),
		MinLatency:     calculateMin(streamLatencies),
		MaxLatency:     calculateMax(streamLatencies),
		ErrorRate:      float64(len(errors)) / float64(ob.config.BenchmarkRuns) * 100,
		Errors:         errors,
	})
}

func (ob *OperatorBenchmarker) benchmarkErrorResilience(providerName string, provider core.Provider) {
	// Test with intentionally problematic requests
	problemRequests := []struct {
		name        string
		model       string
		maxTokens   int
		temperature float64
		content     string
	}{
		{"empty_content", "gpt-3.5-turbo", 10, 0.5, ""},
		{"very_long_content", "gpt-3.5-turbo", 10, 0.5, strings.Repeat("test ", 1000)},
		{"invalid_temperature", "gpt-3.5-turbo", 10, 5.0, "test"},
		{"zero_tokens", "gpt-3.5-turbo", 0, 0.5, "test"},
		{"negative_tokens", "gpt-3.5-turbo", -10, 0.5, "test"},
	}

	var totalTests int
	var handledGracefully int
	var errors []string

	for _, req := range problemRequests {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := ob.performSingleCompletion(ctx, provider, req.maxTokens, req.temperature, &req.model, &req.content, nil, nil)
		cancel()

		totalTests++
		if err != nil {
			// Error is expected and handled gracefully
			handledGracefully++
			errors = append(errors, fmt.Sprintf("%s: %s", req.name, err.Error()))
		}
	}

	ob.recordBenchmarkResult(BenchmarkResult{
		Provider:       providerName,
		Operation:      "error_resilience",
		TotalRequests:  totalTests,
		SuccessfulReqs: handledGracefully, // Successful error handling
		FailedReqs:     totalTests - handledGracefully,
		ErrorRate:      float64(totalTests-handledGracefully) / float64(totalTests) * 100,
		Errors:         errors,
	})
}

func (ob *OperatorBenchmarker) benchmarkStressTest(providerName string, provider core.Provider) {
	fmt.Printf("\n    🔥 Running stress test for %s...\n", providerName)

	// Gradually increase load until breaking point
	concurrencyLevels := []int{10, 25, 50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		var wg sync.WaitGroup
		var successCount int64
		var errorCount int64

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 5; j++ { // 5 requests per goroutine
					_, err := ob.performSingleCompletion(ctx, provider, 20, 0.5, nil, nil, nil, nil)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		cancel()

		duration := time.Since(start)
		totalRequests := concurrency * 5
		errorRate := float64(errorCount) / float64(totalRequests) * 100

		ob.recordBenchmarkResult(BenchmarkResult{
			Provider:        providerName,
			Operation:       fmt.Sprintf("stress_test_%d", concurrency),
			ConcurrentUsers: concurrency,
			TotalRequests:   totalRequests,
			SuccessfulReqs:  int(successCount),
			FailedReqs:      int(errorCount),
			Duration:        duration,
			Throughput:      float64(successCount) / duration.Seconds(),
			ErrorRate:       errorRate,
		})

		// Stop if error rate is too high
		if errorRate > 50 {
			fmt.Printf("      ⚠️  Breaking point reached at %d concurrent users (%.1f%% error rate)\n", concurrency, errorRate)
			break
		}
	}
}

// Helper function to perform a single completion request
func (ob *OperatorBenchmarker) performSingleCompletion(
	ctx context.Context,
	provider core.Provider,
	maxTokens int,
	temperature float64,
	modelOverride *string,
	contentOverride *string,
	systemMessage *string,
	tools []core.Tool,
) (*core.CompletionResponse, error) {

	model := "gpt-3.5-turbo"
	if modelOverride != nil {
		model = *modelOverride
	}

	content := "Write a brief technical explanation about cloud computing."
	if contentOverride != nil {
		content = *contentOverride
	}

	req := &core.CompletionRequest{
		Model: model,
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: content,
		}},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Tools:       tools,
	}

	if systemMessage != nil {
		req.SystemMessage = systemMessage
	}

	return provider.CreateCompletion(ctx, req)
}

func (ob *OperatorBenchmarker) performStreamingRequest(ctx context.Context, streamer core.Streamer) (int, time.Duration, error) {
	req := &core.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: "Count from 1 to 5 with brief explanations",
		}},
		MaxTokens:   intPtr(100),
		Temperature: float64Ptr(0.3),
		Stream:      true,
	}

	start := time.Now()
	streamChan, err := streamer.StreamCompletion(ctx, req)
	if err != nil {
		return 0, 0, err
	}

	chunks := 0
	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				return chunks, time.Since(start), nil
			}
			if chunk.Done {
				return chunks, time.Since(start), nil
			}
			chunks++
		case <-ctx.Done():
			return chunks, time.Since(start), ctx.Err()
		}
	}
}

func (ob *OperatorBenchmarker) recordBenchmarkResult(result BenchmarkResult) {
	result.StartTime = time.Now()

	// Calculate memory usage if enabled
	if ob.config.EnableMemoryProfile {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.MemoryUsage = int64(m.Alloc / 1024 / 1024) // MB
	}

	// Calculate derived metrics
	if result.TotalTokens > 0 && result.TotalCost > 0 {
		result.CostPerToken = result.TotalCost / float64(result.TotalTokens)
	}

	ob.mu.Lock()
	ob.results = append(ob.results, result)
	ob.mu.Unlock()
}

func (ob *OperatorBenchmarker) generateReport() {
	fmt.Println("📊 COMPREHENSIVE BENCHMARK REPORT")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Overall summary
	ob.printOverallSummary()

	// Performance comparison
	ob.printPerformanceComparison()

	// Detailed metrics by operation
	ob.printDetailedMetrics()

	// Cost analysis
	ob.printCostAnalysis()

	// Recommendations
	ob.printRecommendations()
}

func (ob *OperatorBenchmarker) printOverallSummary() {
	fmt.Println("\n🎯 OVERALL SUMMARY")
	fmt.Println(strings.Repeat("-", 40))

	totalTests := len(ob.results)
	successfulTests := 0
	totalRequests := 0
	successfulRequests := 0

	for _, result := range ob.results {
		totalRequests += result.TotalRequests
		successfulRequests += result.SuccessfulReqs
		if result.ErrorRate < 50 { // Consider test successful if error rate < 50%
			successfulTests++
		}
	}

	overallSuccessRate := float64(successfulTests) / float64(totalTests) * 100
	requestSuccessRate := float64(successfulRequests) / float64(totalRequests) * 100

	fmt.Printf("Total benchmark operations: %d\n", totalTests)
	fmt.Printf("Successful operations: %d (%.1f%%)\n", successfulTests, overallSuccessRate)
	fmt.Printf("Total requests made: %d\n", totalRequests)
	fmt.Printf("Successful requests: %d (%.1f%%)\n", successfulRequests, requestSuccessRate)
}

func (ob *OperatorBenchmarker) printPerformanceComparison() {
	fmt.Println("\n🏆 PERFORMANCE COMPARISON")
	fmt.Println(strings.Repeat("-", 40))

	// Group results by provider
	providerStats := make(map[string]struct {
		totalTests   int
		successTests int
		avgLatency   time.Duration
		totalTokens  int
		totalCost    float64
		maxThroughput float64
	})

	for _, result := range ob.results {
		stats := providerStats[result.Provider]
		stats.totalTests++
		if result.ErrorRate < 50 {
			stats.successTests++
		}
		stats.avgLatency += result.AvgLatency
		stats.totalTokens += result.TotalTokens
		stats.totalCost += result.TotalCost
		if result.Throughput > stats.maxThroughput {
			stats.maxThroughput = result.Throughput
		}
		providerStats[result.Provider] = stats
	}

	fmt.Printf("%-12s %-8s %-10s %-8s %-10s %-8s\n", "PROVIDER", "SUCCESS%", "AVG_LATENCY", "TOKENS", "COST", "MAX_RPS")
	fmt.Println(strings.Repeat("-", 70))

	for provider, stats := range providerStats {
		successRate := float64(stats.successTests) / float64(stats.totalTests) * 100
		avgLatency := stats.avgLatency / time.Duration(stats.totalTests)

		fmt.Printf("%-12s %-8.1f %-10v %-8d $%-9.4f %-8.1f\n",
			provider,
			successRate,
			avgLatency.Round(time.Millisecond),
			stats.totalTokens,
			stats.totalCost,
			stats.maxThroughput)
	}
}

func (ob *OperatorBenchmarker) printDetailedMetrics() {
	fmt.Println("\n📈 DETAILED PERFORMANCE METRICS")
	fmt.Println(strings.Repeat("-", 40))

	// Group by operation type
	operationGroups := make(map[string][]BenchmarkResult)
	for _, result := range ob.results {
		opType := strings.Split(result.Operation, "_")[0]
		operationGroups[opType] = append(operationGroups[opType], result)
	}

	for opType, results := range operationGroups {
		fmt.Printf("\n🔍 %s Results:\n", strings.ToUpper(opType))
		fmt.Printf("%-12s %-8s %-10s %-10s %-10s %-8s\n", "Provider", "Requests", "Success%", "AvgLatency", "P95Latency", "Tokens")
		fmt.Println(strings.Repeat("-", 65))

		for _, result := range results {
			successRate := float64(result.SuccessfulReqs) / float64(result.TotalRequests) * 100
			fmt.Printf("%-12s %-8d %-10.1f %-10v %-10v %-8d\n",
				result.Provider,
				result.TotalRequests,
				successRate,
				result.AvgLatency.Round(time.Millisecond),
				result.P95Latency.Round(time.Millisecond),
				result.TotalTokens)
		}
	}
}

func (ob *OperatorBenchmarker) printCostAnalysis() {
	fmt.Println("\n💰 COST ANALYSIS")
	fmt.Println(strings.Repeat("-", 40))

	providerCosts := make(map[string]struct {
		totalCost    float64
		totalTokens  int
		costPerToken float64
	})

	for _, result := range ob.results {
		stats := providerCosts[result.Provider]
		stats.totalCost += result.TotalCost
		stats.totalTokens += result.TotalTokens
		if stats.totalTokens > 0 {
			stats.costPerToken = stats.totalCost / float64(stats.totalTokens)
		}
		providerCosts[result.Provider] = stats
	}

	fmt.Printf("%-12s %-12s %-12s %-15s\n", "PROVIDER", "TOTAL_COST", "TOKENS", "COST_PER_TOKEN")
	fmt.Println(strings.Repeat("-", 55))

	for provider, stats := range providerCosts {
		fmt.Printf("%-12s $%-11.6f %-12d $%-14.8f\n",
			provider,
			stats.totalCost,
			stats.totalTokens,
			stats.costPerToken)
	}
}

func (ob *OperatorBenchmarker) printRecommendations() {
	fmt.Println("\n💡 RECOMMENDATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Find best performers
	var bestLatency, bestThroughput, bestCost BenchmarkResult
	bestLatencySet := false

	for _, result := range ob.results {
		if result.SuccessfulReqs == 0 {
			continue
		}

		if !bestLatencySet || (result.AvgLatency > 0 && result.AvgLatency < bestLatency.AvgLatency) {
			bestLatency = result
			bestLatencySet = true
		}

		if result.Throughput > bestThroughput.Throughput {
			bestThroughput = result
		}

		if result.CostPerToken > 0 && (bestCost.CostPerToken == 0 || result.CostPerToken < bestCost.CostPerToken) {
			bestCost = result
		}
	}

	if bestLatencySet {
		fmt.Printf("⚡ Best Latency: %s (%v average)\n", bestLatency.Provider, bestLatency.AvgLatency.Round(time.Millisecond))
	}
	if bestThroughput.Throughput > 0 {
		fmt.Printf("🚀 Best Throughput: %s (%.1f req/s)\n", bestThroughput.Provider, bestThroughput.Throughput)
	}
	if bestCost.CostPerToken > 0 {
		fmt.Printf("💰 Most Cost Effective: %s ($%.8f per token)\n", bestCost.Provider, bestCost.CostPerToken)
	}

	fmt.Println("\n📋 General Recommendations:")
	fmt.Println("• Use 'mock' provider for development and testing")
	fmt.Println("• Consider provider-specific strengths for different use cases")
	fmt.Println("• Implement proper error handling and retries")
	fmt.Println("• Monitor costs closely in production")
	fmt.Println("• Set appropriate timeouts based on performance characteristics")
	fmt.Println("• Test under realistic concurrent load before deployment")
}

func (ob *OperatorBenchmarker) exportResults() {
	// This function would export results to JSON
	fmt.Println("\n📁 Export functionality not implemented in this demo")
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func calculateMean(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func calculateMin(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Create a copy and sort
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := percentile * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

func sum(ints []int) int {
	total := 0
	for _, i := range ints {
		total += i
	}
	return total
}

func sumFloat64(floats []float64) float64 {
	total := 0.0
	for _, f := range floats {
		total += f
	}
	return total
}
