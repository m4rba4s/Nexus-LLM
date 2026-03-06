// Package anthropic provides an Anthropic LLM provider implementation.
//
// This package implements the Provider interface for Anthropic's Claude models,
// supporting chat completions, streaming responses, and system messages.
//
// Example usage:
//
//	config := anthropic.Config{
//		APIKey:  "sk-ant-...",
//		BaseURL: "https://api.anthropic.com", // optional
//	}
//
//	provider, err := anthropic.New(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	resp, err := provider.CreateCompletion(ctx, &core.CompletionRequest{
//		Model:    "claude-3-sonnet-20240229",
//		Messages: []core.Message{{Role: "user", Content: "Hello"}},
//	})
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

const (
	DefaultBaseURL = "https://api.anthropic.com"
	DefaultTimeout = 30 * time.Second
	UserAgent      = "gollm/1.0"
	APIVersion     = "2023-06-01"
)

// Provider implements the core.Provider interface for Anthropic.
type Provider struct {
	mu      sync.RWMutex
	config  Config
	client  *http.Client
	metrics *core.ProviderMetrics

	// Model capabilities cache
	modelsCache     []core.Model
	modelsCacheTime time.Time
	modelsCacheTTL  time.Duration
}

// Config configures the Anthropic provider.
type Config struct {
	APIKey     string        `json:"api_key" validate:"required"`
	BaseURL    string        `json:"base_url,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	MaxRetries int           `json:"max_retries,omitempty"`

	// HTTP client settings
	UserAgent     string            `json:"user_agent,omitempty"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// API version
	APIVersion string `json:"api_version,omitempty"`
}

// DefaultConfig returns sensible defaults for the Anthropic provider.
func DefaultConfig() Config {
	return Config{
		BaseURL:    DefaultBaseURL,
		Timeout:    DefaultTimeout,
		MaxRetries: 3,
		UserAgent:  UserAgent,
		APIVersion: APIVersion,
	}
}

// New creates a new Anthropic provider with the given configuration.
func New(config Config) (*Provider, error) {
	// Apply defaults
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.UserAgent == "" {
		config.UserAgent = UserAgent
	}
	if config.APIVersion == "" {
		config.APIVersion = APIVersion
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, &core.ValidationError{
			Field:   "api_key",
			Value:   config.APIKey,
			Rule:    "required",
			Message: "Anthropic API key is required",
		}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	provider := &Provider{
		config:         config,
		client:         client,
		metrics:        core.NewProviderMetrics(),
		modelsCacheTTL: 5 * time.Minute,
	}

	return provider, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "anthropic"
}

// CreateCompletion creates a chat completion synchronously.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	start := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to Anthropic format
	anthropicReq, err := p.convertRequest(req)
	if err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API request
	var anthropicResp anthropicCompletionResponse
	if err := p.makeRequest(ctx, "POST", "/v1/messages", anthropicReq, &anthropicResp); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, err
	}

	// Convert response
	response, err := p.convertResponse(&anthropicResp, req)
	if err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	// Calculate cost and record metrics
	cost := p.calculateCost(req.Model, &response.Usage)
	responseTime := time.Since(start)
	response.ResponseTime = responseTime

	p.metrics.RecordRequest(responseTime, &response.Usage, &cost, nil)

	return response, nil
}

// StreamCompletion creates a streaming chat completion.
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to Anthropic format with streaming enabled
	anthropicReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	anthropicReq.Stream = true

	// Create response channel
	chunks := make(chan core.StreamChunk, 10)

	go func() {
		defer close(chunks)

		start := time.Now()

		if err := p.streamRequest(ctx, anthropicReq, chunks, req); err != nil {
			chunks <- core.StreamChunk{
				Error: err,
				Done:  true,
			}
			p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		}
	}()

	return chunks, nil
}

// GetModels returns the list of available models.
func (p *Provider) GetModels(ctx context.Context) ([]core.Model, error) {
	p.mu.RLock()
	if p.modelsCache != nil && time.Since(p.modelsCacheTime) < p.modelsCacheTTL {
		models := make([]core.Model, len(p.modelsCache))
		copy(models, p.modelsCache)
		p.mu.RUnlock()
		return models, nil
	}
	p.mu.RUnlock()

	// Anthropic doesn't have a models endpoint, so we return predefined models
	models := p.getPredefinedModels()

	// Cache results
	p.mu.Lock()
	p.modelsCache = models
	p.modelsCacheTime = time.Now()
	p.mu.Unlock()

	return models, nil
}

// ValidateConfig validates the provider configuration.
func (p *Provider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return &core.ValidationError{
			Field:   "api_key",
			Value:   p.config.APIKey,
			Rule:    "required",
			Message: "API key cannot be empty",
		}
	}

	if p.config.BaseURL == "" {
		return &core.ValidationError{
			Field:   "base_url",
			Value:   p.config.BaseURL,
			Rule:    "required",
			Message: "base URL cannot be empty",
		}
	}

	if p.config.Timeout <= 0 {
		return &core.ValidationError{
			Field:   "timeout",
			Value:   p.config.Timeout,
			Rule:    "positive",
			Message: "timeout must be positive",
		}
	}

	return nil
}

// Private helper methods

// makeRequest makes an HTTP request to the Anthropic API.
func (p *Provider) makeRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := strings.TrimSuffix(p.config.BaseURL, "/") + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("User-Agent", p.config.UserAgent)
	req.Header.Set("anthropic-version", p.config.APIVersion)

	// Add custom headers
	for key, value := range p.config.CustomHeaders {
		req.Header.Set(key, value)
	}

	// Make request with retry logic
	resp, err := p.makeRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return p.handleErrorResponse(resp)
	}

	// Parse successful response
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// makeRequestWithRetry makes an HTTP request with retry logic.
func (p *Provider) makeRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for i := 0; i <= p.config.MaxRetries; i++ {
		// Clone request for retry
		reqCopy := req.Clone(req.Context())

		resp, err := p.client.Do(reqCopy)
		if err != nil {
			lastErr = err
			if i < p.config.MaxRetries {
				backoff := time.Duration(i+1) * time.Second
				time.Sleep(backoff)
				continue
			}
			break
		}

		// Check for retryable status codes
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			// For 4xx errors like 429, keep the response for error handling
			if resp.StatusCode < 500 {
				if lastResp != nil {
					lastResp.Body.Close()
				}
				lastResp = resp
			} else {
				resp.Body.Close()
			}

			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			if i < p.config.MaxRetries {
				backoff := time.Duration(i+1) * time.Second
				time.Sleep(backoff)
				continue
			}
			break
		}

		return resp, nil
	}

	// If we have a 4xx response (like 429), return it for proper error handling
	if lastResp != nil {
		return lastResp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", p.config.MaxRetries, lastErr)
}

// handleErrorResponse handles error responses from the Anthropic API.
func (p *Provider) handleErrorResponse(resp *http.Response) error {
	var errorResp anthropicErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		return &core.APIError{
			StatusCode: resp.StatusCode,
			Code:       "unknown_error",
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			Provider:   "anthropic",
		}
	}

	return &core.APIError{
		StatusCode: resp.StatusCode,
		Code:       errorResp.Error.Type,
		Message:    errorResp.Error.Message,
		Provider:   "anthropic",
	}
}

// streamRequest handles streaming requests.
func (p *Provider) streamRequest(ctx context.Context, req *anthropicCompletionRequest, chunks chan<- core.StreamChunk, originalReq *core.CompletionRequest) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("User-Agent", p.config.UserAgent)
	httpReq.Header.Set("anthropic-version", p.config.APIVersion)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return p.handleErrorResponse(resp)
	}

	// Process stream
	scanner := bufio.NewScanner(resp.Body)
	var totalUsage core.Usage
	start := time.Now()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // Skip malformed chunks
		}

		coreChunk := p.convertStreamEvent(&event, originalReq)

		// Handle usage information
		if event.Type == "message_stop" && event.Message != nil && event.Message.Usage != nil {
			totalUsage = core.Usage{
				PromptTokens:     event.Message.Usage.InputTokens,
				CompletionTokens: event.Message.Usage.OutputTokens,
				TotalTokens:      event.Message.Usage.InputTokens + event.Message.Usage.OutputTokens,
			}
		}

		select {
		case chunks <- coreChunk:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Stop on message_stop event
		if event.Type == "message_stop" {
			break
		}
	}

	// Send final chunk with usage
	if totalUsage.TotalTokens > 0 {
		cost := p.calculateCost(originalReq.Model, &totalUsage)
		responseTime := time.Since(start)

		finalChunk := core.StreamChunk{
			Usage:     &totalUsage,
			Done:      true,
			Provider:  "anthropic",
			RequestID: originalReq.RequestID,
			Timestamp: time.Now(),
		}

		chunks <- finalChunk
		p.metrics.RecordRequest(responseTime, &totalUsage, &cost, nil)
	}

	return scanner.Err()
}

// convertRequest converts a core request to Anthropic format.
func (p *Provider) convertRequest(req *core.CompletionRequest) (*anthropicCompletionRequest, error) {
	anthropicReq := &anthropicCompletionRequest{
		Model:  req.Model,
		Stream: req.Stream,
	}

	// Separate system message from other messages
	var systemMessage string
	var messages []anthropicMessage

	for _, msg := range req.Messages {
		if msg.Role == core.RoleSystem {
			if systemMessage != "" {
				systemMessage += "\n\n" + msg.Content
			} else {
				systemMessage = msg.Content
			}
		} else if msg.Role == core.RoleUser || msg.Role == core.RoleAssistant {
			messages = append(messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Apply system message from request if provided
	if req.SystemMessage != nil && *req.SystemMessage != "" {
		if systemMessage != "" {
			systemMessage = *req.SystemMessage + "\n\n" + systemMessage
		} else {
			systemMessage = *req.SystemMessage
		}
	}

	if systemMessage != "" {
		anthropicReq.System = systemMessage
	}

	anthropicReq.Messages = messages

	// Set optional parameters
	if req.MaxTokens != nil {
		anthropicReq.MaxTokens = *req.MaxTokens
	} else {
		// Anthropic requires max_tokens
		anthropicReq.MaxTokens = 4096
	}

	if req.Temperature != nil {
		anthropicReq.Temperature = req.Temperature
	}

	if req.TopP != nil {
		anthropicReq.TopP = req.TopP
	}

	if len(req.Stop) > 0 {
		anthropicReq.StopSequences = req.Stop
	}

	return anthropicReq, nil
}

// convertResponse converts an Anthropic response to core format.
func (p *Provider) convertResponse(resp *anthropicCompletionResponse, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	coreResp := &core.CompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Usage: core.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Provider:  "anthropic",
		RequestID: req.RequestID,
	}

	// Convert content to choices
	if len(resp.Content) > 0 {
		coreResp.Choices = make([]core.Choice, 1)

		// Combine all text content
		var content strings.Builder
		for _, block := range resp.Content {
			if block.Type == "text" {
				content.WriteString(block.Text)
			}
		}

		coreResp.Choices[0] = core.Choice{
			Index: 0,
			Message: core.Message{
				Role:    core.RoleAssistant,
				Content: content.String(),
			},
			FinishReason: resp.StopReason,
		}
	}

	return coreResp, nil
}

// convertStreamEvent converts an Anthropic stream event to core format.
func (p *Provider) convertStreamEvent(event *anthropicStreamEvent, req *core.CompletionRequest) core.StreamChunk {
	coreChunk := core.StreamChunk{
		Object:    "chat.completion.chunk",
		Created:   time.Now().Unix(),
		Provider:  "anthropic",
		RequestID: req.RequestID,
		Timestamp: time.Now(),
	}

	// Set ID and Model from Message if available
	if event.Message != nil {
		coreChunk.ID = event.Message.ID
		coreChunk.Model = event.Message.Model
	}

	switch event.Type {
	case "message_start":
		// Initial chunk
		coreChunk.Choices = []core.Choice{
			{
				Index: 0,
				Delta: &core.Message{
					Role: core.RoleAssistant,
				},
			},
		}

	case "content_block_delta":
		if event.Delta != nil && event.Delta.Text != "" {
			coreChunk.Choices = []core.Choice{
				{
					Index: 0,
					Delta: &core.Message{
						Content: event.Delta.Text,
					},
				},
			}
		}

	case "message_stop":
		coreChunk.Choices = []core.Choice{
			{
				Index:        0,
				Delta:        &core.Message{},
				FinishReason: "stop",
			},
		}
		coreChunk.Done = true
	}

	return coreChunk
}

// getPredefinedModels returns the predefined Anthropic models.
func (p *Provider) getPredefinedModels() []core.Model {
	models := []core.Model{
		{
			ID:                "claude-3-opus-20240229",
			Object:            "model",
			Created:           time.Now().Unix(),
			OwnedBy:           "anthropic",
			Provider:          "anthropic",
			MaxTokens:         intPtr(4096),
			SupportsFunctions: false,
			SupportsStreaming: true,
			SupportsVision:    true,
			InputCostPer1K:    float64Ptr(0.015),
			OutputCostPer1K:   float64Ptr(0.075),
			Description:       "Anthropic's most powerful model, with top-level performance on highly complex tasks",
			Tags:              []string{"anthropic", "claude", "opus"},
		},
		{
			ID:                "claude-3-sonnet-20240229",
			Object:            "model",
			Created:           time.Now().Unix(),
			OwnedBy:           "anthropic",
			Provider:          "anthropic",
			MaxTokens:         intPtr(4096),
			SupportsFunctions: false,
			SupportsStreaming: true,
			SupportsVision:    true,
			InputCostPer1K:    float64Ptr(0.003),
			OutputCostPer1K:   float64Ptr(0.015),
			Description:       "Balance of intelligence and speed for enterprise workloads",
			Tags:              []string{"anthropic", "claude", "sonnet"},
		},
		{
			ID:                "claude-3-haiku-20240307",
			Object:            "model",
			Created:           time.Now().Unix(),
			OwnedBy:           "anthropic",
			Provider:          "anthropic",
			MaxTokens:         intPtr(4096),
			SupportsFunctions: false,
			SupportsStreaming: true,
			SupportsVision:    true,
			InputCostPer1K:    float64Ptr(0.00025),
			OutputCostPer1K:   float64Ptr(0.00125),
			Description:       "Fastest and most compact model for near-instant responsiveness",
			Tags:              []string{"anthropic", "claude", "haiku"},
		},
	}

	return models
}

// calculateCost calculates the cost for a completion.
func (p *Provider) calculateCost(modelID string, usage *core.Usage) float64 {
	var inputRate, outputRate float64

	switch modelID {
	case "claude-3-opus-20240229":
		inputRate, outputRate = 0.015, 0.075
	case "claude-3-sonnet-20240229":
		inputRate, outputRate = 0.003, 0.015
	case "claude-3-haiku-20240307":
		inputRate, outputRate = 0.00025, 0.00125
	default:
		return 0
	}

	inputCost := float64(usage.PromptTokens) / 1000.0 * inputRate
	outputCost := float64(usage.CompletionTokens) / 1000.0 * outputRate

	return inputCost + outputCost
}

// GetMetrics returns the provider metrics.
func (p *Provider) GetMetrics() *core.ProviderMetrics {
	return p.metrics.Clone()
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// Anthropic API types

type anthropicCompletionRequest struct {
	Model         string             `json:"model"`
	Messages      []anthropicMessage `json:"messages"`
	System        string             `json:"system,omitempty"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicCompletionResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence *string            `json:"stop_sequence"`
	Usage        anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicStreamEvent struct {
	Type    string                  `json:"type"`
	Message *anthropicStreamMessage `json:"message,omitempty"`
	Index   *int                    `json:"index,omitempty"`
	Delta   *anthropicStreamDelta   `json:"delta,omitempty"`
}

type anthropicStreamMessage struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Role  string          `json:"role"`
	Model string          `json:"model"`
	Usage *anthropicUsage `json:"usage,omitempty"`
}

type anthropicStreamDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewFromConfig creates a new Anthropic provider from core configuration.
// This is used by the provider registry for dynamic provider creation.
func NewFromConfig(config core.ProviderConfig) (core.Provider, error) {
	// Convert core config to Anthropic-specific config
	anthropicConfig := Config{
		APIKey:     config.APIKey,
		BaseURL:    config.BaseURL,
		MaxRetries: config.MaxRetries,
		Timeout:    config.Timeout,
	}

	// Apply custom headers if provided
	if config.CustomHeaders != nil {
		anthropicConfig.CustomHeaders = config.CustomHeaders
	}

	// Parse provider-specific extra settings
	if config.Extra != nil {
		if userAgent, ok := config.Extra["user_agent"].(string); ok {
			anthropicConfig.UserAgent = userAgent
		}
		if apiVersion, ok := config.Extra["api_version"].(string); ok {
			anthropicConfig.APIVersion = apiVersion
		}
	}

	return New(anthropicConfig)
}

// init registers the Anthropic provider factory with the global registry.
func init() {
	core.RegisterProviderFactory("anthropic", NewFromConfig)
}
