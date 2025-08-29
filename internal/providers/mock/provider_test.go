package mock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/gollm/internal/core"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected func(*testing.T, *Provider)
	}{
		{
			name:   "default config",
			config: Config{},
			expected: func(t *testing.T, p *Provider) {
				assert.Equal(t, "mock", p.config.Name)
				assert.Equal(t, "mock-gpt-3.5-turbo", p.config.DefaultModel)
				assert.Len(t, p.models, 3)
				assert.True(t, p.config.EnableStreaming)
			},
		},
		{
			name: "custom config",
			config: Config{
				Name:            "test-mock",
				DefaultModel:    "test-model",
				SupportedModels: []string{"test-model", "test-model-2"},
				Latency:         50 * time.Millisecond,
				EnableStreaming: false,
			},
			expected: func(t *testing.T, p *Provider) {
				assert.Equal(t, "test-mock", p.config.Name)
				assert.Equal(t, "test-model", p.config.DefaultModel)
				assert.Len(t, p.models, 2)
				assert.False(t, p.config.EnableStreaming)
				assert.Equal(t, 50*time.Millisecond, p.config.Latency)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(tt.config)
			require.NotNil(t, provider)
			tt.expected(t, provider)
		})
	}
}

func TestProvider_Name(t *testing.T) {
	provider := New(Config{Name: "test-provider"})
	assert.Equal(t, "test-provider", provider.Name())
}

func TestProvider_CreateCompletion(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*Provider)
		request        *core.CompletionRequest
		expectedError  string
		expectedText   string
		checkResponse  func(*testing.T, *core.CompletionResponse)
	}{
		{
			name:  "successful basic completion",
			setup: func(p *Provider) {},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expectedText: "Mock response to: Hello",
			checkResponse: func(t *testing.T, resp *core.CompletionResponse) {
				assert.NotEmpty(t, resp.ID)
				assert.Equal(t, "chat.completion", resp.Object)
				assert.Equal(t, "mock-gpt-3.5-turbo", resp.Model)
				assert.Len(t, resp.Choices, 1)
				assert.Equal(t, core.RoleAssistant, resp.Choices[0].Message.Role)
				assert.Equal(t, core.FinishReasonStop, resp.Choices[0].FinishReason)
				assert.NotZero(t, resp.Usage.TotalTokens)
			},
		},
		{
			name: "custom response",
			setup: func(p *Provider) {
				p.SetResponse("mock-gpt-3.5-turbo", "Custom test response")
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expectedText: "Custom test response",
		},
		{
			name: "global response override",
			setup: func(p *Provider) {
				p.SetGlobalResponse("Global response")
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-4",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedText: "Global response",
		},
		{
			name: "model-specific error",
			setup: func(p *Provider) {
				p.SetError("mock-gpt-4", errors.New("model error"))
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-4",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "model error",
		},
		{
			name: "global error",
			setup: func(p *Provider) {
				p.SetGlobalError(errors.New("global error"))
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "global error",
		},
		{
			name: "invalid request",
			setup: func(p *Provider) {},
			request: &core.CompletionRequest{
				Model:    "", // Empty model
				Messages: []core.Message{},
			},
			expectedError: "invalid request",
		},
		{
			name: "fail after count",
			setup: func(p *Provider) {
				p.SetFailAfterCount(2)
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "", // First request should succeed
		},
		{
			name: "rate limit after count",
			setup: func(p *Provider) {
				p.SetRateLimitAfter(1)
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "", // First request should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(DefaultConfig())
			tt.setup(provider)

			ctx := context.Background()
			resp, err := provider.CreateCompletion(ctx, tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.expectedText != "" {
				assert.Equal(t, tt.expectedText, resp.Choices[0].Message.Content)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestProvider_CreateCompletion_FailAfterCount(t *testing.T) {
	provider := New(DefaultConfig())
	provider.SetFailAfterCount(2)

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test"},
		},
	}

	ctx := context.Background()

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		resp, err := provider.CreateCompletion(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	}

	// Third request should fail
	resp, err := provider.CreateCompletion(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Mock failure after request count")
}

func TestProvider_CreateCompletion_RateLimit(t *testing.T) {
	provider := New(DefaultConfig())
	provider.SetRateLimitAfter(1)

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test"},
		},
	}

	ctx := context.Background()

	// First request should succeed
	resp, err := provider.CreateCompletion(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Second request should be rate limited
	resp, err = provider.CreateCompletion(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *core.APIError
	if assert.True(t, errors.As(err, &apiErr)) {
		assert.Equal(t, 429, apiErr.StatusCode)
		assert.Equal(t, "rate_limit_exceeded", apiErr.Code)
	}
}

func TestProvider_CreateCompletion_Latency(t *testing.T) {
	provider := New(DefaultConfig())
	provider.SetLatency("mock-gpt-3.5-turbo", 100*time.Millisecond)

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test"},
		},
	}

	ctx := context.Background()
	start := time.Now()

	resp, err := provider.CreateCompletion(ctx, req)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, elapsed >= 100*time.Millisecond)
}

func TestProvider_CreateCompletion_ContextCancellation(t *testing.T) {
	provider := New(DefaultConfig())
	provider.SetGlobalLatency(1 * time.Second)

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := provider.CreateCompletion(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestProvider_StreamCompletion(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*Provider)
		request       *core.CompletionRequest
		expectedError string
		checkStream   func(*testing.T, <-chan core.StreamChunk)
	}{
		{
			name:  "successful streaming",
			setup: func(p *Provider) {},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Hello world test"},
				},
			},
			checkStream: func(t *testing.T, chunks <-chan core.StreamChunk) {
				var receivedChunks []core.StreamChunk
				var totalContent string

				for chunk := range chunks {
					assert.NoError(t, chunk.Error)
					receivedChunks = append(receivedChunks, chunk)

					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
						totalContent += chunk.Choices[0].Delta.Content
					}
				}

				assert.True(t, len(receivedChunks) > 1)
				assert.True(t, receivedChunks[len(receivedChunks)-1].Done)
				assert.Contains(t, totalContent, "Mock response to: Hello world test")

				// Check final chunk has usage
				finalChunk := receivedChunks[len(receivedChunks)-1]
				assert.NotNil(t, finalChunk.Usage)
				assert.True(t, finalChunk.Usage.TotalTokens > 0)
			},
		},
		{
			name: "custom stream chunks",
			setup: func(p *Provider) {
				p.SetStreamChunks("mock-gpt-3.5-turbo", []string{"Hello", " there", "!"})
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			checkStream: func(t *testing.T, chunks <-chan core.StreamChunk) {
				var content []string
				for chunk := range chunks {
					if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
						content = append(content, chunk.Choices[0].Delta.Content)
					}
				}
				expected := []string{"Hello", " there", "!"}
				assert.Equal(t, expected, content)
			},
		},
		{
			name: "streaming not supported",
			setup: func(p *Provider) {
				p.config.EnableStreaming = false
			},
			request: &core.CompletionRequest{
				Model: "mock-gpt-3.5-turbo",
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
			},
			expectedError: "streaming not supported",
		},
		{
			name: "invalid request",
			setup: func(p *Provider) {},
			request: &core.CompletionRequest{
				Model:    "",
				Messages: []core.Message{},
			},
			expectedError: "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(DefaultConfig())
			tt.setup(provider)

			ctx := context.Background()
			chunks, err := provider.StreamCompletion(ctx, tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, chunks)

			if tt.checkStream != nil {
				tt.checkStream(t, chunks)
			}
		})
	}
}

func TestProvider_StreamCompletion_ContextCancellation(t *testing.T) {
	provider := New(DefaultConfig())

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "Test message"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	chunks, err := provider.StreamCompletion(ctx, req)
	require.NoError(t, err)

	// Cancel context after receiving first chunk
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	var hasError bool
	for chunk := range chunks {
		if chunk.Error != nil {
			hasError = true
			assert.Equal(t, context.Canceled, chunk.Error)
			break
		}
	}

	assert.True(t, hasError)
}

func TestProvider_GetModels(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedCount int
		checkModels   func(*testing.T, []core.Model)
	}{
		{
			name:          "default models",
			config:        DefaultConfig(),
			expectedCount: 3,
			checkModels: func(t *testing.T, models []core.Model) {
				modelIDs := make([]string, len(models))
				for i, model := range models {
					modelIDs[i] = model.ID
					assert.Equal(t, "mock", model.Provider)
					assert.NotNil(t, model.MaxTokens)
					assert.True(t, model.SupportsFunctions)
					assert.True(t, model.SupportsStreaming)
				}
				assert.Contains(t, modelIDs, "mock-gpt-3.5-turbo")
				assert.Contains(t, modelIDs, "mock-gpt-4")
				assert.Contains(t, modelIDs, "mock-claude-3")
			},
		},
		{
			name: "custom models",
			config: Config{
				Name:            "test",
				SupportedModels: []string{"custom-model-1", "custom-model-2"},
				EnableFunctions: false,
				EnableStreaming: false,
			},
			expectedCount: 2,
			checkModels: func(t *testing.T, models []core.Model) {
				assert.Equal(t, "custom-model-1", models[0].ID)
				assert.Equal(t, "custom-model-2", models[1].ID)
				assert.False(t, models[0].SupportsFunctions)
				assert.False(t, models[0].SupportsStreaming)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(tt.config)
			ctx := context.Background()

			models, err := provider.GetModels(ctx)

			assert.NoError(t, err)
			assert.Len(t, models, tt.expectedCount)

			if tt.checkModels != nil {
				tt.checkModels(t, models)
			}
		})
	}
}

func TestProvider_GetModels_ContextCancellation(t *testing.T) {
	provider := New(DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	models, err := provider.GetModels(ctx)
	assert.Error(t, err)
	assert.Nil(t, models)
	assert.Equal(t, context.Canceled, err)
}

func TestProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorField  string
	}{
		{
			name:        "valid default config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "empty name",
			config: Config{
				SupportedModels: []string{"model-1"},
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "no supported models",
			config: Config{
				Name:            "test",
				SupportedModels: []string{},
			},
			expectError: true,
			errorField:  "supported_models",
		},
		{
			name: "invalid error rate - negative",
			config: Config{
				Name:            "test",
				SupportedModels: []string{"model-1"},
				ErrorRate:       -0.1,
			},
			expectError: true,
			errorField:  "error_rate",
		},
		{
			name: "invalid error rate - too high",
			config: Config{
				Name:            "test",
				SupportedModels: []string{"model-1"},
				ErrorRate:       1.5,
			},
			expectError: true,
			errorField:  "error_rate",
		},
		{
			name: "valid error rate",
			config: Config{
				Name:            "test",
				SupportedModels: []string{"model-1"},
				ErrorRate:       0.1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := New(tt.config)
			err := provider.ValidateConfig()

			if tt.expectError {
				assert.Error(t, err)
				var validationErr *core.ValidationError
				if assert.True(t, errors.As(err, &validationErr)) {
					assert.Contains(t, validationErr.Field, tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_ConfigurationMethods(t *testing.T) {
	provider := New(DefaultConfig())

	// Test SetResponse and SetGlobalResponse
	provider.SetResponse("model-1", "Model 1 response")
	provider.SetGlobalResponse("Global response")

	req := &core.CompletionRequest{
		Model:    "model-1",
		Messages: []core.Message{{Role: "user", Content: "Test"}},
	}

	resp, err := provider.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Model 1 response", resp.Choices[0].Message.Content)

	// Test with different model (should use global)
	req.Model = "model-2"
	resp, err = provider.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Global response", resp.Choices[0].Message.Content)

	// Test SetError
	testErr := errors.New("test error")
	provider.SetError("model-error", testErr)

	req.Model = "model-error"
	resp, err = provider.CreateCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Equal(t, testErr, err)

	// Test SetGlobalError
	globalErr := errors.New("global error")
	provider.SetGlobalError(globalErr)

	req.Model = "any-model"
	resp, err = provider.CreateCompletion(context.Background(), req)
	assert.Error(t, err)
	assert.Equal(t, globalErr, err)

	// Reset global error
	provider.SetGlobalError(nil)

	// Test SetLatency
	provider.SetLatency("slow-model", 50*time.Millisecond)

	req.Model = "slow-model"
	start := time.Now()
	resp, err = provider.CreateCompletion(context.Background(), req)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, elapsed >= 50*time.Millisecond)
}

func TestProvider_ResetCounters(t *testing.T) {
	provider := New(DefaultConfig())
	provider.SetFailAfterCount(2)

	req := &core.CompletionRequest{
		Model:    "mock-gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Test"}},
	}

	ctx := context.Background()

	// Make requests to increase counter
	for i := 0; i < 2; i++ {
		_, err := provider.CreateCompletion(ctx, req)
		assert.NoError(t, err)
	}

	// Next request should fail
	_, err := provider.CreateCompletion(ctx, req)
	assert.Error(t, err)

	// Reset counters
	provider.ResetCounters()

	// Should work again
	_, err = provider.CreateCompletion(ctx, req)
	assert.NoError(t, err)
}

func TestProvider_GetMetrics(t *testing.T) {
	provider := New(DefaultConfig())

	req := &core.CompletionRequest{
		Model:    "mock-gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Test message"}},
	}

	ctx := context.Background()

	// Make a successful request
	_, err := provider.CreateCompletion(ctx, req)
	assert.NoError(t, err)

	// Make a failed request
	provider.SetError("mock-gpt-3.5-turbo", errors.New("test error"))
	_, err = provider.CreateCompletion(ctx, req)
	assert.Error(t, err)

	// Check metrics
	metrics := provider.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessRequests)
	assert.Equal(t, int64(1), metrics.ErrorRequests)
	assert.True(t, metrics.TotalTokensUsed > 0)
	assert.True(t, metrics.TotalCost > 0)
}

func TestProvider_UsageCalculation(t *testing.T) {
	provider := New(DefaultConfig())

	req := &core.CompletionRequest{
		Model: "mock-gpt-3.5-turbo",
		Messages: []core.Message{
			{Role: "user", Content: "This is a test message with multiple words"},
		},
	}

	ctx := context.Background()
	resp, err := provider.CreateCompletion(ctx, req)

	assert.NoError(t, err)
	assert.True(t, resp.Usage.PromptTokens > 0)
	assert.True(t, resp.Usage.CompletionTokens > 0)
	assert.Equal(t, resp.Usage.PromptTokens+resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
}

// Benchmark tests

func BenchmarkProvider_CreateCompletion(b *testing.B) {
	provider := New(DefaultConfig())
	req := &core.CompletionRequest{
		Model:    "mock-gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Test"}},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.CreateCompletion(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProvider_StreamCompletion(b *testing.B) {
	provider := New(DefaultConfig())
	req := &core.CompletionRequest{
		Model:    "mock-gpt-3.5-turbo",
		Messages: []core.Message{{Role: "user", Content: "Test"}},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		chunks, err := provider.StreamCompletion(ctx, req)
		if err != nil {
			b.Fatal(err)
		}

		// Consume all chunks
		for range chunks {
		}
	}
}

func BenchmarkProvider_GetModels(b *testing.B) {
	provider := New(DefaultConfig())
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := provider.GetModels(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
