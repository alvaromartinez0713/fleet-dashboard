package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// RateLimiterImpl implements a sliding-window rate limiter per vehicle_id.
// Default: 12 requests per 60s window.
// Thread-safe with sync.RWMutex.
// Each vehicle_id has an isolated rate limit (no sharing across vehicles).
type RateLimiterImpl struct {
	mu              sync.RWMutex
	config          domain.RateLimitConfig
	requestsPerKey  map[string][]time.Time // vehicle_id → timestamps of requests
	clock           ports.Clock            // Clock for retrieving current time
}

// NewRateLimiter creates a new rate limiter with the given config and clock.
func NewRateLimiter(config domain.RateLimitConfig, clock ports.Clock) *RateLimiterImpl {
	return &RateLimiterImpl{
		config:         config,
		requestsPerKey: make(map[string][]time.Time),
		clock:          clock,
	}
}

// Allow checks if a request from the given vehicle_id is allowed under the rate limit.
// Uses a sliding window: removes timestamps older than the window, then checks if count < limit.
// Returns true if the request is allowed, false if rate limit exceeded.
// Implements ports.RateLimiter interface.
func (rl *RateLimiterImpl) Allow(ctx context.Context, vehicleID string) bool {
	now := rl.clock.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create the request list for this vehicle
	timestamps, exists := rl.requestsPerKey[vehicleID]
	if !exists {
		timestamps = []time.Time{}
	}

	// Remove timestamps outside the sliding window (older than windowSize)
	windowStart := now.Add(-time.Duration(rl.config.WindowSize()) * time.Second)
	validTimestamps := []time.Time{}
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if we can accept another request
	if len(validTimestamps) < int(rl.config.RequestsPerWindow()) {
		// Add the current request timestamp
		validTimestamps = append(validTimestamps, now)
		rl.requestsPerKey[vehicleID] = validTimestamps
		return true
	}

	// Rate limit exceeded
	rl.requestsPerKey[vehicleID] = validTimestamps
	return false
}

// Reset clears all rate limit data for the given vehicle_id.
// Used primarily for testing and cleanup.
func (rl *RateLimiterImpl) Reset(vehicleID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requestsPerKey, vehicleID)
}

// ResetAll clears all rate limit data for all vehicles.
// Used primarily for testing and cleanup.
func (rl *RateLimiterImpl) ResetAll() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requestsPerKey = make(map[string][]time.Time)
}

// GetCount returns the current request count for a vehicle_id within the window.
// Useful for testing and diagnostics.
func (rl *RateLimiterImpl) GetCount(vehicleID string, now time.Time) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	timestamps, exists := rl.requestsPerKey[vehicleID]
	if !exists {
		return 0
	}

	// Count valid timestamps within the window
	windowStart := now.Add(-time.Duration(rl.config.WindowSize()) * time.Second)
	count := 0
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			count++
		}
	}
	return count
}
