// Package core defines the fundamental types and interfaces for the GOLLM system.
//
// This package provides the core abstractions that all other components build upon,
// including provider interfaces, request/response types, and shared data structures.
//
// Example usage:
//
//	provider := openai.New(openai.Config{
//		APIKey: "sk-...",
//	})
//
//	req := &core.CompletionRequest{
//		Model: "gpt-3.5-turbo",
//		Messages: []core.Message{
//			{Role: "user", Content: "Hello, world!"},
//		},
//	}
//
//	resp, err := provider.CreateCompletion(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Common error types used throughout the system.
var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidModel      = errors.New("invalid model specified")
	ErrInvalidProvider   = errors.New("invalid provider specified")
	ErrRateLimited       = errors.New("rate limit exceeded")
	ErrQuotaExceeded     = errors.New("quota exceeded")
	ErrAuthentication    = errors.New("authentication failed")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrProviderUnavail   = errors.New("provider unavailable")
	ErrTimeout           = errors.New("request timeout")
	ErrContextCanceled   = errors.New("context canceled")
	ErrInvalidResponse   = errors.New("invalid response from provider")
	ErrStreamClosed      = errors.New("stream closed")
)

// ValidationError represents a validation failure with detailed context.
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Rule    string      `json:"rule"`
	Message string      `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s (value: %v, rule: %s): %s",
		e.Field, e.Value, e.Rule, e.Message)
}

// NewValidationError creates a new ValidationError with the given parameters.
func NewValidationError(field string, value interface{}, rule string, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// APIError represents an error from an LLM provider API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	Provider   string `json:"provider"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error from %s (%d/%s): %s", e.Provider, e.StatusCode, e.Code, e.Message)
}

// Provider defines the interface that all LLM providers must implement.
//
// Providers handle the actual communication with LLM APIs, including
// authentication, request formatting, and response parsing.
type Provider interface {
	// Name returns the provider's identifier (e.g., "openai", "anthropic").
	Name() string

	// CreateCompletion creates a chat completion synchronously.
	// Returns the complete response or an error.
	CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// StreamCompletion creates a streaming chat completion.
	// Returns a channel that emits completion chunks as they arrive.
	StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)

	// GetModels returns a list of available models for this provider.
	GetModels(ctx context.Context) ([]Model, error)

	// ValidateConfig validates the provider's configuration.
	ValidateConfig() error
}

// Streamer defines the interface for providers that support streaming.
type Streamer interface {
	Provider
	StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error)
}

// ModelLister defines the interface for providers that can list models.
type ModelLister interface {
	Provider
	GetModels(ctx context.Context) ([]Model, error)
}

// Message represents a single message in a conversation.
type Message struct {
	Role       string                 `json:"role" validate:"required,oneof=system user assistant tool"`
	Content    string                 `json:"content,omitempty"`
	Name       string                 `json:"name,omitempty"`
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// String implements fmt.Stringer for debugging (content may be truncated).
func (m *Message) String() string {
	content := m.Content
	if len(content) > 100 {
		content = content[:97] + "..."
	}
	return fmt.Sprintf("Message{Role: %s, Content: %q}", m.Role, content)
}

// ToolCall represents a function/tool call in a message.
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type" validate:"required"`
	Function FunctionCall    `json:"function,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// FunctionCall represents a function call within a tool call.
type FunctionCall struct {
	Name      string          `json:"name" validate:"required"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CompletionRequest represents a request for chat completion.
type CompletionRequest struct {
	// Required fields
	Model    string    `json:"model" validate:"required,min=1,max=100"`
	Messages []Message `json:"messages" validate:"required,min=1,max=100"`

	// Optional completion parameters
	MaxTokens        *int             `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=32768"`
	Temperature      *float64         `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	TopP             *float64         `json:"top_p,omitempty" validate:"omitempty,min=0,max=1"`
	FrequencyPenalty *float64         `json:"frequency_penalty,omitempty" validate:"omitempty,min=-2,max=2"`
	PresencePenalty  *float64         `json:"presence_penalty,omitempty" validate:"omitempty,min=-2,max=2"`
	Stop             []string         `json:"stop,omitempty" validate:"omitempty,max=4"`

	// Streaming and tools
	Stream    bool       `json:"stream,omitempty"`
	Tools     []Tool     `json:"tools,omitempty"`
	ToolChoice *string   `json:"tool_choice,omitempty"`

	// System and user identification
	SystemMessage *string `json:"system_message,omitempty"`
	User          string  `json:"user,omitempty" validate:"omitempty,max=256"`

	// Request metadata
	RequestID string            `json:"request_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// Validate validates the completion request.
func (r *CompletionRequest) Validate() error {
	if r.Model == "" {
		return &ValidationError{
			Field:   "model",
			Value:   r.Model,
			Rule:    "required",
			Message: "model cannot be empty",
		}
	}

	if len(r.Messages) == 0 {
		return &ValidationError{
			Field:   "messages",
			Value:   len(r.Messages),
			Rule:    "min=1",
			Message: "at least one message is required",
		}
	}

	// Validate temperature range
	if r.Temperature != nil && (*r.Temperature < 0 || *r.Temperature > 2) {
		return &ValidationError{
			Field:   "temperature",
			Value:   *r.Temperature,
			Rule:    "range=0-2",
			Message: "temperature must be between 0 and 2",
		}
	}

	// Validate each message
	for i, msg := range r.Messages {
		if msg.Role == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d].role", i),
				Value:   msg.Role,
				Rule:    "required",
				Message: "message role cannot be empty",
			}
		}
		if msg.Role != "system" && msg.Role != "user" && msg.Role != "assistant" && msg.Role != "tool" {
			return &ValidationError{
				Field:   fmt.Sprintf("messages[%d].role", i),
				Value:   msg.Role,
				Rule:    "oneof",
				Message: "role must be one of: system, user, assistant, tool",
			}
		}
	}

	return nil
}

// Tool represents a function/tool that the model can call.
type Tool struct {
	Type     string       `json:"type" validate:"required"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function that can be called by the model.
type ToolFunction struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// CompletionResponse represents a response from the chat completion API.
type CompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`

	// Provider-specific metadata
	Provider     string            `json:"provider,omitempty"`
	RequestID    string            `json:"request_id,omitempty"`
	ResponseTime time.Duration     `json:"response_time,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ChatRequest represents a chat completion request (alias for CompletionRequest).
type ChatRequest = CompletionRequest

// ChatResponse represents a chat completion response with simplified fields for CLI usage.
type ChatResponse struct {
	Content      string `json:"content"`
	TokensUsed   int    `json:"tokens_used"`
	FinishReason string `json:"finish_reason"`
	Model        string `json:"model,omitempty"`
	Provider     string `json:"provider,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason string  `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
}

// Logprobs represents token log probabilities.
type Logprobs struct {
	Tokens   []string             `json:"tokens,omitempty"`
	TokenLogprobs []float64       `json:"token_logprobs,omitempty"`
	TopLogprobs   []map[string]float64 `json:"top_logprobs,omitempty"`
}

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// Cost tracking (if available)
	PromptCost     *float64 `json:"prompt_cost,omitempty"`
	CompletionCost *float64 `json:"completion_cost,omitempty"`
	TotalCost      *float64 `json:"total_cost,omitempty"`
}

// StreamChunk represents a chunk in a streaming response.
type StreamChunk struct {
	ID      string   `json:"id,omitempty"`
	Object  string   `json:"object,omitempty"`
	Created int64    `json:"created,omitempty"`
	Model   string   `json:"model,omitempty"`
	Choices []Choice `json:"choices,omitempty"`
	Usage   *Usage   `json:"usage,omitempty"`

	// Error handling for streams
	Error error `json:"error,omitempty"`
	Done  bool  `json:"done,omitempty"`

	// Stream metadata
	Provider  string        `json:"provider,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
	Timestamp time.Time     `json:"timestamp,omitempty"`
}

// Model represents information about an available model.
type Model struct {
	ID          string            `json:"id"`
	Object      string            `json:"object,omitempty"`
	Created     int64             `json:"created,omitempty"`
	OwnedBy     string            `json:"owned_by,omitempty"`
	Provider    string            `json:"provider"`

	// Model capabilities
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	SupportsFunctions bool     `json:"supports_functions"`
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsVision    bool     `json:"supports_vision,omitempty"`

	// Pricing information (if available)
	InputCostPer1K  *float64 `json:"input_cost_per_1k,omitempty"`
	OutputCostPer1K *float64 `json:"output_cost_per_1k,omitempty"`

	// Additional metadata
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// String implements fmt.Stringer for debugging.
func (m *Model) String() string {
	var maxTokens interface{}
	if m.MaxTokens != nil {
		maxTokens = *m.MaxTokens
	} else {
		maxTokens = nil
	}
	return fmt.Sprintf("Model{ID: %s, Provider: %s, MaxTokens: %v}",
		m.ID, m.Provider, maxTokens)
}

// StreamReader provides a convenient interface for reading streaming responses.
type StreamReader struct {
	chunks <-chan StreamChunk
	err    error
	closed bool
}

// NewStreamReader creates a new stream reader from a chunk channel.
func NewStreamReader(chunks <-chan StreamChunk) *StreamReader {
	return &StreamReader{chunks: chunks}
}

// Read reads the next chunk from the stream.
// Returns io.EOF when the stream is closed.
func (r *StreamReader) Read() (StreamChunk, error) {
	if r.closed {
		return StreamChunk{}, io.EOF
	}

	chunk, ok := <-r.chunks
	if !ok {
		r.closed = true
		return StreamChunk{}, io.EOF
	}

	if chunk.Error != nil {
		r.err = chunk.Error
		return chunk, chunk.Error
	}

	if chunk.Done {
		r.closed = true
	}

	return chunk, nil
}

// Close closes the stream reader.
func (r *StreamReader) Close() error {
	r.closed = true
	return nil
}

// Err returns any error that occurred during streaming.
func (r *StreamReader) Err() error {
	return r.err
}

// ProviderConfig represents configuration for a single provider.
type ProviderConfig struct {
	Type            string            `json:"type" validate:"required"`
	APIKey          string            `json:"api_key,omitempty"`
	BaseURL         string            `json:"base_url,omitempty"`
	Organization    string            `json:"organization,omitempty"`
	MaxRetries      int               `json:"max_retries,omitempty"`
	Timeout         time.Duration     `json:"timeout,omitempty"`
	RateLimit       string            `json:"rate_limit,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	TLSVerify       *bool             `json:"tls_verify,omitempty"`

	// Provider-specific settings
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Validate validates the provider configuration.
func (c *ProviderConfig) Validate() error {
	if c.Type == "" {
		return &ValidationError{
			Field:   "type",
			Value:   c.Type,
			Rule:    "required",
			Message: "provider type cannot be empty",
		}
	}

	return nil
}

// RequestMetadata contains metadata about a request.
type RequestMetadata struct {
	RequestID   string            `json:"request_id"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
	StartTime   time.Time         `json:"start_time"`
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
}

// ResponseMetadata contains metadata about a response.
type ResponseMetadata struct {
	RequestID    string        `json:"request_id"`
	ResponseTime time.Duration `json:"response_time"`
	TokensUsed   int           `json:"tokens_used"`
	Cost         *float64      `json:"cost,omitempty"`
	Cached       bool          `json:"cached,omitempty"`
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	EndTime      time.Time     `json:"end_time"`
}

// String returns a string representation of the Choice.
func (c *Choice) String() string {
	result := fmt.Sprintf("Choice{Index: %d", c.Index)

	if c.Message.Role != "" || c.Message.Content != "" {
		result += fmt.Sprintf(", Message: %s", c.Message.String())
	}

	if c.Delta != nil {
		result += fmt.Sprintf(", Delta: %s", c.Delta.String())
	}

	result += fmt.Sprintf(", FinishReason: %s}", c.FinishReason)
	return result
}

// String returns a string representation of the CompletionResponse.
func (r *CompletionResponse) String() string {
	return fmt.Sprintf("CompletionResponse{ID: %s, Model: %s, Choices: %d}",
		r.ID, r.Model, len(r.Choices))
}

// String returns a string representation of the StreamChunk.
func (s *StreamChunk) String() string {
	result := fmt.Sprintf("StreamChunk{ID: %s, Model: %s, Choices: %d, Done: %t",
		s.ID, s.Model, len(s.Choices), s.Done)

	if s.Error != nil {
		result += fmt.Sprintf(", Error: %s", s.Error.Error())
	}

	result += "}"
	return result
}

// Validate validates the ToolCall.
func (t *ToolCall) Validate() error {
	if t.ID == "" {
		return NewValidationError("id", t.ID, "required", "tool call ID cannot be empty")
	}

	if t.Type != ToolTypeFunction {
		return NewValidationError("type", t.Type, "invalid", "tool call type must be 'function'")
	}

	return t.Function.Validate()
}

// Validate validates the FunctionCall.
func (f *FunctionCall) Validate() error {
	if f.Name == "" {
		return NewValidationError("name", f.Name, "required", "function name cannot be empty")
	}

	if len(f.Arguments) > 0 {
		var temp interface{}
		if err := json.Unmarshal(f.Arguments, &temp); err != nil {
			return NewValidationError("arguments", string(f.Arguments), "invalid_json", "arguments must be valid JSON")
		}
	}

	return nil
}

// Validate validates the Message.
func (m *Message) Validate() error {
	validRoles := map[string]bool{
		RoleSystem:    true,
		RoleUser:      true,
		RoleAssistant: true,
		RoleTool:      true,
	}

	if m.Role == "" {
		return NewValidationError("role", m.Role, "required", "message role cannot be empty")
	}

	if !validRoles[m.Role] {
		return NewValidationError("role", m.Role, "invalid", "message role must be one of: system, user, assistant, tool")
	}

	if m.Role == RoleTool && m.ToolCallID == "" {
		return NewValidationError("tool_call_id", m.ToolCallID, "required", "tool messages must have a tool_call_id")
	}

	return nil
}

// String returns a string representation of the Usage.
func (u *Usage) String() string {
	cost := "N/A"
	if u.TotalCost != nil {
		cost = fmt.Sprintf("$%.3f", *u.TotalCost)
	}

	return fmt.Sprintf("Usage{Prompt: %d, Completion: %d, Total: %d, Cost: %s}",
		u.PromptTokens, u.CompletionTokens, u.TotalTokens, cost)
}

const (
	// Message roles
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"

	// Finish reasons
	FinishReasonStop         = "stop"
	FinishReasonLength       = "length"
	FinishReasonToolCalls    = "tool_calls"
	FinishReasonContentFilter = "content_filter"
	FinishReasonTimeout      = "timeout"

	// Tool types
	ToolTypeFunction = "function"

	// Common model IDs (for reference)
	ModelGPT35Turbo = "gpt-3.5-turbo"
	ModelGPT4       = "gpt-4"
	ModelGPT4Turbo  = "gpt-4-turbo"
	ModelClaude3    = "claude-3-sonnet"
)

// DefaultSettings provides sensible default values.
var DefaultSettings = struct {
	MaxTokens        int
	Temperature      float64
	TopP             float64
	FrequencyPenalty float64
	PresencePenalty  float64
	Timeout          time.Duration
	MaxRetries       int
}{
	MaxTokens:        2048,
	Temperature:      0.7,
	TopP:             1.0,
	FrequencyPenalty: 0.0,
	PresencePenalty:  0.0,
	Timeout:          30 * time.Second,
	MaxRetries:       3,
}
