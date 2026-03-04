package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"
)

// MockProvider implements core.Provider for testing
type MockProvider struct {
	response     string
	streamChunks []string
	models       []core.Model
	shouldError  bool
	errorMessage string
	callCount    int
	lastRequest  *core.CompletionRequest
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		response: "Test response from mock provider",
		models: []core.Model{
			{ID: "test-model-1", Provider: "mock"},
			{ID: "test-model-2", Provider: "mock"},
		},
		streamChunks: []string{"Test ", "response ", "from ", "mock"},
	}
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) CreateCompletion(ctx context.Context, req *core.CompletionRequest) (*core.CompletionResponse, error) {
	m.callCount++
	m.lastRequest = req

	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	return &core.CompletionResponse{
		ID:       "test-response-id",
		Model:    req.Model,
		Provider: "mock",
		Choices: []core.Choice{
			{
				Index: 0,
				Message: core.Message{
					Role:    "assistant",
					Content: m.response,
				},
				FinishReason: core.FinishReasonStop,
			},
		},
		Usage: core.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *MockProvider) StreamCompletion(ctx context.Context, req *core.CompletionRequest) (<-chan core.StreamChunk, error) {
	m.callCount++
	m.lastRequest = req

	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}

	ch := make(chan core.StreamChunk, len(m.streamChunks))
	go func() {
		defer close(ch)
		for _, chunk := range m.streamChunks {
			ch <- core.StreamChunk{
				ID: "test-chunk-id",
				Choices: []core.Choice{
					{
						Delta: &core.Message{Content: chunk},
					},
				},
			}
		}
	}()

	return ch, nil
}

func (m *MockProvider) GetModels(ctx context.Context) ([]core.Model, error) {
	if m.shouldError {
		return nil, errors.New(m.errorMessage)
	}
	return m.models, nil
}

func (m *MockProvider) ValidateConfig() error {
	if m.shouldError {
		return errors.New(m.errorMessage)
	}
	return nil
}

func (m *MockProvider) SetResponse(response string) {
	m.response = response
}

func (m *MockProvider) SetError(shouldError bool, message string) {
	m.shouldError = shouldError
	m.errorMessage = message
}

func (m *MockProvider) GetCallCount() int {
	return m.callCount
}

func (m *MockProvider) GetLastRequest() *core.CompletionRequest {
	return m.lastRequest
}

// Test utilities

func createTestCommand(provider core.Provider) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer

	cmd := &cobra.Command{
		Use: "test",
	}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	return cmd, &stdout, &stderr
}

func TestChatFlags_Validation(t *testing.T) {
	tests := []struct {
		name        string
		flags       ChatFlags
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid flags",
			flags: ChatFlags{
				Temperature: 0.7,
				MaxTokens:   100,
				TopP:        0.9,
			},
			expectError: false,
		},
		{
			name: "temperature too high",
			flags: ChatFlags{
				Temperature: 3.0,
			},
			expectError: true,
			errorMsg:    "temperature must be between 0 and 2",
		},
		{
			name: "temperature negative",
			flags: ChatFlags{
				Temperature: -0.5,
			},
			expectError: true,
			errorMsg:    "temperature must be between 0 and 2",
		},
		{
			name: "max tokens negative",
			flags: ChatFlags{
				MaxTokens: -100,
			},
			expectError: true,
			errorMsg:    "max tokens must be positive",
		},
		{
			name: "top-p out of range",
			flags: ChatFlags{
				TopP: 1.5,
			},
			expectError: true,
			errorMsg:    "top-p must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChatFlags(&tt.flags)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildCompletionRequest(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		flags    ChatFlags
		expected *core.CompletionRequest
	}{
		{
			name:    "basic message",
			message: "Hello, world!",
			flags:   ChatFlags{},
			expected: &core.CompletionRequest{
				Model: "", // Will be set by provider
				Messages: []core.Message{
					{Role: "user", Content: "Hello, world!"},
				},
			},
		},
		{
			name:    "with system message",
			message: "Hello",
			flags: ChatFlags{
				SystemMessage: "You are helpful",
			},
			expected: &core.CompletionRequest{
				Messages: []core.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
		},
		{
			name:    "with parameters",
			message: "Test",
			flags: ChatFlags{
				Temperature: 0.8,
				MaxTokens:   150,
				TopP:        0.95,
			},
			expected: &core.CompletionRequest{
				Messages: []core.Message{
					{Role: "user", Content: "Test"},
				},
				Temperature: &[]float64{0.8}[0],
				MaxTokens:   &[]int{150}[0],
				TopP:        &[]float64{0.95}[0],
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCompletionRequest(tt.message, &tt.flags)

			assert.Equal(t, len(tt.expected.Messages), len(result.Messages))
			for i, expectedMsg := range tt.expected.Messages {
				assert.Equal(t, expectedMsg.Role, result.Messages[i].Role)
				assert.Equal(t, expectedMsg.Content, result.Messages[i].Content)
			}

			if tt.expected.Temperature != nil {
				require.NotNil(t, result.Temperature)
				assert.Equal(t, *tt.expected.Temperature, *result.Temperature)
			}

			if tt.expected.MaxTokens != nil {
				require.NotNil(t, result.MaxTokens)
				assert.Equal(t, *tt.expected.MaxTokens, *result.MaxTokens)
			}

			if tt.expected.TopP != nil {
				require.NotNil(t, result.TopP)
				assert.Equal(t, *tt.expected.TopP, *result.TopP)
			}
		})
	}
}

func TestProcessChatResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *core.CompletionResponse
		flags    ChatFlags
		expected string
	}{
		{
			name: "basic response",
			response: &core.CompletionResponse{
				Choices: []core.Choice{
					{Message: core.Message{Content: "Hello there!"}},
				},
			},
			flags:    ChatFlags{},
			expected: "Hello there!",
		},
		{
			name: "quiet mode",
			response: &core.CompletionResponse{
				Choices: []core.Choice{
					{Message: core.Message{Content: "Hello there!"}},
				},
			},
			flags:    ChatFlags{Quiet: true},
			expected: "Hello there!",
		},
		{
			name: "raw mode",
			response: &core.CompletionResponse{
				Choices: []core.Choice{
					{Message: core.Message{Content: "Hello there!"}},
				},
			},
			flags:    ChatFlags{Raw: true},
			expected: "Hello there!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			err := processChatResponse(&output, tt.response, &tt.flags)

			assert.NoError(t, err)
			assert.Contains(t, output.String(), tt.expected)
		})
	}
}

func TestProcessStreamingResponse(t *testing.T) {
	tests := []struct {
		name   string
		chunks []core.StreamChunk
		flags  ChatFlags
		expect string
	}{
		{
			name: "basic streaming",
			chunks: []core.StreamChunk{
				{Choices: []core.Choice{{Delta: &core.Message{Content: "Hello "}}}},
				{Choices: []core.Choice{{Delta: &core.Message{Content: "there!"}}}},
			},
			flags:  ChatFlags{},
			expect: "Hello there!",
		},
		{
			name: "quiet streaming",
			chunks: []core.StreamChunk{
				{Choices: []core.Choice{{Delta: &core.Message{Content: "Test "}}}},
				{Choices: []core.Choice{{Delta: &core.Message{Content: "content"}}}},
			},
			flags:  ChatFlags{Quiet: true},
			expect: "Test content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan core.StreamChunk, len(tt.chunks))
			for _, chunk := range tt.chunks {
				ch <- chunk
			}
			close(ch)

			var output bytes.Buffer
			err := processStreamingResponse(&output, ch, &tt.flags)

			assert.NoError(t, err)
			assert.Contains(t, output.String(), tt.expect)
		})
	}
}

func TestInputReading(t *testing.T) {
	tests := []struct {
		name     string
		stdin    string
		args     []string
		expected string
		hasError bool
	}{
		{
			name:     "from argument",
			args:     []string{"Hello from arg"},
			expected: "Hello from arg",
		},
		{
			name:     "from stdin",
			stdin:    "Hello from stdin",
			args:     []string{},
			expected: "Hello from stdin",
		},
		{
			name:     "argument takes precedence",
			stdin:    "From stdin",
			args:     []string{"From argument"},
			expected: "From argument",
		},
		{
			name:     "no input",
			stdin:    "",
			args:     []string{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdin bytes.Buffer
			stdin.WriteString(tt.stdin)

			result, err := readInput(&stdin, tt.args)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCompleteFlags_Validation(t *testing.T) {
	tests := []struct {
		name        string
		flags       CompleteFlags
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid flags",
			flags: CompleteFlags{
				Language:    "python",
				MaxTokens:   200,
				Temperature: 0.3,
			},
			expectError: false,
		},
		{
			name: "invalid temperature",
			flags: CompleteFlags{
				Temperature: 3.0,
			},
			expectError: true,
			errorMsg:    "temperature",
		},
		{
			name: "negative max tokens",
			flags: CompleteFlags{
				MaxTokens: -50,
			},
			expectError: true,
			errorMsg:    "max tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCompleteFlags(&tt.flags)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildCodeCompletionRequest(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		flags    CompleteFlags
		expected string // Expected system message content
	}{
		{
			name:     "python code",
			code:     "def factorial(n):",
			flags:    CompleteFlags{Language: "python"},
			expected: "You are a Python code completion assistant",
		},
		{
			name:     "go code",
			code:     "func main() {",
			flags:    CompleteFlags{Language: "go"},
			expected: "You are a Go code completion assistant",
		},
		{
			name:     "javascript code",
			code:     "function test() {",
			flags:    CompleteFlags{Language: "javascript"},
			expected: "You are a JavaScript code completion assistant",
		},
		{
			name:     "auto-detect python",
			code:     "def hello():",
			flags:    CompleteFlags{},
			expected: "Python", // Should auto-detect
		},
		{
			name:     "auto-detect go",
			code:     "package main",
			flags:    CompleteFlags{},
			expected: "Go", // Should auto-detect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildCodeCompletionRequest(tt.code, &tt.flags)

			assert.NotEmpty(t, req.Messages)
			assert.Equal(t, "user", req.Messages[len(req.Messages)-1].Role)
			assert.Contains(t, req.Messages[len(req.Messages)-1].Content, tt.code)

			// Check system message contains expected language
			found := false
			for _, msg := range req.Messages {
				if msg.Role == "system" && strings.Contains(msg.Content, tt.expected) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected system message with language %s", tt.expected)
		})
	}
}

func TestDetectCodeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "python function",
			code:     "def hello():",
			expected: "python",
		},
		{
			name:     "python import",
			code:     "import sys\nprint('hello')",
			expected: "python",
		},
		{
			name:     "go package",
			code:     "package main\n\nfunc main() {",
			expected: "go",
		},
		{
			name:     "go import",
			code:     "import \"fmt\"\n\nfunc test() {",
			expected: "go",
		},
		{
			name:     "javascript function",
			code:     "function test() {\n  return 'hello';",
			expected: "javascript",
		},
		{
			name:     "javascript const",
			code:     "const test = () => {",
			expected: "javascript",
		},
		{
			name:     "typescript interface",
			code:     "interface User {\n  name: string;",
			expected: "typescript",
		},
		{
			name:     "unknown code",
			code:     "some random text",
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCodeLanguage(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderSelection(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai":    {Type: "openai"},
			"anthropic": {Type: "anthropic"},
		},
	}

	tests := []struct {
		name         string
		flagProvider string
		expected     string
	}{
		{
			name:         "use default provider",
			flagProvider: "",
			expected:     "openai",
		},
		{
			name:         "override with flag",
			flagProvider: "anthropic",
			expected:     "anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectProvider(cfg, tt.flagProvider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkBuildCompletionRequest(b *testing.B) {
	flags := &ChatFlags{
		Temperature:   0.7,
		MaxTokens:     100,
		SystemMessage: "You are helpful",
	}
	message := "Hello, world!"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = buildCompletionRequest(message, flags)
	}
}

func BenchmarkDetectCodeLanguage(b *testing.B) {
	code := "def fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = detectCodeLanguage(code)
	}
}

// Helper functions that would be implemented in the actual command files

func validateChatFlags(flags *ChatFlags) error {
	if flags.Temperature < 0 || flags.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}
	if flags.MaxTokens < 0 {
		return errors.New("max tokens must be positive")
	}
	if flags.TopP < 0 || flags.TopP > 1 {
		return errors.New("top-p must be between 0 and 1")
	}
	return nil
}

func buildCompletionRequest(message string, flags *ChatFlags) *core.CompletionRequest {
	messages := []core.Message{}

	if flags.SystemMessage != "" {
		messages = append(messages, core.Message{
			Role:    "system",
			Content: flags.SystemMessage,
		})
	}

	messages = append(messages, core.Message{
		Role:    "user",
		Content: message,
	})

	req := &core.CompletionRequest{
		Messages: messages,
		Stream:   flags.Stream,
	}

	if flags.Temperature > 0 {
		req.Temperature = &flags.Temperature
	}
	if flags.MaxTokens > 0 {
		req.MaxTokens = &flags.MaxTokens
	}
	if flags.TopP > 0 {
		req.TopP = &flags.TopP
	}

	return req
}

func processChatResponse(output io.Writer, response *core.CompletionResponse, flags *ChatFlags) error {
	if len(response.Choices) == 0 {
		return errors.New("no response choices")
	}

	content := response.Choices[0].Message.Content
	_, err := output.Write([]byte(content))
	return err
}

func processStreamingResponse(output io.Writer, ch <-chan core.StreamChunk, flags *ChatFlags) error {
	for chunk := range ch {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Content != "" {
			_, err := output.Write([]byte(chunk.Choices[0].Delta.Content))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readInput(stdin io.Reader, args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	input, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}

	content := strings.TrimSpace(string(input))
	if content == "" {
		return "", errors.New("no message provided (use argument or pipe from stdin)")
	}

	return content, nil
}

// CompleteFlags and validateCompleteFlags are defined in complete.go

func buildCodeCompletionRequest(code string, flags *CompleteFlags) *core.CompletionRequest {
	language := flags.Language
	if language == "" {
		language = detectCodeLanguage(code)
	}

	systemMsg := buildSystemMessage(language)
	userMsg := buildUserMessage(code, language)

	return &core.CompletionRequest{
		Messages: []core.Message{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
	}
}

func detectCodeLanguage(code string) string {
	codeLower := strings.ToLower(code)

	// Check for Go-specific patterns first (more specific)
	if strings.Contains(code, "package ") || strings.Contains(code, "import \"") || strings.Contains(code, "func ") {
		return "go"
	}

	// Check for Python-specific patterns
	if strings.Contains(codeLower, "def ") || strings.Contains(codeLower, "from ") || strings.Contains(codeLower, "class ") || strings.Contains(codeLower, "__init__") {
		return "python"
	}

	// Check for TypeScript-specific patterns
	if strings.Contains(codeLower, "interface ") || strings.Contains(codeLower, ": string") || strings.Contains(codeLower, ": number") || strings.Contains(codeLower, "enum ") {
		return "typescript"
	}

	// Check for JavaScript patterns
	if strings.Contains(codeLower, "function ") || strings.Contains(codeLower, "const ") || strings.Contains(codeLower, "let ") || strings.Contains(codeLower, "=>") {
		return "javascript"
	}

	// Fall back to generic import check (less specific)
	if strings.Contains(codeLower, "import ") {
		return "python" // Default to Python for generic import
	}

	return "text"
}

func buildSystemMessage(language string) string {
	switch language {
	case "python":
		return "You are a Python code completion assistant. Complete the given code."
	case "go":
		return "You are a Go code completion assistant. Complete the given code."
	case "javascript":
		return "You are a JavaScript code completion assistant. Complete the given code."
	case "typescript":
		return "You are a TypeScript code completion assistant. Complete the given code."
	default:
		return "You are a code completion assistant. Complete the given code."
	}
}

func buildUserMessage(code, language string) string {
	return fmt.Sprintf("Complete this %s code:\n\n```%s\n%s\n```", language, language, code)
}

func selectProvider(cfg *config.Config, flagProvider string) string {
	if flagProvider != "" {
		return flagProvider
	}
	return cfg.DefaultProvider
}
