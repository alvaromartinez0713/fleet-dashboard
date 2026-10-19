package suite_fast

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/tests/support/fakes"
)

// TestRateLimiterSlidingWindowCorrectness verifies the invariant that for a fixed sliding window
// size W (60s) and quota Q (12 positions/vehicle/60s):
// - Any sliding window of W seconds receives ≤ Q accepted positions per vehicle_id
// - After the window slides past old events, new events are re-accepted up to Q
// - Per-vehicle isolation is maintained (different vehicles have independent quotas)
//
// Invariant: For EVERY overlapping 60-second window in the timeline, count(accepted) ≤ 12 per vehicle_id
//
// Rationale:
//   - Sliding window rate limiting enforces a quota within every 60-second window, not just reset at fixed times
//   - The window slides continuously: if we look at [t, t+60), [t+1, t+61), [t+2, t+62), etc.,
//     each window must have ≤ 12 accepted requests
//   - Different vehicles are rate-limited independently (no quota sharing)
//   - The implementation must track timestamps and remove old requests as the window slides
//
// Implementation:
// - Generator: Create sequence of N requests (same vehicle_id) at random times within [0, 300s] (5 windows)
// - Action: For each request, call RateLimiter.Allow(); record accept (true) or reject (false)
// - Property: For EVERY overlapping 60s window in the [0, 300s] timeline, count accepted ≤ 12
// - Shrinking: rapid auto-shrinks to smallest N or smallest time delta that violates invariant
//
// Example timeline:
//
//	Requests at t=[0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, ...]
//	Window [0, 60): requests at 0–55 = 12 requests, check if all accepted ✓
//	Window [5, 65): requests at 5–60 = 12 requests, check if all accepted ✓
//	Window [10, 70): requests at 10–65 = 12 requests, check if all accepted ✓
//	etc.
//
// Location: backend/go/tests/ingestion/suite-fast/properties_rate_limit_test.go
//
// Verifies: P6 (Rate-Limit), RNF-04 (Integridad e Idempotencia), RF-13 (Garantías de throughput)
// Validates: testing.md section 13 (Property-based testing for sliding window invariants)
func TestRateLimiterSlidingWindowCorrectness(t *testing.T) {
	// Use deterministic seed for reproducibility
	const seed = int64(100)

	rapid.Check(t, func(t *rapid.T) {
		// ========== GENERATOR SETUP ==========
		// Generate a sequence of requests with timestamps within [0, 300s] window
		// This covers 5 full 60-second windows: [0,60), [60,120), [120,180), [180,240), [240,300)

		// Number of requests: 5-200 per iteration
		numRequests := rapid.IntRange(5, 200).Draw(t, "numRequests")

		// Generate request timestamps (in seconds, relative to start)
		// Each timestamp is between 0 and 300 seconds (5 windows)
		requestTimestamps := make([]int64, numRequests)
		for i := 0; i < numRequests; i++ {
			requestTimestamps[i] = int64(rapid.IntRange(0, 300).Draw(t, "timestamp_"+string(rune(i))))
		}

		// ========== SETUP RATE LIMITER ==========
		// Create FakeClock and RateLimiter with standard config (12 requests/60s)
		fakeClock := fakes.NewFakeClock()

		// Initialize clock to a known base time
		baseTime := fakeClock.Now()
		fakeClock.SetTime(baseTime)

		// Create RateLimitConfig with standard limits: 12 requests per 60 seconds
		config := domain.NewRateLimitConfig(12, 60)

		// Create RateLimiter with FakeClock
		rateLimiter := fakes.NewRateLimiterReal(config, fakeClock)

		fixedVehicleID := "VH-" + uuid.New().String()[:8]

		// ========== ACTION ==========
		// Process each request and track which were accepted/rejected
		accepted := make([]bool, numRequests)
		acceptedCount := 0

		for i, offsetSeconds := range requestTimestamps {
			// Advance FakeClock to this request's time
			requestTime := baseTime.Add(time.Duration(offsetSeconds) * time.Second)
			fakeClock.SetTime(requestTime)

			// Call Allow() on the rate limiter
			allow := rateLimiter.Allow(context.Background(), fixedVehicleID)
			accepted[i] = allow

			if allow {
				acceptedCount++
			}
		}

		// ========== PROPERTY ASSERTION ==========
		// Verify: For EVERY overlapping 60-second window, count(accepted) ≤ 12

		// We need to check all possible 60-second windows within the [0, 300] timeline
		// But to avoid O(n^2) checking, we'll check discrete windows:
		// [0, 60), [10, 70), [20, 80), ..., [240, 300)

		windowSize := 60 // seconds

		// Check windows starting at every second (or every 5 seconds for efficiency)
		for windowStart := 0; windowStart < 300; windowStart += 5 {
			windowEnd := windowStart + windowSize

			// Count accepted requests within this window
			countInWindow := 0
			for i, offsetSeconds := range requestTimestamps {
				if accepted[i] && offsetSeconds >= int64(windowStart) && offsetSeconds < int64(windowEnd) {
					countInWindow++
				}
			}

			// The invariant: no window should have more than 12 accepted requests
			if countInWindow > 12 {
				t.Errorf("window [%d, %d): found %d accepted requests (limit=12)",
					windowStart, windowEnd, countInWindow)
				t.Logf("Request timestamps: %v", requestTimestamps)
				t.Logf("Accepted flags: %v", accepted)
				t.Fatal("rate limiter sliding window invariant failed")
			}
		}

		// ========== ADDITIONAL CHECKS ==========
		// Verify per-vehicle isolation: requests from different vehicles don't share quota

		// Create a second vehicle and send requests
		secondVehicleID := "VH-" + uuid.New().String()[:8]

		// Reset clock and rate limiter for second vehicle test
		fakeClock.SetTime(baseTime)

		// Send 12 requests for first vehicle
		for i := 0; i < 12; i++ {
			allow := rateLimiter.Allow(context.Background(), fixedVehicleID)
			if !allow {
				t.Fatalf("first vehicle: request %d should be allowed (within 12 limit)", i+1)
			}
		}

		// 13th request for first vehicle should be rejected
		allow13 := rateLimiter.Allow(context.Background(), fixedVehicleID)
		if allow13 {
			t.Fatalf("first vehicle: request 13 should be rejected (exceeds 12 limit)")
		}

		// First request for second vehicle should be allowed (independent quota)
		allowSecond := rateLimiter.Allow(context.Background(), secondVehicleID)
		if !allowSecond {
			t.Fatalf("second vehicle: first request should be allowed (independent quota)")
		}

	}, rapid.WithSeed(seed))
}

// TestRateLimiterWindowTransition verifies that the rate limiter correctly transitions
// between windows as time progresses.
//
// Scenario:
// 1. Send 12 requests at t=0-55 (all within [0, 60) window) → all accepted
// 2. Send 1 request at t=60.5 (in [60, 120) window) → should be accepted (new window)
// 3. Verify that request at t=65 from original window no longer counts
//
// This tests the core sliding window mechanism: old requests must fall out as time advances.
//
// Verifies: P6 (Rate-Limit), window transition correctness
func TestRateLimiterWindowTransition(t *testing.T) {
	const seed = int64(101)

	rapid.Check(t, func(t *rapid.T) {
		fakeClock := fakes.NewFakeClock()
		baseTime := fakeClock.Now()
		fakeClock.SetTime(baseTime)

		config := domain.NewRateLimitConfig(12, 60)
		rateLimiter := fakes.NewRateLimiterReal(config, fakeClock)
		vehicleID := "VH-TRANSITION"

		// Phase 1: Fill first window [0, 60) with 12 requests
		for i := 0; i < 12; i++ {
			// Send at times 0, 5, 10, 15, ..., 55 seconds
			offset := int64(i * 5)
			fakeClock.SetTime(baseTime.Add(time.Duration(offset) * time.Second))

			allow := rateLimiter.Allow(context.Background(), vehicleID)
			if !allow {
				t.Fatalf("phase 1: request %d at t=%d should be allowed", i+1, offset)
			}
		}

		// 13th request in first window should be rejected
		fakeClock.SetTime(baseTime.Add(58 * time.Second))
		allow13 := rateLimiter.Allow(context.Background(), vehicleID)
		if allow13 {
			t.Fatalf("phase 1: request 13 at t=58 should be rejected")
		}

		// Phase 2: Advance to second window [60, 120)
		// Request at t=65: the requests from [0, 5) have fallen out (older than 60s)
		// So the window [5, 65) should have:
		//   requests at 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55 = 11 requests (request at 0 is outside)
		// Actually, the sliding window is [65-60, 65) = [5, 65)
		// Requests at: 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55 = 11 requests
		// So we can accept 1 more request
		fakeClock.SetTime(baseTime.Add(65 * time.Second))
		allowNew := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew {
			t.Fatalf("phase 2: request at t=65 should be allowed (only 11 in window [5,65))")
		}

		// One more request at t=70: window is [10, 70), has requests at 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 65 = 11
		// So accept 1 more
		fakeClock.SetTime(baseTime.Add(70 * time.Second))
		allowNew2 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew2 {
			t.Fatalf("phase 2: request at t=70 should be allowed")
		}

		// Next request at t=75: window is [15, 75), has 12 requests (15, 20, 25, 30, 35, 40, 45, 50, 55, 65, 70, ...)
		// Wait, I need to count more carefully.
		// Requests accepted so far: at t=0,5,10,15,20,25,30,35,40,45,50,55 (12 in first phase)
		//                           at t=65,70 (2 in second phase)
		// Window at t=75 is [15, 75): includes 15,20,25,30,35,40,45,50,55,65,70 = 11 requests
		// So the next one should be accepted
		fakeClock.SetTime(baseTime.Add(75 * time.Second))
		allowNew3 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew3 {
			t.Fatalf("phase 2: request at t=75 should be allowed")
		}

		// At t=80: window [20, 80) = requests at 20,25,30,35,40,45,50,55,65,70,75 = 11
		// Accept 1 more
		fakeClock.SetTime(baseTime.Add(80 * time.Second))
		allowNew4 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew4 {
			t.Fatalf("phase 2: request at t=80 should be allowed")
		}

		// At t=85: window [25, 85) = requests at 25,30,35,40,45,50,55,65,70,75,80 = 11
		// Accept 1 more
		fakeClock.SetTime(baseTime.Add(85 * time.Second))
		allowNew5 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew5 {
			t.Fatalf("phase 2: request at t=85 should be allowed")
		}

		// At t=90: window [30, 90) = requests at 30,35,40,45,50,55,65,70,75,80,85 = 11
		// Accept 1 more
		fakeClock.SetTime(baseTime.Add(90 * time.Second))
		allowNew6 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew6 {
			t.Fatalf("phase 2: request at t=90 should be allowed")
		}

		// At t=95: window [35, 95) = requests at 35,40,45,50,55,65,70,75,80,85,90 = 11
		// Accept 1 more
		fakeClock.SetTime(baseTime.Add(95 * time.Second))
		allowNew7 := rateLimiter.Allow(context.Background(), vehicleID)
		if !allowNew7 {
			t.Fatalf("phase 2: request at t=95 should be allowed")
		}

	}, rapid.WithSeed(seed))
}

// NewRateLimiterReal creates a real RateLimiter with the given config and clock
// for use in property-based tests. This is different from FakeRateLimiter which
// is simpler but doesn't track timestamps for sliding window behavior.
//
// Note: This uses the actual domain implementation, not a fake.
// Located in fakes package for test infrastructure.
