package main

import (
	"context"
	"fmt"
	"strings"
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

// TestStatus represents the result of a test
type TestStatus int

const (
	TestPass TestStatus = iota
	TestFail
	TestSkip
)

func (ts TestStatus) String() string {
	switch ts {
	case TestPass:
		return "✅ PASS"
	case TestFail:
		return "❌ FAIL"
	case TestSkip:
		return "⏭️  SKIP"
	default:
		return "❓ UNKNOWN"
	}
}

// QuickTest represents a single test case
type QuickTest struct {
	Name        string
	Provider    string
	Status      TestStatus
	Duration    time.Duration
	Message     string
	TokensUsed  int
}

// QuickTester runs fast integration tests
type QuickTester struct {
	providers map[string]core.Provider
	results   []QuickTest
}

// getModelForProvider returns the appropriate model for each provider
func (qt *QuickTester) getModelForProvider(providerName string) string {
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
	fmt.Println("⚡ GOLLM Quick Operator Integration Test")
	fmt.Println("==========================================")

	tester := &QuickTester{
		providers: make(map[string]core.Provider),
		results:   make([]QuickTest, 0),
	}

	// Initialize providers
	if err := tester.initProviders(); err != nil {
		fmt.Printf("❌ Initialization failed: %v\n", err)
		return
	}

	// Run quick tests
	tester.runQuickTests()

	// Print summary
	tester.printSummary()
}

func (qt *QuickTester) initProviders() error {
	fmt.Println("\n🔧 Initializing providers...")

	// Always include mock provider
	mockProvider := mock.New(mock.DefaultConfig())
	qt.providers["mock"] = mockProvider
	fmt.Println("  ✅ mock")

	// Try to load real providers
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  ⚠️  Config not found, using mock only\n")
		return nil
	}

	providerFactories := map[string]func(config.ProviderConfig) (core.Provider, error){
		"openai": func(c config.ProviderConfig) (core.Provider, error) {
			return openai.New(openai.Config{
				APIKey:     c.APIKey.Value(),
				BaseURL:    c.BaseURL,
				MaxRetries: c.MaxRetries,
				Timeout:    c.Timeout,
			})
		},
		"anthropic": func(c config.ProviderConfig) (core.Provider, error) {
			return anthropic.New(anthropic.Config{
				APIKey:     c.APIKey.Value(),
				BaseURL:    c.BaseURL,
				MaxRetries: c.MaxRetries,
				Timeout:    c.Timeout,
			})
		},
		"gemini": func(c config.ProviderConfig) (core.Provider, error) {
			return gemini.New(gemini.Config{
				APIKey:     c.APIKey.Value(),
				BaseURL:    c.BaseURL,
				MaxRetries: c.MaxRetries,
				Timeout:    c.Timeout,
				Headers:    c.CustomHeaders,
			})
		},
		"deepseek": func(c config.ProviderConfig) (core.Provider, error) {
			return deepseek.New(deepseek.Config{
				APIKey:     c.APIKey.Value(),
				BaseURL:    c.BaseURL,
				MaxRetries: c.MaxRetries,
				Timeout:    c.Timeout,
				Headers:    c.CustomHeaders,
			})
		},
		"openrouter": func(c config.ProviderConfig) (core.Provider, error) {
			return openrouter.New(openrouter.Config{
				APIKey:     c.APIKey.Value(),
				BaseURL:    c.BaseURL,
				MaxRetries: c.MaxRetries,
				Timeout:    c.Timeout,
				Headers:    c.CustomHeaders,
			})
		},
	}

	for name, providerConfig := range cfg.Providers {
		factory, exists := providerFactories[providerConfig.Type]
		if !exists {
			fmt.Printf("  ⚠️  %s (unsupported type: %s)\n", name, providerConfig.Type)
			continue
		}

		provider, err := factory(providerConfig)
		if err != nil {
			fmt.Printf("  ⚠️  %s (init failed)\n", name)
			continue
		}

		qt.providers[name] = provider
		fmt.Printf("  ✅ %s\n", name)
	}

	fmt.Printf("\n📊 Total providers: %d\n", len(qt.providers))
	return nil
}

func (qt *QuickTester) runQuickTests() {
	fmt.Println("\n🧪 Running integration tests...\n")

	tests := []struct {
		name string
		fn   func(string, core.Provider) QuickTest
	}{
		{"Completion", qt.testCompletion},
		{"Validation", qt.testValidation},
		{"Models", qt.testModels},
		{"Streaming", qt.testStreaming},
		{"Concurrency", qt.testConcurrency},
	}

	for _, test := range tests {
		fmt.Printf("📋 %s\n", test.name)
		fmt.Println(strings.Repeat("-", 20))

		for name, provider := range qt.providers {
			result := test.fn(name, provider)
			result.Name = test.name
			result.Provider = name
			qt.results = append(qt.results, result)

			fmt.Printf("  %-12s %s", name, result.Status)
			if result.Duration > 0 {
				fmt.Printf(" (%v)", result.Duration.Round(time.Millisecond))
			}
			if result.TokensUsed > 0 {
				fmt.Printf(" [%dt]", result.TokensUsed)
			}
			if result.Status != TestPass && result.Message != "" {
				fmt.Printf(" - %s", truncate(result.Message, 40))
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func (qt *QuickTester) testCompletion(provider string, p core.Provider) QuickTest {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	req := &core.CompletionRequest{
		Model: qt.getModelForProvider(provider),
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: "Say 'Hello' in one word only",
		}},
		MaxTokens:   intPtr(5),
		Temperature: float64Ptr(0.1),
	}

	resp, err := p.CreateCompletion(ctx, req)
	duration := time.Since(start)

	if err != nil {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  err.Error(),
		}
	}

	if len(resp.Choices) == 0 {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  "no response choices",
		}
	}

	return QuickTest{
		Status:     TestPass,
		Duration:   duration,
		TokensUsed: resp.Usage.TotalTokens,
		Message:    fmt.Sprintf("Response: %s", truncate(resp.Choices[0].Message.Content, 20)),
	}
}

func (qt *QuickTester) testValidation(provider string, p core.Provider) QuickTest {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	// Test with invalid request (should fail gracefully)
	invalidReq := &core.CompletionRequest{
		Model:    "", // Invalid
		Messages: []core.Message{}, // Invalid
	}

	_, err := p.CreateCompletion(ctx, invalidReq)
	duration := time.Since(start)

	if err == nil {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  "should have rejected invalid request",
		}
	}

	// Check if error message is reasonable
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "model") || strings.Contains(errMsg, "message") ||
	   strings.Contains(errMsg, "valid") || strings.Contains(errMsg, "required") {
		return QuickTest{
			Status:   TestPass,
			Duration: duration,
			Message:  "validation works correctly",
		}
	}

	return QuickTest{
		Status:   TestFail,
		Duration: duration,
		Message:  "unexpected error type",
	}
}

func (qt *QuickTester) testModels(provider string, p core.Provider) QuickTest {
	lister, ok := p.(core.ModelLister)
	if !ok {
		return QuickTest{
			Status:  TestSkip,
			Message: "model listing not supported",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	models, err := lister.GetModels(ctx)
	duration := time.Since(start)

	if err != nil {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  err.Error(),
		}
	}

	return QuickTest{
		Status:   TestPass,
		Duration: duration,
		Message:  fmt.Sprintf("%d models available", len(models)),
	}
}

func (qt *QuickTester) testStreaming(provider string, p core.Provider) QuickTest {
	streamer, ok := p.(core.Streamer)
	if !ok {
		return QuickTest{
			Status:  TestSkip,
			Message: "streaming not supported",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	start := time.Now()
	req := &core.CompletionRequest{
		Model: qt.getModelForProvider(provider),
		Messages: []core.Message{{
			Role:    core.RoleUser,
			Content: "Count: 1, 2, 3",
		}},
		MaxTokens:   intPtr(20),
		Temperature: float64Ptr(0.1),
		Stream:      true,
	}

	streamChan, err := streamer.StreamCompletion(ctx, req)
	if err != nil {
		return QuickTest{
			Status:   TestFail,
			Duration: time.Since(start),
			Message:  err.Error(),
		}
	}

	chunks := 0
	for chunks < 10 { // Safety limit
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				break
			}
			if chunk.Done {
				break
			}
			chunks++
		case <-ctx.Done():
			return QuickTest{
				Status:   TestFail,
				Duration: time.Since(start),
				Message:  "streaming timeout",
			}
		}
	}

	duration := time.Since(start)

	if chunks == 0 {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  "no stream chunks received",
		}
	}

	return QuickTest{
		Status:   TestPass,
		Duration: duration,
		Message:  fmt.Sprintf("%d chunks received", chunks),
	}
}

func (qt *QuickTester) testConcurrency(provider string, p core.Provider) QuickTest {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()

	// Simple concurrency test with 3 goroutines
	results := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			req := &core.CompletionRequest{
				Model: qt.getModelForProvider(provider),
				Messages: []core.Message{{
					Role:    core.RoleUser,
					Content: fmt.Sprintf("Worker %d says hi", id),
				}},
				MaxTokens:   intPtr(5),
				Temperature: float64Ptr(0.1),
			}

			_, err := p.CreateCompletion(ctx, req)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	var lastError error
	for i := 0; i < 3; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			lastError = err
		}
	}

	duration := time.Since(start)

	if successCount == 0 {
		return QuickTest{
			Status:   TestFail,
			Duration: duration,
			Message:  lastError.Error(),
		}
	}

	if successCount == 3 {
		return QuickTest{
			Status:   TestPass,
			Duration: duration,
			Message:  "all concurrent requests succeeded",
		}
	}

	return QuickTest{
		Status:   TestPass,
		Duration: duration,
		Message:  fmt.Sprintf("%d/3 requests succeeded", successCount),
	}
}

func (qt *QuickTester) printSummary() {
	fmt.Println("📊 QUICK TEST SUMMARY")
	fmt.Println("=====================")

	// Count results by status
	statusCounts := make(map[TestStatus]int)
	for _, result := range qt.results {
		statusCounts[result.Status]++
	}

	total := len(qt.results)
	fmt.Printf("\nTotal tests: %d\n", total)
	fmt.Printf("✅ Passed: %d\n", statusCounts[TestPass])
	fmt.Printf("❌ Failed: %d\n", statusCounts[TestFail])
	fmt.Printf("⏭️  Skipped: %d\n", statusCounts[TestSkip])

	if total > 0 {
		successRate := float64(statusCounts[TestPass]) / float64(total) * 100
		fmt.Printf("\nSuccess Rate: %.1f%%\n", successRate)
	}

	// Provider summary
	fmt.Println("\n🏆 PROVIDER SUMMARY")
	fmt.Println("-------------------")

	providerStats := make(map[string]struct {
		pass, fail, skip int
	})

	for _, result := range qt.results {
		stats := providerStats[result.Provider]
		switch result.Status {
		case TestPass:
			stats.pass++
		case TestFail:
			stats.fail++
		case TestSkip:
			stats.skip++
		}
		providerStats[result.Provider] = stats
	}

	fmt.Printf("%-12s %-6s %-6s %-6s %-8s\n", "PROVIDER", "PASS", "FAIL", "SKIP", "SUCCESS%")
	fmt.Println(strings.Repeat("-", 45))

	for provider, stats := range providerStats {
		total := stats.pass + stats.fail + stats.skip
		var successRate float64
		if total > 0 {
			successRate = float64(stats.pass) / float64(stats.pass+stats.fail) * 100
		}

		fmt.Printf("%-12s %-6d %-6d %-6d %-8.1f\n",
			provider, stats.pass, stats.fail, stats.skip, successRate)
	}

	// Quick recommendations
	fmt.Println("\n💡 QUICK RECOMMENDATIONS")
	fmt.Println("------------------------")

	if statusCounts[TestFail] == 0 {
		fmt.Println("🎉 All tests passed! Your GOLLM setup is working correctly.")
	} else {
		fmt.Println("• Check failed tests above for specific issues")
		fmt.Println("• Verify API keys are valid and have sufficient quota")
		fmt.Println("• Ensure network connectivity to provider endpoints")
		fmt.Println("• Consider using mock provider for development")
	}

	if statusCounts[TestSkip] > 0 {
		fmt.Println("• Some features not supported by all providers (normal)")
	}

	fmt.Println("• Run full benchmark suite for detailed performance analysis")
}

// Helper functions
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
