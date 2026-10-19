package fakes

import (
	"context"
	"sync"

	"github.com/fleet-platform/backend/internal/ingestion/adapters/outbound/resilience"
	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// FakeRateLimiter is a test implementation of ports.RateLimiter.
// It allows tests to control whether requests are allowed or rejected based on a configurable limit.
// This fake does NOT track timestamps, so it doesn't properly implement sliding window behavior.
// For property-based tests that require real sliding window behavior, use NewRateLimiterReal().
//
// Thread-safe: all operations are protected by sync.Mutex.
type FakeRateLimiter struct {
	vehicleCallCounts    map[string]int
	allowedCallsPerWindow int
	mutex                sync.Mutex
}

// Verify that FakeRateLimiter implements ports.RateLimiter.
var _ ports.RateLimiter = (*FakeRateLimiter)(nil)

// NewFakeRateLimiter creates a new FakeRateLimiter with a specified call limit.
func NewFakeRateLimiter(limit int) *FakeRateLimiter {
	return &FakeRateLimiter{
		vehicleCallCounts:    make(map[string]int),
		allowedCallsPerWindow: limit,
	}
}

// Allow checks if a request from the given vehicle is allowed.
// Returns true if the call count is below the limit, false otherwise.
func (f *FakeRateLimiter) Allow(ctx context.Context, vehicleID string) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	count := f.vehicleCallCounts[vehicleID]
	f.vehicleCallCounts[vehicleID] = count + 1

	return count < f.allowedCallsPerWindow
}

// GetCallCount returns the total number of calls for a vehicle.
func (f *FakeRateLimiter) GetCallCount(vehicleID string) int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.vehicleCallCounts[vehicleID]
}

// Reset clears all call counts for all vehicles.
func (f *FakeRateLimiter) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.vehicleCallCounts = make(map[string]int)
}

// ResetForVehicle clears the call count for a specific vehicle.
func (f *FakeRateLimiter) ResetForVehicle(vehicleID string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	delete(f.vehicleCallCounts, vehicleID)
}

// SetLimit changes the allowed calls per window.
func (f *FakeRateLimiter) SetLimit(limit int) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.allowedCallsPerWindow = limit
}

// NewRateLimiterReal creates a real RateLimiterImpl with the given config and clock.
// This is used in property-based tests that require actual sliding window behavior.
// Unlike FakeRateLimiter, this implementation tracks timestamps and enforces the
// sliding window invariant correctly.
//
// Args:
//   config: RateLimitConfig (e.g., 12 requests per 60 seconds)
//   clock: Clock port (can be FakeClock for deterministic testing)
//
// Returns:
//   *resilience.RateLimiterImpl: Real rate limiter with proper sliding window semantics
func NewRateLimiterReal(config domain.RateLimitConfig, clock ports.Clock) *resilience.RateLimiterImpl {
	return resilience.NewRateLimiter(config, clock)
}
