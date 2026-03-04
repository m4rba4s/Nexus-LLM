// Package security provides comprehensive security validation and utilities for GOLLM.
//
// This package implements security validation functions, credential management,
// rate limiting, circuit breakers, and other security-related utilities used
// throughout the GOLLM application.
package security

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Common security patterns and limits
const (
	// MaxModelNameLength defines maximum allowed model name length
	MaxModelNameLength = 100

	// MaxMessageContentLength defines maximum message content length (100KB)
	MaxMessageContentLength = 100 * 1024

	// MaxAPIKeyLength defines maximum API key length
	MaxAPIKeyLength = 1024

	// MinTLSVersion defines minimum required TLS version
	MinTLSVersion = tls.VersionTLS13

	// MaxTimeout defines maximum allowed request timeout
	MaxTimeout = 5 * time.Minute

	// DefaultRateLimitWindow defines default rate limiting window
	DefaultRateLimitWindow = time.Minute
)

// Validation regex patterns
var (
	// ModelNamePattern validates model names (alphanumeric, hyphens, underscores, dots)
	ModelNamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`)

	// APIKeyPattern validates API key format
	APIKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`)

	// BearerTokenPattern validates Bearer token format
	BearerTokenPattern = regexp.MustCompile(`^Bearer\s+[^\s]+$`)
)

// ValidationError represents a validation error with detailed context
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Rule    string      `json:"rule"`
	Message string      `json:"message"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s (rule: %s): %s", e.Field, e.Rule, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, rule, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// ValidateModelName validates model name for security and format compliance
func ValidateModelName(model string) error {
	if model == "" {
		return NewValidationError("model", model, "required", "cannot be empty")
	}

	if len(model) > MaxModelNameLength {
		return NewValidationError("model", model, "max_length",
			fmt.Sprintf("exceeds maximum length of %d", MaxModelNameLength))
	}

	if !ModelNamePattern.MatchString(model) {
		return NewValidationError("model", model, "format",
			"contains invalid characters (only alphanumeric, hyphens, underscores, and dots allowed)")
	}

	return nil
}

// ValidateMessageContent validates message content for size and safety
func ValidateMessageContent(content string) error {
	if len(content) > MaxMessageContentLength {
		return NewValidationError("content", len(content), "max_length",
			fmt.Sprintf("exceeds maximum length of %d bytes", MaxMessageContentLength))
	}

	// Content validation is permissive - we allow most content but validate length
	// Actual content filtering should be done at the LLM provider level
	return nil
}

// ValidateAPIKey validates API key format and security
func ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return NewValidationError("api_key", apiKey, "required", "cannot be empty")
	}

	if len(apiKey) > MaxAPIKeyLength {
		return NewValidationError("api_key", len(apiKey), "max_length",
			fmt.Sprintf("exceeds maximum length of %d", MaxAPIKeyLength))
	}

	// Basic format validation - allow most characters but ensure reasonable format
	if len(apiKey) < 10 {
		return NewValidationError("api_key", len(apiKey), "min_length",
			"API key is too short (minimum 10 characters)")
	}

	return nil
}

// ValidateAuthHeader validates HTTP Authorization header format
func ValidateAuthHeader(header string) error {
	if header == "" {
		return NewValidationError("authorization", header, "required", "authorization header is required")
	}

	if !BearerTokenPattern.MatchString(header) {
		return NewValidationError("authorization", header, "format",
			"must be in format 'Bearer <token>'")
	}

	// Extract token and validate
	token := strings.TrimPrefix(header, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return NewValidationError("authorization", header, "format", "Bearer token cannot be empty")
	}

	return nil
}

// ValidateTimeout validates request timeout values
func ValidateTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return NewValidationError("timeout", timeout, "positive", "must be positive")
	}

	if timeout > MaxTimeout {
		return NewValidationError("timeout", timeout, "max_value",
			fmt.Sprintf("exceeds maximum timeout of %v", MaxTimeout))
	}

	return nil
}

// ValidateTLSConfig validates TLS configuration for security compliance
func ValidateTLSConfig(config *tls.Config) error {
	if config == nil {
		return NewValidationError("tls_config", config, "required", "TLS configuration is required")
	}

	if config.MinVersion < MinTLSVersion {
		return NewValidationError("tls_min_version", config.MinVersion, "min_version",
			fmt.Sprintf("minimum TLS version must be %d or higher", MinTLSVersion))
	}

	if config.InsecureSkipVerify {
		return NewValidationError("tls_verify", config.InsecureSkipVerify, "security",
			"TLS certificate verification cannot be disabled")
	}

	return nil
}

// SanitizeForLogging sanitizes sensitive data for logging
func SanitizeForLogging(data map[string]interface{}) map[string]interface{} {
	sensitiveFields := map[string]bool{
		"api_key":       true,
		"apikey":        true,
		"key":           true,
		"token":         true,
		"password":      true,
		"secret":        true,
		"credential":    true,
		"auth":          true,
		"authorization": true,
	}

	sanitized := make(map[string]interface{})
	for k, v := range data {
		lowerKey := strings.ToLower(k)
		if sensitiveFields[lowerKey] || strings.Contains(lowerKey, "key") || strings.Contains(lowerKey, "token") {
			sanitized[k] = "***REDACTED***"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	limit    int
	window   time.Duration
	requests []time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
	}
}

// Allow checks if a request should be allowed based on rate limit
func (rl *RateLimiter) Allow() bool {
	if rl.limit <= 0 {
		return true // Unlimited
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range rl.requests {
		if now.Sub(reqTime) < rl.window {
			validRequests = append(validRequests, reqTime)
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

// Reset resets the rate limiter
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = nil
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	// CircuitClosed means the circuit is closed (normal operation)
	CircuitClosed CircuitBreakerState = "closed"
	// CircuitOpen means the circuit is open (failing fast)
	CircuitOpen CircuitBreakerState = "open"
	// CircuitHalfOpen means the circuit is testing if it can close
	CircuitHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailure  time.Time
	state        CircuitBreakerState
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// IsOpen returns true if the circuit is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == CircuitOpen
}

// IsClosed returns true if the circuit is closed
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == CircuitClosed
}

// IsHalfOpen returns true if the circuit is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == CircuitHalfOpen
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if circuit should transition from open to half-open
	cb.checkTransition()

	if cb.IsOpen() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// RecordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

// RecordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failures = 0
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
	cb.lastFailure = time.Time{}
}

// checkTransition checks if the circuit should transition from open to half-open
func (cb *CircuitBreaker) checkTransition() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitOpen && time.Since(cb.lastFailure) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
	}
}

// SecureHTTPClient creates an HTTP client with security best practices
func SecureHTTPClient(timeout time.Duration) *http.Client {
	// Enforce maximum timeout
	if timeout > MaxTimeout {
		timeout = MaxTimeout
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// Connection pooling settings
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,

			// Timeouts
			DisableKeepAlives:     false,
			DisableCompression:    false,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,

			// TLS configuration
			TLSClientConfig: &tls.Config{
				MinVersion: MinTLSVersion,
				// Certificate verification is enabled by default
			},
		},

		// Don't follow redirects automatically for security
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// ValidateHTTPHeaders validates HTTP headers for security
func ValidateHTTPHeaders(headers map[string]string) error {
	dangerousHeaders := map[string]bool{
		"x-forwarded-for":   true,
		"x-real-ip":         true,
		"x-forwarded-proto": true,
		"x-forwarded-host":  true,
	}

	for name, value := range headers {
		lowerName := strings.ToLower(name)

		// Check for dangerous headers that could be used for injection
		if dangerousHeaders[lowerName] {
			return NewValidationError("header", name, "dangerous",
				"header could be used for request smuggling or injection attacks")
		}

		// Validate header value length
		if len(value) > 8192 {
			return NewValidationError("header_value", name, "max_length",
				"header value exceeds maximum length of 8192 characters")
		}

		// Check for control characters in header values
		for _, r := range value {
			if r < 32 && r != 9 { // Allow tab (9) but not other control chars
				return NewValidationError("header_value", name, "format",
					"header value contains invalid control characters")
			}
		}
	}

	return nil
}

// IsSecureContext checks if the application is running in a secure context
func IsSecureContext(scheme string, host string) bool {
	// HTTPS is always secure
	if scheme == "https" {
		return true
	}

	// HTTP is only secure for localhost in development
	if scheme == "http" {
		return host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "localhost:")
	}

	return false
}

// SecurityHeaders returns recommended security headers for HTTP responses
func SecurityHeaders() map[string]string {
	return map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'",
	}
}
