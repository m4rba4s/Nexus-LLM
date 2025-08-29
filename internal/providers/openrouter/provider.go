// Package openrouter provides an OpenRouter API provider implementation.
package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"time"

	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/transport"
)

const (
	// DefaultBaseURL is the default OpenRouter API base URL
	DefaultBaseURL = "https://openrouter.ai/api/v1"

	// DefaultModel is the default OpenRouter model to use
	DefaultModel = "openai/gpt-4o"

	// ProviderName is the name of this provider
	ProviderName = "openrouter"
)

// Config holds the configuration for the OpenRouter provider.
type Config struct {
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url"`
	Model       string            `json:"model"`
	MaxRetries  int               `json:"max_retries"`
	Timeout     time.Duration     `json:"timeout"`
	Headers     map[string]string `json:"headers"`
	SiteURL     string            `json:"site_url"`     // For OpenRouter rankings
	SiteName    string            `json:"site_name"`    // For OpenRouter rankings
}

// Provider implements the OpenRouter API provider.
type Provider struct {
	config Config
	client *transport.HTTPClient
}

// New creates a new OpenRouter provider instance.
func New(config Config) (*Provider, error) {
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
		config.Timeout = 30 * time.Second
	}

	// Create HTTP client config
	httpConfig := transport.HTTPConfig{
		BaseURL: config.BaseURL,
		Timeout: config.Timeout,
	}

	client, err := transport.NewHTTPClient(httpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}


	// Set default headers
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + config.APIKey,
	}

	// Add OpenRouter-specific headers for rankings (optional but recommended)
	if config.SiteURL != "" {
		headers["HTTP-Referer"] = config.SiteURL
	}
	if config.SiteName != "" {
		headers["X-Title"] = config.SiteName
	}

	// Add custom headers
	for k, v := range config.Headers {
		headers[k] = v
	}



	return &Provider{
		config: config,
		client: client,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// CreateCompletion creates a completion using the OpenRouter API.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to OpenRouter request format (OpenAI-compatible)
	openrouterReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make the API call
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.config.APIKey,
	}

	// Add OpenRouter-specific headers
	if p.config.SiteURL != "" {
		headers["HTTP-Referer"] = p.config.SiteURL
	}
	if p.config.SiteName != "" {
		headers["X-Title"] = p.config.SiteName
	}

	// Add custom headers
	for k, v := range p.config.Headers {
		headers[k] = v
	}

	httpResp, err := p.client.Post(ctx, "/chat/completions", openrouterReq, headers)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var openrouterResp OpenRouterResponse
	if err := json.Unmarshal(respBody, &openrouterResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response format
	return p.convertResponse(&openrouterResp, req)
}

// StreamCompletion creates a streaming completion using the OpenRouter API.
func (p *Provider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to OpenRouter request format with streaming enabled
	openrouterReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	openrouterReq.Stream = true

	// Create streaming request
	// Create streaming request (this is a simplified implementation)
	// In a real implementation, you'd need to implement SSE streaming
	return nil, fmt.Errorf("streaming not yet implemented for OpenRouter provider")

}

// GetModels returns available models for the OpenRouter provider.
func (p *Provider) GetModels(ctx context.Context) ([]core.Model, error) {
	// Make API call to get models
	headers := map[string]string{
		"Authorization": "Bearer " + p.config.APIKey,
	}

	httpResp, err := p.client.Get(ctx, "/models", headers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var modelsResp OpenRouterModelsResponse
	if err := json.Unmarshal(respBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	var models []core.Model
	for _, model := range modelsResp.Data {
		maxTokens := model.ContextLength
		inputCost := model.Pricing.Prompt
		outputCost := model.Pricing.Completion

		models = append(models, core.Model{
			ID:              model.ID,
			Provider:        ProviderName,
			MaxTokens:       &maxTokens,
			InputCostPer1K:  &inputCost,
			OutputCostPer1K: &outputCost,
		})
	}

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

// convertRequest converts a standard CompletionRequest to OpenRouter format.
func (p *Provider) convertRequest(req *core.CompletionRequest) (*OpenRouterRequest, error) {
	// Convert messages to OpenRouter format (OpenAI-compatible)
	var messages []OpenRouterMessage
	for _, msg := range req.Messages {
		messages = append(messages, OpenRouterMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
	}

	openrouterReq := &OpenRouterRequest{
		Model:    model,
		Messages: messages,
	}

	// Set optional parameters
	if req.MaxTokens != nil {
		openrouterReq.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		openrouterReq.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		openrouterReq.TopP = *req.TopP
	}
	if req.FrequencyPenalty != nil {
		openrouterReq.FrequencyPenalty = *req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		openrouterReq.PresencePenalty = *req.PresencePenalty
	}

	return openrouterReq, nil
}

// convertResponse converts an OpenRouter response to standard format.
func (p *Provider) convertResponse(openrouterResp *OpenRouterResponse, originalReq *core.CompletionRequest) (*core.CompletionResponse, error) {
	if len(openrouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var choices []core.Choice
	for _, choice := range openrouterResp.Choices {
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
		ID:      openrouterResp.ID,
		Model:   openrouterResp.Model,
		Choices: choices,
		Usage: core.Usage{
			PromptTokens:     openrouterResp.Usage.PromptTokens,
			CompletionTokens: openrouterResp.Usage.CompletionTokens,
			TotalTokens:      openrouterResp.Usage.TotalTokens,
		},
		Provider: ProviderName,
	}

	return response, nil
}

// OpenRouterRequest represents a request to the OpenRouter API.
type OpenRouterRequest struct {
	Model            string                `json:"model"`
	Messages         []OpenRouterMessage   `json:"messages"`
	MaxTokens        int                   `json:"max_tokens,omitempty"`
	Temperature      float64               `json:"temperature,omitempty"`
	TopP             float64               `json:"top_p,omitempty"`
	FrequencyPenalty float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64               `json:"presence_penalty,omitempty"`
	Stream           bool                  `json:"stream,omitempty"`
}

// OpenRouterMessage represents a message in an OpenRouter request.
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents a response from the OpenRouter API.
type OpenRouterResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenRouterChoice    `json:"choices"`
	Usage   OpenRouterUsage       `json:"usage"`
}

// OpenRouterChoice represents a choice in an OpenRouter response.
type OpenRouterChoice struct {
	Index        int                   `json:"index"`
	Message      OpenRouterMessage     `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

// OpenRouterUsage represents usage information in an OpenRouter response.
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterStreamResponse represents a streaming response from OpenRouter.
type OpenRouterStreamResponse struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []OpenRouterStreamChoice    `json:"choices"`
}

// OpenRouterStreamChoice represents a streaming choice.
type OpenRouterStreamChoice struct {
	Index        int                       `json:"index"`
	Delta        OpenRouterStreamDelta     `json:"delta"`
	FinishReason string                    `json:"finish_reason"`
}

// OpenRouterStreamDelta represents a streaming delta.
type OpenRouterStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// OpenRouterModelsResponse represents the models list response.
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// OpenRouterModel represents a model in the OpenRouter models list.
type OpenRouterModel struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	ContextLength int                    `json:"context_length"`
	Pricing       OpenRouterModelPricing `json:"pricing"`
}

// OpenRouterModelPricing represents pricing information for a model.
type OpenRouterModelPricing struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
}
