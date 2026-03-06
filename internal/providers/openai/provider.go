// Package openai provides an OpenAI LLM provider implementation.
//
// This package implements the Provider interface for OpenAI's GPT models,
// supporting chat completions, streaming responses, function calling, and
// vision capabilities.
//
// Example usage:
//
//	config := openai.Config{
//		APIKey:  "sk-...",
//		BaseURL: "https://api.openai.com/v1", // optional
//	}
//
//	provider, err := openai.New(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	resp, err := provider.CreateCompletion(ctx, &core.CompletionRequest{
//		Model:    "gpt-3.5-turbo",
//		Messages: []core.Message{{Role: "user", Content: "Hello"}},
//	})
package openai

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
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultTimeout = 30 * time.Second
	UserAgent      = "gollm/1.0"
)

// Provider implements the core.Provider interface for OpenAI.
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

// Config configures the OpenAI provider.
type Config struct {
	APIKey       string        `json:"api_key" validate:"required"`
	BaseURL      string        `json:"base_url,omitempty"`
	Organization string        `json:"organization,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	MaxRetries   int           `json:"max_retries,omitempty"`

	// HTTP client settings
	UserAgent     string            `json:"user_agent,omitempty"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// Features
	EnableFunctionCalling bool `json:"enable_function_calling,omitempty"`
	EnableVision          bool `json:"enable_vision,omitempty"`
}

// DefaultConfig returns sensible defaults for the OpenAI provider.
func DefaultConfig() Config {
	return Config{
		BaseURL:               DefaultBaseURL,
		Timeout:               DefaultTimeout,
		MaxRetries:            3,
		UserAgent:             UserAgent,
		EnableFunctionCalling: true,
		EnableVision:          true,
	}
}

// New creates a new OpenAI provider with the given configuration.
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

	// Validate required fields
	if config.APIKey == "" {
		return nil, &core.ValidationError{
			Field:   "api_key",
			Value:   config.APIKey,
			Rule:    "required",
			Message: "OpenAI API key is required",
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

// NewFromConfig creates a new OpenAI provider from core configuration.
// This is used by the provider registry for dynamic provider creation.
func NewFromConfig(config core.ProviderConfig) (core.Provider, error) {
	// Convert core config to OpenAI-specific config
	openaiConfig := Config{
		APIKey:       config.APIKey,
		BaseURL:      config.BaseURL,
		Organization: config.Organization,
		Timeout:      config.Timeout,
		MaxRetries:   config.MaxRetries,
	}

	// Apply custom headers if provided
	if config.CustomHeaders != nil {
		openaiConfig.CustomHeaders = config.CustomHeaders
	}

	// Parse provider-specific extra settings
	if config.Extra != nil {
		if userAgent, ok := config.Extra["user_agent"].(string); ok {
			openaiConfig.UserAgent = userAgent
		}
		if enableFunctions, ok := config.Extra["enable_function_calling"].(bool); ok {
			openaiConfig.EnableFunctionCalling = enableFunctions
		}
		if enableVision, ok := config.Extra["enable_vision"].(bool); ok {
			openaiConfig.EnableVision = enableVision
		}
	}

	return New(openaiConfig)
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai"
}

// CreateCompletion creates a chat completion synchronously.
func (p *Provider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	start := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to OpenAI format
	openaiReq, err := p.convertRequest(req)
	if err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API request
	var openaiResp openaiCompletionResponse
	if err := p.makeRequest(ctx, "POST", "/chat/completions", openaiReq, &openaiResp); err != nil {
		p.metrics.RecordRequest(time.Since(start), nil, nil, err)
		return nil, err
	}

	// Convert response
	response, err := p.convertResponse(&openaiResp, req)
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

	// Convert to OpenAI format with streaming enabled
	openaiReq, err := p.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}
	openaiReq.Stream = true

	// Create response channel
	chunks := make(chan core.StreamChunk, 10)

	go func() {
		defer close(chunks)

		start := time.Now()

		if err := p.streamRequest(ctx, openaiReq, chunks, req); err != nil {
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

	// Fetch models from API
	var openaiResp openaiModelsResponse
	if err := p.makeRequest(ctx, "GET", "/models", nil, &openaiResp); err != nil {
		return nil, err
	}

	// Convert to core models
	models := make([]core.Model, 0, len(openaiResp.Data))
	for _, openaiModel := range openaiResp.Data {
		model := p.convertModel(&openaiModel)
		models = append(models, model)
	}

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

// makeRequest makes an HTTP request to the OpenAI API.
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
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", p.config.UserAgent)

	if p.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", p.config.Organization)
	}

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

// handleErrorResponse handles error responses from the OpenAI API.
func (p *Provider) handleErrorResponse(resp *http.Response) error {
	var errorResp openaiErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		return &core.APIError{
			StatusCode: resp.StatusCode,
			Code:       "unknown_error",
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			Provider:   "openai",
		}
	}

	return &core.APIError{
		StatusCode: resp.StatusCode,
		Code:       errorResp.Error.Code,
		Message:    errorResp.Error.Message,
		Type:       errorResp.Error.Type,
		Provider:   "openai",
	}
}

// streamRequest handles streaming requests.
func (p *Provider) streamRequest(ctx context.Context, req *openaiCompletionRequest, chunks chan<- core.StreamChunk, originalReq *core.CompletionRequest) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("User-Agent", p.config.UserAgent)
	httpReq.Header.Set("Accept", "text/event-stream")

	if p.config.Organization != "" {
		httpReq.Header.Set("OpenAI-Organization", p.config.Organization)
	}

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

		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // Skip malformed chunks
		}

		coreChunk := p.convertStreamChunk(&chunk, originalReq)

		// Accumulate usage if present
		if chunk.Usage != nil {
			totalUsage = core.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		select {
		case chunks <- coreChunk:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Send final chunk with usage
	if totalUsage.TotalTokens > 0 {
		cost := p.calculateCost(originalReq.Model, &totalUsage)
		responseTime := time.Since(start)

		finalChunk := core.StreamChunk{
			Usage:     &totalUsage,
			Done:      true,
			Provider:  "openai",
			RequestID: originalReq.RequestID,
			Timestamp: time.Now(),
		}

		chunks <- finalChunk
		p.metrics.RecordRequest(responseTime, &totalUsage, &cost, nil)
	}

	return scanner.Err()
}

// convertRequest converts a core request to OpenAI format.
func (p *Provider) convertRequest(req *core.CompletionRequest) (*openaiCompletionRequest, error) {
	openaiReq := &openaiCompletionRequest{
		Model:    req.Model,
		Messages: make([]openaiMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Convert messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openaiMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			openaiReq.Messages[i].ToolCalls = make([]openaiToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openaiReq.Messages[i].ToolCalls[j] = openaiToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openaiFunction{
						Name:      tc.Function.Name,
						Arguments: string(tc.Function.Arguments),
					},
				}
			}
		}

		if msg.ToolCallID != "" {
			openaiReq.Messages[i].ToolCallID = msg.ToolCallID
		}
	}

	// Set optional parameters
	if req.MaxTokens != nil {
		openaiReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		openaiReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		openaiReq.TopP = req.TopP
	}
	if req.FrequencyPenalty != nil {
		openaiReq.FrequencyPenalty = req.FrequencyPenalty
	}
	if req.PresencePenalty != nil {
		openaiReq.PresencePenalty = req.PresencePenalty
	}
	if len(req.Stop) > 0 {
		openaiReq.Stop = req.Stop
	}
	if req.User != "" {
		openaiReq.User = req.User
	}

	// Convert tools
	if len(req.Tools) > 0 && p.config.EnableFunctionCalling {
		openaiReq.Tools = make([]openaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openaiTool{
				Type: tool.Type,
				Function: openaiToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}

		if req.ToolChoice != nil {
			openaiReq.ToolChoice = req.ToolChoice
		}
	}

	return openaiReq, nil
}

// convertResponse converts an OpenAI response to core format.
func (p *Provider) convertResponse(resp *openaiCompletionResponse, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	coreResp := &core.CompletionResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: make([]core.Choice, len(resp.Choices)),
		Usage: core.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Provider:  "openai",
		RequestID: req.RequestID,
	}

	// Convert choices
	for i, choice := range resp.Choices {
		coreResp.Choices[i] = core.Choice{
			Index:        choice.Index,
			FinishReason: choice.FinishReason,
			Message: core.Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
				Name:    choice.Message.Name,
			},
		}

		// Convert tool calls if present
		if len(choice.Message.ToolCalls) > 0 {
			coreResp.Choices[i].Message.ToolCalls = make([]core.ToolCall, len(choice.Message.ToolCalls))
			for j, tc := range choice.Message.ToolCalls {
				coreResp.Choices[i].Message.ToolCalls[j] = core.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: core.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: json.RawMessage(tc.Function.Arguments),
					},
				}
			}
		}
	}

	return coreResp, nil
}

// convertStreamChunk converts an OpenAI stream chunk to core format.
func (p *Provider) convertStreamChunk(chunk *openaiStreamChunk, req *core.CompletionRequest) core.StreamChunk {
	coreChunk := core.StreamChunk{
		ID:        chunk.ID,
		Object:    chunk.Object,
		Created:   chunk.Created,
		Model:     chunk.Model,
		Provider:  "openai",
		RequestID: req.RequestID,
		Timestamp: time.Now(),
	}

	if len(chunk.Choices) > 0 {
		coreChunk.Choices = make([]core.Choice, len(chunk.Choices))
		for i, choice := range chunk.Choices {
			coreChunk.Choices[i] = core.Choice{
				Index:        choice.Index,
				FinishReason: choice.FinishReason,
			}

			if choice.Delta != nil {
				coreChunk.Choices[i].Delta = &core.Message{
					Role:    choice.Delta.Role,
					Content: choice.Delta.Content,
					Name:    choice.Delta.Name,
				}

				// Convert delta tool calls if present
				if len(choice.Delta.ToolCalls) > 0 {
					coreChunk.Choices[i].Delta.ToolCalls = make([]core.ToolCall, len(choice.Delta.ToolCalls))
					for j, tc := range choice.Delta.ToolCalls {
						coreChunk.Choices[i].Delta.ToolCalls[j] = core.ToolCall{
							ID:   tc.ID,
							Type: tc.Type,
							Function: core.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: json.RawMessage(tc.Function.Arguments),
							},
						}
					}
				}
			}
		}
	}

	if chunk.Usage != nil {
		usage := &core.Usage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
		coreChunk.Usage = usage
	}

	return coreChunk
}

// convertModel converts an OpenAI model to core format.
func (p *Provider) convertModel(model *openaiModel) core.Model {
	coreModel := core.Model{
		ID:       model.ID,
		Object:   model.Object,
		Created:  model.Created,
		OwnedBy:  model.OwnedBy,
		Provider: "openai",
	}

	// Set capabilities based on model ID
	if strings.Contains(model.ID, "gpt-4") {
		maxTokens := 8192
		if strings.Contains(model.ID, "32k") {
			maxTokens = 32768
		} else if strings.Contains(model.ID, "turbo") {
			maxTokens = 4096
		}
		coreModel.MaxTokens = &maxTokens
		coreModel.SupportsFunctions = p.config.EnableFunctionCalling
		coreModel.SupportsStreaming = true
		coreModel.SupportsVision = p.config.EnableVision && strings.Contains(model.ID, "vision")
	} else if strings.Contains(model.ID, "gpt-3.5") {
		maxTokens := 4096
		if strings.Contains(model.ID, "16k") {
			maxTokens = 16384
		}
		coreModel.MaxTokens = &maxTokens
		coreModel.SupportsFunctions = p.config.EnableFunctionCalling
		coreModel.SupportsStreaming = true
	}

	// Set pricing (approximate values)
	p.setPricing(&coreModel, model.ID)

	coreModel.Description = fmt.Sprintf("OpenAI %s model", model.ID)
	coreModel.Tags = []string{"openai", "chat"}

	return coreModel
}

// setPricing sets approximate pricing for models.
func (p *Provider) setPricing(model *core.Model, modelID string) {
	switch {
	case strings.Contains(modelID, "gpt-4-turbo"):
		input := 0.01
		output := 0.03
		model.InputCostPer1K = &input
		model.OutputCostPer1K = &output
	case strings.Contains(modelID, "gpt-4"):
		input := 0.03
		output := 0.06
		model.InputCostPer1K = &input
		model.OutputCostPer1K = &output
	case strings.Contains(modelID, "gpt-3.5-turbo"):
		input := 0.0015
		output := 0.002
		model.InputCostPer1K = &input
		model.OutputCostPer1K = &output
	}
}

// calculateCost calculates the cost for a completion.
func (p *Provider) calculateCost(modelID string, usage *core.Usage) float64 {
	var inputRate, outputRate float64

	switch {
	case strings.Contains(modelID, "gpt-4-turbo"):
		inputRate, outputRate = 0.01, 0.03
	case strings.Contains(modelID, "gpt-4"):
		inputRate, outputRate = 0.03, 0.06
	case strings.Contains(modelID, "gpt-3.5-turbo"):
		inputRate, outputRate = 0.0015, 0.002
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

// OpenAI API types

type openaiCompletionRequest struct {
	Model            string          `json:"model"`
	Messages         []openaiMessage `json:"messages"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Stop             []string        `json:"stop,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	Tools            []openaiTool    `json:"tools,omitempty"`
	ToolChoice       *string         `json:"tool_choice,omitempty"`
	User             string          `json:"user,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type openaiCompletionResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *openaiUsage         `json:"usage,omitempty"`
}

type openaiStreamChoice struct {
	Index        int            `json:"index"`
	Delta        *openaiMessage `json:"delta,omitempty"`
	FinishReason string         `json:"finish_reason,omitempty"`
}

type openaiModelsResponse struct {
	Object string        `json:"object"`
	Data   []openaiModel `json:"data"`
}

type openaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// init registers the OpenAI provider factory with the global registry.
func init() {
	core.RegisterProviderFactory("openai", NewFromConfig)
}
