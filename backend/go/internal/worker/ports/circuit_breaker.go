package ports

import (
	"context"
)

// CircuitBreakerState represents the state of a circuit breaker protecting a remote dependency.
// See RF-07, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7.
type CircuitBreakerState string

const (
	// StateClosed indicates the circuit breaker is CLOSED: normal operation,
	// requests are allowed to proceed to the protected dependency.
	StateClosed CircuitBreakerState = "CLOSED"

	// StateOpen indicates the circuit breaker is OPEN: the protected dependency
	// has accumulated consecutive recoverable failures >= WORKER_CB_FAILURE_THRESHOLD.
	// Requests are rejected immediately without calling the function.
	StateOpen CircuitBreakerState = "OPEN"

	// StateHalfOpen indicates the circuit breaker is in test/recovery mode: after
	// WORKER_CB_RECOVERY_INTERVAL has elapsed from the OPEN → HALF_OPEN transition,
	// a single probe request is allowed. Success returns to CLOSED; failure returns to OPEN.
	StateHalfOpen CircuitBreakerState = "HALF_OPEN"
)

// CircuitBreakerOpenError is returned when Execute() is called while the breaker is OPEN
// and the protected operation cannot proceed (RF-07).
type CircuitBreakerOpenError struct {
	// Message describes the error (always "circuit breaker is open" or similar).
	Message string
}

// Error implements the error interface.
func (e CircuitBreakerOpenError) Error() string {
	return e.Message
}

// CircuitBreaker protects a remote dependency (TimescaleDB in the context of the worker)
// from cascading failures using a state machine.
//
// ## State Transitions and Protective Semantics (RF-07–8.7)
//
// The circuit breaker implements the following state machine to prevent repeated
// failed requests to a temporarily unavailable dependency:
//
// ### State Transitions
//
// **CLOSED (Normal Operation):**
//
// - Initial state when the circuit breaker starts.
// - Requests are allowed to proceed to the protected function via Execute().
// - Each call to Execute() either succeeds (failure counter resets to 0) or fails
//   with a recoverable error (failure counter increments by 1).
// - On success: failure counter is reset; the circuit remains CLOSED.
// - On recoverable failure: failure counter is incremented.
// - When the failure counter reaches or exceeds WORKER_CB_FAILURE_THRESHOLD
//   (configurable, default 5), the circuit transitions to OPEN (RF-07).
// - Non-recoverable errors do not affect the failure counter and do not advance
//   the state machine; they are returned to the caller immediately.
//
// **OPEN (Protecting the Dependency):**
//
// - Entered when consecutive recoverable failures reach the threshold.
// - The breaker remains OPEN for a duration of WORKER_CB_RECOVERY_INTERVAL
//   (configurable, default 30 seconds) to give the dependency time to recover
//   (RF-07).
// - While OPEN, Execute() immediately returns CircuitBreakerOpenError without
//   calling the protected function (RF-07).
// - The worker MUST observe this error and pause consuming from Kafka
//   (RF-07), ensuring that unconsumed messages remain unflushed to the
//   offset store and will be reprocessed when the breaker recovers.
// - After WORKER_CB_RECOVERY_INTERVAL has elapsed, the circuit transitions to HALF_OPEN.
//
// **HALF_OPEN (Testing the Dependency):**
//
// - Entered after the recovery interval has elapsed while in OPEN state.
// - In HALF_OPEN, a single probe request is allowed to test whether the dependency
//   has recovered (RF-07, 8.5).
// - Only one attempt is permitted in HALF_OPEN; concurrent or additional attempts
//   MUST be rejected or postponed until the current attempt resolves
//   (RF-07).
// - If the probe succeeds: the circuit transitions back to CLOSED, failure counter
//   resets to 0, and normal operation resumes (RF-07).
// - If the probe fails with a recoverable error: the circuit transitions back to OPEN,
//   the failure counter is reset to the threshold (ready to open again if new attempts
//   fail), and the recovery interval restarts (RF-07).
// - Non-recoverable errors in HALF_OPEN are returned to the caller without changing
//   the circuit state (similar to CLOSED behavior).
//
// ### Failure Counting and Threshold
//
// RF-07 states: only recoverable failures count toward the threshold.
// Non-recoverable failures (e.g., invalid input, constraint violation unrelated
// to deduplication) MUST NOT increment the failure counter or cause state transitions.
// This ensures that transient issues trigger the circuit breaker, but permanent errors
// do not accumulate artificial state.
//
// The failure threshold is read from configuration WORKER_CB_FAILURE_THRESHOLD
// (default 5, RF-07). The threshold MUST be ≥ 1.
//
// ### Timeout and Configuration
//
// The recovery interval is read from configuration WORKER_CB_RECOVERY_INTERVAL
// (default 30 seconds, RF-07).
//
// Each database operation within Execute() is subject to a timeout read from
// configuration WORKER_DB_OPERATION_TIMEOUT (default 5 seconds, RF-03).
// If the operation exceeds the timeout, it is cancelled and treated as a recoverable
// error (RF-070).
//
// ### Integration with the Worker's Kafka Consumer
//
// When the circuit breaker transitions to OPEN:
// - The worker's Kafka consumer (RF-07) MUST pause fetching new messages.
// - Outstanding messages that were accepted but not yet persisted MUST NOT have
//   their offsets committed (RF-05), ensuring they are reprocessed.
// - When the breaker recovers (HALF_OPEN → CLOSED), the consumer resumes fetching.
// - Offset commits resume for messages successfully persisted during recovery.
//
// This integration ensures that no events are silently lost while the dependency
// is unavailable, and that the consumer does not waste bandwidth fetching messages
// that cannot be processed.
//
// ## Execute() Behavior
//
// Execute(ctx, fn) runs the protected function fn under circuit breaker protection.
//
// - If the circuit is OPEN: returns CircuitBreakerOpenError immediately without
//   calling fn (RF-07).
// - If the circuit is CLOSED or HALF_OPEN: calls fn and observes the result.
//   - If fn() returns nil (success): advances the state machine as described above.
//   - If fn() returns a non-nil error: the caller (typically the use case) classifies
//     the error as recoverable or non-recoverable (domain.AppError.IsRecoverable).
//     Only recoverable errors advance the failure counter and state machine;
//     non-recoverable errors are returned without affecting state.
//
// Execute() is thread-safe when called concurrently from multiple goroutines
// (e.g., when the worker processes multiple messages in parallel). However,
// implementations MUST ensure that the HALF_OPEN state enforces the single-probe
// semantics: at most one attempt is in-flight at any time (RF-07).
//
// ## Documentation References
//
// - Circuit Breaker implementation details: full requirements for Circuit Breaker on TimescaleDB.
// - testing.md section 14: circuit breaker resilience testing with Toxiproxy.
// - product.md RF-10, RNF-06: resilience and error handling.
type CircuitBreaker interface {
	// Execute runs fn under circuit breaker protection.
	//
	// If the circuit is OPEN, returns CircuitBreakerOpenError without calling fn.
	// If the circuit is CLOSED or HALF_OPEN, calls fn and returns its result.
	// The caller is responsible for classifying the error returned by fn as
	// recoverable or non-recoverable; only recoverable errors affect the state machine.
	//
	// Execute() is thread-safe for concurrent calls.
	//
	// RF-07–8.7, 8.9, 8.10.
	Execute(ctx context.Context, fn func() error) error

	// State returns the current state of the circuit breaker.
	// May be called concurrently from multiple goroutines.
	// Returns one of StateClosed, StateOpen, or StateHalfOpen.
	//
	// RF-07–8.7.
	State() CircuitBreakerState
}
