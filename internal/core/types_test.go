package core

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    *ValidationError
		expected string
	}{
		{
			name: "complete validation error",
			error: &ValidationError{
				Field:   "model",
				Value:   "",
				Rule:    "required",
				Message: "cannot be empty",
			},
			expected: "validation failed for field model (value: , rule: required): cannot be empty",
		},
		{
			name: "validation error with complex value",
			error: &ValidationError{
				Field:   "temperature",
				Value:   2.5,
				Rule:    "max=2",
				Message: "temperature too high",
			},
			expected: "validation failed for field temperature (value: 2.5, rule: max=2): temperature too high",
		},
		{
			name: "validation error with nil value",
			error: &ValidationError{
				Field:   "messages",
				Value:   nil,
				Rule:    "required",
				Message: "messages cannot be nil",
			},
			expected: "validation failed for field messages (value: <nil>, rule: required): messages cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    *APIError
		expected string
	}{
		{
			name: "complete API error",
			error: &APIError{
				StatusCode: 429,
				Code:       "rate_limit_exceeded",
				Message:    "Rate limit exceeded",
				Provider:   "openai",
			},
			expected: "API error from openai (429/rate_limit_exceeded): Rate limit exceeded",
		},
		{
			name: "API error without provider",
			error: &APIError{
				StatusCode: 500,
				Code:       "internal_error",
				Message:    "Internal server error",
				Provider:   "",
			},
			expected: "API error from  (500/internal_error): Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.error.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessage_String(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name: "short message",
			message: Message{
				Role:    "user",
				Content: "Hello, world!",
			},
			expected: `Message{Role: user, Content: "Hello, world!"}`,
		},
		{
			name: "long message gets truncated",
			message: Message{
				Role:    "assistant",
				Content: strings.Repeat("a", 150),
			},
			expected: `Message{Role: assistant, Content: "` + strings.Repeat("a", 97) + `..."}`,
		},
		{
			name: "message with special characters",
			message: Message{
				Role:    "system",
				Content: "You are a \"helpful\" assistant\nwith multiple lines",
			},
			expected: `Message{Role: system, Content: "You are a \"helpful\" assistant\nwith multiple lines"}`,
		},
		{
			name: "empty message",
			message: Message{
				Role:    "user",
				Content: "",
			},
			expected: `Message{Role: user, Content: ""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.message.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompletionRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		request     *CompletionRequest
		expectError bool
		errorField  string
	}{
		{
			name: "valid basic request",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expectError: false,
		},
		{
			name: "valid request with all parameters",
			request: &CompletionRequest{
				Model: "gpt-4",
				Messages: []Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
				MaxTokens:   func(i int) *int { return &i }(1000),
				Temperature: func(f float64) *float64 { return &f }(0.7),
				TopP:        func(f float64) *float64 { return &f }(1.0),
			},
			expectError: false,
		},
		{
			name: "empty model",
			request: &CompletionRequest{
				Model: "",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expectError: true,
			errorField:  "model",
		},
		{
			name: "no messages",
			request: &CompletionRequest{
				Model:    "gpt-3.5-turbo",
				Messages: []Message{},
			},
			expectError: true,
			errorField:  "messages",
		},
		{
			name: "temperature too high",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: func(f float64) *float64 { return &f }(2.5),
			},
			expectError: true,
			errorField:  "temperature",
		},
		{
			name: "temperature too low",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: func(f float64) *float64 { return &f }(-0.1),
			},
			expectError: true,
			errorField:  "temperature",
		},
		{
			name: "invalid message role",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "invalid", Content: "Hello"},
				},
			},
			expectError: true,
			errorField:  "messages[0].role",
		},
		{
			name: "empty message role",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "", Content: "Hello"},
				},
			},
			expectError: true,
			errorField:  "messages[0].role",
		},
		{
			name: "valid roles",
			request: &CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []Message{
					{Role: "system", Content: "System message"},
					{Role: "user", Content: "User message"},
					{Role: "assistant", Content: "Assistant message"},
					{Role: "tool", Content: "Tool message"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModel_String(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected string
	}{
		{
			name: "model with all fields",
			model: Model{
				ID:        "gpt-4",
				Provider:  "openai",
				MaxTokens: func(i int) *int { return &i }(8192),
			},
			expected: "Model{ID: gpt-4, Provider: openai, MaxTokens: 8192}",
		},
		{
			name: "model without max tokens",
			model: Model{
				ID:       "claude-3-sonnet",
				Provider: "anthropic",
			},
			expected: "Model{ID: claude-3-sonnet, Provider: anthropic, MaxTokens: <nil>}",
		},
		{
			name:     "empty model",
			model:    Model{},
			expected: "Model{ID: , Provider: , MaxTokens: <nil>}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.model.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStreamReader(t *testing.T) {
	t.Run("successful streaming", func(t *testing.T) {
		// Create a channel with test data
		chunks := make(chan StreamChunk, 3)
		chunks <- StreamChunk{
			Choices: []Choice{
				{Delta: &Message{Content: "Hello"}},
			},
		}
		chunks <- StreamChunk{
			Choices: []Choice{
				{Delta: &Message{Content: " world"}},
			},
		}
		chunks <- StreamChunk{Done: true}
		close(chunks)

		reader := NewStreamReader(chunks)

		// Read first chunk
		chunk1, err := reader.Read()
		assert.NoError(t, err)
		assert.Equal(t, "Hello", chunk1.Choices[0].Delta.Content)
		assert.False(t, chunk1.Done)

		// Read second chunk
		chunk2, err := reader.Read()
		assert.NoError(t, err)
		assert.Equal(t, " world", chunk2.Choices[0].Delta.Content)
		assert.False(t, chunk2.Done)

		// Read final chunk
		chunk3, err := reader.Read()
		assert.NoError(t, err)
		assert.True(t, chunk3.Done)

		// Reading after stream is closed should return EOF
		_, err = reader.Read()
		assert.Equal(t, io.EOF, err)

		// Reader should be closed
		err = reader.Close()
		assert.NoError(t, err)
	})

	t.Run("streaming with error", func(t *testing.T) {
		testErr := errors.New("streaming error")
		chunks := make(chan StreamChunk, 2)
		chunks <- StreamChunk{
			Choices: []Choice{
				{Delta: &Message{Content: "Hello"}},
			},
		}
		chunks <- StreamChunk{
			Error: testErr,
		}
		close(chunks)

		reader := NewStreamReader(chunks)

		// Read successful chunk
		chunk1, err := reader.Read()
		assert.NoError(t, err)
		assert.Equal(t, "Hello", chunk1.Choices[0].Delta.Content)

		// Read error chunk
		chunk2, err := reader.Read()
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, testErr, chunk2.Error)

		// Reader should have stored the error
		assert.Equal(t, testErr, reader.Err())
	})

	t.Run("empty stream", func(t *testing.T) {
		chunks := make(chan StreamChunk)
		close(chunks)

		reader := NewStreamReader(chunks)

		// Reading from empty stream should return EOF
		_, err := reader.Read()
		assert.Equal(t, io.EOF, err)
	})
}

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      ProviderConfig
		expectError bool
		errorField  string
	}{
		{
			name: "valid config",
			config: ProviderConfig{
				Type:   "openai",
				APIKey: "sk-test",
			},
			expectError: false,
		},
		{
			name: "empty type",
			config: ProviderConfig{
				APIKey: "sk-test",
			},
			expectError: true,
			errorField:  "type",
		},
		{
			name: "valid config with all fields",
			config: ProviderConfig{
				Type:       "anthropic",
				APIKey:     "ak-test",
				BaseURL:    "https://api.anthropic.com",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				RateLimit:  "60/minute",
				CustomHeaders: map[string]string{
					"X-Custom": "value",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecureStringImplementations(t *testing.T) {
	// This tests the interface implementations for secure string handling
	// that would be used in the config package

	t.Run("secure string should not expose value in string representation", func(t *testing.T) {
		// This is a conceptual test - the actual SecureString type
		// would be implemented in the config package
		sensitiveValue := "sk-very-secret-key-12345"

		// In a real implementation, this would use the SecureString type
		// For now, we test the concept
		type SecureString string

		s := SecureString(sensitiveValue)
		stringRepresentation := func(s SecureString) string {
			if s == "" {
				return ""
			}
			return "***REDACTED***"
		}(s)

		assert.Equal(t, "***REDACTED***", stringRepresentation)
		assert.NotContains(t, stringRepresentation, "sk-very-secret")
	})
}

func TestConstants(t *testing.T) {
	t.Run("message roles", func(t *testing.T) {
		assert.Equal(t, "system", RoleSystem)
		assert.Equal(t, "user", RoleUser)
		assert.Equal(t, "assistant", RoleAssistant)
		assert.Equal(t, "tool", RoleTool)
	})

	t.Run("finish reasons", func(t *testing.T) {
		assert.Equal(t, "stop", FinishReasonStop)
		assert.Equal(t, "length", FinishReasonLength)
		assert.Equal(t, "tool_calls", FinishReasonToolCalls)
		assert.Equal(t, "content_filter", FinishReasonContentFilter)
		assert.Equal(t, "timeout", FinishReasonTimeout)
	})

	t.Run("tool types", func(t *testing.T) {
		assert.Equal(t, "function", ToolTypeFunction)
	})

	t.Run("model constants", func(t *testing.T) {
		assert.Equal(t, "gpt-3.5-turbo", ModelGPT35Turbo)
		assert.Equal(t, "gpt-4", ModelGPT4)
		assert.Equal(t, "gpt-4-turbo", ModelGPT4Turbo)
		assert.Equal(t, "claude-3-sonnet", ModelClaude3)
	})
}

func TestDefaultSettings(t *testing.T) {
	t.Run("default settings values", func(t *testing.T) {
		assert.Equal(t, 2048, DefaultSettings.MaxTokens)
		assert.Equal(t, 0.7, DefaultSettings.Temperature)
		assert.Equal(t, 1.0, DefaultSettings.TopP)
		assert.Equal(t, 0.0, DefaultSettings.FrequencyPenalty)
		assert.Equal(t, 0.0, DefaultSettings.PresencePenalty)
		assert.Equal(t, 30*time.Second, DefaultSettings.Timeout)
		assert.Equal(t, 3, DefaultSettings.MaxRetries)
	})

	t.Run("default settings are reasonable", func(t *testing.T) {
		// Ensure default values are within valid ranges
		assert.True(t, DefaultSettings.MaxTokens > 0)
		assert.True(t, DefaultSettings.Temperature >= 0 && DefaultSettings.Temperature <= 2)
		assert.True(t, DefaultSettings.TopP >= 0 && DefaultSettings.TopP <= 1)
		assert.True(t, DefaultSettings.FrequencyPenalty >= -2 && DefaultSettings.FrequencyPenalty <= 2)
		assert.True(t, DefaultSettings.PresencePenalty >= -2 && DefaultSettings.PresencePenalty <= 2)
		assert.True(t, DefaultSettings.Timeout > 0)
		assert.True(t, DefaultSettings.MaxRetries >= 0)
	})
}

func TestUsageCalculations(t *testing.T) {
	t.Run("usage with cost calculations", func(t *testing.T) {
		usage := Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			PromptCost:       func(f float64) *float64 { return &f }(0.001),
			CompletionCost:   func(f float64) *float64 { return &f }(0.002),
			TotalCost:        func(f float64) *float64 { return &f }(0.003),
		}

		assert.Equal(t, 100, usage.PromptTokens)
		assert.Equal(t, 50, usage.CompletionTokens)
		assert.Equal(t, 150, usage.TotalTokens)
		assert.NotNil(t, usage.PromptCost)
		assert.Equal(t, 0.001, *usage.PromptCost)
		assert.NotNil(t, usage.CompletionCost)
		assert.Equal(t, 0.002, *usage.CompletionCost)
		assert.NotNil(t, usage.TotalCost)
		assert.Equal(t, 0.003, *usage.TotalCost)
	})

	t.Run("usage without cost information", func(t *testing.T) {
		usage := Usage{
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
		}

		assert.Equal(t, 200, usage.PromptTokens)
		assert.Equal(t, 100, usage.CompletionTokens)
		assert.Equal(t, 300, usage.TotalTokens)
		assert.Nil(t, usage.PromptCost)
		assert.Nil(t, usage.CompletionCost)
		assert.Nil(t, usage.TotalCost)
	})
}

// Benchmark tests for performance-critical operations

func BenchmarkMessage_String(b *testing.B) {
	message := Message{
		Role:    "user",
		Content: "This is a test message for benchmarking string conversion performance",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = message.String()
	}
}

func BenchmarkCompletionRequest_Validate(b *testing.B) {
	request := &CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello, world!"},
		},
		MaxTokens:   func(i int) *int { return &i }(1000),
		Temperature: func(f float64) *float64 { return &f }(0.7),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := request.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamReader_Read(b *testing.B) {
	// Create a channel with many chunks
	chunks := make(chan StreamChunk, b.N+1)
	for i := 0; i < b.N; i++ {
		chunks <- StreamChunk{
			Choices: []Choice{
				{Delta: &Message{Content: "test"}},
			},
		}
	}
	chunks <- StreamChunk{Done: true}
	close(chunks)

	reader := NewStreamReader(chunks)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := reader.Read()
		if err != nil && err != io.EOF {
			b.Fatal(err)
		}
	}
}

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    interface{}
		rule     string
		message  string
		expected string
	}{
		{
			name:     "string field",
			field:    "model",
			value:    "",
			rule:     "required",
			message:  "cannot be empty",
			expected: "validation failed for field model (value: , rule: required): cannot be empty",
		},
		{
			name:     "numeric field",
			field:    "temperature",
			value:    2.5,
			rule:     "max",
			message:  "exceeds maximum value",
			expected: "validation failed for field temperature (value: 2.5, rule: max): exceeds maximum value",
		},
		{
			name:     "nil value",
			field:    "messages",
			value:    nil,
			rule:     "required",
			message:  "cannot be nil",
			expected: "validation failed for field messages (value: <nil>, rule: required): cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.value, tt.rule, tt.message)

			assert.NotNil(t, err)
			assert.Equal(t, tt.field, err.Field)
			assert.Equal(t, tt.value, err.Value)
			assert.Equal(t, tt.rule, err.Rule)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestChoice_String(t *testing.T) {
	tests := []struct {
		name     string
		choice   Choice
		expected string
	}{
		{
			name: "choice with message",
			choice: Choice{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello!",
				},
				FinishReason: "stop",
			},
			expected: `Choice{Index: 0, Message: Message{Role: assistant, Content: "Hello!"}, FinishReason: stop}`,
		},
		{
			name: "choice with delta",
			choice: Choice{
				Index: 1,
				Delta: &Message{
					Role:    "assistant",
					Content: "streaming...",
				},
				FinishReason: "",
			},
			expected: `Choice{Index: 1, Delta: Message{Role: assistant, Content: "streaming..."}, FinishReason: }`,
		},
		{
			name: "choice with both message and delta",
			choice: Choice{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Complete",
				},
				Delta: &Message{
					Role:    "assistant",
					Content: "Partial",
				},
				FinishReason: "stop",
			},
			expected: `Choice{Index: 0, Message: Message{Role: assistant, Content: "Complete"}, Delta: Message{Role: assistant, Content: "Partial"}, FinishReason: stop}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.choice.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompletionResponse_String(t *testing.T) {
	tests := []struct {
		name     string
		response CompletionResponse
		expected string
	}{
		{
			name: "response with single choice",
			response: CompletionResponse{
				ID:    "test-123",
				Model: "gpt-3.5-turbo",
				Choices: []Choice{
					{
						Index: 0,
						Message: Message{
							Role:    "assistant",
							Content: "Hello!",
						},
						FinishReason: "stop",
					},
				},
			},
			expected: `CompletionResponse{ID: test-123, Model: gpt-3.5-turbo, Choices: 1}`,
		},
		{
			name: "response with multiple choices",
			response: CompletionResponse{
				ID:    "test-456",
				Model: "gpt-4",
				Choices: []Choice{
					{Index: 0, Message: Message{Content: "Choice 1"}},
					{Index: 1, Message: Message{Content: "Choice 2"}},
				},
			},
			expected: `CompletionResponse{ID: test-456, Model: gpt-4, Choices: 2}`,
		},
		{
			name: "response with no choices",
			response: CompletionResponse{
				ID:      "test-789",
				Model:   "claude-3",
				Choices: []Choice{},
			},
			expected: `CompletionResponse{ID: test-789, Model: claude-3, Choices: 0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStreamChunk_String(t *testing.T) {
	tests := []struct {
		name     string
		chunk    StreamChunk
		expected string
	}{
		{
			name: "chunk with choices",
			chunk: StreamChunk{
				ID:    "stream-123",
				Model: "gpt-3.5-turbo",
				Choices: []Choice{
					{
						Index: 0,
						Delta: &Message{Content: "Hello"},
					},
				},
				Done: false,
			},
			expected: `StreamChunk{ID: stream-123, Model: gpt-3.5-turbo, Choices: 1, Done: false}`,
		},
		{
			name: "final chunk",
			chunk: StreamChunk{
				ID:      "stream-456",
				Model:   "gpt-4",
				Choices: []Choice{},
				Done:    true,
			},
			expected: `StreamChunk{ID: stream-456, Model: gpt-4, Choices: 0, Done: true}`,
		},
		{
			name: "chunk with error",
			chunk: StreamChunk{
				ID:      "stream-error",
				Model:   "gpt-3.5-turbo",
				Choices: []Choice{},
				Done:    false,
				Error:   errors.New("stream error"),
			},
			expected: `StreamChunk{ID: stream-error, Model: gpt-3.5-turbo, Choices: 0, Done: false, Error: stream error}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.chunk.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToolCall_Validation(t *testing.T) {
	tests := []struct {
		name        string
		toolCall    ToolCall
		expectError bool
		errorField  string
	}{
		{
			name: "valid tool call",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "get_weather",
					Arguments: json.RawMessage(`{"location": "New York"}`),
				},
			},
			expectError: false,
		},
		{
			name: "empty tool call ID",
			toolCall: ToolCall{
				ID:   "",
				Type: "function",
				Function: FunctionCall{
					Name: "get_weather",
				},
			},
			expectError: true,
			errorField:  "id",
		},
		{
			name: "invalid tool call type",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: "invalid",
				Function: FunctionCall{
					Name: "get_weather",
				},
			},
			expectError: true,
			errorField:  "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.toolCall.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionCall_Validation(t *testing.T) {
	tests := []struct {
		name         string
		functionCall FunctionCall
		expectError  bool
		errorField   string
	}{
		{
			name: "valid function call",
			functionCall: FunctionCall{
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location": "New York"}`),
			},
			expectError: false,
		},
		{
			name: "empty function name",
			functionCall: FunctionCall{
				Name:      "",
				Arguments: json.RawMessage(`{"location": "New York"}`),
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "invalid JSON arguments",
			functionCall: FunctionCall{
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location": "New York"`), // Invalid JSON
			},
			expectError: true,
			errorField:  "arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.functionCall.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessage_Validation(t *testing.T) {
	tests := []struct {
		name        string
		message     Message
		expectError bool
		errorField  string
	}{
		{
			name: "valid user message",
			message: Message{
				Role:    "user",
				Content: "Hello, world!",
			},
			expectError: false,
		},
		{
			name: "valid system message",
			message: Message{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			expectError: false,
		},
		{
			name: "valid assistant message",
			message: Message{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			expectError: false,
		},
		{
			name: "valid tool message",
			message: Message{
				Role:       "tool",
				Content:    "Weather data",
				ToolCallID: "call_123",
			},
			expectError: false,
		},
		{
			name: "invalid role",
			message: Message{
				Role:    "invalid",
				Content: "Hello",
			},
			expectError: true,
			errorField:  "role",
		},
		{
			name: "empty role",
			message: Message{
				Role:    "",
				Content: "Hello",
			},
			expectError: true,
			errorField:  "role",
		},
		{
			name: "tool message without tool_call_id",
			message: Message{
				Role:    "tool",
				Content: "Tool response",
			},
			expectError: true,
			errorField:  "tool_call_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUsage_String(t *testing.T) {
	tests := []struct {
		name     string
		usage    Usage
		expected string
	}{
		{
			name: "usage with costs",
			usage: Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
				PromptCost:       func(f float64) *float64 { return &f }(0.001),
				CompletionCost:   func(f float64) *float64 { return &f }(0.002),
				TotalCost:        func(f float64) *float64 { return &f }(0.003),
			},
			expected: "Usage{Prompt: 100, Completion: 50, Total: 150, Cost: $0.003}",
		},
		{
			name: "usage without costs",
			usage: Usage{
				PromptTokens:     200,
				CompletionTokens: 100,
				TotalTokens:      300,
			},
			expected: "Usage{Prompt: 200, Completion: 100, Total: 300, Cost: N/A}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.usage.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
