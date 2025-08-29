package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
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

type TestResult struct {
	Provider   string
	Test       string
	Success    bool
	Duration   time.Duration
	Error      error
	TokensUsed int
	Cost       float64
	Details    map[string]interface{}
}

type OperatorTester struct {
	providers map[string]core.Provider
	results   []TestResult
	mu        sync.RWMutex
}

// getModelForProvider returns the appropriate model for each provider
func (ot *OperatorTester) getModelForProvider(providerName string) string {
	switch strings.ToLower(providerName) {
	case "deepseek":
		return "deepseek-chat"
	case "openai":
		return "gpt-3.5-turbo"
	case "anthropic":
		return "claude-3-sonnet-20240229"
	case "gemini":
		return "gemini-1.5-pro"
	case "openrouter":
		return "openai/gpt-3.5-turbo"
	case "mock":
		return "mock-gpt-3.5-turbo"
	default:
		return "gpt-3.5-turbo" // fallback
	}
}

func main() {
	fmt.Println("🧪 GOLLM Operator Testing Tool")
	fmt.Println("========================================")

	tester := &OperatorTester{
		providers: make(map[string]core.Provider),
		results:   make([]TestResult, 0),
	}

	// Initialize providers
	if err := tester.initProviders(); err != nil {
		log.Fatalf("❌ Initialization failed: %v", err)
	}

	// Run all tests
	tester.runAllTests()

	// Print comprehensive report
	tester.printReport()
}

func (ot *OperatorTester) initProviders() error {
	fmt.Println("\n🔧 Initializing providers...")

	// Always add mock provider for baseline testing
	mockProvider := mock.New(mock.DefaultConfig())
	ot.providers["mock"] = mockProvider
	fmt.Println("  ✅ Mock provider ready")

	// Load config and add real providers
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  ⚠️  No config found, using mock only: %v\n", err)
		return nil
	}

	for name, providerConfig := range cfg.Providers {
		provider, err := ot.createProvider(name, providerConfig)
		if err != nil {
			fmt.Printf("  ⚠️  Skip %s: %v\n", name, err)
			continue
		}

		ot.providers[name] = provider
		fmt.Printf("  ✅ %s ready\n", name)
	}

	fmt.Printf("\n📋 Total providers: %d\n", len(ot.providers))
	return nil
}

func (ot *OperatorTester) createProvider(name string, cfg config.ProviderConfig) (core.Provider, error) {
	switch cfg.Type {
	case "openai":
		return openai.New(openai.Config{
			APIKey:     cfg.APIKey.Value(),
			BaseURL:    cfg.BaseURL,
			MaxRetries: cfg.MaxRetries,
			Timeout:    cfg.Timeout,
		})
	case "anthropic":
		return anthropic.New(anthropic.Config{
			APIKey:     cfg.APIKey.Value(),
			BaseURL:    cfg.BaseURL,
			MaxRetries: cfg.MaxRetries,
			Timeout:    cfg.Timeout,
		})
	case "gemini":
		return gemini.New(gemini.Config{
			APIKey:     cfg.APIKey.Value(),
			BaseURL:    cfg.BaseURL,
			MaxRetries: cfg.MaxRetries,
			Timeout:    cfg.Timeout,
			Headers:    cfg.CustomHeaders,
		})
	case "deepseek":
		return deepseek.New(deepseek.Config{
			APIKey:     cfg.APIKey.Value(),
			BaseURL:    cfg.BaseURL,
			MaxRetries: cfg.MaxRetries,
			Timeout:    cfg.Timeout,
			Headers:    cfg.CustomHeaders,
		})
	case "openrouter":
		return openrouter.New(openrouter.Config{
			APIKey:     cfg.APIKey.Value(),
			BaseURL:    cfg.BaseURL,
			MaxRetries: cfg.MaxRetries,
			Timeout:    cfg.Timeout,
			Headers:    cfg.CustomHeaders,
		})
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Type)
	}
}

func (ot *OperatorTester) runAllTests() {
	fmt.Println("\n🚀 Running operator tests...")

	tests := []struct {
		name string
		fn   func(string, core.Provider) TestResult
	}{
		{"Basic Completion", ot.testBasicCompletion},
		{"Streaming", ot.testStreaming},
		{"Model Discovery", ot.testModelDiscovery},
		{"Error Handling", ot.testErrorHandling},
		{"Concurrent Load", ot.testConcurrentLoad},
		{"Context Cancel", ot.testContextCancel},
	}

	for _, test := range tests {
		fmt.Printf("\n📋 %s\n", test.name)
		fmt.Println(strings.Repeat("-", 25))

		for name, provider := range ot.providers {
			result := test.fn(name, provider)
			result.Provider = name
			result.Test = test.name

			ot.mu.Lock()
			ot.results = append(ot.results, result)
			ot.mu.Unlock()

			// Print immediate feedback
			status := "❌"
			if result.Success {
				status = "✅"
			}

			details := fmt.Sprintf("%v", result.Duration.Round(time.Millisecond))
			if result.TokensUsed > 0 {
				details += fmt.Sprintf(" (%dt)", result.TokensUsed)
			}

			fmt.Printf("  %s %-12s %s", status, name, details)
			if !result.Success && result.Error != nil {
				fmt.Printf(" - %s", truncateError(result.Error.Error(), 40))
			}
			fmt.Println()
		}
	}
}

func (ot *OperatorTester) testBasicCompletion(provider string, p core.Provider) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	start := time.Now()
	req := &core.CompletionRequest{
		Model: ot.getModelForProvider(provider),
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: fmt.Sprintf("Say 'Hello from %s!' exactly", provider),
		}},
		MaxTokens:   intPtr(20),
		Temperature: float64Ptr(0.1),
	}

	resp, err := p.CreateCompletion(ctx, req)
	duration := time.Since(start)

	result := TestResult{Duration: duration, Success: false, Error: err}

	if err == nil && len(resp.Choices) > 0 {
		result.Success = true
		result.TokensUsed = resp.Usage.TotalTokens
		if resp.Usage.TotalCost != nil {
			result.Cost = *resp.Usage.TotalCost
		}
		result.Details = map[string]interface{}{
			"response": truncateString(resp.Choices[0].Message.Content, 50),
		}
	}

	return result
}

func (ot *OperatorTester) testStreaming(provider string, p core.Provider) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()

	streamer, ok := p.(core.Streamer)
	if !ok {
		return TestResult{
			Duration: time.Since(start),
			Success:  false,
			Error:    fmt.Errorf("streaming not supported"),
		}
	}

	req := &core.CompletionRequest{
		Model: ot.getModelForProvider(provider),
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: "Count from 1 to 3",
		}},
		MaxTokens:   intPtr(30),
		Temperature: float64Ptr(0.1),
		Stream:      true,
	}

	streamChan, err := streamer.StreamCompletion(ctx, req)
	if err != nil {
		return TestResult{Duration: time.Since(start), Success: false, Error: err}
	}

	chunks := 0
	var response strings.Builder

	for chunks < 50 { // Safety limit
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				break
			}

			if chunk.Done {
				break
			}

			chunks++
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
				response.WriteString(chunk.Choices[0].Delta.Content)
			}
		case <-ctx.Done():
			return TestResult{Duration: time.Since(start), Success: false, Error: ctx.Err()}
		}
	}

	return TestResult{
		Duration: time.Since(start),
		Success:  chunks > 0,
		Details: map[string]interface{}{
			"chunks":   chunks,
			"response": truncateString(response.String(), 40),
		},
	}
}

func (ot *OperatorTester) testModelDiscovery(provider string, p core.Provider) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	lister, ok := p.(core.ModelLister)
	if !ok {
		return TestResult{
			Duration: time.Since(start),
			Success:  false,
			Error:    fmt.Errorf("model listing not supported"),
		}
	}

	models, err := lister.GetModels(ctx)
	duration := time.Since(start)

	result := TestResult{Duration: duration, Success: false, Error: err}

	if err == nil {
		result.Success = len(models) > 0
		result.Details = map[string]interface{}{
			"model_count": len(models),
			"sample_models": extractModelNames(models, 3),
		}
	}

	return result
}

func (ot *OperatorTester) testErrorHandling(provider string, p core.Provider) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	// Intentionally invalid request
	invalidReq := &core.CompletionRequest{
		Model:    "", // Invalid
		Messages: []core.Message{}, // Invalid
	}

	_, err := p.CreateCompletion(ctx, invalidReq)
	duration := time.Since(start)

	// We EXPECT this to fail with a validation error
	if err == nil {
		return TestResult{
			Duration: duration,
			Success:  false,
			Error:    fmt.Errorf("expected validation error but got success"),
		}
	}

	// Check if error is appropriate
	errMsg := strings.ToLower(err.Error())
	isValidError := strings.Contains(errMsg, "model") ||
		strings.Contains(errMsg, "message") ||
		strings.Contains(errMsg, "valid") ||
		strings.Contains(errMsg, "required")

	return TestResult{
		Duration: duration,
		Success:  isValidError,
		Details: map[string]interface{}{
			"error_msg": truncateString(err.Error(), 60),
		},
	}
}

func (ot *OperatorTester) testConcurrentLoad(provider string, p core.Provider) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	const workers = 3
	const requestsPerWorker = 2

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successes, errors int

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				req := &core.CompletionRequest{
					Model: ot.getModelForProvider(provider),
					Messages: []core.Message{{
						Role:    core.RoleUser,
						Content: fmt.Sprintf("Worker %d says hello", workerID),
					}},
					MaxTokens:   intPtr(5),
					Temperature: float64Ptr(0.1),
				}

				_, err := p.CreateCompletion(ctx, req)

				mu.Lock()
				if err != nil {
					errors++
				} else {
					successes++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	total := workers * requestsPerWorker
	successRate := float64(successes) / float64(total) * 100

	return TestResult{
		Duration: duration,
		Success:  successRate >= 60, // 60% threshold for concurrent operations
		Details: map[string]interface{}{
			"total_requests": total,
			"successes":      successes,
			"errors":         errors,
			"success_rate":   successRate,
			"rps":           float64(total) / duration.Seconds(),
		},
	}
}

func (ot *OperatorTester) testContextCancel(provider string, p core.Provider) TestResult {
	start := time.Now()

	// Very short timeout to force cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &core.CompletionRequest{
		Model: ot.getModelForProvider(provider),
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: "Write a very long detailed story about space exploration. " + strings.Repeat("Make it extremely detailed and comprehensive. ", 20),
		}},
		MaxTokens:   intPtr(1000),
		Temperature: float64Ptr(0.5),
	}

	_, err := p.CreateCompletion(ctx, req)
	duration := time.Since(start)

	// We expect cancellation/timeout
	if err == nil {
		return TestResult{
			Duration: duration,
			Success:  false,
			Error:    fmt.Errorf("expected timeout but got success"),
		}
	}

	errMsg := strings.ToLower(err.Error())
	isValidCancel := strings.Contains(errMsg, "context") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "cancel") ||
		strings.Contains(errMsg, "deadline")

	return TestResult{
		Duration: duration,
		Success:  isValidCancel,
		Details: map[string]interface{}{
			"cancel_time": duration,
			"error_type":  fmt.Sprintf("%T", err),
		},
	}
}

func (ot *OperatorTester) printReport() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 OPERATOR TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	// Group results by provider
	providerResults := make(map[string][]TestResult)
	for _, result := range ot.results {
		providerResults[result.Provider] = append(providerResults[result.Provider], result)
	}

	// Summary table
	fmt.Printf("\n%-12s %-15s %-6s %-8s %-10s %-20s\n",
		"PROVIDER", "TEST", "STATUS", "TIME", "TOKENS", "NOTES")
	fmt.Println(strings.Repeat("-", 75))

	totalTests := 0
	totalSuccesses := 0

	for provider, results := range providerResults {
		for _, result := range results {
			totalTests++

			status := "FAIL"
			statusIcon := "❌"
			if result.Success {
				status = "PASS"
				statusIcon = "✅"
				totalSuccesses++
			}

			tokens := ""
			if result.TokensUsed > 0 {
				tokens = fmt.Sprintf("%d", result.TokensUsed)
			}

			notes := ""
			if result.Cost > 0 {
				notes = fmt.Sprintf("$%.4f", result.Cost)
			}
			if result.Error != nil && !result.Success {
				if notes != "" {
					notes += " | "
				}
				notes += truncateError(result.Error.Error(), 20)
			}

			fmt.Printf("%-12s %-15s %s %-5s %-8v %-10s %-20s\n",
				provider,
				result.Test,
				statusIcon,
				status,
				result.Duration.Round(time.Millisecond),
				tokens,
				notes)
		}
	}

	// Overall statistics
	fmt.Println(strings.Repeat("-", 75))
	successRate := float64(totalSuccesses) / float64(totalTests) * 100
	fmt.Printf("📈 OVERALL: %d/%d passed (%.1f%% success rate)\n",
		totalSuccesses, totalTests, successRate)

	// Provider ranking
	ot.printProviderRanking(providerResults)

	// Performance insights
	ot.printPerformanceInsights()
}

func (ot *OperatorTester) printProviderRanking(providerResults map[string][]TestResult) {
	fmt.Println("\n🏆 PROVIDER RANKING")
	fmt.Println(strings.Repeat("-", 30))

	type ProviderScore struct {
		Name        string
		SuccessRate float64
		AvgTime     time.Duration
		TotalTokens int
		TotalCost   float64
	}

	var scores []ProviderScore

	for provider, results := range providerResults {
		var successes int
		var totalDuration time.Duration
		var totalTokens int
		var totalCost float64

		for _, result := range results {
			if result.Success {
				successes++
			}
			totalDuration += result.Duration
			totalTokens += result.TokensUsed
			totalCost += result.Cost
		}

		successRate := float64(successes) / float64(len(results)) * 100
		avgTime := totalDuration / time.Duration(len(results))

		scores = append(scores, ProviderScore{
			Name:        provider,
			SuccessRate: successRate,
			AvgTime:     avgTime,
			TotalTokens: totalTokens,
			TotalCost:   totalCost,
		})
	}

	// Sort by success rate, then by speed
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].SuccessRate < scores[j].SuccessRate ||
				(scores[i].SuccessRate == scores[j].SuccessRate && scores[i].AvgTime > scores[j].AvgTime) {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	for i, score := range scores {
		medal := ""
		switch i {
		case 0: medal = "🥇"
		case 1: medal = "🥈"
		case 2: medal = "🥉"
		default: medal = fmt.Sprintf("#%d", i+1)
		}

		fmt.Printf("%s %-10s %6.1f%% %8v %6dt $%.4f\n",
			medal,
			score.Name,
			score.SuccessRate,
			score.AvgTime.Round(time.Millisecond),
			score.TotalTokens,
			score.TotalCost)
	}
}

func (ot *OperatorTester) printPerformanceInsights() {
	fmt.Println("\n⚡ PERFORMANCE INSIGHTS")
	fmt.Println(strings.Repeat("-", 30))

	var fastest, slowest TestResult
	var totalDuration time.Duration
	var successfulTests int
	var fastestInitialized bool

	for _, result := range ot.results {
		if !result.Success {
			continue
		}

		if !fastestInitialized || result.Duration < fastest.Duration {
			fastest = result
			fastestInitialized = true
		}
		if result.Duration > slowest.Duration {
			slowest = result
		}

		totalDuration += result.Duration
		successfulTests++
	}

	if successfulTests > 0 {
		avgTime := totalDuration / time.Duration(successfulTests)
		fmt.Printf("⚡ Fastest: %s/%s (%v)\n", fastest.Provider, fastest.Test, fastest.Duration.Round(time.Millisecond))
		fmt.Printf("🐌 Slowest: %s/%s (%v)\n", slowest.Provider, slowest.Test, slowest.Duration.Round(time.Millisecond))
		fmt.Printf("📊 Average: %v\n", avgTime.Round(time.Millisecond))
	}

	// Calculate total metrics
	var totalTokens int
	var totalCost float64
	for _, result := range ot.results {
		if result.Success {
			totalTokens += result.TokensUsed
			totalCost += result.Cost
		}
	}

	if totalTokens > 0 {
		fmt.Printf("🎯 Total tokens: %d\n", totalTokens)
		fmt.Printf("💰 Total cost: $%.6f\n", totalCost)
		if totalTokens > 0 {
			fmt.Printf("📈 Cost per token: $%.8f\n", totalCost/float64(totalTokens))
		}
	}

	// Error analysis
	ot.printErrorAnalysis()

	fmt.Println("\n💡 Quick recommendations:")
	fmt.Println("  • Use 'mock' provider for development")
	fmt.Println("  • Monitor token usage to control costs")
	fmt.Println("  • Implement timeout and error handling")
	fmt.Println("  • Test concurrent scenarios before production")
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func truncateError(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return truncateString(s, maxLen)
}

func extractModelNames(models []core.Model, limit int) []string {
	var names []string
	for i, model := range models {
		if i >= limit {
			names = append(names, "...")
			break
		}
		names = append(names, model.ID)
	}
	return names
}

// Helper functions for pointer conversion
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

func (ot *OperatorTester) printErrorAnalysis() {
	fmt.Println("\n🔍 ERROR ANALYSIS")
	fmt.Println(strings.Repeat("-", 30))

	// Count error types
	errorTypes := make(map[string]int)
	authErrors := 0
	timeoutErrors := 0
	configErrors := 0
	networkErrors := 0

	for _, result := range ot.results {
		if result.Success || result.Error == nil {
			continue
		}

		errMsg := strings.ToLower(result.Error.Error())

		// Categorize errors
		if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "api key") {
			authErrors++
			errorTypes["Authentication"] = authErrors
		} else if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") || strings.Contains(errMsg, "context") {
			timeoutErrors++
			errorTypes["Timeout/Context"] = timeoutErrors
		} else if strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "invalid character") {
			networkErrors++
			errorTypes["Network/Parsing"] = networkErrors
		} else if strings.Contains(errMsg, "config") || strings.Contains(errMsg, "valid") {
			configErrors++
			errorTypes["Configuration"] = configErrors
		} else {
			errorTypes["Other"]++
		}
	}

	if len(errorTypes) == 0 {
		fmt.Println("🎉 No errors detected!")
		return
	}

	fmt.Println("Common error patterns:")
	for errorType, count := range errorTypes {
		fmt.Printf("  • %s: %d occurrences\n", errorType, count)
	}

	// Specific recommendations based on errors
	if authErrors > 0 {
		fmt.Println("\n🔑 Authentication Issues:")
		fmt.Println("  • Check API keys in config.yaml")
		fmt.Println("  • Verify API key permissions and quotas")
		fmt.Println("  • Ensure keys are active and not expired")
	}

	if timeoutErrors > 0 {
		fmt.Println("\n⏱️ Timeout Issues:")
		fmt.Println("  • Consider increasing timeout values")
		fmt.Println("  • Check network connectivity")
		fmt.Println("  • Some providers may be slower than others")
	}

	if networkErrors > 0 {
		fmt.Println("\n🌐 Network/Parsing Issues:")
		fmt.Println("  • Verify provider base URLs")
		fmt.Println("  • Check for proxy or firewall issues")
		fmt.Println("  • Provider API may have changed format")
	}

	// Provider-specific analysis
	fmt.Println("\n📊 Provider-specific error summary:")
	providerErrors := make(map[string]int)
	for _, result := range ot.results {
		if !result.Success && result.Error != nil {
			providerErrors[result.Provider]++
		}
	}

	for provider, errorCount := range providerErrors {
		fmt.Printf("  • %s: %d failed tests\n", provider, errorCount)
	}
}
