package ports

import "time"

// Clock provides the current time, injectable for deterministic tests (Req 10.5, 10.7).
// Implementations must return UTC timestamps; the use case converts the result to UTC
// with millisecond precision before assigning stored_at.
type Clock interface {
	// Now returns the current instant in UTC.
	// For production, implementations return time.Now().UTC().
	// For tests, implementations can be fixed (always return the same time) or controlled
	// to advance time deterministically, enabling property-based tests of time-dependent
	// logic without external dependencies (Req 10.5).
	Now() time.Time
}
