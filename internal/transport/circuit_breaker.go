// Package transport provides circuit breaker implementation for HTTP resilience.
//
// The circuit breaker implements the Circuit Breaker pattern to prevent cascading
// failures and provide fast failure detection. It has three states:
//   - Closed: Normal operation, requests pass through
//   - Open: Failing fast, requests are rejected immediately
//   - Half-Open: Testing if the service has recovered
//
// Example usage:
//
//	config := transport.CircuitBreakerConfig{
//		MaxFailures:  5,
//		ResetTimeout: 60 * time.Second,
//		Timeout:     30 * time.Second,
//	}
//
//	breaker := transport.NewCircuitBreaker(config)
//
//	if breaker.IsOpen() {
//		return errors.New("circuit breaker is open")
//	}
//
//	// Execute operation
//	err := doOperation()
//	if err != nil {
//		breaker.RecordFailure()
//	} else {
//		breaker.RecordSuccess()
//	}
package transport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/gollm/internal/core"
)

// CircuitBreakerState represents the current state of the circuit breaker.
type CircuitBreakerState int

const (
	// StateClosed indicates the circuit breaker is closed (normal operation).
	StateClosed CircuitBreakerState = iota
	// StateOpen indicates the circuit breaker is open (failing fast).
	StateOpen
	// StateHalfOpen indicates the circuit breaker is testing recovery.
	StateHalfOpen
)

// String returns the string representation of the circuit breaker state.
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

var (
	// Circuit breaker specific errors
	ErrCircuitBreakerOpen     = errors.New("circuit breaker is open")
	ErrCircuitBreakerTimeout  = errors.New("circuit breaker timeout")
	ErrInvalidConfiguration   = errors.New("invalid circuit breaker configuration")
)

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before opening the circuit
	MaxFailures int `json:"max_failures" validate:"min=1,max=100"`

	// ResetTimeout is how long to wait before transitioning from Open to Half-Open
	ResetTimeout time.Duration `json:"reset_timeout" validate:"min=1s,max=300s"`

	// Timeout is how long to wait for a request in Half-Open state
	Timeout time.Duration `json:"timeout" validate:"min=1s,max=60s"`

	// FailureThreshold is the percentage of failures that trigger opening (0.0-1.0)
	FailureThreshold float64 `json:"failure_threshold" validate:"min=0.0,max=1.0"`

	// MinRequests is the minimum number of requests before calculating failure rate
	MinRequests int `json:"min_requests" validate:"min=1,max=1000"`

	// SlidingWindowSize is the size of the sliding window for failure rate calculation
	SlidingWindowSize time.Duration `json:"sliding_window_size" validate:"min=10s,max=600s"`
}

// DefaultCircuitBreakerConfig returns sensible defaults for circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:       5,
		ResetTimeout:      60 * time.Second,
		Timeout:           30 * time.Second,
		FailureThreshold:  0.5, // 50% failure rate
		MinRequests:       10,
		SlidingWindowSize: 60 * time.Second,
	}
}

// Validate validates the circuit breaker configuration.
func (c *CircuitBreakerConfig) Validate() error {
	if c.MaxFailures <= 0 {
		return &core.ValidationError{
			Field:   "max_failures",
			Value:   c.MaxFailures,
			Rule:    "min=1",
			Message: "max failures must be at least 1",
		}
	}

	if c.ResetTimeout <= 0 {
		return &core.ValidationError{
			Field:   "reset_timeout",
			Value:   c.ResetTimeout,
			Rule:    "min=1s",
			Message: "reset timeout must be positive",
		}
	}

	if c.Timeout <= 0 {
		return &core.ValidationError{
			Field:   "timeout",
			Value:   c.Timeout,
			Rule:    "min=1s",
			Message: "timeout must be positive",
		}
	}

	if c.FailureThreshold < 0 || c.FailureThreshold > 1 {
		return &core.ValidationError{
			Field:   "failure_threshold",
			Value:   c.FailureThreshold,
			Rule:    "range=0.0-1.0",
			Message: "failure threshold must be between 0.0 and 1.0",
		}
	}

	return nil
}

// CircuitBreaker implements the circuit breaker pattern for resilience.
type CircuitBreaker struct {
	mu       sync.RWMutex
	config   CircuitBreakerConfig
	state    CircuitBreakerState
	failures int
	requests int

	// State transitions
	lastFailureTime time.Time
	lastStateChange time.Time

	// Sliding window for failure rate calculation
	requestWindow []requestRecord

	// Metrics
	metrics *CircuitBreakerMetrics
}

// requestRecord represents a single request record in the sliding window.
type requestRecord struct {
	timestamp time.Time
	success   bool
}

// CircuitBreakerMetrics tracks circuit breaker performance metrics.
type CircuitBreakerMetrics struct {
	mu sync.RWMutex

	TotalRequests     int64         `json:"total_requests"`
	TotalFailures     int64         `json:"total_failures"`
	TotalSuccesses    int64         `json:"total_successes"`
	StateTransitions  int64         `json:"state_transitions"`
	TimeInClosed      time.Duration `json:"time_in_closed"`
	TimeInOpen        time.Duration `json:"time_in_open"`
	TimeInHalfOpen    time.Duration `json:"time_in_half_open"`
	LastStateChange   time.Time     `json:"last_state_change"`
	ConsecutiveFailures int         `json:"consecutive_failures"`
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if err := config.Validate(); err != nil {
		// Use default configuration if validation fails
		config = DefaultCircuitBreakerConfig()
	}

	now := time.Now()
	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: now,
		requestWindow:   make([]requestRecord, 0, config.MinRequests*2),
		metrics: &CircuitBreakerMetrics{
			LastStateChange: now,
		},
	}
}

// IsOpen returns true if the circuit breaker is in the Open state.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state == StateOpen
}

// IsClosed returns true if the circuit breaker is in the Closed state.
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state == StateClosed
}

// IsHalfOpen returns true if the circuit breaker is in the Half-Open state.
func (cb *CircuitBreaker) IsHalfOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state == StateHalfOpen
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}

// CanExecute checks if a request can be executed based on the current state.
func (cb *CircuitBreaker) CanExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if enough time has passed to try half-open
		if now.Sub(cb.lastStateChange) >= cb.config.ResetTimeout {
			cb.transitionTo(StateHalfOpen, now)
			return nil
		}
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		// Allow limited requests in half-open state
		return nil

	default:
		return fmt.Errorf("unknown circuit breaker state: %v", cb.state)
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.recordRequest(now, true)

	cb.requests++
	cb.failures = 0 // Reset consecutive failures

	// Update metrics
	cb.metrics.mu.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.TotalSuccesses++
	cb.metrics.ConsecutiveFailures = 0
	cb.metrics.mu.Unlock()

	// State transitions
	switch cb.state {
	case StateHalfOpen:
		// Transition back to closed after successful request
		cb.transitionTo(StateClosed, now)

	case StateOpen:
		// This shouldn't happen, but handle gracefully
		cb.transitionTo(StateClosed, now)
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.recordRequest(now, false)

	cb.requests++
	cb.failures++
	cb.lastFailureTime = now

	// Update metrics
	cb.metrics.mu.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.TotalFailures++
	cb.metrics.ConsecutiveFailures = cb.failures
	cb.metrics.mu.Unlock()

	// Check if we should open the circuit
	if cb.shouldOpen() {
		cb.transitionTo(StateOpen, now)
	}
}

// recordRequest adds a request to the sliding window.
func (cb *CircuitBreaker) recordRequest(timestamp time.Time, success bool) {
	// Add new record
	cb.requestWindow = append(cb.requestWindow, requestRecord{
		timestamp: timestamp,
		success:   success,
	})

	// Clean old records outside the sliding window
	cutoff := timestamp.Add(-cb.config.SlidingWindowSize)
	var validRecords []requestRecord

	for _, record := range cb.requestWindow {
		if record.timestamp.After(cutoff) {
			validRecords = append(validRecords, record)
		}
	}

	cb.requestWindow = validRecords
}

// shouldOpen determines if the circuit should be opened based on failure criteria.
func (cb *CircuitBreaker) shouldOpen() bool {
	if cb.state == StateOpen {
		return false // Already open
	}

	// Check consecutive failures threshold
	if cb.failures >= cb.config.MaxFailures {
		return true
	}

	// Check failure rate in sliding window
	if len(cb.requestWindow) >= cb.config.MinRequests {
		failureRate := cb.calculateFailureRate()
		if failureRate >= cb.config.FailureThreshold {
			return true
		}
	}

	return false
}

// calculateFailureRate calculates the failure rate in the current sliding window.
func (cb *CircuitBreaker) calculateFailureRate() float64 {
	if len(cb.requestWindow) == 0 {
		return 0.0
	}

	failures := 0
	for _, record := range cb.requestWindow {
		if !record.success {
			failures++
		}
	}

	return float64(failures) / float64(len(cb.requestWindow))
}

// transitionTo changes the circuit breaker state and updates metrics.
func (cb *CircuitBreaker) transitionTo(newState CircuitBreakerState, now time.Time) {
	if cb.state == newState {
		return // No state change
	}

	oldState := cb.state
	timeSinceLastChange := now.Sub(cb.lastStateChange)

	// Update time spent in previous state
	cb.metrics.mu.Lock()
	switch oldState {
	case StateClosed:
		cb.metrics.TimeInClosed += timeSinceLastChange
	case StateOpen:
		cb.metrics.TimeInOpen += timeSinceLastChange
	case StateHalfOpen:
		cb.metrics.TimeInHalfOpen += timeSinceLastChange
	}
	cb.metrics.StateTransitions++
	cb.metrics.LastStateChange = now
	cb.metrics.mu.Unlock()

	// Change state
	cb.state = newState
	cb.lastStateChange = now

	// Reset counters on state change
	switch newState {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		// Keep failure count but reset requests to allow testing
		cb.requests = 0
	}
}

// GetMetrics returns a copy of the current circuit breaker metrics.
func (cb *CircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.metrics.mu.RLock()
	defer cb.metrics.mu.RUnlock()

	// Calculate current state duration
	now := time.Now()
	currentStateDuration := now.Sub(cb.metrics.LastStateChange)

	metrics := &CircuitBreakerMetrics{
		TotalRequests:       cb.metrics.TotalRequests,
		TotalFailures:       cb.metrics.TotalFailures,
		TotalSuccesses:      cb.metrics.TotalSuccesses,
		StateTransitions:    cb.metrics.StateTransitions,
		TimeInClosed:        cb.metrics.TimeInClosed,
		TimeInOpen:          cb.metrics.TimeInOpen,
		TimeInHalfOpen:      cb.metrics.TimeInHalfOpen,
		LastStateChange:     cb.metrics.LastStateChange,
		ConsecutiveFailures: cb.metrics.ConsecutiveFailures,
	}

	// Add current state time
	cb.mu.RLock()
	switch cb.state {
	case StateClosed:
		metrics.TimeInClosed += currentStateDuration
	case StateOpen:
		metrics.TimeInOpen += currentStateDuration
	case StateHalfOpen:
		metrics.TimeInHalfOpen += currentStateDuration
	}
	cb.mu.RUnlock()

	return metrics
}

// GetFailureRate returns the current failure rate in the sliding window.
func (cb *CircuitBreaker) GetFailureRate() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.calculateFailureRate()
}

// Reset resets the circuit breaker to the closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.transitionTo(StateClosed, now)
	cb.failures = 0
	cb.requests = 0
	cb.requestWindow = cb.requestWindow[:0] // Clear sliding window
}

// GetConsecutiveFailures returns the current number of consecutive failures.
func (cb *CircuitBreaker) GetConsecutiveFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.failures
}

// GetTimeSinceLastFailure returns the duration since the last failure.
func (cb *CircuitBreaker) GetTimeSinceLastFailure() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.lastFailureTime.IsZero() {
		return 0
	}

	return time.Since(cb.lastFailureTime)
}

// String implements fmt.Stringer for debugging.
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return fmt.Sprintf("CircuitBreaker{state: %s, failures: %d, requests: %d, failure_rate: %.2f}",
		cb.state, cb.failures, cb.requests, cb.calculateFailureRate())
}
