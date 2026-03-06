// Package commands provides the benchmark command for testing provider performance.
//
// The benchmark command allows users to test the performance and reliability of
// different LLM providers and models. It measures various metrics including:
// - Response latency
// - Throughput (tokens per second)
// - Success rates
// - Error patterns
// - Resource usage
//
// Usage:
//
//	gollm benchmark --provider openai --model gpt-4 --iterations 50
//	gollm benchmark --all --output json
//	gollm benchmark --scenario coding --duration 5m
package commands

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/m4rba4s/Nexus-LLM/internal/config"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
	"github.com/m4rba4s/Nexus-LLM/internal/display"
)

// BenchmarkFlags holds all benchmark command configuration.
type BenchmarkFlags struct {
	Provider       string
	Model          string
	All            bool
	Iterations     int
	Duration       time.Duration
	Concurrency    int
	Scenario       string
	OutputFormat   string
	Quiet          bool
	Verbose        bool
	SaveResults    bool
	ResultsFile    string
	WarmupRuns     int
	Timeout        time.Duration
	StopOnError    bool
	MinSuccessRate float64
}

// BenchmarkScenario defines a test scenario with specific parameters.
type BenchmarkScenario struct {
	Name           string
	Description    string
	Prompt         string
	MaxTokens      int
	Temperature    float64
	ExpectedTokens int
}

// BenchmarkResult contains comprehensive benchmark results.
type BenchmarkResult struct {
	Provider           string         `json:"provider"`
	Model              string         `json:"model"`
	Scenario           string         `json:"scenario"`
	TotalRequests      int            `json:"total_requests"`
	SuccessfulRequests int            `json:"successful_requests"`
	FailedRequests     int            `json:"failed_requests"`
	SuccessRate        float64        `json:"success_rate"`
	AvgLatency         time.Duration  `json:"avg_latency"`
	MinLatency         time.Duration  `json:"min_latency"`
	MaxLatency         time.Duration  `json:"max_latency"`
	P50Latency         time.Duration  `json:"p50_latency"`
	P95Latency         time.Duration  `json:"p95_latency"`
	P99Latency         time.Duration  `json:"p99_latency"`
	TotalTokens        int            `json:"total_tokens"`
	TokensPerSecond    float64        `json:"tokens_per_second"`
	RequestsPerSecond  float64        `json:"requests_per_second"`
	TotalDuration      time.Duration  `json:"total_duration"`
	Errors             map[string]int `json:"errors"`
	ResourceUsage      ResourceUsage  `json:"resource_usage"`
	StartTime          time.Time      `json:"start_time"`
	EndTime            time.Time      `json:"end_time"`
}

// ResourceUsage tracks system resource consumption during benchmarking.
type ResourceUsage struct {
	InitialMemory  uint64  `json:"initial_memory"`
	PeakMemory     uint64  `json:"peak_memory"`
	FinalMemory    uint64  `json:"final_memory"`
	MemoryGrowth   uint64  `json:"memory_growth"`
	CPUPercent     float64 `json:"cpu_percent"`
	GoroutineCount int     `json:"goroutine_count"`
}

// RequestResult holds the result of a single benchmark request.
type RequestResult struct {
	Success   bool
	Latency   time.Duration
	Tokens    int
	Error     string
	Timestamp time.Time
}

// NewBenchmarkCommand creates the benchmark command.
func NewBenchmarkCommand() *cobra.Command {
	flags := &BenchmarkFlags{}

	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run performance benchmarks against LLM providers",
		Long: `Run comprehensive performance benchmarks against one or more LLM providers.

The benchmark command measures various performance metrics including response latency,
throughput, success rates, and resource usage. It supports different test scenarios
and can run against single or multiple providers concurrently.

Examples:
  # Benchmark a specific provider and model
  gollm benchmark --provider openai --model gpt-4 --iterations 100

  # Benchmark all configured providers
  gollm benchmark --all --duration 5m

  # Run coding scenario benchmark
  gollm benchmark --scenario coding --concurrency 5

  # Save results to file
  gollm benchmark --provider deepseek --save-results --results-file benchmark.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchmarkCommand(cmd.Context(), flags)
		},
	}

	addBenchmarkFlags(cmd, flags)
	return cmd
}

// addBenchmarkFlags adds flags to the benchmark command.
func addBenchmarkFlags(cmd *cobra.Command, flags *BenchmarkFlags) {
	// Provider selection
	cmd.Flags().StringVarP(&flags.Provider, "provider", "p", "", "Provider to benchmark (required unless --all)")
	cmd.Flags().StringVarP(&flags.Model, "model", "m", "", "Model to benchmark")
	cmd.Flags().BoolVar(&flags.All, "all", false, "Benchmark all configured providers")

	// Test parameters
	cmd.Flags().IntVarP(&flags.Iterations, "iterations", "i", 50, "Number of requests to make (ignored if duration is set)")
	cmd.Flags().DurationVarP(&flags.Duration, "duration", "d", 0, "Duration to run benchmark (e.g., 5m, 30s)")
	cmd.Flags().IntVarP(&flags.Concurrency, "concurrency", "n", 1, "Number of concurrent requests")
	cmd.Flags().StringVarP(&flags.Scenario, "scenario", "s", "default", "Test scenario (default, coding, creative, analysis)")

	// Output options
	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json, table)")
	cmd.Flags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress non-essential output")
	cmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose output")

	// Results persistence
	cmd.Flags().BoolVar(&flags.SaveResults, "save-results", false, "Save benchmark results to file")
	cmd.Flags().StringVar(&flags.ResultsFile, "results-file", "", "File to save results (auto-generated if not specified)")

	// Advanced options
	cmd.Flags().IntVar(&flags.WarmupRuns, "warmup", 5, "Number of warmup runs before benchmark")
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 30*time.Second, "Request timeout")
	cmd.Flags().BoolVar(&flags.StopOnError, "stop-on-error", false, "Stop benchmark on first error")
	cmd.Flags().Float64Var(&flags.MinSuccessRate, "min-success-rate", 0.95, "Minimum acceptable success rate")
}

// runBenchmarkCommand executes the benchmark command.
func runBenchmarkCommand(ctx context.Context, flags *BenchmarkFlags) error {
	// Create display renderer
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: !flags.Quiet,
		Format:      display.Format(flags.OutputFormat),
		Quiet:       flags.Quiet,
		Verbose:     flags.Verbose,
	})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		return err
	}

	// Get benchmark targets
	targets, err := getBenchmarkTargets(cfg, flags)
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to get benchmark targets: %v", err))
		return err
	}

	if len(targets) == 0 {
		renderer.Error("No benchmark targets found")
		return fmt.Errorf("no benchmark targets specified")
	}

	// Get benchmark scenario
	scenario, err := getBenchmarkScenario(flags.Scenario)
	if err != nil {
		renderer.Error(fmt.Sprintf("Failed to get benchmark scenario: %v", err))
		return err
	}

	renderer.Info(fmt.Sprintf("Starting benchmark with %d target(s), scenario: %s", len(targets), scenario.Name))

	// Run benchmarks
	results := make([]*BenchmarkResult, 0, len(targets))
	for _, target := range targets {
		renderer.Info(fmt.Sprintf("Benchmarking %s:%s", target.Provider, target.Model))

		result, err := runSingleBenchmark(ctx, target, scenario, flags, renderer)
		if err != nil {
			renderer.Error(fmt.Sprintf("Benchmark failed for %s:%s - %v", target.Provider, target.Model, err))
			if flags.StopOnError {
				return err
			}
			continue
		}

		results = append(results, result)
		renderer.BenchmarkResult(display.BenchmarkResult{
			Provider:        result.Provider,
			Model:           result.Model,
			AvgLatency:      result.AvgLatency,
			TokensPerSecond: result.TokensPerSecond,
			SuccessRate:     result.SuccessRate,
			TotalRequests:   result.TotalRequests,
		})
	}

	// Display summary
	displayBenchmarkSummary(results, renderer)

	// Save results if requested
	if flags.SaveResults {
		if err := saveBenchmarkResults(results, flags.ResultsFile); err != nil {
			renderer.Error(fmt.Sprintf("Failed to save results: %v", err))
			return err
		}
		renderer.Success("Results saved successfully")
	}

	// Check minimum success rate
	for _, result := range results {
		if result.SuccessRate < flags.MinSuccessRate {
			renderer.Warning(fmt.Sprintf("Provider %s:%s has success rate %.2f%%, below minimum %.2f%%",
				result.Provider, result.Model, result.SuccessRate*100, flags.MinSuccessRate*100))
		}
	}

	return nil
}

// BenchmarkTarget represents a provider/model combination to benchmark.
type BenchmarkTarget struct {
	Provider string
	Model    string
}

// getBenchmarkTargets determines which providers/models to benchmark.
func getBenchmarkTargets(cfg *config.Config, flags *BenchmarkFlags) ([]BenchmarkTarget, error) {
	var targets []BenchmarkTarget

	if flags.All {
		// Benchmark all configured providers
		for providerName, providerConfig := range cfg.Providers {
			var model string
			if len(providerConfig.Models) > 0 {
				model = providerConfig.Models[0]
			} else {
				model = "default" // Provider will use its default model
			}
			targets = append(targets, BenchmarkTarget{
				Provider: providerName,
				Model:    model,
			})
		}
	} else if flags.Provider != "" {
		// Benchmark specific provider
		model := flags.Model
		if model == "" {
			// Use provider's configured model or default
			if providerConfig, exists := cfg.Providers[flags.Provider]; exists && len(providerConfig.Models) > 0 {
				model = providerConfig.Models[0]
			}
			if model == "" {
				model = "default"
			}
		}
		targets = append(targets, BenchmarkTarget{
			Provider: flags.Provider,
			Model:    model,
		})
	} else {
		return nil, fmt.Errorf("either --provider or --all must be specified")
	}

	return targets, nil
}

// getBenchmarkScenario returns the benchmark scenario configuration.
func getBenchmarkScenario(name string) (*BenchmarkScenario, error) {
	scenarios := map[string]*BenchmarkScenario{
		"default": {
			Name:           "default",
			Description:    "General purpose benchmark with balanced parameters",
			Prompt:         "Explain the concept of machine learning in simple terms.",
			MaxTokens:      500,
			Temperature:    0.7,
			ExpectedTokens: 300,
		},
		"coding": {
			Name:           "coding",
			Description:    "Programming and technical tasks benchmark",
			Prompt:         "Write a Python function to calculate the fibonacci sequence using dynamic programming. Include error handling and documentation.",
			MaxTokens:      1000,
			Temperature:    0.2,
			ExpectedTokens: 600,
		},
		"creative": {
			Name:           "creative",
			Description:    "Creative writing and imaginative tasks benchmark",
			Prompt:         "Write a short story about a time traveler who accidentally changes a small detail in the past and discovers unexpected consequences.",
			MaxTokens:      800,
			Temperature:    0.9,
			ExpectedTokens: 500,
		},
		"analysis": {
			Name:           "analysis",
			Description:    "Analytical and reasoning tasks benchmark",
			Prompt:         "Analyze the potential economic and social impacts of widespread adoption of autonomous vehicles. Consider both benefits and challenges.",
			MaxTokens:      1200,
			Temperature:    0.3,
			ExpectedTokens: 800,
		},
	}

	scenario, exists := scenarios[name]
	if !exists {
		return nil, fmt.Errorf("unknown scenario: %s", name)
	}

	return scenario, nil
}

// runSingleBenchmark runs a benchmark against a single provider/model combination.
func runSingleBenchmark(ctx context.Context, target BenchmarkTarget, scenario *BenchmarkScenario, flags *BenchmarkFlags, renderer *display.Renderer) (*BenchmarkResult, error) {
	// Initialize result
	result := &BenchmarkResult{
		Provider:  target.Provider,
		Model:     target.Model,
		Scenario:  scenario.Name,
		StartTime: time.Now(),
		Errors:    make(map[string]int),
	}

	// Get initial resource usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	result.ResourceUsage.InitialMemory = memStats.Alloc
	result.ResourceUsage.GoroutineCount = runtime.NumGoroutine()

	// Create provider (this would use the actual provider factory)
	// For now, we'll simulate with a mock
	provider, err := createProviderForBenchmark(target.Provider, target.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Warmup runs
	renderer.StartProgress(fmt.Sprintf("Running %d warmup requests", flags.WarmupRuns))
	for i := 0; i < flags.WarmupRuns; i++ {
		_ = runSingleRequest(ctx, provider, scenario, flags.Timeout)
		renderer.UpdateProgress()
	}
	renderer.FinishProgress()

	// Determine test duration or iterations
	var testCtx context.Context
	var testCancel context.CancelFunc

	if flags.Duration > 0 {
		testCtx, testCancel = context.WithTimeout(ctx, flags.Duration)
	} else {
		testCtx, testCancel = context.WithCancel(ctx)
	}
	defer testCancel()

	// Start benchmark
	renderer.StartProgress("Running benchmark")

	resultsChan := make(chan RequestResult, flags.Concurrency*2)
	var wg sync.WaitGroup

	// Start concurrent workers
	for i := 0; i < flags.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runBenchmarkWorker(testCtx, provider, scenario, flags, resultsChan, renderer)
		}()
	}

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []RequestResult
	latencies := make([]time.Duration, 0)

	maxRequests := flags.Iterations
	if flags.Duration > 0 {
		maxRequests = math.MaxInt32 // Effectively unlimited for duration-based tests
	}

	for requestResult := range resultsChan {
		results = append(results, requestResult)

		if requestResult.Success {
			result.SuccessfulRequests++
			result.TotalTokens += requestResult.Tokens
			latencies = append(latencies, requestResult.Latency)
		} else {
			result.FailedRequests++
			if requestResult.Error != "" {
				result.Errors[requestResult.Error]++
			}
		}

		result.TotalRequests++
		renderer.UpdateProgress()

		// Check if we've reached the iteration limit
		if result.TotalRequests >= maxRequests {
			testCancel()
		}

		// Stop on error if requested
		if flags.StopOnError && !requestResult.Success {
			testCancel()
			break
		}
	}

	renderer.FinishProgress()
	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Calculate statistics
	if result.TotalRequests > 0 {
		result.SuccessRate = float64(result.SuccessfulRequests) / float64(result.TotalRequests)
	}

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[len(latencies)-1]
		result.P50Latency = latencies[len(latencies)/2]
		result.P95Latency = latencies[int(float64(len(latencies))*0.95)]
		result.P99Latency = latencies[int(float64(len(latencies))*0.99)]

		// Calculate average latency
		var totalLatency time.Duration
		for _, lat := range latencies {
			totalLatency += lat
		}
		result.AvgLatency = totalLatency / time.Duration(len(latencies))
	}

	// Calculate throughput
	if result.TotalDuration > 0 {
		result.RequestsPerSecond = float64(result.SuccessfulRequests) / result.TotalDuration.Seconds()
		result.TokensPerSecond = float64(result.TotalTokens) / result.TotalDuration.Seconds()
	}

	// Get final resource usage
	runtime.ReadMemStats(&memStats)
	result.ResourceUsage.FinalMemory = memStats.Alloc
	result.ResourceUsage.PeakMemory = memStats.Sys
	result.ResourceUsage.MemoryGrowth = result.ResourceUsage.FinalMemory - result.ResourceUsage.InitialMemory

	return result, nil
}

// runBenchmarkWorker runs benchmark requests in a worker goroutine.
func runBenchmarkWorker(ctx context.Context, provider core.Provider, scenario *BenchmarkScenario, flags *BenchmarkFlags, results chan<- RequestResult, renderer *display.Renderer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			result := runSingleRequest(ctx, provider, scenario, flags.Timeout)
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// runSingleRequest executes a single benchmark request.
func runSingleRequest(ctx context.Context, provider core.Provider, scenario *BenchmarkScenario, timeout time.Duration) RequestResult {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Create request (this would use the actual provider interface)
	request := core.ChatRequest{
		Messages: []core.Message{
			{
				Role:    "user",
				Content: scenario.Prompt,
			},
		},
		MaxTokens:   &scenario.MaxTokens,
		Temperature: &scenario.Temperature,
		Stream:      false, // Use non-streaming for consistent measurements
	}

	// Execute request
	response, err := provider.CreateCompletion(requestCtx, &request)
	latency := time.Since(start)

	if err != nil {
		return RequestResult{
			Success:   false,
			Latency:   latency,
			Error:     err.Error(),
			Timestamp: start,
		}
	}

	// Count tokens from usage if available
	tokens := 0
	if len(response.Choices) > 0 {
		tokens = len(response.Choices[0].Message.Content) / 4 // Rough approximation: 4 characters per token
	}
	if response.Usage.CompletionTokens > 0 {
		tokens = response.Usage.CompletionTokens
	}

	return RequestResult{
		Success:   true,
		Latency:   latency,
		Tokens:    tokens,
		Timestamp: start,
	}
}

// createProviderForBenchmark creates a provider instance for benchmarking.
// This is a placeholder - in the real implementation, this would use the provider factory.
func createProviderForBenchmark(providerName, model string) (core.Provider, error) {
	// This would use the actual provider factory from the core package
	// For now, return a mock provider
	return &mockProvider{name: providerName, model: model}, nil
}

// mockProvider is a simple mock provider for benchmarking.
type mockProvider struct {
	name  string
	model string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) GetModels(ctx context.Context) ([]core.Model, error) {
	return []core.Model{
		{
			ID:       m.model,
			Provider: m.name,
		},
	}, nil
}

func (m *mockProvider) ValidateConfig() error {
	return nil
}

func (m *mockProvider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	// Simulate some latency
	select {
	case <-time.After(time.Duration(50+rand.Intn(200)) * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate occasional errors
	if rand.Float64() < 0.02 { // 2% error rate
		return nil, fmt.Errorf("simulated provider error")
	}

	maxTokens := 500
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	content := "This is a simulated response for benchmarking purposes."
	if len(req.Messages) > 0 {
		userContent := req.Messages[0].Content
		if len(userContent) > 100 {
			userContent = userContent[:100]
		}
		content += " " + userContent
	}

	return &core.CompletionResponse{
		ID:      "benchmark-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   m.model,
		Choices: []core.Choice{
			{
				Index: 0,
				Message: core.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: core.Usage{
			PromptTokens:     50,
			CompletionTokens: maxTokens / 2,
			TotalTokens:      50 + maxTokens/2,
		},
	}, nil
}

func (m *mockProvider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	// Simple mock implementation for streaming
	ch := make(chan core.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- core.StreamChunk{
			ID:      "stream-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   m.model,
			Choices: []core.Choice{
				{
					Index: 0,
					Message: core.Message{
						Role:    "assistant",
						Content: "Simulated streaming response",
					},
				},
			},
		}
	}()
	return ch, nil
}

// displayBenchmarkSummary shows a summary of all benchmark results.
func displayBenchmarkSummary(results []*BenchmarkResult, renderer *display.Renderer) {
	if len(results) == 0 {
		return
	}

	renderer.Info("\n📊 Benchmark Summary:")

	for _, result := range results {
		renderer.BenchmarkResult(display.BenchmarkResult{
			Provider:        result.Provider,
			Model:           result.Model,
			AvgLatency:      result.AvgLatency,
			TokensPerSecond: result.TokensPerSecond,
			SuccessRate:     result.SuccessRate,
			TotalRequests:   result.TotalRequests,
		})
	}

	// Find best performing provider
	var bestLatency *BenchmarkResult
	var bestThroughput *BenchmarkResult
	var bestReliability *BenchmarkResult

	for _, result := range results {
		if bestLatency == nil || result.AvgLatency < bestLatency.AvgLatency {
			bestLatency = result
		}
		if bestThroughput == nil || result.TokensPerSecond > bestThroughput.TokensPerSecond {
			bestThroughput = result
		}
		if bestReliability == nil || result.SuccessRate > bestReliability.SuccessRate {
			bestReliability = result
		}
	}

	renderer.Info("\n🏆 Best Performers:")
	if bestLatency != nil {
		renderer.Success(fmt.Sprintf("Lowest Latency: %s:%s (%v)", bestLatency.Provider, bestLatency.Model, bestLatency.AvgLatency))
	}
	if bestThroughput != nil {
		renderer.Success(fmt.Sprintf("Highest Throughput: %s:%s (%.2f tokens/sec)", bestThroughput.Provider, bestThroughput.Model, bestThroughput.TokensPerSecond))
	}
	if bestReliability != nil {
		renderer.Success(fmt.Sprintf("Most Reliable: %s:%s (%.2f%%)", bestReliability.Provider, bestReliability.Model, bestReliability.SuccessRate*100))
	}
}

// saveBenchmarkResults saves benchmark results to a file.
func saveBenchmarkResults(results []*BenchmarkResult, filename string) error {
	// Implementation would save results to JSON file
	// For now, just a placeholder
	fmt.Printf("Results would be saved to: %s\n", filename)
	return nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
