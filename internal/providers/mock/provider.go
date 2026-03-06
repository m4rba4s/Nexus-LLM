// Package mock provides a mock LLM provider implementation for testing and development.
//
// The mock provider allows configuring various response scenarios including
// successful completions, error conditions, latency simulation, and streaming behavior.
// This is essential for testing the GOLLM system without requiring actual API calls
// to external LLM providers.
//
// Example usage:
//
//	config := mock.Config{
//		Name:         "test-mock",
//		DefaultModel: "mock-gpt-3.5",
//		Latency:      100 * time.Millisecond,
//	}
//
//	provider := mock.New(config)
//
//	// Configure custom response
//	provider.SetResponse("Hello, world!")
//
//	resp, err := provider.CreateCompletion(ctx, &core.CompletionRequest{
//		Model:    "mock-gpt-3.5",
//		Messages: []core.Message{{Role: "user", Content: "Hello"}},
//	})
package mock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

// Provider implements a mock LLM provider for testing and development.
type Provider struct {
	mu      sync.RWMutex
	config  Config
	models  []core.Model
	metrics *core.ProviderMetrics

	// Configurable responses and behaviors
	responses    map[string]string        // Model -> response mapping
	errors       map[string]error         // Model -> error mapping
	latencies    map[string]time.Duration // Model -> latency mapping
	streamChunks map[string][]string      // Model -> stream chunks mapping
	requestCount map[string]int           // Model -> request count

	// Global behaviors
	globalResponse string
	globalError    error
	globalLatency  time.Duration
	failAfterCount int // Fail after N requests
	rateLimitAfter int // Rate limit after N requests
	shouldStream   bool
	chunkDelay     time.Duration
}

// Config configures the mock provider behavior.
type Config struct {
	Name             string        `json:"name"`
	DefaultModel     string        `json:"default_model"`
	SupportedModels  []string      `json:"supported_models"`
	Latency          time.Duration `json:"latency"`
	ErrorRate        float64       `json:"error_rate"` // 0.0-1.0
	EnableStreaming  bool          `json:"enable_streaming"`
	StreamChunkDelay time.Duration `json:"stream_chunk_delay"`
	MaxTokens        int           `json:"max_tokens"`
	EnableFunctions  bool          `json:"enable_functions"`
	EnableVision     bool          `json:"enable_vision"`

	// Cost simulation
	InputCostPer1K  float64 `json:"input_cost_per_1k"`
	OutputCostPer1K float64 `json:"output_cost_per_1k"`
}

// DefaultConfig returns sensible defaults for the mock provider.
func DefaultConfig() Config {
	return Config{
		Name:             "mock",
		DefaultModel:     "mock-gpt-3.5-turbo",
		SupportedModels:  []string{"mock-gpt-3.5-turbo", "mock-gpt-4", "mock-claude-3"},
		Latency:          100 * time.Millisecond,
		ErrorRate:        0.0,
		EnableStreaming:  true,
		StreamChunkDelay: 50 * time.Millisecond,
		MaxTokens:        4096,
		EnableFunctions:  true,
		EnableVision:     false,
		InputCostPer1K:   0.001,
		OutputCostPer1K:  0.002,
	}
}

// New creates a new mock provider with the given configuration.
func New(config Config) *Provider {
	// Apply defaults for completely empty config
	if config.Name == "" && len(config.SupportedModels) == 0 {
		config = DefaultConfig()
	} else {
		// Apply defaults selectively for missing non-critical fields
		defaults := DefaultConfig()

		// Only set default model if we have supported models but no default
		if config.DefaultModel == "" && len(config.SupportedModels) > 0 {
			// Use first supported model as default
			config.DefaultModel = config.SupportedModels[0]
		}
		if config.Latency == 0 {
			config.Latency = defaults.Latency
		}
		if config.StreamChunkDelay == 0 {
			config.StreamChunkDelay = defaults.StreamChunkDelay
		}
		if config.MaxTokens == 0 {
			config.MaxTokens = defaults.MaxTokens
		}
		if config.InputCostPer1K == 0 {
			config.InputCostPer1K = defaults.InputCostPer1K
		}
		if config.OutputCostPer1K == 0 {
			config.OutputCostPer1K = defaults.OutputCostPer1K
		}
	}

	// Create supported models
	models := make([]core.Model, 0, len(config.SupportedModels))
	for _, modelID := range config.SupportedModels {
		models = append(models, core.Model{
			ID:                modelID,
			Object:            "model",
			Created:           time.Now().Unix(),
			OwnedBy:           "mock-org",
			Provider:          config.Name,
			MaxTokens:         &config.MaxTokens,
			SupportsFunctions: config.EnableFunctions,
			SupportsStreaming: config.EnableStreaming,
			SupportsVision:    config.EnableVision,
			InputCostPer1K:    &config.InputCostPer1K,
			OutputCostPer1K:   &config.OutputCostPer1K,
			Description:       fmt.Sprintf("Mock model %s for testing", modelID),
			Tags:              []string{"mock", "testing"},
		})
	}

	return &Provider{
		config:       config,
		models:       models,
		metrics:      core.NewProviderMetrics(),
		responses:    make(map[string]string),
		errors:       make(map[string]error),
		latencies:    make(map[string]time.Duration),
		streamChunks: make(map[string][]string),
		requestCount: make(map[string]int),
		shouldStream: config.EnableStreaming,
		chunkDelay:   config.StreamChunkDelay,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return p.config.Name
}

// CreateCompletion creates a chat completion synchronously.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	start := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	p.mu.Lock()
	p.requestCount[req.Model]++
	count := p.requestCount[req.Model]
	p.mu.Unlock()

	// Check for configured failures
	if err := p.checkForConfiguredFailure(req.Model, count); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, err
	}

	// Simulate latency
	latency := p.getLatency(req.Model)
	if latency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(latency):
			// Continue
		}
	}

	// Generate response
	response := p.generateResponse(req)

	// Calculate usage and cost
	usage := p.calculateUsage(req, response)
	cost := p.calculateCost(&usage)

	// Set usage in response
	response.Usage = usage

	p.metrics.RecordRequest(time.Since(start), &usage, &cost, nil)

	return response, nil
}

// StreamCompletion creates a streaming chat completion.
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if !p.config.EnableStreaming {
		return nil, errors.New("streaming not supported by this mock provider")
	}

	p.mu.Lock()
	p.requestCount[req.Model]++
	count := p.requestCount[req.Model]
	p.mu.Unlock()

	// Check for configured failures
	if err := p.checkForConfiguredFailure(req.Model, count); err != nil {
		return nil, err
	}

	// Create response channel
	chunks := make(chan core.StreamChunk, 10)

	go func() {
		defer close(chunks)

		start := time.Now()
		responseText := p.getResponse(req.Model, req)
		streamChunks := p.getStreamChunks(req.Model, responseText)

		// Send chunks with delay
		for i, chunk := range streamChunks {
			select {
			case <-ctx.Done():
				chunks <- core.StreamChunk{
					Error: ctx.Err(),
				}
				return
			case <-time.After(p.chunkDelay):
				isLast := i == len(streamChunks)-1

				streamChunk := core.StreamChunk{
					ID:      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []core.Choice{
						{
							Index: 0,
							Delta: &core.Message{
								Role:    core.RoleAssistant,
								Content: chunk,
							},
						},
					},
					Done:      isLast,
					Provider:  p.config.Name,
					RequestID: req.RequestID,
					Timestamp: time.Now(),
				}

				if isLast {
					// Add usage information to final chunk
					usage := p.calculateUsage(req, &core.CompletionResponse{
						Choices: []core.Choice{{Message: core.Message{Content: responseText}}},
					})
					streamChunk.Usage = &usage
					streamChunk.Choices[0].FinishReason = core.FinishReasonStop
				}

				chunks <- streamChunk
			}
		}

		// Record metrics for streaming
		totalResponseTime := time.Since(start)
		usage := p.calculateUsage(req, &core.CompletionResponse{
			Choices: []core.Choice{{Message: core.Message{Content: responseText}}},
		})
		cost := p.calculateCost(&usage)
		p.metrics.RecordRequest(totalResponseTime, &usage, &cost, nil)
	}()

	return chunks, nil
}

// GetModels returns the list of available models.
func (p *Provider) GetModels(ctx context.Context) ([]core.Model, error) {
	// Simulate some latency
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
		// Continue
	}

	p.mu.RLock()
	models := make([]core.Model, len(p.models))
	copy(models, p.models)
	p.mu.RUnlock()

	return models, nil
}

// ValidateConfig validates the provider configuration.
func (p *Provider) ValidateConfig() error {
	if p.config.Name == "" {
		return &core.ValidationError{
			Field:   "name",
			Value:   p.config.Name,
			Rule:    "required",
			Message: "provider name cannot be empty",
		}
	}

	if len(p.config.SupportedModels) == 0 {
		return &core.ValidationError{
			Field:   "supported_models",
			Value:   len(p.config.SupportedModels),
			Rule:    "required",
			Message: "at least one model must be supported",
		}
	}

	if p.config.ErrorRate < 0 || p.config.ErrorRate > 1 {
		return &core.ValidationError{
			Field:   "error_rate",
			Value:   p.config.ErrorRate,
			Rule:    "range",
			Message: "error rate must be between 0.0 and 1.0",
		}
	}

	return nil
}

// Configuration methods for testing

// SetResponse sets a custom response for a specific model.
func (p *Provider) SetResponse(model, response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses[model] = response
}

// SetGlobalResponse sets a global response for all models.
func (p *Provider) SetGlobalResponse(response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.globalResponse = response
}

// SetError sets a custom error for a specific model.
func (p *Provider) SetError(model string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors[model] = err
}

// SetGlobalError sets a global error for all models.
func (p *Provider) SetGlobalError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.globalError = err
}

// SetLatency sets custom latency for a specific model.
func (p *Provider) SetLatency(model string, latency time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.latencies[model] = latency
}

// SetGlobalLatency sets global latency for all models.
func (p *Provider) SetGlobalLatency(latency time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.globalLatency = latency
}

// SetStreamChunks sets custom stream chunks for a specific model.
func (p *Provider) SetStreamChunks(model string, chunks []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streamChunks[model] = chunks
}

// SetFailAfterCount configures the provider to fail after N requests.
func (p *Provider) SetFailAfterCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failAfterCount = count
}

// SetRateLimitAfter configures the provider to rate limit after N requests.
func (p *Provider) SetRateLimitAfter(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimitAfter = count
}

// ResetCounters resets all request counters.
func (p *Provider) ResetCounters() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requestCount = make(map[string]int)
}

// GetMetrics returns the provider metrics.
func (p *Provider) GetMetrics() *core.ProviderMetrics {
	return p.metrics.Clone()
}

// Private helper methods

// checkForConfiguredFailure checks if a failure should be triggered.
func (p *Provider) checkForConfiguredFailure(model string, count int) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check global error
	if p.globalError != nil {
		return p.globalError
	}

	// Check model-specific error
	if err, exists := p.errors[model]; exists {
		return err
	}

	// Check fail after count
	if p.failAfterCount > 0 && count > p.failAfterCount {
		return &core.APIError{
			StatusCode: 500,
			Code:       "internal_error",
			Message:    "Mock failure after request count",
			Provider:   p.config.Name,
		}
	}

	// Check rate limiting
	if p.rateLimitAfter > 0 && count > p.rateLimitAfter {
		return &core.APIError{
			StatusCode: 429,
			Code:       "rate_limit_exceeded",
			Message:    "Mock rate limit exceeded",
			Provider:   p.config.Name,
		}
	}

	// Random error based on error rate
	if p.config.ErrorRate > 0 && rand.Float64() < p.config.ErrorRate {
		return &core.APIError{
			StatusCode: 500,
			Code:       "random_error",
			Message:    "Random error for testing",
			Provider:   p.config.Name,
		}
	}

	return nil
}

// getLatency returns the configured latency for a model.
func (p *Provider) getLatency(model string) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if latency, exists := p.latencies[model]; exists {
		return latency
	}

	if p.globalLatency > 0 {
		return p.globalLatency
	}

	return p.config.Latency
}

// generateResponse generates a mock response.
func (p *Provider) generateResponse(req *core.CompletionRequest) *core.CompletionResponse {
	responseText := p.getResponse(req.Model, req)

	return &core.CompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []core.Choice{
			{
				Index: 0,
				Message: core.Message{
					Role:    core.RoleAssistant,
					Content: responseText,
				},
				FinishReason: core.FinishReasonStop,
			},
		},
		Provider:  p.config.Name,
		RequestID: req.RequestID,
	}
}

// getResponse returns the configured response for a model.
func (p *Provider) getResponse(model string, req *core.CompletionRequest) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Extract last user message
	last := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == core.RoleUser {
			last = req.Messages[i].Content
			break
		}
	}
	if last == "" {
		return "Hello! I'm a mock assistant. How can I help you?"
	}

    // Helpers
    // containsCode detects if any line in the prompt looks like a code start.
    containsCode := func(s string) bool {
        // Quick lowercased scan across lines to tolerate prompt prefixes
        for _, line := range strings.Split(s, "\n") {
            l := strings.TrimSpace(strings.ToLower(line))
            if strings.HasPrefix(l, "def ") || strings.HasPrefix(l, "func ") || strings.HasPrefix(l, "class ") {
                return true
            }
        }
        return false
    }

    // 1) Code prompt → prefer model-specific response if present
    if containsCode(last) {
        if resp, ok := p.responses[model]; ok {
            return resp
        }
        // Default illustrative code completion when none is configured
        return "def fibonacci(n):\n    if n <= 1:\n        return n\n\nfunc factorial(n int) int {\n    if n <= 1 {\n        return 1\n    }\n}\n\nclass Calculator:\n    def __init__(self):\n        self.total = 0\n"
    }

	// 2) Heuristic canned replies for common prompts
	lower := strings.ToLower(last)
	if strings.Contains(lower, "hello, world") || strings.Contains(lower, "привет") {
		return "Hello there!"
	}
	if strings.Contains(lower, "what is go") {
		return "Go is a programming language developed by Google"
	}
	if strings.Contains(lower, "tell me a story") {
		return "Once upon a time there was a blazing fast Go CLI named GOLLM..."
	}

	// 3) Prefer model-specific mapping over global for general prompts
	if resp, ok := p.responses[model]; ok {
		return resp
	}

	// 4) Global response fallback
	if p.globalResponse != "" {
		return p.globalResponse
	}

	// 5) Contextual default
	return fmt.Sprintf("Mock response to: %s", last)
}

// getStreamChunks returns stream chunks for a model.
func (p *Provider) getStreamChunks(model, responseText string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check for configured chunks
	if chunks, exists := p.streamChunks[model]; exists {
		return chunks
	}

	// Split response into words for streaming
	words := strings.Fields(responseText)
	if len(words) == 0 {
		return []string{responseText}
	}

	// Group words into chunks
	var chunks []string
	chunkSize := 3 // Words per chunk

	for i := 0; i < len(words); i += chunkSize {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}

		chunk := strings.Join(words[i:end], " ")
		if i+chunkSize < len(words) {
			chunk += " "
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}

// calculateUsage calculates token usage for billing simulation.
func (p *Provider) calculateUsage(req *core.CompletionRequest, resp *core.CompletionResponse) core.Usage {
	// Simple word-based token estimation (1 token ≈ 0.75 words)
	promptText := ""
	for _, msg := range req.Messages {
		promptText += msg.Content + " "
	}

	completionText := ""
	if len(resp.Choices) > 0 {
		completionText = resp.Choices[0].Message.Content
	}

	// Ensure we have at least some tokens for realistic simulation
	promptWords := strings.Fields(promptText)
	completionWords := strings.Fields(completionText)

	promptTokens := int(float64(len(promptWords)) / 0.75)
	if promptTokens == 0 && len(promptText) > 0 {
		promptTokens = 1 // Minimum 1 token for non-empty content
	}

	completionTokens := int(float64(len(completionWords)) / 0.75)
	if completionTokens == 0 && len(completionText) > 0 {
		completionTokens = 1 // Minimum 1 token for non-empty content
	}

	return core.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// calculateCost calculates the cost based on usage and pricing.
func (p *Provider) calculateCost(usage *core.Usage) float64 {
	inputCost := float64(usage.PromptTokens) / 1000.0 * p.config.InputCostPer1K
	outputCost := float64(usage.CompletionTokens) / 1000.0 * p.config.OutputCostPer1K
	return inputCost + outputCost
}

// NewFromConfig creates a new mock provider from core configuration.
// This is used by the provider registry for dynamic provider creation.
func NewFromConfig(config core.ProviderConfig) (core.Provider, error) {
	// Convert core config to mock-specific config
	mockConfig := Config{
		Name:            "mock",
		DefaultModel:    "mock-gpt-3.5-turbo",
		EnableStreaming: true,
		SupportedModels: []string{
			"mock-gpt-3.5-turbo",
			"mock-gpt-4",
			"mock-claude-3-sonnet",
		},
	}

	// Apply settings from core config if provided
	if config.Timeout > 0 {
		mockConfig.Latency = config.Timeout / 10 // Use 1/10th of timeout as latency
	}

	// Parse provider-specific extra settings
	if config.Extra != nil {
		if latency, ok := config.Extra["latency"].(time.Duration); ok {
			mockConfig.Latency = latency
		}
		if latencyStr, ok := config.Extra["latency"].(string); ok {
			if parsed, err := time.ParseDuration(latencyStr); err == nil {
				mockConfig.Latency = parsed
			}
		}
		if maxTokens, ok := config.Extra["max_tokens"].(int); ok {
			mockConfig.MaxTokens = maxTokens
		}
		if errorRate, ok := config.Extra["error_rate"].(float64); ok {
			mockConfig.ErrorRate = errorRate
		}
		// Note: default response handling is managed internally by the mock provider
	}

	return New(mockConfig), nil
}

// init registers the mock provider factory with the global registry.
func init() {
	core.RegisterProviderFactory("mock", NewFromConfig)
}
