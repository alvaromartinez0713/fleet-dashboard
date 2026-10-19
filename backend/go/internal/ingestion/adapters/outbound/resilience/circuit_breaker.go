package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState string

const (
	StateClosed     CircuitBreakerState = "CLOSED"
	StateOpen       CircuitBreakerState = "OPEN"
	StateHalfOpen   CircuitBreakerState = "HALF_OPEN"
	defaultTimeout  time.Duration       = 30 * time.Second
	failureThreshold uint32             = 5
	successThreshold uint32             = 2
)

// CircuitBreaker manages resilience for remote calls using a state machine:
// CLOSED (normal) → OPEN (5 failures) → HALF_OPEN (30s timeout) → CLOSED (2 successes)
// Thread-safe with sync.RWMutex.
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    uint32
	successCount    uint32
	lastFailureTime time.Time
	timeout         time.Duration
	clock           ports.Clock
}

// NewCircuitBreaker creates a new circuit breaker in CLOSED state with system clock.
func NewCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreakerWithClock(nil)
}

// NewCircuitBreakerWithClock creates a new circuit breaker with a custom clock.
// If clock is nil, uses system time.
func NewCircuitBreakerWithClock(clock ports.Clock) *CircuitBreaker {
	if clock == nil {
		// Use a simple clock wrapper around time.Now
		clock = &systemClock{}
	}
	return &CircuitBreaker{
		state:   StateClosed,
		timeout: defaultTimeout,
		clock:   clock,
	}
}

// systemClock wraps time.Now() to implement ports.Clock
type systemClock struct{}

func (sc *systemClock) Now() time.Time {
	return time.Now()
}

// RecordSuccess records a successful call.
// If in HALF_OPEN, increments success counter. After 2 successes, transitions to CLOSED.
// If in CLOSED, resets failure counter.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure counter on success while closed
		cb.failureCount = 0
	case StateHalfOpen:
		// Increment success counter in half-open state
		cb.successCount++
		if cb.successCount >= successThreshold {
			// Transition to CLOSED after 2 successes
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	case StateOpen:
		// Ignore successes while open
	}
}

// RecordFailure records a failed call.
// If in CLOSED, increments failure counter. After 5 failures, transitions to OPEN.
// If in HALF_OPEN, reverts to OPEN immediately.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		cb.lastFailureTime = cb.clock.Now()
		if cb.failureCount >= failureThreshold {
			// Transition to OPEN after 5 failures
			cb.state = StateOpen
			cb.successCount = 0
		}
	case StateHalfOpen:
		// Any failure in half-open reverts to open
		cb.state = StateOpen
		cb.failureCount = failureThreshold // Immediate reset to threshold
		cb.successCount = 0
		cb.lastFailureTime = cb.clock.Now()
	case StateOpen:
		// Update last failure time but don't increment counter
		cb.lastFailureTime = cb.clock.Now()
	}
}

// IsOpen returns true if the circuit breaker is OPEN or HALF_OPEN has timed out.
// Automatically transitions from OPEN to HALF_OPEN after timeout.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateClosed {
		return false
	}

	if cb.state == StateOpen {
		// Check if timeout has elapsed to transition to HALF_OPEN
		if cb.clock.Now().Sub(cb.lastFailureTime) >= cb.timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			return false // Half-open allows next attempt
		}
		return true // Still open
	}

	// StateHalfOpen
	return false // Half-open allows attempts
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// String returns a string representation of the circuit breaker state.
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return fmt.Sprintf("CircuitBreaker{state=%s, failures=%d, successes=%d}", cb.state, cb.failureCount, cb.successCount)
}

// Execute runs a function with circuit breaker protection.
// Implements ports.CircuitBreakerManager interface.
// If the circuit is OPEN, returns CircuitBreakerOpenError immediately.
// If the circuit is CLOSED or HALF_OPEN, calls fn and tracks the result.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if circuit is open
	if cb.IsOpen() {
		return ports.CircuitBreakerOpenError{
			Message: "circuit breaker is open: dependency unavailable",
		}
	}

	// Execute the function
	err := fn()

	// Track result
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// CircuitBreakerManager manages multiple circuit breakers by name.
// Thread-safe with sync.RWMutex.
type CircuitBreakerManager struct {
	mu           sync.RWMutex
	breakers     map[string]*CircuitBreaker
	defaultBCB   *CircuitBreaker // Default circuit breaker for unnamed operations
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers:   make(map[string]*CircuitBreaker),
		defaultBCB: NewCircuitBreaker(),
	}
}

// Get returns the circuit breaker for the given name, creating one if necessary.
func (cbm *CircuitBreakerManager) Get(name string) *CircuitBreaker {
	if name == "" {
		return cbm.defaultBCB
	}

	cbm.mu.RLock()
	breaker, exists := cbm.breakers[name]
	cbm.mu.RUnlock()

	if exists {
		return breaker
	}

	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Double-check pattern: another goroutine may have created it while we waited for lock
	if breaker, exists := cbm.breakers[name]; exists {
		return breaker
	}

	breaker = NewCircuitBreaker()
	cbm.breakers[name] = breaker
	return breaker
}

// RecordSuccess records a successful call for the given breaker.
func (cbm *CircuitBreakerManager) RecordSuccess(name string) {
	cbm.Get(name).RecordSuccess()
}

// RecordFailure records a failed call for the given breaker.
func (cbm *CircuitBreakerManager) RecordFailure(name string) {
	cbm.Get(name).RecordFailure()
}

// IsOpen checks if the given breaker is open.
func (cbm *CircuitBreakerManager) IsOpen(name string) bool {
	return cbm.Get(name).IsOpen()
}
