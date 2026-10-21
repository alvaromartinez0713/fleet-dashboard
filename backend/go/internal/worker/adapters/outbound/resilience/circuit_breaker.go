package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

type CircuitBreakerState string

const (
	StateClosed   CircuitBreakerState = "CLOSED"
	StateOpen     CircuitBreakerState = "OPEN"
	StateHalfOpen CircuitBreakerState = "HALF_OPEN"
)

// CircuitBreaker implements ports.CircuitBreaker.
// Receives: config values (thresholds, intervals), clock port
type CircuitBreaker struct {
	mu                  sync.RWMutex
	state               CircuitBreakerState
	failureCount        uint32
	successCount        uint32
	lastFailureTime     time.Time
	failureThreshold    uint32
	recoveryInterval    time.Duration
	successThreshold    uint32
	maxHalfOpenAttempts uint32
	clock               ports.Clock
}

// NewCircuitBreaker creates a circuit breaker.
// Receives: config values and clock (port)
func NewCircuitBreaker(
	failureThreshold uint32,
	recoveryInterval time.Duration,
	successThreshold uint32,
	maxHalfOpenAttempts uint32,
	clock ports.Clock,
) *CircuitBreaker {
	if clock == nil {
		panic("clock is nil")
	}
	if successThreshold == 0 {
		successThreshold = 2
	}
	if maxHalfOpenAttempts == 0 {
		maxHalfOpenAttempts = 1
	}

	return &CircuitBreaker{
		state:               StateClosed,
		failureThreshold:    failureThreshold,
		recoveryInterval:    recoveryInterval,
		successThreshold:    successThreshold,
		maxHalfOpenAttempts: maxHalfOpenAttempts,
		clock:               clock,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if cb.IsOpen() {
		return ports.CircuitBreakerOpenError{Message: "circuit open"}
	}
	err := fn()
	if err != nil {
		cb.RecordFailure(err)
		return err
	}
	cb.RecordSuccess()
	return nil
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state != StateOpen {
		return false
	}

	elapsed := cb.clock.Now().Sub(cb.lastFailureTime)
	return elapsed < cb.recoveryInterval
}

func (cb *CircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return string(cb.state)
}

func (cb *CircuitBreaker) RecordFailure(err error) {
	if err == nil {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastFailureTime = cb.clock.Now()
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.lastFailureTime = cb.clock.Now()
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	case StateOpen:
		elapsed := cb.clock.Now().Sub(cb.lastFailureTime)
		if elapsed >= cb.recoveryInterval {
			cb.state = StateHalfOpen
			cb.successCount = 1
		}
	}
}

var _ ports.CircuitBreaker = (*CircuitBreaker)(nil)
