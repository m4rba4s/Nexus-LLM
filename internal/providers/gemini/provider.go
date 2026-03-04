// Package gemini provides a Google Gemini API provider implementation.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/gollm/internal/core"
)

const (
	// DefaultBaseURL is the default Gemini API base URL
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

	// DefaultModel is the default Gemini model to use
	DefaultModel = "gemini-2.0-flash"

	// ProviderName is the name of this provider
	ProviderName = "gemini"

	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second

	// UserAgent is the default user agent
	UserAgent = "gollm/1.0"
)

// Config holds the configuration for the Gemini provider.
type Config struct {
	APIKey     string            `json:"api_key" validate:"required"`
	BaseURL    string            `json:"base_url,omitempty"`
	Model      string            `json:"model,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
}

// Provider implements the Gemini API provider.
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

// New creates a new Gemini provider instance.
func New(config Config) (*Provider, error) {
	// Apply defaults
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.UserAgent == "" {
		config.UserAgent = UserAgent
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, &core.ValidationError{
			Field:   "api_key",
			Value:   config.APIKey,
			Rule:    "required",
			Message: "Gemini API key is required",
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

	return &Provider{
		config:         config,
		client:         client,
		modelsCacheTTL: 10 * time.Minute,
		metrics:        &core.ProviderMetrics{},
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// CreateCompletion creates a completion using the Gemini API.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to Gemini request format
	geminiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make the API call
	model := req.Model
	if model == "" {
		model = p.config.Model
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", p.config.BaseURL, model)

	requestBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-goog-api-key", p.config.APIKey)
	httpReq.Header.Set("User-Agent", p.config.UserAgent)

	// Add custom headers
	for k, v := range p.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Make the request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response format
	return p.convertResponse(&geminiResp, req)
}

// StreamCompletion creates a streaming completion using the Gemini API.
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	// TODO: Implement streaming for Gemini API when supported
	// For now, fall back to regular completion and stream the result
	response, err := p.CreateCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create a channel and send the complete response as chunks
	chunks := make(chan core.StreamChunk, 1)

	go func() {
		defer close(chunks)

		if len(response.Choices) > 0 {
			content := response.Choices[0].Message.Content

			// Create a single chunk with the complete response
			chunk := core.StreamChunk{
				ID:     response.ID,
				Object: "chat.completion.chunk",
				Model:  response.Model,
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
				Done: true,
			}
			chunks <- chunk
		}
	}()

	return chunks, nil
}

// GetModels returns available models for the Gemini provider.
func (p *Provider) GetModels(ctx context.Context) ([]core.Model, error) {
	p.mu.RLock()
	if time.Since(p.modelsCacheTime) < p.modelsCacheTTL && len(p.modelsCache) > 0 {
		models := make([]core.Model, len(p.modelsCache))
		copy(models, p.modelsCache)
		p.mu.RUnlock()
		return models, nil
	}
	p.mu.RUnlock()

	// Static list of known Gemini models
	models := []core.Model{
		{
			ID:                "gemini-2.0-flash",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{1048576}[0],
			SupportsFunctions: true,
			SupportsStreaming: false,
			InputCostPer1K:    &[]float64{0.075}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{0.30}[0],  // per 1M tokens -> per 1K
			Description:       "Gemini 2.0 Flash - Fast and efficient model",
			Tags:              []string{"fast", "multimodal"},
		},
		{
			ID:                "gemini-1.5-pro",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{2097152}[0],
			SupportsFunctions: true,
			SupportsStreaming: false,
			SupportsVision:    true,
			InputCostPer1K:    &[]float64{1.25}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{5.00}[0], // per 1M tokens -> per 1K
			Description:       "Gemini 1.5 Pro - High-performance model with vision",
			Tags:              []string{"pro", "multimodal", "vision"},
		},
		{
			ID:                "gemini-1.5-flash",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{1048576}[0],
			SupportsFunctions: true,
			SupportsStreaming: false,
			InputCostPer1K:    &[]float64{0.075}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{0.30}[0],  // per 1M tokens -> per 1K
			Description:       "Gemini 1.5 Flash - Fast and cost-effective",
			Tags:              []string{"fast", "economical"},
		},
	}

	// Cache the models
	p.mu.Lock()
	p.modelsCache = make([]core.Model, len(models))
	copy(p.modelsCache, models)
	p.modelsCacheTime = time.Now()
	p.mu.Unlock()

	return models, nil
}

// ValidateConfig validates the provider configuration.
func (p *Provider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if p.config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if p.config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if p.config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	return nil
}

// convertRequest converts a standard CompletionRequest to Gemini format.
func (p *Provider) convertRequest(req *core.CompletionRequest) (*GeminiRequest, error) {
	// Convert messages to Gemini content format
	var contents []GeminiContent

	for _, msg := range req.Messages {
		// Map roles
		var role string
		switch msg.Role {
		case "system":
			// Gemini doesn't have system role, prepend to user message
			role = "user"
		case "user":
			role = "user"
		case "assistant":
			role = "model"
		default:
			role = "user"
		}

		content := GeminiContent{
			Role: role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		}
		contents = append(contents, content)
	}

	geminiReq := &GeminiRequest{
		Contents: contents,
	}

	// Add generation config if specified
	if req.MaxTokens != nil || req.Temperature != nil || req.TopP != nil {
		config := &GeminiGenerationConfig{}

		if req.MaxTokens != nil {
			config.MaxOutputTokens = *req.MaxTokens
		}
		if req.Temperature != nil {
			config.Temperature = *req.Temperature
		}
		if req.TopP != nil {
			config.TopP = *req.TopP
		}

		geminiReq.GenerationConfig = config
	}

	return geminiReq, nil
}

// convertResponse converts a Gemini response to standard format.
func (p *Provider) convertResponse(geminiResp *GeminiResponse, originalReq *core.CompletionRequest) (*core.CompletionResponse, error) {
	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	var choices []core.Choice

	for i, candidate := range geminiResp.Candidates {
		if len(candidate.Content.Parts) == 0 {
			continue
		}

		// Combine all text parts
		var content strings.Builder
		for _, part := range candidate.Content.Parts {
			content.WriteString(part.Text)
		}

		choice := core.Choice{
			Index: i,
			Message: core.Message{
				Role:    "assistant",
				Content: content.String(),
			},
			FinishReason: p.mapFinishReason(candidate.FinishReason),
		}
		choices = append(choices, choice)
	}

	response := &core.CompletionResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Model:   originalReq.Model,
		Choices: choices,
		Usage: core.Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}

	return response, nil
}

// mapFinishReason maps Gemini finish reasons to standard format.
func (p *Provider) mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}

// GeminiRequest represents a request to the Gemini API.
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`
}

// GeminiContent represents content in a Gemini request.
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content.
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration.
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GeminiSafetySetting represents safety settings.
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents a response from the Gemini API.
type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata GeminiUsage       `json:"usageMetadata"`
}

// GeminiCandidate represents a candidate response.
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

// GeminiUsage represents usage information.
type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewFromConfig creates a new Gemini provider from core configuration.
// This is used by the provider registry for dynamic provider creation.
func NewFromConfig(config core.ProviderConfig) (core.Provider, error) {
	// Convert core config to Gemini-specific config
	geminiConfig := Config{
		APIKey:     config.APIKey,
		BaseURL:    DefaultBaseURL,
		Model:      DefaultModel,
		MaxRetries: 3,
		Timeout:    DefaultTimeout,
		UserAgent:  UserAgent,
	}

	// Apply settings from core config if provided
	if config.BaseURL != "" {
		geminiConfig.BaseURL = config.BaseURL
	}
	if config.Timeout > 0 {
		geminiConfig.Timeout = config.Timeout
	}
	if config.MaxRetries > 0 {
		geminiConfig.MaxRetries = config.MaxRetries
	}

	// Parse provider-specific extra settings
	if config.Extra != nil {
		if model, ok := config.Extra["default_model"].(string); ok {
			geminiConfig.Model = model
		}
		if headers, ok := config.Extra["headers"].(map[string]string); ok {
			geminiConfig.Headers = headers
		}
		if userAgent, ok := config.Extra["user_agent"].(string); ok {
			geminiConfig.UserAgent = userAgent
		}
	}

	return New(geminiConfig)
}

// init registers the Gemini provider factory with the global registry.
func init() {
	core.RegisterProviderFactory("gemini", NewFromConfig)
}
