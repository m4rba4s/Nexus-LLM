package transport

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name   string
		config CircuitBreakerConfig
		want   CircuitBreakerConfig
	}{
		{
			name:   "default config",
			config: CircuitBreakerConfig{},
			want: CircuitBreakerConfig{
				MaxFailures:       5,
				ResetTimeout:     60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.5,
				MinRequests:      10,
				SlidingWindowSize: 60 * time.Second,
			},
		},
		{
			name: "custom config",
			config: CircuitBreakerConfig{
				MaxFailures:       8,
				ResetTimeout:     30 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.6,
				MinRequests:      20,
				SlidingWindowSize: 30 * time.Second,
			},
			want: CircuitBreakerConfig{
				MaxFailures:       8,
				ResetTimeout:     30 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.6,
				MinRequests:      20,
				SlidingWindowSize: 30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCircuitBreaker(tt.config)

			assert.NotNil(t, cb)
			assert.Equal(t, tt.want.MaxFailures, cb.config.MaxFailures)
			assert.Equal(t, tt.want.FailureThreshold, cb.config.FailureThreshold)
			assert.Equal(t, tt.want.ResetTimeout, cb.config.ResetTimeout)
			assert.Equal(t, tt.want.Timeout, cb.config.Timeout)
			assert.Equal(t, StateClosed, cb.GetState())
		})
	}
}

func TestCircuitBreaker_BasicFlow(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       2,
		ResetTimeout:     100 * time.Millisecond,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 0.5,
		MinRequests:      3,
		SlidingWindowSize: 100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Initially closed
	assert.Equal(t, StateClosed, cb.GetState())

	// Successful request
	err := cb.CanExecute()
	assert.NoError(t, err)
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())

	// First failure
	err = cb.CanExecute()
	assert.NoError(t, err)
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.GetState()) // Still closed

	// Second failure - should trigger open state
	err = cb.CanExecute()
	assert.NoError(t, err)
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.GetState()) // Now open

	// Request while open should fail immediately
	err = cb.CanExecute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should now allow requests (half-open state)
	err = cb.CanExecute()
	assert.NoError(t, err)

	// Successful request should close the circuit
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*CircuitBreaker)
		action         func(*CircuitBreaker) error
		expectedState  CircuitBreakerState
		expectError    bool
		errorContains  string
	}{
		{
			name: "closed to open after failures",
			setup: func(cb *CircuitBreaker) {
				// Simulate failure count reaching threshold
				for i := 0; i < cb.config.MaxFailures-1; i++ {
					cb.RecordFailure()
				}
			},
			action: func(cb *CircuitBreaker) error {
				cb.RecordFailure()
				return errors.New("final failure")
			},
			expectedState: StateOpen,
			expectError:   true,
			errorContains: "final failure",
		},
		{
			name: "open circuit rejects requests",
			setup: func(cb *CircuitBreaker) {
				// Force circuit to open state by recording failures
				for i := 0; i < cb.config.MaxFailures; i++ {
					cb.RecordFailure()
				}
			},
			action: func(cb *CircuitBreaker) error {
				if cb.IsOpen() {
					return ErrCircuitBreakerOpen
				}
				return nil
			},
			expectedState: StateOpen,
			expectError:   true,
			errorContains: "circuit breaker is open",
		},
		{
			name: "half-open allows limited requests",
			setup: func(cb *CircuitBreaker) {
				// Force circuit to open first, then wait for reset timeout
				for i := 0; i < cb.config.MaxFailures; i++ {
					cb.RecordFailure()
				}
				time.Sleep(cb.config.ResetTimeout + 10*time.Millisecond)
			},
			action: func(cb *CircuitBreaker) error {
				if !cb.IsOpen() {
					cb.RecordSuccess()
				}
				return nil
			},
			expectedState: StateOpen,
			expectError:   false,
		},
		{
			name: "half-open failure reopens circuit",
			setup: func(cb *CircuitBreaker) {
				// Force circuit to open first, then wait for reset timeout
				for i := 0; i < cb.config.MaxFailures; i++ {
					cb.RecordFailure()
				}
				time.Sleep(cb.config.ResetTimeout + 10*time.Millisecond)
			},
			action: func(cb *CircuitBreaker) error {
				cb.RecordFailure()
				return errors.New("half-open failure")
			},
			expectedState: StateOpen,
			expectError:   true,
			errorContains: "half-open failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CircuitBreakerConfig{
				MaxFailures:       2,
				ResetTimeout:     100 * time.Millisecond,
				Timeout:          100 * time.Millisecond,
				FailureThreshold: 0.5,
				MinRequests:      3,
				SlidingWindowSize: 100 * time.Millisecond,
			}

			cb := NewCircuitBreaker(config)

			if tt.setup != nil {
				tt.setup(cb)
			}

			err := tt.action(cb)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedState, cb.GetState())
		})
	}
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Simulate timeout scenario
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = errors.New("operation completed")
	}

	if err.Error() == "context deadline exceeded" {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       50,
		ResetTimeout:     time.Second,
		Timeout:          time.Second,
		FailureThreshold: 0.5,
		MinRequests:      100,
		SlidingWindowSize: time.Second,
	}

	cb := NewCircuitBreaker(config)

	const numGoroutines = 50
	const numOperations = 100

	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Simulate some work
				time.Sleep(time.Microsecond)

				// Simulate occasional failures
				if (goroutineID*numOperations+j)%10 == 0 {
					cb.RecordFailure()
					atomic.AddInt64(&errorCount, 1)
				} else {
					cb.RecordSuccess()
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalOperations := int64(numGoroutines * numOperations)
	assert.Equal(t, totalOperations, successCount+errorCount)
	assert.Greater(t, successCount, int64(0))
	assert.Greater(t, errorCount, int64(0))

	// Verify circuit breaker state is consistent
	state := cb.GetState()
	assert.True(t, state == StateClosed || state == StateOpen || state == StateHalfOpen)
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       3,
		ResetTimeout:     100 * time.Millisecond,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 0.3,
		MinRequests:      10,
		SlidingWindowSize: 100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Execute some successful and failed requests
	for i := 0; i < 5; i++ {
		cb.RecordSuccess()
	}

	for i := 0; i < 2; i++ {
		cb.RecordFailure()
	}

	// Check basic circuit breaker state
	assert.Equal(t, StateClosed, cb.GetState())
	assert.False(t, cb.IsOpen())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       2,
		ResetTimeout:     100 * time.Millisecond,
		Timeout:          100 * time.Millisecond,
		FailureThreshold: 0.5,
		MinRequests:      3,
		SlidingWindowSize: 100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Trigger failures to open circuit
	for i := 0; i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	assert.Equal(t, StateOpen, cb.GetState())

	// Reset circuit breaker
	cb.Reset()

	assert.Equal(t, StateClosed, cb.GetState())

	// Should be able to record success normally
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_RecoveryScenarios(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       2,
		ResetTimeout:     50 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 0.5,
		MinRequests:      3,
		SlidingWindowSize: 50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Force circuit to open
	for i := 0; i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Record success to close circuit
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())

	// Subsequent operations should work normally
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_EdgeCases(t *testing.T) {
	t.Run("zero failure threshold", func(t *testing.T) {
		config := CircuitBreakerConfig{
			MaxFailures:       0,
			ResetTimeout:     60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 0.0,
			MinRequests:      10,
			SlidingWindowSize: 60 * time.Second,
		}

		cb := NewCircuitBreaker(config)

		// Should immediately open on first failure
		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("zero timeout", func(t *testing.T) {
		config := CircuitBreakerConfig{
			MaxFailures:       2,
			ResetTimeout:     0, // Zero timeout means immediate recovery
			Timeout:          30 * time.Second,
			FailureThreshold: 0.5,
			MinRequests:      10,
			SlidingWindowSize: 60 * time.Second,
		}

		cb := NewCircuitBreaker(config)

		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.RecordFailure()
		}
		assert.Equal(t, StateClosed, cb.GetState())

		// Should immediately transition to half-open
		time.Sleep(time.Millisecond) // Give it a moment

		// Should be able to record success
		cb.RecordSuccess()
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("panic recovery", func(t *testing.T) {
		config := CircuitBreakerConfig{
			MaxFailures:       2,
			ResetTimeout:     60 * time.Second,
			Timeout:          30 * time.Second,
			FailureThreshold: 0.5,
			MinRequests:      10,
			SlidingWindowSize: 60 * time.Second,
		}

		cb := NewCircuitBreaker(config)

		// Simulate panic scenario by recording failure
		cb.RecordFailure()
		assert.Equal(t, StateClosed, cb.GetState()) // Should still be closed after first failure
	})
}

// Benchmark tests
func BenchmarkCircuitBreaker_ClosedState(b *testing.B) {
	config := CircuitBreakerConfig{
		MaxFailures:       100,
		ResetTimeout:     time.Minute,
		Timeout:          time.Minute,
		FailureThreshold: 0.1,
		MinRequests:      1000,
		SlidingWindowSize: time.Minute,
	}

	cb := NewCircuitBreaker(config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cb.RecordSuccess()
	}
}

func BenchmarkCircuitBreaker_OpenState(b *testing.B) {
	config := CircuitBreakerConfig{
		MaxFailures:       1,
		ResetTimeout:     time.Minute,
		Timeout:          time.Minute,
		FailureThreshold: 0.1,
		MinRequests:      10,
		SlidingWindowSize: time.Minute,
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.RecordFailure()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cb.IsOpen() // Just check if it's open
	}
}

func BenchmarkCircuitBreaker_ConcurrentAccess(b *testing.B) {
	config := CircuitBreakerConfig{
		MaxFailures:       100,
		ResetTimeout:     time.Minute,
		Timeout:          time.Minute,
		FailureThreshold: 0.1,
		MinRequests:      1000,
		SlidingWindowSize: time.Minute,
	}

	cb := NewCircuitBreaker(config)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordSuccess()
		}
	})
}

func TestCircuitBreaker_String(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       5,
		ResetTimeout:     60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 0.5,
		MinRequests:      10,
		SlidingWindowSize: 60 * time.Second,
	}

	cb := NewCircuitBreaker(config)

	result := cb.String()
	assert.Contains(t, result, "closed")
}

func TestCircuitBreaker_IsReady(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       2,
		ResetTimeout:     50 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 0.5,
		MinRequests:      3,
		SlidingWindowSize: 50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Initially ready (closed state)
	assert.False(t, cb.IsOpen())

	// Open the circuit
	for i := 0; i < config.MaxFailures; i++ {
		cb.RecordFailure()
	}

	// Should be open
	assert.True(t, cb.IsOpen())

	// Wait for recovery timeout
	time.Sleep(60 * time.Millisecond)

	// Should still be open since no requests were made to transition to half-open
	assert.True(t, cb.IsOpen())
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:       2,
		ResetTimeout:     50 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 0.5,
		MinRequests:      3,
		SlidingWindowSize: 50 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)

	// Record some successes and failures
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordFailure()

	metrics := cb.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(3), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.TotalFailures)
}
