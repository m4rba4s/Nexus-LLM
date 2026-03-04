package core

import (
	"errors"
	"log"
	"sync"
	"time"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is OPEN")
)

type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal operations
	StateOpen                         // Failing, reject immediately
	StateHalfOpen                     // Testing recovery
)

// CircuitBreaker implements the L7 resilient routing pattern
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitState
	failures     int
	maxFailures  int
	resetTimeout time.Duration
	openedAt     time.Time
}

// NewCircuitBreaker initializes a thread-safe breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// Allow checks if the circuit is closed or half-open and allows the request.
// If Open, it checks if the cooldown has elapsed to transition to Half-Open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.openedAt) > cb.resetTimeout {
			log.Println("[CIRCUIT BREAKER] State transitioning to HALF-OPEN. Testing API...")
			cb.state = StateHalfOpen
			return true // Allow one test request
		}
		return false
	case StateHalfOpen:
		// If half-open, only one request should be trying at a time ideally
		return true
	}
	return false
}

// ReportSuccess resets the breaker if it was HalfOpen
func (cb *CircuitBreaker) ReportSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		log.Println("[CIRCUIT BREAKER] API test successful. State transitioning to CLOSED.")
		cb.state = StateClosed
	}
	cb.failures = 0
}

// ReportFailure increments the failure count and opens the circuit if threshold is reached
func (cb *CircuitBreaker) ReportFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.state == StateHalfOpen || (cb.state == StateClosed && cb.failures >= cb.maxFailures) {
		log.Printf("[CIRCUIT BREAKER] Threshold reached (%d failures). State transitioning to OPEN.", cb.failures)
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}

// State returns the current State for monitoring
func (cb *CircuitBreaker) State() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return int(cb.state)
}
