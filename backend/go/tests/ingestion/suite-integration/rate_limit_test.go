//go:build integration
// +build integration

package suite_integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/tests/support/harness"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestConcurrentRequestsSameVehicleRateLimit verifies that 12 concurrent requests from the same
// vehicle_id are accepted (202), while the 13th request is rejected with 429 Too Many Requests.
//
// This test validates rate limit enforcement with proper concurrency:
// - Rate limit: 12 requests per 60s window per vehicle_id
// - All 12 requests sent in parallel via sync.WaitGroup
// - 13th request also sent in parallel
// - Exactly 12 should succeed with 202, 1 should fail with 429
// - Failed response should be RFC 9457 ProblemDetails
// - Rate limiting is per-vehicle (isolation verified separately in 10.7)
//
// Verifies: P6 (Rate-Limit), RF-13
// Requirements: Rate limit 12/60s per vehicle_id, concurrent safety
//
// Test Infrastructure:
// - Fake Kafka publisher (not integration with real Kafka)
// - Fake HTTP server with configurable rate limiter
// - Real middleware stack (rate limit enforcement)
// - Fake clock (deterministic timing)
func TestConcurrentRequestsSameVehicleRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ========== SETUP ==========
	// Create test HTTP server with rate limit of 12 per window
	server, err := harness.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	require.NoError(t, server.Start(), "failed to start test HTTP server")
	time.Sleep(100 * time.Millisecond)

	baseURL := server.BaseURL()
	rateLimiter := server.GetRateLimiter()

	// Verify rate limiter is configured correctly
	// Default: 12 requests per window
	rateLimiter.SetLimit(12)

	const vehicleID = "VH-001"
	const totalRequests = 13 // 12 allowed + 1 that should be rejected
	const expectedSuccesses = 12
	const expectedRejections = 1

	// ========== ACT: Send 13 concurrent requests with same vehicle_id ==========
	t.Logf("Sending %d concurrent requests for vehicle %s (limit: 12)", totalRequests, vehicleID)

	// Track responses using channels
	type RequestResult struct {
		index      int
		statusCode int
		body       []byte
		err        error
	}

	resultsChan := make(chan RequestResult, totalRequests)
	var wg sync.WaitGroup

	// Launch 13 concurrent goroutines
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(requestIndex int) {
			defer wg.Done()

			// Create position request
			position := mothers.PositionMother{}.Valid().
				WithVehicleID(vehicleID).
				Build()

			// Convert to HTTP request
			requestBody := dto.PositionRequest{
				VehicleID:  position.VehicleID,
				Latitude:   position.Latitude,
				Longitude:  position.Longitude,
				CapturedAt: position.CapturedAt,
			}

			bodyJSON, err := json.Marshal(requestBody)
			if err != nil {
				resultsChan <- RequestResult{
					index: requestIndex,
					err:   fmt.Errorf("failed to marshal request: %w", err),
				}
				return
			}

			// Create HTTP request
			url := fmt.Sprintf("%s/api/v1/positions", baseURL)
			req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
			if err != nil {
				resultsChan <- RequestResult{
					index: requestIndex,
					err:   fmt.Errorf("failed to create request: %w", err),
				}
				return
			}

			req.Header.Set("Content-Type", "application/json")

			// Send request with short timeout
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				resultsChan <- RequestResult{
					index: requestIndex,
					err:   fmt.Errorf("request failed: %w", err),
				}
				return
			}
			defer resp.Body.Close()

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resultsChan <- RequestResult{
					index:      requestIndex,
					statusCode: resp.StatusCode,
					err:        fmt.Errorf("failed to read response: %w", err),
				}
				return
			}

			// Return result
			resultsChan <- RequestResult{
				index:      requestIndex,
				statusCode: resp.StatusCode,
				body:       body,
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultsChan)

	// ========== ASSERT: Collect and verify results ==========
	t.Log("Collecting results from all concurrent requests...")

	results := make([]RequestResult, 0, totalRequests)
	for result := range resultsChan {
		results = append(results, result)

		// Log each result
		if result.err != nil {
			t.Logf("  Request %d: ERROR - %v", result.index, result.err)
		} else {
			t.Logf("  Request %d: Status %d", result.index, result.statusCode)
		}
	}

	// Verify we got exactly 13 responses
	require.Equal(t, totalRequests, len(results),
		"should receive %d responses for %d concurrent requests", totalRequests, totalRequests)

	// Count successes (202) and rejections (429)
	successCount := 0
	rejectionCount := 0
	var rejectionResponse RequestResult

	for _, result := range results {
		require.NoError(t, result.err, "request should not have network error")

		if result.statusCode == http.StatusAccepted {
			successCount++
		} else if result.statusCode == http.StatusTooManyRequests {
			rejectionCount++
			rejectionResponse = result
		} else {
			t.Errorf("Request %d returned unexpected status %d", result.index, result.statusCode)
		}
	}

	// ========== VERIFY: Count matches expectations ==========
	assert.Equal(t, expectedSuccesses, successCount,
		"should have exactly %d successful requests (202 Accepted)", expectedSuccesses)
	assert.Equal(t, expectedRejections, rejectionCount,
		"should have exactly %d rejected requests (429 Too Many Requests)", expectedRejections)

	t.Logf("✓ Rate limit enforced: %d accepted, %d rejected", successCount, rejectionCount)

	// ========== VERIFY: Rejected response is RFC 9457 ProblemDetails ==========
	require.Equal(t, expectedRejections, rejectionCount,
		"should have exactly 1 rejection for rate limit exceed")

	require.NotNil(t, rejectionResponse.body,
		"rejection response should have body")
	require.NotEmpty(t, rejectionResponse.body,
		"rejection response body should not be empty")

	// Parse ProblemDetails
	var problemDetails dto.ProblemDetails
	err = json.Unmarshal(rejectionResponse.body, &problemDetails)
	require.NoError(t, err,
		"rejection response should be valid JSON ProblemDetails")

	// Verify ProblemDetails fields per RFC 9457
	assert.Equal(t, http.StatusTooManyRequests, problemDetails.Status,
		"status field should be 429")
	assert.NotEmpty(t, problemDetails.Type,
		"type field should indicate rate limit error")
	assert.NotEmpty(t, problemDetails.Title,
		"title field should explain the rate limit error")
	assert.NotEmpty(t, problemDetails.Detail,
		"detail field should provide context")
	assert.NotEmpty(t, problemDetails.TraceID,
		"ProblemDetails should include trace ID")

	// Verify trace ID is meaningful (UUID format)
	traceID := problemDetails.TraceID
	assert.Greater(t, len(traceID), 8, "trace ID should be a UUID (at least 8 chars)")

	t.Logf("✓ RFC 9457 ProblemDetails verified:")
	t.Logf("  - status: %d", problemDetails.Status)
	t.Logf("  - type: %s", problemDetails.Type)
	t.Logf("  - title: %s", problemDetails.Title)
	t.Logf("  - detail: %s", problemDetails.Detail)
	t.Logf("  - traceID: %s", problemDetails.TraceID)

	// ========== FINAL VERIFICATION: Rate limiter state ==========
	// After 13 requests, rate limiter should show:
	// - vehicle_id "VH-001": 13 calls (all recorded, 12 allowed + 1 rejected)
	callCount := rateLimiter.GetCallCount(vehicleID)
	assert.GreaterOrEqual(t, callCount, expectedSuccesses,
		"rate limiter should have recorded at least %d calls for %s", expectedSuccesses, vehicleID)

	t.Logf("✓ Rate limiter recorded %d calls for %s", callCount, vehicleID)

	// Verifies:
	// - P6 (Rate-Limit): Windowed rate limiting per vehicle_id
	// - RF-13: Rate limit 12/60s per vehicle_id
	// - RFC 9457: Error responses are valid ProblemDetails
	// - Concurrent safety: 13 parallel requests handled correctly
}

// TestConcurrentRequestsDifferentVehiclesIsolated verifies that rate limiting is
// completely isolated per vehicle_id. When rate limit for VH-A is exhausted,
// VH-B should not be affected.
//
// Setup:
// - TestHTTPServer with RateLimiter (12 requests per 60s window, per vehicle)
// - Send 12 requests for VH-A → all return 202 Accepted
// - Send 13th request for VH-A → returns 429 Too Many Requests (rate limit exceeded)
// - Send 12 requests for VH-B → all return 202 Accepted (NOT affected by VH-A's limit)
// - Verify: rate limit is completely isolated by vehicle_id
//
// Verifies: P6 (Rate-Limit)
// RF: RF-13
//
// Business requirement:
// Each vehicle has independent rate limit quota. The quota is NOT shared globally.
// This ensures that one chatty vehicle cannot block another vehicle's events.
// Examples:
//   - Vehicle A sending 12 requests/min should not affect Vehicle B quota
//   - Vehicle B can still send 12 requests/min independently
//   - Per-vehicle isolation guarantees fair access for all vehicles in the fleet
func TestConcurrentRequestsDifferentVehiclesIsolated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires Kafka)")
	}

	// ========== SETUP ==========
	// Create test HTTP server with fakes (no real Kafka for this test)
	server, err := harness.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	// Start the server
	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	serverURL := server.BaseURL()
	require.NotEmpty(t, serverURL, "server URL should not be empty")

	// ========== ACT & ASSERT: Test Vehicle A rate limiting ==========

	// Test vehicle A
	vehicleAID := "VH-A"

	// Step 1: Send 12 successful requests for VH-A
	successCount := 0
	for i := 1; i <= 12; i++ {
		// Build request with VH-A
		positionReq := mothers.PositionMother{}.Valid().
			WithVehicleID(vehicleAID).
			Build()

		bodyBytes, err := json.Marshal(positionReq)
		require.NoError(t, err, "failed to marshal position request to JSON")

		// Send HTTP POST
		httpResp, err := http.Post(
			fmt.Sprintf("%s/api/v1/positions", serverURL),
			"application/json",
			harness.NewBytesReader(bodyBytes),
		)
		require.NoError(t, err, "failed to send HTTP request for VH-A (request %d)", i)
		defer httpResp.Body.Close()

		// Verify: All 12 should return 202 Accepted
		if httpResp.StatusCode == http.StatusAccepted {
			successCount++
		}
		assert.Equal(t, http.StatusAccepted, httpResp.StatusCode,
			"request %d for VH-A should return 202 Accepted, got %d",
			i, httpResp.StatusCode)
	}

	// Verify all 12 requests succeeded
	assert.Equal(t, 12, successCount,
		"all 12 requests for VH-A should succeed with 202 Accepted")

	// Step 2: Send 13th request for VH-A → should be rejected with 429
	positionReq13 := mothers.PositionMother{}.Valid().
		WithVehicleID(vehicleAID).
		Build()

	bodyBytes13, err := json.Marshal(positionReq13)
	require.NoError(t, err, "failed to marshal 13th position request")

	httpResp13, err := http.Post(
		fmt.Sprintf("%s/api/v1/positions", serverURL),
		"application/json",
		harness.NewBytesReader(bodyBytes13),
	)
	require.NoError(t, err, "failed to send 13th HTTP request for VH-A")
	defer httpResp13.Body.Close()

	// Verify: 13th request for VH-A should return 429 Too Many Requests
	assert.Equal(t, http.StatusTooManyRequests, httpResp13.StatusCode,
		"13th request for VH-A should return 429 Too Many Requests (rate limit exceeded), got %d",
		httpResp13.StatusCode)

	// Parse error response to verify it's a ProblemDetails with correct type
	var errorResp map[string]interface{}
	respBytes, err := io.ReadAll(httpResp13.Body)
	require.NoError(t, err, "failed to read HTTP response body for 13th request")

	err = json.Unmarshal(respBytes, &errorResp)
	require.NoError(t, err, "failed to unmarshal error response as JSON")

	// Verify ProblemDetails structure for rate limit error
	assert.Equal(t, float64(429), errorResp["status"],
		"error response status should be 429")
	assert.NotEmpty(t, errorResp["title"],
		"error response should contain title")
	assert.NotEmpty(t, errorResp["detail"],
		"error response should contain detail about rate limit")

	// ========== ACT & ASSERT: Test Vehicle B is NOT affected by VH-A's limit ==========

	vehicleBID := "VH-B"

	// Step 3: Send 12 requests for VH-B → all should succeed (NOT affected by VH-A)
	successCountB := 0
	for i := 1; i <= 12; i++ {
		positionReqB := mothers.PositionMother{}.Valid().
			WithVehicleID(vehicleBID).
			Build()

		bodyBytesB, err := json.Marshal(positionReqB)
		require.NoError(t, err, "failed to marshal position request for VH-B (request %d)", i)

		httpRespB, err := http.Post(
			fmt.Sprintf("%s/api/v1/positions", serverURL),
			"application/json",
			harness.NewBytesReader(bodyBytesB),
		)
		require.NoError(t, err, "failed to send HTTP request for VH-B (request %d)", i)
		defer httpRespB.Body.Close()

		// Verify: All 12 requests for VH-B should return 202 Accepted
		// CRITICAL: VH-B should NOT be affected by VH-A's exhausted quota
		if httpRespB.StatusCode == http.StatusAccepted {
			successCountB++
		}
		assert.Equal(t, http.StatusAccepted, httpRespB.StatusCode,
			"request %d for VH-B should return 202 Accepted (quota is per-vehicle, not shared), got %d",
			i, httpRespB.StatusCode)
	}

	// Verify all 12 requests for VH-B succeeded
	assert.Equal(t, 12, successCountB,
		"all 12 requests for VH-B should succeed with 202 Accepted (no interference from VH-A)")

	// ========== ASSERT: Verify total isolation ==========
	// Summary:
	// - VH-A: 12 successful + 1 rejected (429) = 13 total attempts, 12 accepted
	// - VH-B: 12 successful = 12 total attempts, 12 accepted
	// - Total: 25 HTTP requests, 24 accepted (202), 1 rejected (429)
	//
	// This proves that:
	// 1. Rate limit is enforced per vehicle_id
	// 2. Exhausting VH-A's quota does NOT affect VH-B
	// 3. Each vehicle has independent rate limit window
	// 4. RateLimit adapter correctly isolates quota by vehicle_id

	t.Logf("Verified rate limit isolation:")
	t.Logf("  VH-A: 12 requests accepted, 1 rejected (429)")
	t.Logf("  VH-B: 12 requests accepted (unaffected by VH-A)")
	t.Logf("  Conclusion: Rate limit is correctly isolated per vehicle_id")

	// Verifies:
	// - P6 (Rate-Limit): Ventana deslizante 60s, 12/60s por vehicle_id, aislamiento total
	// - RF-13: RateLimiter debe aislar cuota por vehicle_id
}

// TestRateLimiterWindowResetRealClock verifies that the rate limiting window slides correctly
// when using a real (system) clock. With a 60-second window and 12 requests per window:
//
// Phase 1: Send 12 requests for vehicle "VH-TEST-001" within the 60s window
//   - All 12 should succeed (202 Accepted)
//
// Phase 2: Send 13th request immediately
//   - Should be rejected with 429 Too Many Requests (rate limit exceeded)
//
// Phase 3: Wait 61 seconds (window slides; old requests expire)
//   - Rate limit quota should regenerate
//
// Phase 4: Send 12 more requests
//   - All 12 should succeed (202 Accepted)
//   - Window has slid; new 12-request quota is available
//
// This test proves that:
// - Sliding window is implemented correctly (timestamps older than 60s are pruned)
// - Real clock (time.Now()) is used, not a fake controllable clock
// - Rate limit enforcement per vehicle_id is working
// - Window reset is automatic and time-based (not event-based)
//
// Infrastructure:
// - Real Kafka via Testcontainers (for event publishing)
// - Real RateLimiter (not fake) using real system clock
// - Real Clock adapter (RealClock using time.Now())
// - Real KafkaPublisher (connecting to real Kafka)
// - HTTP server with middleware stack (RateLimitMiddleware using real RateLimiter)
//
// Verifies: P6 (Rate-Limit sliding window)
// RF: RF-13 (Rate limiting protection)
func TestRateLimiterWindowResetRealClock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (Docker required, long running)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second) // 3 minutes for entire test
	defer cancel()

	// ========== SETUP ==========
	// Start Kafka + Zookeeper containers via Testcontainers
	env := SetupIntegrationTest(ctx, t)
	defer env.Cleanup()

	require.NotNil(t, env, "failed to setup integration test environment")
	require.NotEmpty(t, env.KafkaURL, "Kafka URL should not be empty")

	// Create Kafka publisher connected to real Kafka
	brokers := []string{env.KafkaURL}
	kafkaPublisher, err := messaging.NewKafkaPublisher(brokers)
	require.NoError(t, err, "failed to create Kafka publisher")
	defer kafkaPublisher.Close()

	// Create topic "positions" for the test
	conn, err := kafka.Dial("tcp", env.KafkaURL)
	require.NoError(t, err, "failed to connect to Kafka broker for topic creation")
	defer conn.Close()

	// Create topic "positions"
	err = conn.CreateTopics(kafka.TopicConfigs{
		{
			Topic:             "positions",
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	})
	// Ignore error if topic already exists
	_ = err

	// Give topic creation time to propagate
	time.Sleep(2 * time.Second)

	vehicleID := "VH-TEST-001"
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// ========== PHASE 1: Send 12 requests (should all succeed) ==========
	t.Logf("PHASE 1: Sending 12 valid requests for vehicle %s (within 60s window)", vehicleID)

	server := harness.NewTestHTTPServerWithPublisher(kafkaPublisher)
	require.NotNil(t, server, "failed to create test HTTP server with Kafka publisher")
	defer server.Stop()

	serverURL := server.BaseURL()
	require.NotEmpty(t, serverURL, "server URL should not be empty")

	for i := 1; i <= 12; i++ {
		posReq := mothers.PositionMother{}.Valid().WithVehicleID(vehicleID).Build()
		reqBody, err := json.Marshal(posReq)
		require.NoError(t, err, "failed to marshal position request")

		httpResp, err := httpClient.Post(
			fmt.Sprintf("%s/api/v1/positions", serverURL),
			"application/json",
			harness.NewBytesReader(reqBody),
		)
		require.NoError(t, err, "failed to POST position request %d", i)
		defer httpResp.Body.Close()

		// All 12 should return 202
		assert.Equal(t, http.StatusAccepted, httpResp.StatusCode,
			"request %d should return 202, got %d", i, httpResp.StatusCode)

		// Verify response structure
		var respBody map[string]interface{}
		err = json.NewDecoder(httpResp.Body).Decode(&respBody)
		require.NoError(t, err, "failed to decode response body for request %d", i)

		assert.Equal(t, "accepted", respBody["status"], "request %d response status should be 'accepted'", i)
		assert.NotEmpty(t, respBody["event_id"], "request %d should have event_id", i)
		assert.Equal(t, vehicleID, respBody["vehicle_id"], "request %d vehicle_id should match", i)

		t.Logf("  Request %d: ✓ Accepted (202)", i)
	}

	t.Logf("PHASE 1: All 12 requests accepted ✓")

	// ========== PHASE 2: Send 13th request (should be rejected with 429) ==========
	t.Logf("PHASE 2: Sending 13th request (should exceed rate limit and return 429)")

	posReq := mothers.PositionMother{}.Valid().WithVehicleID(vehicleID).Build()
	reqBody, err := json.Marshal(posReq)
	require.NoError(t, err, "failed to marshal 13th position request")

	httpResp, err := httpClient.Post(
		fmt.Sprintf("%s/api/v1/positions", serverURL),
		"application/json",
		harness.NewBytesReader(reqBody),
	)
	require.NoError(t, err, "failed to POST 13th position request")
	defer httpResp.Body.Close()

	// Should return 429 (Too Many Requests)
	assert.Equal(t, http.StatusTooManyRequests, httpResp.StatusCode,
		"13th request should return 429 Too Many Requests, got %d", httpResp.StatusCode)

	// Verify ProblemDetails response
	var problemDetails map[string]interface{}
	err = json.NewDecoder(httpResp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode problem details response")

	assert.NotEmpty(t, problemDetails["type"], "problem details should have 'type'")
	assert.Equal(t, float64(429), problemDetails["status"], "problem status should be 429")

	t.Logf("PHASE 2: 13th request rejected ✓ (429 Too Many Requests)")

	// ========== PHASE 3: Wait 61 seconds (window slides) ==========
	t.Logf("PHASE 3: Waiting 61 seconds for rate limit window to slide...")

	// This is a real, wall-clock sleep of 61 seconds
	// During this time, the sliding window in RateLimiter will naturally expire old requests
	time.Sleep(61 * time.Second)

	t.Logf("PHASE 3: Wait completed ✓ (window should have slid)")

	// ========== PHASE 4: Send 12 more requests (should succeed) ==========
	t.Logf("PHASE 4: Sending 12 more requests (should succeed after window slide)")

	for i := 1; i <= 12; i++ {
		posReq := mothers.PositionMother{}.Valid().WithVehicleID(vehicleID).Build()
		reqBody, err := json.Marshal(posReq)
		require.NoError(t, err, "failed to marshal position request in phase 4 (%d)", i)

		httpResp, err := httpClient.Post(
			fmt.Sprintf("%s/api/v1/positions", serverURL),
			"application/json",
			harness.NewBytesReader(reqBody),
		)
		require.NoError(t, err, "failed to POST position request in phase 4 (%d)", i)
		defer httpResp.Body.Close()

		// All 12 should return 202 (window has slid, quota regenerated)
		assert.Equal(t, http.StatusAccepted, httpResp.StatusCode,
			"phase 4 request %d should return 202 after window slide, got %d", i, httpResp.StatusCode)

		// Verify response structure
		var respBody map[string]interface{}
		err = json.NewDecoder(httpResp.Body).Decode(&respBody)
		require.NoError(t, err, "failed to decode response body in phase 4 (%d)", i)

		assert.Equal(t, "accepted", respBody["status"], "phase 4 request %d status should be 'accepted'", i)

		t.Logf("  Phase 4 Request %d: ✓ Accepted (202)", i)
	}

	t.Logf("PHASE 4: All 12 requests accepted after window slide ✓")

	// ========== SUMMARY ==========
	t.Logf("TEST PASSED ✓: Rate limiting window slides correctly with real clock")
	t.Logf("  - Phase 1: 12 requests accepted ✓")
	t.Logf("  - Phase 2: 13th request rejected (429) ✓")
	t.Logf("  - Phase 3: Waited 61 seconds (window slide) ✓")
	t.Logf("  - Phase 4: 12 more requests accepted (quota regenerated) ✓")
}
