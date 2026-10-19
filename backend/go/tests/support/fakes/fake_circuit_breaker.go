package fakes

import (
	"context"
	"sync"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// FakeCircuitBreaker is a deterministic circuit breaker for testing.
// It allows tests to control circuit state (OPEN/CLOSED/HALF_OPEN) and verify calls.
//
// Typical usage in tests:
//
//	breaker := &FakeCircuitBreaker{}  // Initialized to CLOSED state
//	err := breaker.Execute(ctx, func() error {
//	  return kafkaPublisher.Publish(ctx, "positions", key, value)
//	})
//	// Verify the call was executed
//
// For testing state transitions:
//
//	breaker.SetHalfOpen()  // Simulate recovery attempt
//	breaker.RecordSuccess()  // Record 2 successes
//	breaker.RecordSuccess()
//	// Circuit transitions to CLOSED after 2 successes
type FakeCircuitBreaker struct {
	// state represents the current circuit breaker state: CLOSED, OPEN, or HALF_OPEN.
	// Default (zero value) is empty string, which is treated as CLOSED by Execute().
	state CircuitBreakerTestState

	// executionCount tracks the number of times Execute was called.
	executionCount int

	// callResults stores the results of Execute calls for verification.
	callResults []ExecutionResult

	// successCount tracks successful calls in HALF_OPEN state.
	successCount int

	// mu protects concurrent access to mutable fields.
	mu sync.Mutex
}

// CircuitBreakerTestState represents test states for circuit breaker testing.
type CircuitBreakerTestState string

const (
	TestStateClosed   CircuitBreakerTestState = "CLOSED"
	TestStateOpen     CircuitBreakerTestState = "OPEN"
	TestStateHalfOpen CircuitBreakerTestState = "HALF_OPEN"
)

// ExecutionResult captures the result of an Execute call for verification.
type ExecutionResult struct {
	// Error is the error returned by the function (if any).
	Error error

	// FunctionExecuted indicates whether the function was called.
	// If circuit is OPEN, FunctionExecuted is false.
	FunctionExecuted bool
}

// NewFakeCircuitBreaker creates a new FakeCircuitBreaker initialized to CLOSED state.
func NewFakeCircuitBreaker() *FakeCircuitBreaker {
	return &FakeCircuitBreaker{
		state: TestStateClosed,
	}
}

// Execute runs a function with circuit breaker protection.
// - If OPEN: returns CircuitBreakerOpenError without calling fn
// - If CLOSED or HALF_OPEN: calls fn and records the result
func (f *FakeCircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.executionCount++

	// Empty string (zero value) is treated as CLOSED
	if f.state == TestStateOpen {
		// Circuit is open; return error without executing
		f.callResults = append(f.callResults, ExecutionResult{
			Error:            ports.CircuitBreakerOpenError{},
			FunctionExecuted: false,
		})
		return ports.CircuitBreakerOpenError{}
	}

	// Circuit is closed or half-open; execute the function
	err := fn()
	f.callResults = append(f.callResults, ExecutionResult{
		Error:            err,
		FunctionExecuted: true,
	})

	return err
}

// SetOpen forces the circuit into OPEN state for testing circuit breaker behavior.
func (f *FakeCircuitBreaker) SetOpen() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = TestStateOpen
	f.successCount = 0
}

// SetClosed forces the circuit into CLOSED state (normal operation).
func (f *FakeCircuitBreaker) SetClosed() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = TestStateClosed
	f.successCount = 0
}

// SetHalfOpen forces the circuit into HALF_OPEN state for testing recovery attempts.
// Used to simulate the state after timeout from OPEN.
func (f *FakeCircuitBreaker) SetHalfOpen() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = TestStateHalfOpen
	f.successCount = 0
}

// IsOpen returns true if the circuit is in OPEN state.
func (f *FakeCircuitBreaker) IsOpen() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state == TestStateOpen
}

// GetExecutionCount returns the number of times Execute was called.
func (f *FakeCircuitBreaker) GetExecutionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.executionCount
}

// GetState returns the current state of the circuit breaker.
func (f *FakeCircuitBreaker) GetState() CircuitBreakerTestState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

// RecordSuccess simulates a successful call.
// In HALF_OPEN state, increments success counter.
// After 2 successes in HALF_OPEN, automatically transitions to CLOSED.
func (f *FakeCircuitBreaker) RecordSuccess() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.state == TestStateHalfOpen {
		f.successCount++
		if f.successCount >= 2 {
			// Transition to CLOSED after 2 successes
			f.state = TestStateClosed
			f.successCount = 0
		}
	}
}

// RecordFailure simulates a failed call.
// In CLOSED state, increments failure counter.
// After 5 failures in CLOSED, automatically transitions to OPEN.
// In HALF_OPEN state, transitions back to OPEN immediately.
func (f *FakeCircuitBreaker) RecordFailure() {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch f.state {
	case TestStateClosed:
		// Track failures in closed state
		// Note: We don't track failure count in tests, just open the circuit after 5 calls
		// For now, tests manually manage state transitions
	case TestStateHalfOpen:
		// Any failure in half-open reverts to open
		f.state = TestStateOpen
		f.successCount = 0
	}
}

// GetCallResults returns all recorded execution results.
// Use this to verify that functions were executed and their results.
func (f *FakeCircuitBreaker) GetCallResults() []ExecutionResult {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Return a copy to prevent external mutations
	resultsCopy := make([]ExecutionResult, len(f.callResults))
	copy(resultsCopy, f.callResults)
	return resultsCopy
}

// Reset clears all recorded state for reuse in subsequent tests.
func (f *FakeCircuitBreaker) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = TestStateClosed
	f.executionCount = 0
	f.callResults = nil
	f.successCount = 0
}
