package ports

import "context"

// CircuitBreakerManager defines the contract for circuit breaker protection.
// Circuit breaker prevents cascading failures by stopping requests to failing
// dependencies (e.g., Kafka broker) after a threshold of failures is reached.
//
// The circuit breaker operates in three states:
//   - CLOSED: normal operation, requests pass through
//   - OPEN: too many failures detected, requests fail immediately
//   - HALF_OPEN: testing if dependency recovered, limited requests allowed
//
// Implementations are responsible for:
// - Tracking success/failure counts
// - Transitioning between states based on thresholds
// - Resetting state after recovery timeout
// - Thread-safe operation under concurrent requests
//
// CircuitBreakerManager is injectable to enable testing with FakeCircuitBreaker
// that can be manually controlled (forced OPEN, forced CLOSED) for deterministic tests.
type CircuitBreakerManager interface {
	// Execute runs a function with circuit breaker protection.
	// If the circuit is OPEN, Execute returns CircuitBreakerOpenError immediately
	// without calling fn. This prevents requests from reaching a failing dependency.
	//
	// If the circuit is CLOSED or HALF_OPEN, Execute calls fn and tracks the result:
	// - If fn returns nil (success), increments success counter
	// - If fn returns error, increments failure counter
	//
	// Parameters:
	//   ctx - context for cancellation or timeout (inherited from HTTP request)
	//   fn  - the operation to execute (e.g., publish to Kafka)
	//
	// Returns:
	//   nil              if fn succeeds
	//   fn's error       if fn fails (failure is tracked but error is propagated)
	//   CircuitBreakerOpenError if circuit is OPEN (no attempt made)
	//
	// Example:
	//   err := cb.Execute(ctx, func() error {
	//     return kafkaPublisher.Publish(ctx, "positions", key, value)
	//   })
	//   if err != nil {
	//     // Could be CircuitBreakerOpenError (transient, retry later)
	//     // or network error from Kafka (permanent, fail immediately)
	//   }
	Execute(ctx context.Context, fn func() error) error
}

// CircuitBreakerOpenError indicates the circuit breaker is in OPEN state
// and the protected operation was not attempted.
// This is a transient error; the client may retry after a delay.
type CircuitBreakerOpenError struct {
	Message string
}

func (e CircuitBreakerOpenError) Error() string {
	if e.Message == "" {
		return "circuit breaker is open: dependency unavailable"
	}
	return e.Message
}
