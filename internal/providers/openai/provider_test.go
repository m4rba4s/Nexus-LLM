package openai

import (
	"context"
	"encoding/json"

	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorField  string
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "sk-test",
			},
			expectError: false,
		},
		{
			name: "valid config with custom settings",
			config: Config{
				APIKey:       "sk-test",
				BaseURL:      "https://custom.api.com/v1",
				Organization: "org-123",
				Timeout:      60 * time.Second,
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: Config{
				BaseURL: "https://api.openai.com/v1",
			},
			expectError: true,
			errorField:  "api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := New(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *core.ValidationError
				if assert.ErrorAs(t, err, &validationErr) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "openai", provider.Name())

				// Check defaults were applied
				if tt.config.BaseURL == "" {
					assert.Equal(t, DefaultBaseURL, provider.config.BaseURL)
				}
				if tt.config.Timeout == 0 {
					assert.Equal(t, DefaultTimeout, provider.config.Timeout)
				}
				if tt.config.UserAgent == "" {
					assert.Equal(t, UserAgent, provider.config.UserAgent)
				}
			}
		})
	}
}

func TestProvider_Name(t *testing.T) {
	provider := createTestProvider(t)
	assert.Equal(t, "openai", provider.Name())
}

func TestProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorField  string
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:  "sk-test",
				BaseURL: "https://api.openai.com/v1",
				Timeout: 30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty API key",
			config: Config{
				APIKey:  "",
				BaseURL: "https://api.openai.com/v1",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorField:  "api_key",
		},
		{
			name: "empty base URL",
			config: Config{
				APIKey:  "sk-test",
				BaseURL: "",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorField:  "base_url",
		},
		{
			name: "zero timeout",
			config: Config{
				APIKey:  "sk-test",
				BaseURL: "https://api.openai.com/v1",
				Timeout: 0,
			},
			expectError: true,
			errorField:  "timeout",
		},
		{
			name: "negative timeout",
			config: Config{
				APIKey:  "sk-test",
				BaseURL: "https://api.openai.com/v1",
				Timeout: -5 * time.Second,
			},
			expectError: true,
			errorField:  "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{config: tt.config}
			err := provider.ValidateConfig()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *core.ValidationError
				if assert.ErrorAs(t, err, &validationErr) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_CreateCompletion(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	tests := []struct {
		name           string
		request        *core.CompletionRequest
		mockResponse   string
		mockStatusCode int
		expectedError  string
		checkResponse  func(t *testing.T, resp *core.CompletionResponse)
	}{
		{
			name: "successful completion",
			request: &core.CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello, world!"},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-3.5-turbo",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you today?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 12,
					"completion_tokens": 10,
					"total_tokens": 22
				}
			}`,
			mockStatusCode: 200,
			checkResponse: func(t *testing.T, resp *core.CompletionResponse) {
				assert.Equal(t, "chatcmpl-123", resp.ID)
				assert.Equal(t, "chat.completion", resp.Object)
				assert.Equal(t, "gpt-3.5-turbo", resp.Model)
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
				assert.Equal(t, "Hello! How can I help you today?", resp.Choices[0].Message.Content)
				assert.Equal(t, "stop", resp.Choices[0].FinishReason)
				assert.Equal(t, 12, resp.Usage.PromptTokens)
				assert.Equal(t, 10, resp.Usage.CompletionTokens)
				assert.Equal(t, 22, resp.Usage.TotalTokens)
				assert.Equal(t, "openai", resp.Provider)
			},
		},
		{
			name: "completion with function calling",
			request: &core.CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "What's the weather like?"},
				},
				Tools: []core.Tool{
					{
						Type: "function",
						Function: core.ToolFunction{
							Name:        "get_weather",
							Description: "Get current weather",
							Parameters: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"location": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			mockResponse: `{
				"id": "chatcmpl-456",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-3.5-turbo",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": null,
						"tool_calls": [{
							"id": "call_123",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\": \"San Francisco\"}"
							}
						}]
					},
					"finish_reason": "tool_calls"
				}],
				"usage": {
					"prompt_tokens": 15,
					"completion_tokens": 8,
					"total_tokens": 23
				}
			}`,
			mockStatusCode: 200,
			checkResponse: func(t *testing.T, resp *core.CompletionResponse) {
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
				assert.Equal(t, "tool_calls", resp.Choices[0].FinishReason)
				assert.Len(t, resp.Choices[0].Message.ToolCalls, 1)
				assert.Equal(t, "call_123", resp.Choices[0].Message.ToolCalls[0].ID)
				assert.Equal(t, "function", resp.Choices[0].Message.ToolCalls[0].Type)
				assert.Equal(t, "get_weather", resp.Choices[0].Message.ToolCalls[0].Function.Name)
			},
		},
		{
			name: "API error response",
			request: &core.CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"error": {
					"message": "Invalid model specified",
					"type": "invalid_request_error",
					"code": "model_not_found"
				}
			}`,
			mockStatusCode: 400,
			expectedError:  "API error from openai (400/model_not_found): Invalid model specified",
		},
		{
			name: "rate limit error",
			request: &core.CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"error": {
					"message": "Rate limit reached",
					"type": "requests",
					"code": "rate_limit_exceeded"
				}
			}`,
			mockStatusCode: 429,
			expectedError:  "API error from openai (429/rate_limit_exceeded): Rate limit reached",
		},
		{
			name: "invalid request",
			request: &core.CompletionRequest{
				Model:    "", // Invalid empty model
				Messages: []core.Message{},
			},
			mockStatusCode: 200,
			expectedError:  "invalid request:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/chat/completions", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer sk-"))

				// Verify request body if not expecting validation error
				if !strings.Contains(tt.expectedError, "invalid request") {
					var req openaiCompletionRequest
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(body, &req))
					assert.Equal(t, tt.request.Model, req.Model)
					assert.Len(t, req.Messages, len(tt.request.Messages))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			provider := createTestProviderWithServer(t, server.URL)
			ctx := context.Background()

			resp, err := provider.CreateCompletion(ctx, tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestProvider_CreateCompletion_ContextCancellation(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	provider := createTestProviderWithServer(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &core.CompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.CreateCompletion(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") ||
		strings.Contains(err.Error(), "timeout"))
}

func TestProvider_StreamCompletion(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	tests := []struct {
		name          string
		request       *core.CompletionRequest
		mockResponse  string
		expectedError string
		checkChunks   func(t *testing.T, chunks []core.StreamChunk)
	}{
		{
			name: "successful streaming",
			request: &core.CompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
				Stream: true,
			},
			mockResponse: `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":" there!"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":8,"completion_tokens":5,"total_tokens":13}}

data: [DONE]

`,
			checkChunks: func(t *testing.T, chunks []core.StreamChunk) {
				assert.True(t, len(chunks) >= 3)

				// Check first chunk has role
				firstChunk := chunks[0]
				assert.Equal(t, "chatcmpl-123", firstChunk.ID)
				assert.Equal(t, "gpt-3.5-turbo", firstChunk.Model)
				assert.Equal(t, "openai", firstChunk.Provider)
				if len(firstChunk.Choices) > 0 && firstChunk.Choices[0].Delta != nil {
					assert.Equal(t, "assistant", firstChunk.Choices[0].Delta.Role)
				}

				// Check that we have content chunks
				var hasContent bool
				for _, chunk := range chunks {
					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Content != "" {
						hasContent = true
						break
					}
				}
				assert.True(t, hasContent, "Expected at least one chunk with content")

				// Check final chunk has usage
				lastChunk := chunks[len(chunks)-1]
				if lastChunk.Done && lastChunk.Usage != nil {
					assert.Equal(t, 8, lastChunk.Usage.PromptTokens)
					assert.Equal(t, 5, lastChunk.Usage.CompletionTokens)
					assert.Equal(t, 13, lastChunk.Usage.TotalTokens)
				}
			},
		},
		{
			name: "streaming error",
			request: &core.CompletionRequest{
				Model: "invalid-model",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse:  `{"error": {"message": "Model not found", "type": "invalid_request_error", "code": "model_not_found"}}`,
			expectedError: "API error from openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/chat/completions", r.URL.Path)
				assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

				if tt.expectedError != "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
				} else {
					w.Header().Set("Content-Type", "text/event-stream")
					w.WriteHeader(200)
				}
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			provider := createTestProviderWithServer(t, server.URL)
			ctx := context.Background()

			chunksChan, err := provider.StreamCompletion(ctx, tt.request)

			// For streaming, errors can be returned immediately or through the channel
			if tt.expectedError != "" && err != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			if tt.expectedError == "" {
				assert.NoError(t, err)
			}
			assert.NotNil(t, chunksChan)

			// Collect all chunks
			var chunks []core.StreamChunk
			var streamErr error

			for chunk := range chunksChan {
				if chunk.Error != nil {
					streamErr = chunk.Error
					break
				}
				chunks = append(chunks, chunk)
				if chunk.Done {
					break
				}
			}

			if tt.expectedError != "" {
				if streamErr != nil {
					assert.Error(t, streamErr)
					assert.Contains(t, streamErr.Error(), tt.expectedError)
				} else if err != nil {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					t.Errorf("Expected error containing '%s', but got no error", tt.expectedError)
				}
			} else {
				assert.NoError(t, streamErr)
				if tt.checkChunks != nil {
					tt.checkChunks(t, chunks)
				}
			}
		})
	}
}

func TestProvider_GetModels(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	mockResponse := `{
		"object": "list",
		"data": [
			{
				"id": "gpt-3.5-turbo",
				"object": "model",
				"created": 1677610602,
				"owned_by": "openai"
			},
			{
				"id": "gpt-4",
				"object": "model",
				"created": 1687882411,
				"owned_by": "openai"
			},
			{
				"id": "gpt-4-vision-preview",
				"object": "model",
				"created": 1698894917,
				"owned_by": "system"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/models", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer sk-"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// Create provider with features enabled for testing
	config := Config{
		APIKey:                "sk-test",
		BaseURL:               server.URL,
		Timeout:               5 * time.Second,
		EnableFunctionCalling: true,
		EnableVision:          true,
	}
	provider, err := New(config)
	require.NoError(t, err)

	ctx := context.Background()

	models, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Len(t, models, 3)

	// Check GPT-3.5 Turbo
	gpt35 := findModel(models, "gpt-3.5-turbo")
	assert.NotNil(t, gpt35)
	assert.Equal(t, "openai", gpt35.Provider)
	assert.True(t, gpt35.SupportsFunctions)
	assert.True(t, gpt35.SupportsStreaming)
	assert.NotNil(t, gpt35.MaxTokens)
	assert.Equal(t, 4096, *gpt35.MaxTokens)

	// Check GPT-4
	gpt4 := findModel(models, "gpt-4")
	assert.NotNil(t, gpt4)
	assert.Equal(t, "openai", gpt4.Provider)
	assert.True(t, gpt4.SupportsFunctions)
	assert.True(t, gpt4.SupportsStreaming)
	assert.NotNil(t, gpt4.MaxTokens)
	assert.Equal(t, 8192, *gpt4.MaxTokens)

	// Check GPT-4 Vision
	gpt4v := findModel(models, "gpt-4-vision-preview")
	assert.NotNil(t, gpt4v)
	assert.Equal(t, "openai", gpt4v.Provider)
	assert.True(t, gpt4v.SupportsVision)
}

func TestProvider_GetModels_Caching(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"object": "list", "data": []}`))
	}))
	defer server.Close()

	provider := createTestProviderWithServer(t, server.URL)
	provider.modelsCacheTTL = 1 * time.Second
	ctx := context.Background()

	// First call
	models1, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call within cache TTL
	models2, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount) // Should not increment
	assert.Equal(t, len(models1), len(models2))

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third call after cache expiry
	_, err = provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount) // Should increment
}

func TestProvider_CalculateCost(t *testing.T) {
	provider := createTestProvider(t)

	tests := []struct {
		model    string
		usage    core.Usage
		expected float64
	}{
		{
			model: "gpt-3.5-turbo",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.0015/1000 + 500*0.002/1000, // 0.0025
		},
		{
			model: "gpt-4",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.03/1000 + 500*0.06/1000, // 0.06
		},
		{
			model: "gpt-4-turbo",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.01/1000 + 500*0.03/1000, // 0.025
		},
		{
			model: "unknown-model",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cost := provider.calculateCost(tt.model, &tt.usage)
			assert.InDelta(t, tt.expected, cost, 0.000001)
		})
	}
}

func TestProvider_RequestRetries(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	retryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 {
			w.WriteHeader(500) // Trigger retry
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-3.5-turbo",
			"choices": [{"index": 0, "message": {"role": "assistant", "content": "Success"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
		}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:     "sk-test",
		BaseURL:    server.URL,
		MaxRetries: 3,
		Timeout:    5 * time.Second,
	}
	provider, err := New(config)
	require.NoError(t, err)

	req := &core.CompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, retryCount) // Should have retried twice, succeeded on third
	assert.Equal(t, "Success", resp.Choices[0].Message.Content)
}

// Helper functions

func createTestProviderWithServer(t *testing.T, serverURL string) *Provider {
	config := Config{
		APIKey:  "sk-test",
		BaseURL: serverURL,
		Timeout: 5 * time.Second,
	}
	provider, err := New(config)
	require.NoError(t, err)
	return provider
}

func findModel(models []core.Model, id string) *core.Model {
	for _, model := range models {
		if model.ID == id {
			return &model
		}
	}
	return nil
}

// Benchmark tests

func BenchmarkProvider_CreateCompletion(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-3.5-turbo",
			"choices": [{"index": 0, "message": {"role": "assistant", "content": "Test response"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
		}`))
	}))
	defer server.Close()

	provider := &Provider{}
	provider.config = Config{
		APIKey:  "sk-test",
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}
	provider.client = &http.Client{Timeout: 30 * time.Second}
	provider.metrics = core.NewProviderMetrics()

	req := &core.CompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Hello"}},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.CreateCompletion(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProvider_ConvertRequest(b *testing.B) {
	provider := createTestProvider(b)
	req := &core.CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello, world!"},
		},
		MaxTokens:   intPtr(100),
		Temperature: float64Ptr(0.7),
		Tools: []core.Tool{
			{
				Type: "function",
				Function: core.ToolFunction{
					Name:        "get_weather",
					Description: "Get weather information",
					Parameters:  map[string]interface{}{"type": "object"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.convertRequest(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for tests

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func createTestProvider(tb testing.TB) *Provider {
	config := Config{
		APIKey:  "sk-test",
		BaseURL: "https://api.openai.com/v1",
		Timeout: 30 * time.Second,
	}
	provider, err := New(config)
	require.NoError(tb, err)
	return provider
}
