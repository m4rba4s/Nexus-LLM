// Package tui provides AI service integration for simple chat mode
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"

	// Import providers to trigger their init() registration
	_ "github.com/yourusername/gollm/internal/providers/anthropic"
	_ "github.com/yourusername/gollm/internal/providers/deepseek"
	_ "github.com/yourusername/gollm/internal/providers/gemini"
	_ "github.com/yourusername/gollm/internal/providers/mock"
	_ "github.com/yourusername/gollm/internal/providers/openai"
	_ "github.com/yourusername/gollm/internal/providers/openrouter"
)

// AIService provides AI integration for the simple chat interface
type AIService struct {
	config       *config.Config
	tuiConfig    *Config
	provider     core.Provider
	providerName string
	model        string
	conversation []core.Message
	totalTokens  int
	requestCount int
}

// AIResponse represents a response from the AI service
type AIResponse struct {
	Content   string
	Tokens    int
	Latency   time.Duration
	Error     error
	IsError   bool
	Streaming bool
}

// NewAIService creates a new AI service instance
func NewAIService(cfg *config.Config, tuiConfig *Config) (*AIService, error) {
	service := &AIService{
		config:       cfg,
		tuiConfig:    tuiConfig,
		conversation: []core.Message{},
		totalTokens:  0,
		requestCount: 0,
	}

	// Initialize provider
	if err := service.initializeProvider(); err != nil {
		return nil, fmt.Errorf("failed to initialize provider: %w", err)
	}

	return service, nil
}

// initializeProvider sets up the AI provider based on configuration
func (a *AIService) initializeProvider() error {
	// Determine provider name
	providerName := a.tuiConfig.Provider
	if providerName == "" {
		providerName = a.config.DefaultProvider
	}
	if providerName == "" {
		return fmt.Errorf("no provider specified and no default configured")
	}

	// Validate provider exists in config
	if !a.config.HasProvider(providerName) {
		available := a.config.ListProviders()
		return fmt.Errorf("provider %q not configured; available providers: %s",
			providerName, strings.Join(available, ", "))
	}

	// Get provider configuration
	providerConfig, _, err := a.config.GetProvider(providerName)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	// Determine model
	model := a.tuiConfig.Model
	if model == "" && providerConfig.DefaultModel != "" {
		model = providerConfig.DefaultModel
	}
	if model == "" {
		return fmt.Errorf("no model specified and no default configured for provider %s", providerName)
	}

	// Create provider instance
	coreConfig := providerConfig.ToProviderConfig()
	provider, err := core.CreateProviderFromConfig(providerConfig.Type, coreConfig)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Validate provider
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("provider configuration invalid: %w", err)
	}

	a.provider = provider
	a.providerName = providerName
	a.model = model

	return nil
}

// SendMessage sends a message to the AI and returns the response
func (a *AIService) SendMessage(message string) (*AIResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	startTime := time.Now()

	// Add user message to conversation
	userMsg := core.Message{
		Role:    core.RoleUser,
		Content: message,
	}
	a.conversation = append(a.conversation, userMsg)

	// Build request
	request := &core.CompletionRequest{
		Model:     a.model,
		Messages:  a.conversation,
		CreatedAt: time.Now(),
		Stream:    false, // Use non-streaming for simple chat
	}

	// Apply default settings from config
	if a.config.Settings.Temperature > 0 {
		temp := a.config.Settings.Temperature
		request.Temperature = &temp
	}
	if a.config.Settings.MaxTokens > 0 {
		tokens := a.config.Settings.MaxTokens
		request.MaxTokens = &tokens
	}

	// Create context with timeout
	timeout := a.config.Settings.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Send request
	response, err := a.provider.CreateCompletion(ctx, request)
	if err != nil {
		return &AIResponse{
			Content: "",
			Error:   err,
			IsError: true,
			Latency: time.Since(startTime),
		}, nil
	}

	// Extract response content
	var content string
	if len(response.Choices) > 0 {
		content = response.Choices[0].Message.Content

		// Add AI response to conversation
		aiMsg := core.Message{
			Role:    core.RoleAssistant,
			Content: content,
		}
		a.conversation = append(a.conversation, aiMsg)
	}

	// Update statistics
	if response.Usage.TotalTokens > 0 {
		a.totalTokens += response.Usage.TotalTokens
	}
	a.requestCount++

	return &AIResponse{
		Content:   content,
		Tokens:    response.Usage.TotalTokens,
		Latency:   time.Since(startTime),
		Error:     nil,
		IsError:   false,
		Streaming: false,
	}, nil
}

// SendMessageStreaming sends a message with streaming response
func (a *AIService) SendMessageStreaming(message string, callback func(chunk string)) (*AIResponse, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Check if provider supports streaming
	streamer, supportsStreaming := a.provider.(core.Streamer)
	if !supportsStreaming {
		// Fall back to non-streaming
		return a.SendMessage(message)
	}

	startTime := time.Now()

	// Add user message to conversation
	userMsg := core.Message{
		Role:    core.RoleUser,
		Content: message,
	}
	a.conversation = append(a.conversation, userMsg)

	// Build streaming request
	request := &core.CompletionRequest{
		Model:     a.model,
		Messages:  a.conversation,
		CreatedAt: time.Now(),
		Stream:    true,
	}

	// Apply default settings
	if a.config.Settings.Temperature > 0 {
		temp := a.config.Settings.Temperature
		request.Temperature = &temp
	}
	if a.config.Settings.MaxTokens > 0 {
		tokens := a.config.Settings.MaxTokens
		request.MaxTokens = &tokens
	}

	// Create context with timeout
	timeout := a.config.Settings.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // Longer timeout for streaming
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Start streaming
	chunks, err := streamer.StreamCompletion(ctx, request)
	if err != nil {
		return &AIResponse{
			Content: "",
			Error:   err,
			IsError: true,
			Latency: time.Since(startTime),
		}, nil
	}

	// Process streaming chunks
	var fullContent strings.Builder
	var totalUsage *core.Usage

	for chunk := range chunks {
		if chunk.Error != nil {
			return &AIResponse{
				Content: fullContent.String(),
				Error:   chunk.Error,
				IsError: true,
				Latency: time.Since(startTime),
			}, nil
		}

		// Extract content from chunk
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fullContent.WriteString(content)
				if callback != nil {
					callback(content)
				}
			}
		}

		// Store usage information from final chunk
		if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
			totalUsage = chunk.Usage
		}

		if chunk.Done {
			break
		}
	}

	// Add AI response to conversation
	content := fullContent.String()
	if content != "" {
		aiMsg := core.Message{
			Role:    core.RoleAssistant,
			Content: content,
		}
		a.conversation = append(a.conversation, aiMsg)
	}

	// Update statistics
	if totalUsage != nil && totalUsage.TotalTokens > 0 {
		a.totalTokens += totalUsage.TotalTokens
	}
	a.requestCount++

	return &AIResponse{
		Content: content,
		Tokens: func() int {
			if totalUsage != nil {
				return totalUsage.TotalTokens
			}
			return 0
		}(),
		Latency:   time.Since(startTime),
		Error:     nil,
		IsError:   false,
		Streaming: true,
	}, nil
}

// GetProviderInfo returns information about the current provider
func (a *AIService) GetProviderInfo() (string, string, error) {
	if a.provider == nil {
		return "", "", fmt.Errorf("provider not initialized")
	}
	return a.providerName, a.model, nil
}

// GetStats returns usage statistics
func (a *AIService) GetStats() (int, int) {
	return a.totalTokens, a.requestCount
}

// ClearConversation clears the conversation history
func (a *AIService) ClearConversation() {
	a.conversation = []core.Message{}
}

// SetSystemMessage sets a system message for the conversation
func (a *AIService) SetSystemMessage(message string) {
	// Remove existing system message if any
	if len(a.conversation) > 0 && a.conversation[0].Role == core.RoleSystem {
		a.conversation = a.conversation[1:]
	}

	// Add new system message at the beginning
	if message != "" {
		systemMsg := core.Message{
			Role:    core.RoleSystem,
			Content: message,
		}
		a.conversation = append([]core.Message{systemMsg}, a.conversation...)
	}
}

// GetConversationLength returns the number of messages in conversation
func (a *AIService) GetConversationLength() int {
	return len(a.conversation)
}

// TestConnection tests the connection to the AI provider
func (a *AIService) TestConnection() error {
	if a.provider == nil {
		return fmt.Errorf("provider not initialized")
	}

	// Send a simple test message
	testMessage := "Hello, this is a connection test."
	response, err := a.SendMessage(testMessage)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	if response.IsError {
		return fmt.Errorf("connection test failed: %s", response.Error.Error())
	}

	// Remove test messages from conversation to keep it clean
	if len(a.conversation) >= 2 {
		a.conversation = a.conversation[:len(a.conversation)-2]
	}

	return nil
}

// ValidateConfig validates the AI service configuration
func (a *AIService) ValidateConfig() error {
	if a.config == nil {
		return fmt.Errorf("config is nil")
	}

	if a.tuiConfig == nil {
		return fmt.Errorf("tuiConfig is nil")
	}

	if a.provider == nil {
		return fmt.Errorf("provider not initialized")
	}

	return a.provider.ValidateConfig()
}
