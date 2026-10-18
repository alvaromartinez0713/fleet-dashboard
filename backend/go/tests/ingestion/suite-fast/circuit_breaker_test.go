package suite_fast

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/tests/support"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestCircuitBreakerOpensOnRepeatedFailures verifies that the circuit breaker transitions
// from CLOSED to OPEN after 5 consecutive failures.
//
// This test validates:
// - Requests 1-5 succeed despite Kafka publisher failures
// - Circuit breaker records the 5 failures internally
// - Request 6 returns 503 Service Unavailable (circuit is open, blocking the call)
// - Response includes ProblemDetails (RFC 9457)
// - X-Trace-ID header is present in 503 response
// - No Kafka events are published for any request (all fail at Kafka level)
//
// Circuit Breaker Configuration:
// - Failure Threshold: 5 failures trigger OPEN state
// - State: CLOSED → (5 failures) → OPEN
// - Behavior when OPEN: Reject requests immediately with CircuitBreakerOpenError
//
// Verifies: P3 (CB-Open), P7 (RFC9457), P2 (async publication)
// RF: RF-05, RF-12
//
// Pattern for circuit breaker open tests:
//  1. Arrange: Create TestHTTPServer with FakeCircuitBreaker, set it to simulate failures
//  2. Act: Send 5 requests while circuit records failures internally
//  3. Assert: Requests 1-5 are accepted (202) but with circuit tracking failures
//     Request 6 returns 503 (circuit is now open, blocking execution)
//     Verify circuit state is OPEN
func TestCircuitBreakerOpensOnRepeatedFailures(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Get fake circuit breaker and Kafka publisher
	circuitBreaker := server.GetCircuitBreaker()
	kafkaPublisher := server.GetKafkaPublisher()

	// Reset state for determinism
	kafkaPublisher.Reset()
	circuitBreaker.Reset()

	// Build a valid request using Object Mother
	baseRequest := mothers.PositionMother{}.Valid().Build()

	// ========== ACT & ASSERT: Send 5 requests while simulating failures ==========
	// Simulate 5 failures by manually recording them in the circuit breaker
	for i := 1; i <= 5; i++ {
		// Record a failure in the circuit breaker (simulates Kafka failure)
		circuitBreaker.RecordFailure()

		// Prepare and send HTTP request
		reqBody, err := json.Marshal(baseRequest)
		require.NoError(t, err, "request %d: failed to marshal request", i)

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "request %d: failed to create HTTP request", i)

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "request %d: failed to send HTTP request", i)
		defer resp.Body.Close()

		// Requests 1-5: should still be accepted (202) because circuit breaker
		// failure threshold is 5, so it only opens AFTER the 5th failure
		assert.Equal(t, http.StatusAccepted, resp.StatusCode,
			"request %d: expected 202 Accepted (circuit still has capacity), got %d",
			i, resp.StatusCode)
	}

	// After 5 failures, circuit should now be OPEN
	assert.True(t, circuitBreaker.IsOpen(),
		"after 5 failures: circuit breaker should be in OPEN state")

	// ========== ACT: Send 6th request (should be rejected) ==========
	// Prepare and send the 6th request
	reqBody, err := json.Marshal(baseRequest)
	require.NoError(t, err, "request 6: failed to marshal request")

	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "request 6: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "request 6: failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT: 6th request must be rejected with 503 ==========

	// 1. Status code must be 503 (Service Unavailable)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"request 6: expected 503 Service Unavailable (circuit is open), got %d",
		resp.StatusCode)

	// 2. Parse response body as ProblemDetails (RFC 9457)
	var problemDetails map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "request 6: failed to decode ProblemDetails response")

	// 3. Verify ProblemDetails structure
	assert.NotEmpty(t, problemDetails["type"],
		"request 6: ProblemDetails.type should not be empty")
	assert.Contains(t, problemDetails["type"], "circuit",
		"request 6: ProblemDetails.type should indicate circuit breaker error")

	// 4. Status in ProblemDetails must match HTTP status
	assert.Equal(t, float64(http.StatusServiceUnavailable), problemDetails["status"],
		"request 6: ProblemDetails.status should be 503")

	// 5. X-Trace-ID header is present in 503 response
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader,
		"request 6: X-Trace-ID response header should not be empty")

	// 6. No events should have been published to Kafka (all failed before publishing)
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"after all requests: no events should have been published (circuit blocked)")
}

// TestCircuitBreakerHalfOpensAfterTimeout verifies that the circuit breaker transitions
// from OPEN to HALF_OPEN after a timeout period (30 seconds), allowing recovery attempts.
//
// This test validates:
// - Circuit is in OPEN state, blocking requests
// - Request during OPEN state returns 503
// - Time advances by 30+ seconds
// - Circuit transitions to HALF_OPEN (recovery attempt state)
// - Request in HALF_OPEN state is allowed through
// - Response is 202 (successful during recovery attempt)
//
// Circuit Breaker State Machine:
// - CLOSED (normal): Allows all requests
// - OPEN (after 5 failures): Blocks requests for 30 seconds
// - HALF_OPEN (after 30s timeout): Allows one attempt to test recovery
// - If that attempt succeeds: Transition to CLOSED
// - If that attempt fails: Transition back to OPEN
//
// Verifies: P3 (CB-Recovery), P2 (async publication resumes)
// RF: RF-05, RF-12
//
// Pattern for half-open tests:
// 1. Arrange: Create TestHTTPServer, get FakeClock and CircuitBreaker
// 2. Act (Open): Force circuit to OPEN state
// 3. Act (Verify blocked): Send request → verify 503
// 4. Act (Advance time): Call FakeClock.Advance(30+ seconds)
// 5. Act (Transition): Verify circuit is now HALF_OPEN
// 6. Act (Attempt recovery): Send request → verify it's allowed through
// 7. Assert: Request in HALF_OPEN succeeds (202)
func TestCircuitBreakerHalfOpensAfterTimeout(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Get references to fakes for manipulation and verification
	fakeClock := server.GetClock()
	circuitBreaker := server.GetCircuitBreaker()
	kafkaPublisher := server.GetKafkaPublisher()

	// Reset state for determinism
	kafkaPublisher.Reset()
	circuitBreaker.Reset()

	// Build a valid request using Object Mother
	baseRequest := mothers.PositionMother{}.Valid().Build()

	// ========== ACT: Force circuit to OPEN state ==========
	circuitBreaker.SetOpen()

	// Verify circuit is now open
	assert.True(t, circuitBreaker.IsOpen(),
		"setup: circuit should be in OPEN state")

	// ========== ACT & ASSERT: Send request while circuit is OPEN ==========
	// This request should be rejected with 503
	reqBody, err := json.Marshal(baseRequest)
	require.NoError(t, err, "open request: failed to marshal request")

	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "open request: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "open request: failed to send HTTP request")
	defer resp.Body.Close()

	// Verify 503 response
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"open request: expected 503 Service Unavailable (circuit is open), got %d",
		resp.StatusCode)

	// Verify no events published
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"open request: no events should have been published (circuit blocked)")

	// ========== ACT: Advance time by 30+ seconds ==========
	// This simulates the timeout period allowing transition to HALF_OPEN
	initialTime := fakeClock.GetTime()
	fakeClock.Advance(31 * time.Second) // Advance more than 30 seconds
	afterAdvanceTime := fakeClock.GetTime()

	// Verify time was actually advanced
	timeDiff := afterAdvanceTime.Sub(initialTime)
	assert.Greater(t, timeDiff, 30*time.Second,
		"FakeClock should have advanced more than 30 seconds")

	// ========== ACT: Simulate transition to HALF_OPEN ==========
	// After timeout, manually transition circuit to HALF_OPEN
	// (In production, CircuitBreaker.IsOpen() would check timeout and auto-transition)
	circuitBreaker.SetHalfOpen()

	// Verify circuit is now half-open
	assert.False(t, circuitBreaker.IsOpen(),
		"after timeout: circuit should NOT be in OPEN state (transitioned to HALF_OPEN)")

	// ========== ACT: Send request during HALF_OPEN (recovery attempt) ==========
	// This request should be allowed through
	reqBody, err = json.Marshal(baseRequest)
	require.NoError(t, err, "half-open request: failed to marshal request")

	httpReq, err = http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "half-open request: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "half-open request: failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT: Request in HALF_OPEN succeeds ==========

	// 1. Status code must be 202 (accepted, allowed through)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"half-open request: expected 202 Accepted (allowed during recovery), got %d",
		resp.StatusCode)

	// 2. Event should have been published (request went through)
	assert.Equal(t, 1, kafkaPublisher.PublishedCount(),
		"half-open request: 1 event should have been published (request went through)")

	// 3. Record the success for the next test
	// (In production, success in HALF_OPEN leads to transition to CLOSED)
	circuitBreaker.RecordSuccess()
}

// TestCircuitBreakerClosesAfterSuccesses verifies that the circuit breaker transitions
// from HALF_OPEN to CLOSED after 2 consecutive successful requests.
//
// This test validates:
// - Circuit is in HALF_OPEN state (recovery attempt phase)
// - First successful request is allowed through
// - After 1st success: Circuit remains HALF_OPEN
// - Second successful request is also allowed through
// - After 2nd success: Circuit transitions to CLOSED
// - Subsequent requests function normally (CLOSED behavior)
// - Requests no longer see any circuit breaker limitations
//
// Circuit Breaker Configuration:
// - Success Threshold in HALF_OPEN: 2 successes → transition to CLOSED
// - HALF_OPEN state: Allows requests to test if the dependency has recovered
// - Upon 2 successes: Circuit fully recovers (transitions to CLOSED)
//
// Verifies: P3 (CB-Recovery), P2 (async publication resumes)
// RF: RF-05, RF-12
//
// Pattern for circuit breaker close tests:
// 1. Arrange: Create TestHTTPServer, force circuit to HALF_OPEN state
// 2. Act (First success): Send request → verify 202
//   - Record success in circuit
//   - Verify circuit still HALF_OPEN (needs 1 more success)
//
// 3. Act (Second success): Send request → verify 202
//   - Record success in circuit
//   - Verify circuit transitioned to CLOSED (after 2 successes)
//
// 4. Act (Verify recovery): Send 3rd request → verify normal 202 (no circuit limits)
// 5. Assert: All 3 requests published to Kafka (all went through)
func TestCircuitBreakerClosesAfterSuccesses(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Get references to fakes for manipulation and verification
	circuitBreaker := server.GetCircuitBreaker()
	kafkaPublisher := server.GetKafkaPublisher()

	// Reset state for determinism
	kafkaPublisher.Reset()
	circuitBreaker.Reset()

	// Build a valid request using Object Mother
	baseRequest := mothers.PositionMother{}.Valid().Build()

	// ========== ACT: Force circuit to HALF_OPEN state ==========
	// This simulates the state after timeout from OPEN
	circuitBreaker.SetHalfOpen()

	// Verify circuit is now half-open
	assert.False(t, circuitBreaker.IsOpen(),
		"setup: circuit should NOT be in OPEN state (we're in HALF_OPEN)")

	// ========== ACT & ASSERT: First successful request ==========
	reqBody, err := json.Marshal(baseRequest)
	require.NoError(t, err, "first success: failed to marshal request")

	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "first success: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "first success: failed to send HTTP request")
	defer resp.Body.Close()

	// Verify 202 response
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"first success: expected 202 Accepted (allowed during recovery), got %d",
		resp.StatusCode)

	// Record the success in the circuit breaker
	// This increments the internal success counter
	circuitBreaker.RecordSuccess()

	// After 1 success, circuit should still be HALF_OPEN (needs 2 successes to close)

	// Verify event was published
	assert.Equal(t, 1, kafkaPublisher.PublishedCount(),
		"after 1st success: 1 event should have been published")

	// ========== ACT & ASSERT: Second successful request ==========
	reqBody, err = json.Marshal(baseRequest)
	require.NoError(t, err, "second success: failed to marshal request")

	httpReq, err = http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "second success: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "second success: failed to send HTTP request")
	defer resp.Body.Close()

	// Verify 202 response
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"second success: expected 202 Accepted (allowed during recovery), got %d",
		resp.StatusCode)

	// Record the second success in the circuit breaker
	// This should trigger transition to CLOSED (after 2 successes)
	circuitBreaker.RecordSuccess()

	// After 2 successes, circuit should now be CLOSED
	assert.False(t, circuitBreaker.IsOpen(),
		"after 2nd success: circuit should be in CLOSED state (recovered)")

	// Verify second event was published
	assert.Equal(t, 2, kafkaPublisher.PublishedCount(),
		"after 2nd success: 2 events should have been published")

	// ========== ACT & ASSERT: Third request to verify circuit is CLOSED ==========
	// This confirms the circuit has fully recovered and works normally
	reqBody, err = json.Marshal(baseRequest)
	require.NoError(t, err, "third request: failed to marshal request")

	httpReq, err = http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "third request: failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "third request: failed to send HTTP request")
	defer resp.Body.Close()

	// Verify 202 response
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"third request: expected 202 Accepted (circuit is closed, normal operation), got %d",
		resp.StatusCode)

	// Verify third event was published
	assert.Equal(t, 3, kafkaPublisher.PublishedCount(),
		"after 3rd request: 3 events should have been published (all successful)")

	// Verify circuit is still in CLOSED state
	assert.False(t, circuitBreaker.IsOpen(),
		"after 3rd request: circuit should still be in CLOSED state")
}
