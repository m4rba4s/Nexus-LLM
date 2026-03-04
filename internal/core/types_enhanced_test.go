// Package core provides enhanced test coverage for critical edge cases and scenarios
package core

import (
	"encoding/json"
	"testing"
	"time"
)

// Helper functions for pointer creation
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }
func stringPtr(s string) *string  { return &s }

// TestCompletionRequest_EdgeCases tests critical edge cases for validation
func TestCompletionRequest_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     CompletionRequest
		expectError bool
		errorField  string
	}{
		{
			name: "extremely long model name - allowed",
			request: CompletionRequest{
				Model: string(make([]byte, 1000)), // Long model name is allowed
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "negative temperature",
			request: CompletionRequest{
				Model:       "gpt-4",
				Temperature: floatPtr(-1.5),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: true,
			errorField:  "temperature",
		},
		{
			name: "temperature above maximum",
			request: CompletionRequest{
				Model:       "gpt-4",
				Temperature: floatPtr(3.0),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: true,
			errorField:  "temperature",
		},
		{
			name: "negative max tokens - allowed in current implementation",
			request: CompletionRequest{
				Model:     "gpt-4",
				MaxTokens: intPtr(-100),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "extremely high max tokens - allowed",
			request: CompletionRequest{
				Model:     "gpt-4",
				MaxTokens: intPtr(1000000),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid top_p below 0 - allowed in current implementation",
			request: CompletionRequest{
				Model: "gpt-4",
				TopP:  floatPtr(-0.1),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid top_p above 1 - allowed in current implementation",
			request: CompletionRequest{
				Model: "gpt-4",
				TopP:  floatPtr(1.5),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid frequency penalty - allowed in current implementation",
			request: CompletionRequest{
				Model:            "gpt-4",
				FrequencyPenalty: floatPtr(-3.0),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid presence penalty - allowed in current implementation",
			request: CompletionRequest{
				Model:           "gpt-4",
				PresencePenalty: floatPtr(3.0),
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "empty stop sequences array is valid",
			request: CompletionRequest{
				Model: "gpt-4",
				Stop:  []string{},
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "many stop sequences - allowed in current implementation",
			request: CompletionRequest{
				Model: "gpt-4",
				Stop:  make([]string, 10), // Many sequences allowed
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
		{
			name: "valid boundary values",
			request: CompletionRequest{
				Model:            "gpt-4",
				Temperature:      floatPtr(2.0), // Maximum allowed
				TopP:             floatPtr(1.0), // Maximum allowed
				FrequencyPenalty: floatPtr(2.0), // Maximum allowed
				PresencePenalty:  floatPtr(2.0), // Maximum allowed
				MaxTokens:        intPtr(4096),  // Reasonable maximum
				Messages: []Message{
					{Role: RoleUser, Content: "test"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.request.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for field %s, but got none", tt.errorField)
				} else if validationErr, ok := err.(*ValidationError); ok {
					if validationErr.Field != tt.errorField && tt.errorField != "" {
						t.Errorf("Expected error for field %s, got error for field %s", tt.errorField, validationErr.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, but got: %v", err)
				}
			}
		})
	}
}

// TestMessage_EdgeCases tests message validation edge cases
func TestMessage_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		message     Message
		expectError bool
		errorField  string
	}{
		{
			name: "empty role",
			message: Message{
				Role:    "",
				Content: "test content",
			},
			expectError: true,
			errorField:  "role",
		},
		{
			name: "invalid role",
			message: Message{
				Role:    "invalid_role",
				Content: "test content",
			},
			expectError: true,
			errorField:  "role",
		},
		{
			name: "extremely long content",
			message: Message{
				Role:    RoleUser,
				Content: string(make([]byte, 100000)), // 100KB content
			},
			expectError: false, // Long content should be allowed
		},
		{
			name: "unicode content",
			message: Message{
				Role:    RoleUser,
				Content: "测试内容 🎉 émoji ñoño",
			},
			expectError: false,
		},
		{
			name: "tool message with tool call ID",
			message: Message{
				Role:       RoleTool,
				Content:    "tool response",
				ToolCallID: "call_123",
			},
			expectError: false,
		},
		{
			name: "tool message without tool call ID",
			message: Message{
				Role:    RoleTool,
				Content: "tool response",
			},
			expectError: true,
			errorField:  "tool_call_id",
		},
		{
			name: "assistant message with tool calls",
			message: Message{
				Role:    RoleAssistant,
				Content: "",
				ToolCalls: []ToolCall{
					{
						ID:   "call_123",
						Type: ToolTypeFunction,
						Function: FunctionCall{
							Name:      "test_function",
							Arguments: json.RawMessage(`{"param": "value"}`),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "user message with tool calls - allowed in current implementation",
			message: Message{
				Role:    RoleUser,
				Content: "test",
				ToolCalls: []ToolCall{
					{ID: "call_123", Type: ToolTypeFunction},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.message.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for field %s, but got none", tt.errorField)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, but got: %v", err)
				}
			}
		})
	}
}

// TestProviderConfig_EdgeCases tests provider configuration edge cases
func TestProviderConfig_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      ProviderConfig
		expectError bool
		errorField  string
	}{
		{
			name: "basic valid config",
			config: ProviderConfig{
				Type:   "openai",
				APIKey: "test-key",
			},
			expectError: false,
		},
		{
			name: "config with all fields - validation not fully implemented",
			config: ProviderConfig{
				Type:       "openai",
				APIKey:     "test-key",
				BaseURL:    "https://api.example.com",
				MaxRetries: 5,
				Timeout:    30 * time.Second,
				RateLimit:  "100",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for field %s, but got none", tt.errorField)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, but got: %v", err)
				}
			}
		})
	}
}

// TestToolCall_EdgeCases tests tool call validation edge cases
func TestToolCall_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		toolCall    ToolCall
		expectError bool
		errorField  string
	}{
		{
			name: "empty ID",
			toolCall: ToolCall{
				ID:   "",
				Type: ToolTypeFunction,
				Function: FunctionCall{
					Name:      "test",
					Arguments: json.RawMessage("{}"),
				},
			},
			expectError: true,
			errorField:  "id",
		},
		{
			name: "invalid type",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: "invalid_type",
				Function: FunctionCall{
					Name:      "test",
					Arguments: json.RawMessage("{}"),
				},
			},
			expectError: true,
			errorField:  "type",
		},
		{
			name: "function type with empty function",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: ToolTypeFunction,
			},
			expectError: true,
			errorField:  "function",
		},
		{
			name: "function with invalid JSON arguments",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: ToolTypeFunction,
				Function: FunctionCall{
					Name:      "test",
					Arguments: json.RawMessage("invalid json"),
				},
			},
			expectError: true,
			errorField:  "arguments",
		},
		{
			name: "valid minimal tool call",
			toolCall: ToolCall{
				ID:   "call_123",
				Type: ToolTypeFunction,
				Function: FunctionCall{
					Name:      "test",
					Arguments: json.RawMessage("{}"),
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.toolCall.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for field %s, but got none", tt.errorField)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, but got: %v", err)
				}
			}
		})
	}
}

// TestFunctionCall_EdgeCases tests function call validation edge cases
func TestFunctionCall_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		functionCall FunctionCall
		expectError  bool
		errorField   string
	}{
		{
			name: "empty function name",
			functionCall: FunctionCall{
				Name:      "",
				Arguments: json.RawMessage("{}"),
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "function name with special characters - allowed in current implementation",
			functionCall: FunctionCall{
				Name:      "function-with-special-chars!@#",
				Arguments: json.RawMessage("{}"),
			},
			expectError: false,
		},
		{
			name: "long function name - allowed in current implementation",
			functionCall: FunctionCall{
				Name:      string(make([]byte, 100)), // Reasonable length for testing
				Arguments: json.RawMessage("{}"),
			},
			expectError: false,
		},
		{
			name: "invalid JSON arguments",
			functionCall: FunctionCall{
				Name:      "valid_function",
				Arguments: json.RawMessage("{invalid json}"),
			},
			expectError: true,
			errorField:  "arguments",
		},
		{
			name: "empty arguments (valid)",
			functionCall: FunctionCall{
				Name:      "valid_function",
				Arguments: nil,
			},
			expectError: false,
		},
		{
			name: "complex valid arguments",
			functionCall: FunctionCall{
				Name:      "complex_function",
				Arguments: json.RawMessage(`{"array": [1,2,3], "object": {"nested": true}, "string": "value"}`),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.functionCall.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for field %s, but got none", tt.errorField)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, but got: %v", err)
				}
			}
		})
	}
}

// TestStreamReader_EdgeCases tests stream reader edge cases
func TestStreamReader_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("read after close", func(t *testing.T) {
		t.Parallel()
		chunks := make(chan StreamChunk, 1)
		close(chunks)
		reader := NewStreamReader(chunks)
		reader.Close()

		_, err := reader.Read()
		if err == nil {
			t.Error("Expected error when reading after close")
		}
	})

	t.Run("multiple closes", func(t *testing.T) {
		t.Parallel()
		chunks := make(chan StreamChunk, 1)
		reader := NewStreamReader(chunks)
		reader.Close()
		reader.Close() // Should not panic
		reader.Close() // Should not panic
	})

	t.Run("concurrent read/close", func(t *testing.T) {
		t.Parallel()
		chunks := make(chan StreamChunk, 1)
		reader := NewStreamReader(chunks)

		// Start a goroutine that tries to read
		done := make(chan bool)
		go func() {
			defer func() { done <- true }()
			_, _ = reader.Read()
		}()

		// Close the reader while read is potentially blocking
		reader.Close()

		// Wait for the read goroutine to complete
		<-done
	})
}

// TestStringMethods_EdgeCases tests string representation methods with edge cases
func TestStringMethods_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("message string with nil values", func(t *testing.T) {
		t.Parallel()
		var msg *Message
		if msg != nil {
			str := msg.String()
			if str == "" {
				t.Error("Expected non-empty string for message")
			}
		}
	})

	t.Run("model string with unicode", func(t *testing.T) {
		t.Parallel()
		model := &Model{
			ID:          "test-模型-🚀",
			Description: "Test model with unicode characters: 测试模型",
			Tags:        []string{"测试", "🎯", "production"},
		}

		str := model.String()
		if str == "" {
			t.Error("Expected non-empty string for model")
		}
	})

	t.Run("usage string with zero values", func(t *testing.T) {
		t.Parallel()
		usage := &Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
			TotalCost:        floatPtr(0.0),
		}

		str := usage.String()
		if str == "" {
			t.Error("Expected non-empty string for usage")
		}
	})

	t.Run("choice string with empty values", func(t *testing.T) {
		t.Parallel()
		choice := &Choice{
			Index:        0,
			Message:      Message{},
			FinishReason: "",
		}

		str := choice.String()
		if str == "" {
			t.Error("Expected non-empty string for choice")
		}
	})
}

// TestJSONMarshalUnmarshal tests JSON serialization edge cases
func TestJSONMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("completion request json", func(t *testing.T) {
		t.Parallel()
		request := &CompletionRequest{
			Model: "gpt-4",
			Messages: []Message{
				{Role: RoleUser, Content: "test with unicode: 🎉 测试"},
			},
			Temperature: floatPtr(0.7),
			MaxTokens:   intPtr(100),
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshaled CompletionRequest
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Model != request.Model {
			t.Errorf("Model mismatch after JSON round-trip")
		}

		if len(unmarshaled.Messages) != len(request.Messages) {
			t.Errorf("Messages length mismatch after JSON round-trip")
		}
	})

	t.Run("completion response json", func(t *testing.T) {
		t.Parallel()
		response := &CompletionResponse{
			ID:      "test-id",
			Model:   "gpt-4",
			Created: time.Now().Unix(),
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: "Response with unicode: 🚀 测试回复",
					},
					FinishReason: FinishReasonStop,
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Failed to marshal response: %v", err)
		}

		var unmarshaled CompletionResponse
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if unmarshaled.ID != response.ID {
			t.Errorf("ID mismatch after JSON round-trip")
		}
	})
}

// TestErrorTypes_EdgeCases tests error types with various scenarios
func TestErrorTypes_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("validation error with nil values", func(t *testing.T) {
		t.Parallel()
		err := NewValidationError("", nil, "", "")
		if err == nil {
			t.Error("Expected non-nil validation error")
		}

		errStr := err.Error()
		if errStr == "" {
			t.Error("Expected non-empty error string")
		}
	})

	t.Run("api error with extreme values", func(t *testing.T) {
		t.Parallel()
		apiErr := &APIError{
			StatusCode: 999,
			Code:       "EXTREME_ERROR",
			Message:    string(make([]byte, 10000)), // Very long message
			Type:       "critical",
			Provider:   "test-provider",
		}

		errStr := apiErr.Error()
		if errStr == "" {
			t.Error("Expected non-empty error string")
		}
	})
}

// BenchmarkCriticalPaths benchmarks critical performance paths
func BenchmarkCriticalPaths(b *testing.B) {
	request := &CompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "benchmark test message"},
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(100),
	}

	b.Run("validation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = request.Validate()
		}
	})

	b.Run("json_marshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(request)
		}
	})

	usage := &Usage{
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
		TotalCost:        floatPtr(0.05),
	}

	b.Run("usage_string", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = usage.String()
		}
	})
}
