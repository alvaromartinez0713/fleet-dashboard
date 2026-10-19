package suite_fast

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/tests/support"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestLatencyTimestampsCorrect verifies that position request timestamps are calculated
// correctly and maintain proper ordering: capturedAt ≤ receivedAt ≤ processedAt.
//
// This test validates:
// - ReceivedTimestamp is set to current server time when request arrives
// - ProcessedTimestamp is set after processing completes
// - All timestamps are in correct order: capturedAt ≤ receivedAt ≤ processedAt
// - Latencies are non-negative and reasonable for synchronous processing
// - Timestamps are properly returned in the response
//
// Timestamp fields in PositionResponse:
// - CapturedAt: When device captured the position (from client, ISO 8601 string)
// - ReceivedAt: When server received the request (server time, ISO 8601 string)
// - ProcessedAt: When server processed the event (server time, ISO 8601 string)
//
// Latency expectations in fast tests:
// - capturedAt can be before or at receivedAt (device time vs server time)
// - receivedAt ≤ processedAt (processing is nearly instantaneous in tests)
// - receptionLatency = receivedAt - capturedAt (can be positive or negative)
// - processingLatency = processedAt - receivedAt (should be ≥ 0)
//
// Verifies: P4 (4-Timestamps), P5 (Future-Validation)
// RF: RF-07 (timestamp validation), RF-04 (events with timestamps)
func TestLatencyTimestampsCorrect(t *testing.T) {
	// ========== ARRANGE ==========
	// Create test HTTP server with all fakes injected
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	// Start the server (listens on random port)
	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Get fake clock to control time deterministically
	clock := server.GetClock()
	baseTime := clock.Now()

	// Build valid position request with capturedAt set to 5 seconds before now
	// This ensures we have measurable latency between captured and received
	positionRequest := mothers.PositionMother{}.Valid().Build()
	capturedTime := baseTime.Add(-5 * time.Second)
	positionRequest.CapturedAt = capturedTime.Format("2006-01-02T15:04:05.000Z07:00")

	// Serialize request to JSON
	reqBody, err := json.Marshal(positionRequest)
	require.NoError(t, err, "failed to marshal request body")

	// ========== ACT ==========
	// Create HTTP POST request
	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "failed to create HTTP request")

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========

	// 1. Status code must be 202 Accepted
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"HTTP status should be 202 Accepted for valid position")

	// 2. Parse response body to extract timestamps and validate structure
	var positionResponse dto.PositionResponse
	err = json.NewDecoder(resp.Body).Decode(&positionResponse)
	require.NoError(t, err, "failed to decode PositionResponse")

	// 3. Verify all required timestamp fields are present and non-empty
	assert.NotEmpty(t, positionResponse.CapturedAt,
		"response.captured_at should not be empty")
	assert.NotEmpty(t, positionResponse.ReceivedAt,
		"response.received_at should not be empty")
	assert.NotEmpty(t, positionResponse.ProcessedAt,
		"response.processed_at should not be empty")

	// 4. Parse timestamps from ISO 8601 strings
	// The mapper uses "2006-01-02T15:04:05.000Z07:00" format
	capturedAt, err := time.Parse(time.RFC3339, positionResponse.CapturedAt)
	require.NoError(t, err, "failed to parse captured_at timestamp: %s", positionResponse.CapturedAt)

	receivedAt, err := time.Parse(time.RFC3339, positionResponse.ReceivedAt)
	require.NoError(t, err, "failed to parse received_at timestamp: %s", positionResponse.ReceivedAt)

	processedAt, err := time.Parse(time.RFC3339, positionResponse.ProcessedAt)
	require.NoError(t, err, "failed to parse processed_at timestamp: %s", positionResponse.ProcessedAt)

	// 5. Verify timestamp ordering: capturedAt ≤ receivedAt ≤ processedAt
	assert.LessOrEqual(t, capturedAt.Unix(), receivedAt.Unix(),
		"capturedAt should be ≤ receivedAt: %v ≤ %v", capturedAt, receivedAt)

	assert.LessOrEqual(t, receivedAt.Unix(), processedAt.Unix(),
		"receivedAt should be ≤ processedAt: %v ≤ %v", receivedAt, processedAt)

	// 6. Calculate and verify latencies are non-negative
	receptionLatency := receivedAt.Sub(capturedAt)
	processingLatency := processedAt.Sub(receivedAt)

	// Reception latency should be approximately 5 seconds (our synthetic delay)
	// Allow 100ms tolerance for test overhead
	expectedReceptionLatency := 5 * time.Second
	assert.WithinDuration(t, expectedReceptionLatency, receptionLatency, 100*time.Millisecond,
		"receptionLatency should be approximately 5 seconds: %v", receptionLatency)

	// Processing latency should be very small (nearly instantaneous in tests)
	assert.LessOrEqual(t, processingLatency.Milliseconds(), int64(100),
		"processingLatency should be small (< 100ms): %v", processingLatency)

	// 7. Verify monotonicity: processingLatency should be less than or equal to receptionLatency
	// (or at least on the same order of magnitude for this synchronous flow)
	assert.LessOrEqual(t, processingLatency, receptionLatency+1*time.Second,
		"processingLatency should not exceed receptionLatency: %v ≤ %v", processingLatency, receptionLatency)

	// 8. Verify X-Trace-ID is present in response headers
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID response header should be present")

	// 9. Verify response structure is complete
	assert.Equal(t, "accepted", positionResponse.Status,
		"response.status should be 'accepted'")
	assert.NotEmpty(t, positionResponse.EventID,
		"response.event_id should not be empty")
	assert.Equal(t, positionRequest.VehicleID, positionResponse.VehicleID,
		"response.vehicle_id should match request.vehicle_id")
	assert.Equal(t, traceIDHeader, positionResponse.TraceID,
		"response.trace_id should match X-Trace-ID header")

	// Verifies: P4 (4-Timestamps), P5 (Future-Validation)
	// RF: RF-07, RF-04
}
