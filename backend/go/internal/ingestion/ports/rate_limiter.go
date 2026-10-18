package ports

import "context"

// RateLimiter defines the contract for rate limiting per vehicle.
// Rate limiting protects the ingestion service from request floods and ensures
// fair resource allocation across clients (vehicles).
//
// The rate limiter uses a rolling-window algorithm (not fixed buckets):
// - Tracks requests per vehicle_id over a time window (default: 60 seconds)
// - Allows N requests per window (default: 12 requests per 60 seconds = 0.2 req/sec)
// - Rejects additional requests until the window slides
//
// Implementations are responsible for:
// - Maintaining per-vehicle request counts
// - Time-based window management (sliding window)
// - Cleaning up old entries to prevent memory leaks
// - Thread-safe concurrent access
//
// RateLimiter is injectable to enable testing with FakeRateLimiter
// that can be configured with custom limits or forced to reject requests.
type RateLimiter interface {
	// Allow checks if a request from the given vehicle_id is within rate limit.
	// Returns true if allowed, false if rate limit exceeded.
	//
	// Parameters:
	//   ctx       - context for cancellation or timeout
	//   vehicleID - unique identifier of the requesting vehicle
	//
	// Returns:
	//   true  if the request is within the rate limit (request is counted)
	//   false if rate limit exceeded (request is rejected, not counted)
	//
	// Example:
	//   if !limiter.Allow(ctx, "VH-001") {
	//     return RateLimitExceededError
	//   }
	//   // Request allowed, proceed
	//
	// Idempotence:
	//   Calling Allow multiple times with same vehicleID may return different results
	//   because time advances and the window slides.
	//   For testing, use FakeRateLimiter that returns deterministic results.
	Allow(ctx context.Context, vehicleID string) bool
}

// RateLimitExceededError indicates the rate limit for this vehicle has been exceeded.
// This is a transient error; the client should retry after waiting.
type RateLimitExceededError struct {
	VehicleID     string
	Limit         int
	WindowSeconds int
}

func (e RateLimitExceededError) Error() string {
	if e.Limit == 0 {
		return "rate limit exceeded for vehicle: " + e.VehicleID
	}
	return "rate limit exceeded for vehicle: " + e.VehicleID +
		" (max " + string(rune(e.Limit)) + " requests per " +
		string(rune(e.WindowSeconds)) + "s)"
}
