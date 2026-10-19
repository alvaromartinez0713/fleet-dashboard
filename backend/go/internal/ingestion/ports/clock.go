package ports

import "time"

// Clock defines the contract for time queries within the ingestion service.
// Implementations allow injecting a clock for deterministic testing.
// Production uses RealClock which wraps time.Now().
//
// Clock is intentionally minimal with a single method:
// - Now() returns current UTC time as time.Time
//
// This minimal interface enables:
// - Deterministic time in tests (FakeClock can return controlled time)
// - No dependency on a specific time library
// - Minimal surface area for changes
type Clock interface {
	// Now returns the current time in UTC.
	// Production implementations return time.Now().UTC().
	// Test implementations return controlled or fixed time.
	Now() time.Time
}
