// Package deepseek provides a DeepSeek API provider implementation.
package deepseek

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

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

const (
	// DefaultBaseURL is the default DeepSeek API base URL
	DefaultBaseURL = "https://api.deepseek.com/v1"

	// DefaultModel is the default DeepSeek model to use
	DefaultModel = "deepseek-chat"

	// ProviderName is the name of this provider
	ProviderName = "deepseek"

	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second

	// UserAgent is the default user agent
	UserAgent = "gollm/1.0"
)

// Config holds the configuration for the DeepSeek provider.
type Config struct {
	APIKey     string            `json:"api_key" validate:"required"`
	BaseURL    string            `json:"base_url,omitempty"`
	Model      string            `json:"model,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
}

// Provider implements the DeepSeek API provider.
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

// New creates a new DeepSeek provider instance.
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
			Message: "DeepSeek API key is required",
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

// CreateCompletion creates a completion using the DeepSeek API.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to DeepSeek request format (OpenAI-compatible)
	deepseekReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)

	requestBody, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
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
	var deepseekResp DeepSeekResponse
	if err := json.Unmarshal(body, &deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response format
	return p.convertResponse(&deepseekResp, req)
}

// StreamCompletion creates a streaming completion using the DeepSeek API.
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to DeepSeek request format with streaming enabled
	deepseekReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	deepseekReq.Stream = true

	endpoint := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)

	requestBody, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("User-Agent", p.config.UserAgent)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Add custom headers
	for k, v := range p.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Make the streaming request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	chunks := make(chan core.StreamChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			chunks <- core.StreamChunk{
				Error: fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body)),
			}
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var line string
			if err := decoder.Decode(&line); err != nil {
				if err == io.EOF {
					break
				}
				chunks <- core.StreamChunk{Error: err}
				return
			}

			// Parse SSE data
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				data = strings.TrimSpace(data)

				if data == "[DONE]" {
					chunks <- core.StreamChunk{Done: true}
					return
				}

				var streamResp DeepSeekStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					chunks <- core.StreamChunk{Error: err}
					return
				}

				// Convert to standard stream chunk
				if len(streamResp.Choices) > 0 {
					choice := streamResp.Choices[0]
					chunk := core.StreamChunk{
						ID:     streamResp.ID,
						Object: "chat.completion.chunk",
						Model:  streamResp.Model,
						Choices: []core.Choice{
							{
								Index: choice.Index,
								Message: core.Message{
									Role:    choice.Delta.Role,
									Content: choice.Delta.Content,
								},
								FinishReason: choice.FinishReason,
							},
						},
						Done: choice.FinishReason != "",
					}
					chunks <- chunk
				}
			}
		}
	}()

	return chunks, nil
}

// GetModels returns available models for the DeepSeek provider.
func (p *Provider) GetModels(ctx context.Context) ([]core.Model, error) {
	p.mu.RLock()
	if time.Since(p.modelsCacheTime) < p.modelsCacheTTL && len(p.modelsCache) > 0 {
		models := make([]core.Model, len(p.modelsCache))
		copy(models, p.modelsCache)
		p.mu.RUnlock()
		return models, nil
	}
	p.mu.RUnlock()

	// Static list of known DeepSeek models
	models := []core.Model{
		{
			ID:                "deepseek-chat",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{32768}[0],
			SupportsFunctions: true,
			SupportsStreaming: true,
			InputCostPer1K:    &[]float64{0.14}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{0.28}[0], // per 1M tokens -> per 1K
			Description:       "DeepSeek Chat - General purpose conversational model",
			Tags:              []string{"chat", "general"},
		},
		{
			ID:                "deepseek-coder",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{16384}[0],
			SupportsFunctions: true,
			SupportsStreaming: true,
			InputCostPer1K:    &[]float64{0.14}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{0.28}[0], // per 1M tokens -> per 1K
			Description:       "DeepSeek Coder - Specialized coding model",
			Tags:              []string{"code", "programming"},
		},
		{
			ID:                "deepseek-v2.5",
			Object:            "model",
			Provider:          ProviderName,
			MaxTokens:         &[]int{32768}[0],
			SupportsFunctions: true,
			SupportsStreaming: true,
			InputCostPer1K:    &[]float64{0.14}[0], // per 1M tokens -> per 1K
			OutputCostPer1K:   &[]float64{0.28}[0], // per 1M tokens -> per 1K
			Description:       "DeepSeek V2.5 - Latest version with improved capabilities",
			Tags:              []string{"latest", "improved"},
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

// convertRequest converts a standard CompletionRequest to DeepSeek format.
func (p *Provider) convertRequest(req *core.CompletionRequest) (*DeepSeekRequest, error) {
	// Convert messages to DeepSeek format (OpenAI-compatible)
	var messages []DeepSeekMessage
	for _, msg := range req.Messages {
		messages = append(messages, DeepSeekMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
	}

	deepseekReq := &DeepSeekRequest{
		Model:    model,
		Messages: messages,
	}

	// Set optional parameters
	if req.MaxTokens != nil {
		deepseekReq.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		deepseekReq.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		deepseekReq.TopP = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		deepseekReq.FrequencyPenalty = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		deepseekReq.PresencePenalty = *req.PresencePenalty
	}

	return deepseekReq, nil
}

// convertResponse converts a DeepSeek response to standard format.
func (p *Provider) convertResponse(deepseekResp *DeepSeekResponse, originalReq *core.CompletionRequest) (*core.CompletionResponse, error) {
	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var choices []core.Choice
	for _, choice := range deepseekResp.Choices {
		choices = append(choices, core.Choice{
			Index: choice.Index,
			Message: core.Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		})
	}

	response := &core.CompletionResponse{
		ID:      deepseekResp.ID,
		Object:  "chat.completion",
		Model:   deepseekResp.Model,
		Choices: choices,
		Usage: core.Usage{
			PromptTokens:     deepseekResp.Usage.PromptTokens,
			CompletionTokens: deepseekResp.Usage.CompletionTokens,
			TotalTokens:      deepseekResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// DeepSeekRequest represents a request to the DeepSeek API.
type DeepSeekRequest struct {
	Model            string            `json:"model"`
	Messages         []DeepSeekMessage `json:"messages"`
	MaxTokens        int               `json:"max_tokens,omitempty"`
	Temperature      float64           `json:"temperature,omitempty"`
	TopP             float64           `json:"top_p,omitempty"`
	FrequencyPenalty float64           `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64           `json:"presence_penalty,omitempty"`
	Stream           bool              `json:"stream,omitempty"`
}

// DeepSeekMessage represents a message in a DeepSeek request.
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse represents a response from the DeepSeek API.
type DeepSeekResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []DeepSeekChoice `json:"choices"`
	Usage   DeepSeekUsage    `json:"usage"`
}

// DeepSeekChoice represents a choice in a DeepSeek response.
type DeepSeekChoice struct {
	Index        int             `json:"index"`
	Message      DeepSeekMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// DeepSeekUsage represents usage information in a DeepSeek response.
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeepSeekStreamResponse represents a streaming response from DeepSeek.
type DeepSeekStreamResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []DeepSeekStreamChoice `json:"choices"`
}

// DeepSeekStreamChoice represents a streaming choice.
type DeepSeekStreamChoice struct {
	Index        int                 `json:"index"`
	Delta        DeepSeekStreamDelta `json:"delta"`
	FinishReason string              `json:"finish_reason"`
}

// DeepSeekStreamDelta represents a streaming delta.
type DeepSeekStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// NewFromConfig creates a new DeepSeek provider from core configuration.
// This is used by the provider registry for dynamic provider creation.
func NewFromConfig(config core.ProviderConfig) (core.Provider, error) {
	// Convert core config to DeepSeek-specific config
	deepseekConfig := Config{
		APIKey:     config.APIKey,
		BaseURL:    DefaultBaseURL,
		Model:      DefaultModel,
		MaxRetries: 3,
		Timeout:    DefaultTimeout,
		UserAgent:  UserAgent,
	}

	// Apply settings from core config if provided
	if config.BaseURL != "" {
		deepseekConfig.BaseURL = config.BaseURL
	}
	if config.Timeout > 0 {
		deepseekConfig.Timeout = config.Timeout
	}
	if config.MaxRetries > 0 {
		deepseekConfig.MaxRetries = config.MaxRetries
	}

	// Parse provider-specific extra settings
	if config.Extra != nil {
		if model, ok := config.Extra["default_model"].(string); ok {
			deepseekConfig.Model = model
		}
		if headers, ok := config.Extra["headers"].(map[string]string); ok {
			deepseekConfig.Headers = headers
		}
		if userAgent, ok := config.Extra["user_agent"].(string); ok {
			deepseekConfig.UserAgent = userAgent
		}
	}

	provider, err := New(deepseekConfig)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// init registers the DeepSeek provider factory with the global registry.
func init() {
	core.RegisterProviderFactory("deepseek", NewFromConfig)
}
