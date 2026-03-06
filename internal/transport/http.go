// Package transport provides high-performance HTTP client implementations
// with advanced features like connection pooling, retry logic, and compression.
//
// The HTTP client is designed for maximum performance and reliability, supporting:
//   - HTTP/2 with HTTP/1.1 fallback
//   - Intelligent connection pooling
//   - Request/response interceptors
//   - Exponential backoff retry logic
//   - Multiple compression algorithms
//   - Circuit breaker pattern
//   - Comprehensive metrics
//
// Example usage:
//
//	client := transport.NewHTTPClient(transport.HTTPConfig{
//		BaseURL: "https://api.openai.com/v1",
//		Timeout: 30 * time.Second,
//		MaxRetries: 3,
//	})
//
//	resp, err := client.Post(ctx, "/completions", requestBody, headers)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer resp.Body.Close()
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/time/rate"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

// HTTPClient provides a high-performance HTTP client with advanced features.
type HTTPClient struct {
	client         *http.Client
	baseURL        string
	config         HTTPConfig
	interceptors   []Interceptor
	rateLimiter    *rate.Limiter
	circuitBreaker *core.CircuitBreaker
	metrics        *HTTPMetrics
	bufferPool     *sync.Pool
	mu             sync.RWMutex
}

// HTTPConfig configures the HTTP client behavior.
type HTTPConfig struct {
	// Base configuration
	BaseURL   string        `json:"base_url" validate:"required,url"`
	Timeout   time.Duration `json:"timeout" validate:"min=1s,max=300s"`
	UserAgent string        `json:"user_agent,omitempty"`

	// Connection pooling
	MaxIdleConns        int           `json:"max_idle_conns" validate:"min=1,max=1000"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host" validate:"min=1,max=100"`
	MaxConnsPerHost     int           `json:"max_conns_per_host" validate:"min=1,max=100"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout" validate:"min=30s,max=300s"`

	// TLS configuration
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" validate:"min=1s,max=30s"`
	TLSMinVersion       uint16        `json:"tls_min_version"`
	InsecureSkipVerify  bool          `json:"insecure_skip_verify"`
	CertificatePins     []string      `json:"certificate_pins,omitempty"`

	// Retry configuration
	MaxRetries   int           `json:"max_retries" validate:"min=0,max=10"`
	RetryDelay   time.Duration `json:"retry_delay" validate:"min=100ms,max=60s"`
	RetryBackoff float64       `json:"retry_backoff" validate:"min=1,max=5"`

	// Rate limiting
	RateLimit float64 `json:"rate_limit" validate:"min=0"`
	RateBurst int     `json:"rate_burst" validate:"min=1"`

	// Compression
	EnableGzip   bool `json:"enable_gzip"`
	EnableBrotli bool `json:"enable_brotli"`

	// Circuit breaker
	CircuitBreakerMaxFailures int           `json:"circuit_breaker_max_failures"`
	CircuitBreakerTimeout     time.Duration `json:"circuit_breaker_timeout"`

	// Headers
	DefaultHeaders map[string]string `json:"default_headers,omitempty"`

	// Advanced options
	DisableKeepAlives      bool  `json:"disable_keep_alives"`
	DisableCompression     bool  `json:"disable_compression"`
	ForceAttemptHTTP2      bool  `json:"force_attempt_http2"`
	MaxResponseHeaderBytes int64 `json:"max_response_header_bytes" validate:"min=4096,max=1048576"`
}

// DefaultHTTPConfig returns sensible defaults for HTTP client configuration.
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:                   30 * time.Second,
		UserAgent:                 "gollm/1.0",
		MaxIdleConns:              100,
		MaxIdleConnsPerHost:       10,
		MaxConnsPerHost:           50,
		IdleConnTimeout:           90 * time.Second,
		TLSHandshakeTimeout:       10 * time.Second,
		TLSMinVersion:             tls.VersionTLS13,
		MaxRetries:                3,
		RetryDelay:                1 * time.Second,
		RetryBackoff:              2.0,
		RateLimit:                 100.0, // requests per second
		RateBurst:                 10,
		EnableGzip:                true,
		EnableBrotli:              false, // Enable when widely supported
		ForceAttemptHTTP2:         true,
		MaxResponseHeaderBytes:    64 * 1024, // 64KB
		CircuitBreakerMaxFailures: 5,
		CircuitBreakerTimeout:     60 * time.Second,
		DefaultHeaders: map[string]string{
			"Accept":          "application/json",
			"Accept-Encoding": "gzip, deflate",
		},
	}
}

// Interceptor defines the interface for request/response interceptors.
type Interceptor interface {
	// Intercept processes the request before it's sent and the response after it's received.
	Intercept(ctx context.Context, req *http.Request, next RoundTripFunc) (*http.Response, error)
}

// RoundTripFunc represents a function that executes an HTTP round trip.
type RoundTripFunc func(ctx context.Context, req *http.Request) (*http.Response, error)

// NewHTTPClient creates a new HTTP client with the specified configuration.
func NewHTTPClient(config HTTPConfig) (*HTTPClient, error) {
	if err := validateHTTPConfig(config); err != nil {
		return nil, fmt.Errorf("invalid HTTP config: %w", err)
	}

	// Create transport with optimized settings
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:    config.TLSHandshakeTimeout,
		IdleConnTimeout:        config.IdleConnTimeout,
		MaxIdleConns:           config.MaxIdleConns,
		MaxIdleConnsPerHost:    config.MaxIdleConnsPerHost,
		MaxConnsPerHost:        config.MaxConnsPerHost,
		DisableKeepAlives:      config.DisableKeepAlives,
		DisableCompression:     config.DisableCompression,
		ForceAttemptHTTP2:      config.ForceAttemptHTTP2,
		MaxResponseHeaderBytes: config.MaxResponseHeaderBytes,
		TLSClientConfig: &tls.Config{
			MinVersion:         config.TLSMinVersion,
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	// Enable HTTP/2
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, fmt.Errorf("failed to configure HTTP/2: %w", err)
	}

	// Create HTTP client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	// Create rate limiter if configured
	var rateLimiter *rate.Limiter
	if config.RateLimit > 0 {
		rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit), config.RateBurst)
	}

	// Create circuit breaker
	circuitBreaker := core.NewCircuitBreaker(config.CircuitBreakerMaxFailures, config.CircuitBreakerTimeout)

	// Create buffer pool for request/response bodies
	bufferPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}

	client := &HTTPClient{
		client:         httpClient,
		baseURL:        strings.TrimSuffix(config.BaseURL, "/"),
		config:         config,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		metrics:        NewHTTPMetrics(),
		bufferPool:     bufferPool,
	}

	return client, nil
}

// AddInterceptor adds a request/response interceptor.
func (c *HTTPClient) AddInterceptor(interceptor Interceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

// Get performs a GET request.
func (c *HTTPClient) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "GET", path, nil, headers)
}

// Post performs a POST request with JSON body.
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "POST", path, body, headers)
}

// Put performs a PUT request with JSON body.
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "PUT", path, body, headers)
}

// Delete performs a DELETE request.
func (c *HTTPClient) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "DELETE", path, nil, headers)
}

// Request performs an HTTP request with full control over method, path, body, and headers.
func (c *HTTPClient) Request(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	// Build full URL
	fullURL, err := c.buildURL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create request
	req, err := c.createRequest(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply headers
	c.applyHeaders(req, headers)

	// Execute request with retry logic
	return c.executeWithRetry(ctx, req)
}

// buildURL constructs the full URL from base URL and path.
func (c *HTTPClient) buildURL(path string) (string, error) {
	if path == "" {
		return c.baseURL, nil
	}

	// Handle absolute URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Parse and join URLs properly
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return base.ResolveReference(pathURL).String(), nil
}

// createRequest creates an HTTP request with the specified parameters.
func (c *HTTPClient) createRequest(ctx context.Context, method, url string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader

	// Handle different body types
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			// JSON marshal for other types
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for JSON bodies
	if body != nil {
		if _, hasContentType := req.Header["Content-Type"]; !hasContentType {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	return req, nil
}

// applyHeaders applies headers to the request.
func (c *HTTPClient) applyHeaders(req *http.Request, headers map[string]string) {
	// Apply default headers
	for key, value := range c.config.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// Apply request-specific headers (override defaults)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set User-Agent if configured
	if c.config.UserAgent != "" {
		req.Header.Set("User-Agent", c.config.UserAgent)
	}

	// Handle compression
	if c.config.EnableGzip || c.config.EnableBrotli {
		var encodings []string
		if c.config.EnableBrotli {
			encodings = append(encodings, "br")
		}
		if c.config.EnableGzip {
			encodings = append(encodings, "gzip")
		}
		encodings = append(encodings, "deflate")
		req.Header.Set("Accept-Encoding", strings.Join(encodings, ", "))
	}
}

// executeWithRetry executes the request with retry logic and circuit breaker.
func (c *HTTPClient) executeWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply rate limiting
		if c.rateLimiter != nil {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limit wait failed: %w", err)
			}
		}

		// Check circuit breaker
		if !c.circuitBreaker.Allow() {
			return nil, fmt.Errorf("circuit breaker is open: %w", core.ErrProviderUnavail)
		}

		// Execute the request
		resp, err := c.executeRequest(ctx, req)

		// Determine if this is a successful response
		isSuccess := err == nil && (resp == nil || resp.StatusCode < 400)

		// Record metrics
		c.metrics.RecordRequest(time.Since(time.Now()), isSuccess)

		// Check if we should retry
		shouldRetry := c.shouldRetry(err, resp)

		// Update circuit breaker and decide what to do
		if isSuccess {
			c.circuitBreaker.ReportSuccess()
			return resp, nil
		} else {
			c.circuitBreaker.ReportFailure()
			if err != nil {
				lastErr = err
			} else {
				// Create error for HTTP status codes >= 400
				lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			}
		}

		// Don't retry on certain errors
		if !shouldRetry {
			return resp, lastErr
		}

		// Wait before retry (except on last attempt)
		if attempt < c.config.MaxRetries {
			backoffDelay := c.calculateBackoffDelay(attempt)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDelay):
				// Continue with retry
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// executeRequest executes a single HTTP request through the interceptor chain.
func (c *HTTPClient) executeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Clone request for potential retries
	clonedReq := c.cloneRequest(req)

	// Execute through interceptor chain
	roundTrip := c.buildRoundTripChain()

	start := time.Now()
	resp, err := roundTrip(ctx, clonedReq)
	duration := time.Since(start)

	// Record metrics
	c.metrics.RecordRequest(duration, err == nil)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// buildRoundTripChain builds the interceptor chain with the actual HTTP call at the end.
func (c *HTTPClient) buildRoundTripChain() RoundTripFunc {
	// Start with the actual HTTP client call
	roundTrip := RoundTripFunc(func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return c.client.Do(req)
	})

	// Apply interceptors in reverse order (last registered runs first)
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		next := roundTrip
		roundTrip = RoundTripFunc(func(ctx context.Context, req *http.Request) (*http.Response, error) {
			return interceptor.Intercept(ctx, req, next)
		})
	}

	return roundTrip
}

// cloneRequest creates a copy of the request for retries.
func (c *HTTPClient) cloneRequest(req *http.Request) *http.Request {
	cloned := req.Clone(req.Context())

	// Handle body for retries
	if req.Body != nil {
		if req.GetBody != nil {
			var err error
			cloned.Body, err = req.GetBody()
			if err != nil {
				// Fallback: can't retry requests with non-rewindable bodies
				cloned.Body = req.Body
			}
		}
	}

	return cloned
}

// shouldRetry determines if a request should be retried based on the error and response.
func (c *HTTPClient) shouldRetry(err error, resp *http.Response) bool {
	// Don't retry context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Retry on network errors
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return netErr.Temporary() || netErr.Timeout()
		}
		return true // Retry on other errors
	}

	// Retry on specific HTTP status codes
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusTooManyRequests, // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout:      // 504
			return true
		}
	}

	return false
}

// calculateBackoffDelay calculates the delay for exponential backoff.
func (c *HTTPClient) calculateBackoffDelay(attempt int) time.Duration {
	delay := float64(c.config.RetryDelay)
	for i := 0; i < attempt; i++ {
		delay *= c.config.RetryBackoff
	}
	return time.Duration(delay)
}

// GetMetrics returns a copy of the current HTTP metrics.
func (c *HTTPClient) GetMetrics() *HTTPMetrics {
	return c.metrics.Clone()
}

// Close closes the HTTP client and releases resources.
func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// validateHTTPConfig validates the HTTP configuration.
func validateHTTPConfig(config HTTPConfig) error {
	if config.BaseURL == "" {
		return &core.ValidationError{
			Field:   "base_url",
			Rule:    "required",
			Message: "base URL cannot be empty",
		}
	}

	if config.Timeout <= 0 {
		return &core.ValidationError{
			Field:   "timeout",
			Value:   config.Timeout,
			Rule:    "min",
			Message: "timeout must be positive",
		}
	}

	if config.MaxRetries < 0 {
		return &core.ValidationError{
			Field:   "max_retries",
			Value:   config.MaxRetries,
			Rule:    "min",
			Message: "max retries cannot be negative",
		}
	}

	return nil
}

// HTTPMetrics tracks HTTP client performance metrics.
type HTTPMetrics struct {
	mu sync.RWMutex

	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	TotalTime       time.Duration `json:"total_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`

	LastRequestTime time.Time `json:"last_request_time"`
}

// NewHTTPMetrics creates new HTTP metrics.
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{}
}

// RecordRequest records a request in the metrics.
func (m *HTTPMetrics) RecordRequest(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.TotalTime += duration
	m.LastRequestTime = time.Now()

	if m.MinResponseTime == 0 || duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}

	if success {
		m.SuccessRequests++
	} else {
		m.FailedRequests++
	}
}

// GetAverageResponseTime calculates the average response time.
func (m *HTTPMetrics) GetAverageResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}
	return m.TotalTime / time.Duration(m.TotalRequests)
}

// GetSuccessRate calculates the success rate as a percentage.
func (m *HTTPMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}
	return float64(m.SuccessRequests) / float64(m.TotalRequests) * 100
}

// Clone creates a copy of the metrics for safe reading.
func (m *HTTPMetrics) Clone() *HTTPMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &HTTPMetrics{
		TotalRequests:   m.TotalRequests,
		SuccessRequests: m.SuccessRequests,
		FailedRequests:  m.FailedRequests,
		TotalTime:       m.TotalTime,
		MinResponseTime: m.MinResponseTime,
		MaxResponseTime: m.MaxResponseTime,
		LastRequestTime: m.LastRequestTime,
	}
}
