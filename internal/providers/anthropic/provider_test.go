package anthropic

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
				APIKey: "sk-ant-test",
			},
			expectError: false,
		},
		{
			name: "valid config with custom settings",
			config: Config{
				APIKey:     "sk-ant-test",
				BaseURL:    "https://custom.api.com",
				Timeout:    60 * time.Second,
				APIVersion: "2023-01-01",
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: Config{
				BaseURL: "https://api.anthropic.com",
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
				assert.Equal(t, "anthropic", provider.Name())

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
				if tt.config.APIVersion == "" {
					assert.Equal(t, APIVersion, provider.config.APIVersion)
				}
			}
		})
	}
}

func TestProvider_Name(t *testing.T) {
	provider := createTestProvider(t)
	assert.Equal(t, "anthropic", provider.Name())
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
				APIKey:  "sk-ant-test",
				BaseURL: "https://api.anthropic.com",
				Timeout: 30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty API key",
			config: Config{
				APIKey:  "",
				BaseURL: "https://api.anthropic.com",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorField:  "api_key",
		},
		{
			name: "empty base URL",
			config: Config{
				APIKey:  "sk-ant-test",
				BaseURL: "",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorField:  "base_url",
		},
		{
			name: "zero timeout",
			config: Config{
				APIKey:  "sk-ant-test",
				BaseURL: "https://api.anthropic.com",
				Timeout: 0,
			},
			expectError: true,
			errorField:  "timeout",
		},
		{
			name: "negative timeout",
			config: Config{
				APIKey:  "sk-ant-test",
				BaseURL: "https://api.anthropic.com",
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
				Model: "claude-3-sonnet-20240229",
				Messages: []core.Message{
					{Role: "user", Content: "Hello, Claude!"},
				},
			},
			mockResponse: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "Hello! How can I assist you today?"
				}],
				"model": "claude-3-sonnet-20240229",
				"stop_reason": "end_turn",
				"stop_sequence": null,
				"usage": {
					"input_tokens": 10,
					"output_tokens": 12
				}
			}`,
			mockStatusCode: 200,
			checkResponse: func(t *testing.T, resp *core.CompletionResponse) {
				assert.Equal(t, "msg_123", resp.ID)
				assert.Equal(t, "chat.completion", resp.Object)
				assert.Equal(t, "claude-3-sonnet-20240229", resp.Model)
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
				assert.Equal(t, "Hello! How can I assist you today?", resp.Choices[0].Message.Content)
				assert.Equal(t, "end_turn", resp.Choices[0].FinishReason)
				assert.Equal(t, 10, resp.Usage.PromptTokens)
				assert.Equal(t, 12, resp.Usage.CompletionTokens)
				assert.Equal(t, 22, resp.Usage.TotalTokens)
				assert.Equal(t, "anthropic", resp.Provider)
			},
		},
		{
			name: "completion with system message",
			request: &core.CompletionRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []core.Message{
					{Role: "system", Content: "You are a helpful assistant."},
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"id": "msg_456",
				"type": "message",
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "Hi there! I'm here to help."
				}],
				"model": "claude-3-sonnet-20240229",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 15,
					"output_tokens": 10
				}
			}`,
			mockStatusCode: 200,
			checkResponse: func(t *testing.T, resp *core.CompletionResponse) {
				assert.Equal(t, "msg_456", resp.ID)
				assert.Equal(t, "Hi there! I'm here to help.", resp.Choices[0].Message.Content)
			},
		},
		{
			name: "API error response",
			request: &core.CompletionRequest{
				Model: "invalid-model",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"type": "error",
				"error": {
					"type": "invalid_request_error",
					"message": "model: invalid model name 'invalid-model'"
				}
			}`,
			mockStatusCode: 400,
			expectedError:  "API error from anthropic (400/invalid_request_error)",
		},
		{
			name: "rate limit error",
			request: &core.CompletionRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			mockResponse: `{
				"type": "error",
				"error": {
					"type": "rate_limit_error",
					"message": "Rate limit exceeded"
				}
			}`,
			mockStatusCode: 429,
			expectedError:  "API error from anthropic (429/rate_limit_error): Rate limit exceeded",
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
				assert.Equal(t, "/v1/messages", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.True(t, strings.HasPrefix(r.Header.Get("x-api-key"), "sk-ant-"))
				assert.Equal(t, APIVersion, r.Header.Get("anthropic-version"))

				// Verify request body if not expecting validation error
				if !strings.Contains(tt.expectedError, "invalid request") {
					var req anthropicCompletionRequest
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(body, &req))
					assert.Equal(t, tt.request.Model, req.Model)

					// Check that system messages are properly converted
					if hasSystemMessage(tt.request.Messages) {
						assert.NotEmpty(t, req.System)
					}
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
		w.Write([]byte(`{"id": "test", "type": "message", "content": []}`))
	}))
	defer server.Close()

	provider := createTestProviderWithServer(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &core.CompletionRequest{
		Model:    "claude-3-sonnet-20240229",
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
				Model: "claude-3-sonnet-20240229",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
				Stream: true,
			},
			mockResponse: `data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-3-sonnet-20240229","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}

data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" there!"}}

data: {"type":"content_block_stop","index":0}

data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}

data: {"type":"message_stop","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":5}}}

`,
			checkChunks: func(t *testing.T, chunks []core.StreamChunk) {
				assert.True(t, len(chunks) >= 3)

				// Check first chunk (message_start)
				firstChunk := chunks[0]
				assert.Equal(t, "msg_123", firstChunk.ID)
				assert.Equal(t, "claude-3-sonnet-20240229", firstChunk.Model)
				assert.Equal(t, "anthropic", firstChunk.Provider)

				// Check that we have content chunks
				var hasContent bool
				for _, chunk := range chunks {
					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Content != "" {
						hasContent = true
						break
					}
				}
				assert.True(t, hasContent, "Expected at least one chunk with content")

				// Check final chunk has usage and is marked done
				lastChunk := chunks[len(chunks)-1]
				if lastChunk.Done && lastChunk.Usage != nil {
					assert.Equal(t, 10, lastChunk.Usage.PromptTokens)
					assert.Equal(t, 5, lastChunk.Usage.CompletionTokens)
					assert.Equal(t, 15, lastChunk.Usage.TotalTokens)
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
			mockResponse:  `{"type": "error", "error": {"message": "Model not found", "type": "invalid_request_error"}}`,
			expectedError: "API error from anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v1/messages", r.URL.Path)
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
	provider := createTestProvider(t)
	ctx := context.Background()

	models, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Len(t, models, 3) // opus, sonnet, haiku

	// Check Claude-3-Opus
	opus := findModel(models, "claude-3-opus-20240229")
	assert.NotNil(t, opus)
	assert.Equal(t, "anthropic", opus.Provider)
	assert.False(t, opus.SupportsFunctions) // Anthropic doesn't support function calling
	assert.True(t, opus.SupportsStreaming)
	assert.True(t, opus.SupportsVision)
	assert.NotNil(t, opus.MaxTokens)
	assert.Equal(t, 4096, *opus.MaxTokens)
	assert.NotNil(t, opus.InputCostPer1K)
	assert.Equal(t, 0.015, *opus.InputCostPer1K)

	// Check Claude-3-Sonnet
	sonnet := findModel(models, "claude-3-sonnet-20240229")
	assert.NotNil(t, sonnet)
	assert.Equal(t, "anthropic", sonnet.Provider)
	assert.True(t, sonnet.SupportsStreaming)
	assert.True(t, sonnet.SupportsVision)
	assert.NotNil(t, sonnet.InputCostPer1K)
	assert.Equal(t, 0.003, *sonnet.InputCostPer1K)

	// Check Claude-3-Haiku
	haiku := findModel(models, "claude-3-haiku-20240307")
	assert.NotNil(t, haiku)
	assert.Equal(t, "anthropic", haiku.Provider)
	assert.True(t, haiku.SupportsStreaming)
	assert.True(t, haiku.SupportsVision)
	assert.NotNil(t, haiku.InputCostPer1K)
	assert.Equal(t, 0.00025, *haiku.InputCostPer1K)
}

func TestProvider_GetModels_Caching(t *testing.T) {
    if os.Getenv("CI_SANDBOX") == "1" {
        t.Skip("skipping listener-based test in sandbox (CI_SANDBOX=1)")
    }
	provider := createTestProvider(t)
	provider.modelsCacheTTL = 1 * time.Second
	ctx := context.Background()

	// First call
	models1, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	firstCallTime := time.Now()

	// Second call within cache TTL
	models2, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(models1), len(models2))

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Third call after cache expiry
	models3, err := provider.GetModels(ctx)
	assert.NoError(t, err)
	assert.True(t, time.Since(firstCallTime) > 1*time.Second)
	assert.Equal(t, len(models1), len(models3))
}

func TestProvider_CalculateCost(t *testing.T) {
	provider := createTestProvider(t)

	tests := []struct {
		model    string
		usage    core.Usage
		expected float64
	}{
		{
			model: "claude-3-opus-20240229",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.015/1000 + 500*0.075/1000, // 0.0525
		},
		{
			model: "claude-3-sonnet-20240229",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.003/1000 + 500*0.015/1000, // 0.0105
		},
		{
			model: "claude-3-haiku-20240307",
			usage: core.Usage{
				PromptTokens:     1000,
				CompletionTokens: 500,
			},
			expected: 1000*0.00025/1000 + 500*0.00125/1000, // 0.000875
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

func TestProvider_ConvertRequest_SystemMessages(t *testing.T) {
	provider := createTestProvider(t)

	// Test system message conversion
	req := &core.CompletionRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []core.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "system", Content: "Be concise."},
		},
	}

	anthropicReq, err := provider.convertRequest(req)
	assert.NoError(t, err)

	// System messages should be combined
	expectedSystem := "You are a helpful assistant.\n\nBe concise."
	assert.Equal(t, expectedSystem, anthropicReq.System)

	// Only user and assistant messages should remain
	assert.Len(t, anthropicReq.Messages, 2)
	assert.Equal(t, "user", anthropicReq.Messages[0].Role)
	assert.Equal(t, "Hello", anthropicReq.Messages[0].Content)
	assert.Equal(t, "assistant", anthropicReq.Messages[1].Role)
	assert.Equal(t, "Hi there!", anthropicReq.Messages[1].Content)
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
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Success"}],
			"model": "claude-3-sonnet-20240229",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 5, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:     "sk-ant-test",
		BaseURL:    server.URL,
		MaxRetries: 3,
		Timeout:    5 * time.Second,
	}
	provider, err := New(config)
	require.NoError(t, err)

	req := &core.CompletionRequest{
		Model:    "claude-3-sonnet-20240229",
		Messages: []core.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := provider.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, retryCount) // Should have retried twice, succeeded on third
	assert.Equal(t, "Success", resp.Choices[0].Message.Content)
}

// Helper functions

func createTestProvider(t testing.TB) *Provider {
	config := Config{
		APIKey:  "sk-ant-test",
		BaseURL: "https://api.anthropic.com",
		Timeout: 30 * time.Second,
	}
	provider, err := New(config)
	require.NoError(t, err)
	return provider
}

func createTestProviderWithServer(t *testing.T, serverURL string) *Provider {
	config := Config{
		APIKey:  "sk-ant-test",
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

func hasSystemMessage(messages []core.Message) bool {
	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			return true
		}
	}
	return false
}

// Benchmark tests

func BenchmarkProvider_CreateCompletion(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{"type": "text", "text": "Test response"}],
			"model": "claude-3-sonnet-20240229",
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 5, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider := &Provider{}
	provider.config = Config{
		APIKey:  "sk-ant-test",
		BaseURL: server.URL,
		Timeout: 30 * time.Second,
	}
	provider.client = &http.Client{Timeout: 30 * time.Second}
	provider.metrics = core.NewProviderMetrics()

	req := &core.CompletionRequest{
		Model:    "claude-3-sonnet-20240229",
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
		Model: "claude-3-sonnet-20240229",
		Messages: []core.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello, world!"},
		},
		MaxTokens:   intPtr(100),
		Temperature: float64Ptr(0.7),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.convertRequest(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
