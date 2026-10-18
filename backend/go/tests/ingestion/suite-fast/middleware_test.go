package suite_fast

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/tests/support"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestInvalidContentTypeReturns415 verifies that a position request with Content-Type
// other than "application/json" is rejected with status 415 (Unsupported Media Type).
//
// This test validates:
// - HTTP status code is 415 (Unsupported Media Type)
// - Response body is ProblemDetails (RFC 9457)
// - ProblemDetails.Type indicates unsupported media type error
// - ProblemDetails.Title describes the issue
// - ProblemDetails.Status == 415
// - ProblemDetails.Detail mentions Content-Type requirement
// - X-Trace-ID header present in response
// - Response Content-Type is application/problem+json
//
// The ContentTypeMiddleware rejects requests with:
// - Content-Type: text/plain
// - Content-Type: application/xml
// - Content-Type: application/octet-stream
// - Missing Content-Type header for POST requests
//
// Verifies: P7 (RFC9457), P9 (Trace-ID)
// RF: RF-01, RF-08, RF-12
//
// Pattern for middleware validation tests:
// 1. Arrange: Create TestHTTPServer with fakes, build valid PositionRequest
// 2. Act: POST to /api/v1/positions with wrong Content-Type header
// 3. Assert: Verify observable outputs (status, ProblemDetails, headers)
func TestInvalidContentTypeReturns415(t *testing.T) {
	// ========== ARRANGE ==========
	// Create test HTTP server with all fakes injected
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	// Start the server (listens on random port)
	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Build valid position request using Object Mother
	// Valid() returns a builder with all fields set to valid values (VH-001, Manhattan, 10 seconds ago)
	request := mothers.PositionMother{}.Valid().Build()

	// Serialize request to JSON
	reqBody, err := json.Marshal(request)
	require.NoError(t, err, "failed to marshal request")

	// ========== ACT ==========
	// Create HTTP POST request
	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "failed to create HTTP request")

	// Set INVALID Content-Type header (text/plain instead of application/json)
	httpReq.Header.Set("Content-Type", "text/plain")

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========

	// 1. Status code must be 415 (Unsupported Media Type)
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode,
		"expected 415 Unsupported Media Type for invalid Content-Type, got %d", resp.StatusCode)

	// 2. Response Content-Type must be application/problem+json
	responseContentType := resp.Header.Get("Content-Type")
	assert.Equal(t, "application/problem+json", responseContentType,
		"response Content-Type should be application/problem+json, got %s", responseContentType)

	// 3. Parse response body as ProblemDetails (RFC 9457)
	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode ProblemDetails response")

	// 4. ProblemDetails.Type must indicate unsupported media type error
	assert.NotEmpty(t, problemDetails.Type, "ProblemDetails.type should not be empty")
	assert.Contains(t, problemDetails.Type, "unsupported-media-type",
		"ProblemDetails.type should indicate unsupported media type: %s", problemDetails.Type)

	// 5. ProblemDetails.Title describes the issue
	assert.NotEmpty(t, problemDetails.Title, "ProblemDetails.title should not be empty")
	assert.Contains(t, problemDetails.Title, "Media Type",
		"ProblemDetails.title should mention media type")

	// 6. ProblemDetails.Status == 415
	assert.Equal(t, 415, problemDetails.Status,
		"ProblemDetails.status should be 415")

	// 7. ProblemDetails.Detail mentions Content-Type requirement
	assert.NotEmpty(t, problemDetails.Detail, "ProblemDetails.detail should not be empty")
	assert.Contains(t, problemDetails.Detail, "application/json",
		"ProblemDetails.detail should mention application/json requirement: %s", problemDetails.Detail)

	// 8. X-Trace-ID header must be present in response
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID response header should not be empty")
}


// TestIdempotencyKeyInvalidUUIDReturns400 verifies that a request with an invalid
// X-Idempotency-Key header (non-UUID v4 format) is rejected with status 400 Bad Request.
//
// This test validates:
// - HTTP status code is 400 (Bad Request) when header is malformed
// - Response body is ProblemDetails (RFC 9457)
// - ProblemDetails.Type indicates validation/client error
// - ProblemDetails.Status == 400
// - ProblemDetails.Detail mentions idempotency key
// - X-Trace-ID response header is present
// - No event is published to Kafka (request rejected before processing)
//
// The X-Idempotency-Key header format:
// - If absent: request is allowed (optional header)
// - If present and invalid UUID v4 format: request rejected with 400
// - If present and valid UUID v4: request is processed normally
//
// Test case: X-Idempotency-Key: "invalid-uuid" (not a UUID)
//
// Verifies: P1 (Validation), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-07
func TestIdempotencyKeyInvalidUUIDReturns400(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	request := mothers.PositionMother{}.Valid().Build()
	reqBody, err := json.Marshal(request)
	require.NoError(t, err, "failed to marshal request")

	// ========== ACT ==========
	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Idempotency-Key", "invalid-uuid")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"expected 400 Bad Request for invalid X-Idempotency-Key, got %d", resp.StatusCode)

	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode response body as ProblemDetails")

	assert.NotEmpty(t, problemDetails.Type,
		"ProblemDetails.Type should not be empty")
	assert.Contains(t, problemDetails.Type, "invalid",
		"ProblemDetails.Type should indicate invalid header: %s", problemDetails.Type)
	assert.Equal(t, http.StatusBadRequest, problemDetails.Status,
		"ProblemDetails.Status should be 400")
	assert.NotEmpty(t, problemDetails.Detail,
		"ProblemDetails.Detail should not be empty")
	assert.Contains(t, problemDetails.Detail, "idempotency",
		"ProblemDetails.Detail should mention 'idempotency': %s", problemDetails.Detail)

	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty")

	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published to Kafka (request was rejected)")
}

// TestIdempotencyKeyAbsentAllowed verifies that a request WITHOUT the X-Idempotency-Key header
// is allowed and processed normally with status 202 Accepted.
//
// This test validates:
// - HTTP status code is 202 (Accepted) when X-Idempotency-Key header is absent
// - Response body has correct PositionResponse structure
// - Response.Status == "accepted"
// - Response.EventID is present and UUID v4
// - X-Trace-ID header is present
// - Event was published to Kafka (request processed normally)
//
// The X-Idempotency-Key header is OPTIONAL. Absence should not block the request.
//
// Test case: No X-Idempotency-Key header in the request
//
// Verifies: P1 (Validation), P2 (Async+202), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-07, RF-01, RF-04
func TestIdempotencyKeyAbsentAllowed(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	request := mothers.PositionMother{}.Valid().Build()
	reqBody, err := json.Marshal(request)
	require.NoError(t, err, "failed to marshal request")

	// ========== ACT ==========
	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")
	// DO NOT set X-Idempotency-Key header

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"expected 202 Accepted when X-Idempotency-Key is absent, got %d", resp.StatusCode)

	var response dto.PositionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "failed to decode response body as PositionResponse")

	assert.Equal(t, "accepted", response.Status,
		"response.status should be 'accepted'")
	assert.NotEmpty(t, response.EventID, "response.event_id should not be empty")
	assert.Equal(t, request.VehicleID, response.VehicleID,
		"response.vehicle_id should match request.vehicle_id")

	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty")

	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 1, kafkaPublisher.PublishedCount(),
		"exactly 1 event should have been published to Kafka")
}

// TestIdempotencyKeyValidUUIDAccepted verifies that a request with a valid UUID v4 format
// X-Idempotency-Key header is accepted and processed normally with status 202 Accepted.
//
// This test validates:
// - HTTP status code is 202 (Accepted) when X-Idempotency-Key is valid UUID v4
// - Response body has correct PositionResponse structure
// - Response.Status == "accepted"
// - Response.EventID is present and UUID v4
// - X-Trace-ID header is present
// - Event was published to Kafka (request processed)
//
// The X-Idempotency-Key header, when present, MUST be a valid UUID v4.
// If valid, the request is processed normally as if the header provides
// idempotency guarantees.
//
// Test case: X-Idempotency-Key: <valid-uuid-v4>
//
// Verifies: P1 (Validation), P2 (Async+202), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-07, RF-01, RF-04
func TestIdempotencyKeyValidUUIDAccepted(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	request := mothers.PositionMother{}.Valid().Build()
	reqBody, err := json.Marshal(request)
	require.NoError(t, err, "failed to marshal request")

	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// ========== ACT ==========
	httpReq, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err, "failed to create HTTP request")

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Idempotency-Key", validUUID)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========
	assert.Equal(t, http.StatusAccepted, resp.StatusCode,
		"expected 202 Accepted for valid X-Idempotency-Key, got %d", resp.StatusCode)

	var response dto.PositionResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "failed to decode response body as PositionResponse")

	assert.Equal(t, "accepted", response.Status,
		"response.status should be 'accepted'")
	assert.NotEmpty(t, response.EventID, "response.event_id should not be empty")
	assert.Equal(t, request.VehicleID, response.VehicleID,
		"response.vehicle_id should match request.vehicle_id")

	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty")

	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 1, kafkaPublisher.PublishedCount(),
		"exactly 1 event should have been published to Kafka")
}

// TestResponseIncludesTraceIDHeader verifies that all responses include the X-Trace-ID header
// with a valid UUID v4 value.
//
// This test validates:
// - Request WITHOUT X-Trace-ID header: middleware generates a new UUID v4
// - Request WITH valid X-Trace-ID header: middleware uses the provided UUID v4
// - Response ALWAYS includes X-Trace-ID header with UUID v4
// - Success responses (202) include X-Trace-ID header
// - Error responses (422, 429, 415, etc.) include X-Trace-ID header
// - X-Trace-ID values are valid UUID v4 format
//
// Test cases:
// 1. POST /api/v1/positions without X-Trace-ID → 202, response includes generated X-Trace-ID header
// 2. POST /api/v1/positions with valid X-Trace-ID → 202, response includes same X-Trace-ID header
// 3. POST /api/v1/positions (invalid) without X-Trace-ID → 422, response includes generated X-Trace-ID header
// 4. POST /api/v1/positions (invalid) with valid X-Trace-ID → 422, response includes same X-Trace-ID header
// 5. POST /api/v1/positions (rate limited) → 429, response includes X-Trace-ID header
//
// The TraceIDMiddleware runs FIRST (outermost) in the middleware chain:
// - Generates or validates X-Trace-ID from X-Trace-ID header
// - Writes X-Trace-ID to response header before c.Next() to ensure it's present in all responses
// - Stores trace ID in gin.Context for downstream use
// - Errors from downstream middleware are caught by ErrorHandlerMiddleware, which still has the trace ID
//
// Verifies: P9 (Trace-ID), P7 (RFC9457)
// RF: RF-08, RF-07, RF-12
func TestResponseIncludesTraceIDHeader(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Test Case 1: Request WITHOUT X-Trace-ID header → middleware generates UUID
	t.Run("case_1_no_trace_id_generates_new", func(t *testing.T) {
		request := mothers.PositionMother{}.Valid().Build()
		reqBody, err := json.Marshal(request)
		require.NoError(t, err, "failed to marshal request")

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "failed to create HTTP request")

		httpReq.Header.Set("Content-Type", "application/json")
		// DO NOT set X-Trace-ID header

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "failed to send HTTP request")
		defer resp.Body.Close()

		// Assert: Response includes X-Trace-ID header
		traceIDHeader := resp.Header.Get("X-Trace-ID")
		assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty")

		// Assert: X-Trace-ID is valid UUID v4
		assert.True(t, isValidUUIDv4(traceIDHeader),
			"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)

		// Assert: Response status is 202 Accepted
		assert.Equal(t, http.StatusAccepted, resp.StatusCode,
			"expected 202 Accepted, got %d", resp.StatusCode)
	})

	// Test Case 2: Request WITH valid X-Trace-ID header → middleware uses it
	t.Run("case_2_with_trace_id_uses_it", func(t *testing.T) {
		request := mothers.PositionMother{}.Valid().Build()
		reqBody, err := json.Marshal(request)
		require.NoError(t, err, "failed to marshal request")

		providedTraceID := "550e8400-e29b-41d4-a716-446655440000"

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "failed to create HTTP request")

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Trace-ID", providedTraceID)

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "failed to send HTTP request")
		defer resp.Body.Close()

		// Assert: Response includes X-Trace-ID header with SAME value as request
		traceIDHeader := resp.Header.Get("X-Trace-ID")
		assert.Equal(t, providedTraceID, traceIDHeader,
			"X-Trace-ID header should match request value")

		// Assert: X-Trace-ID is valid UUID v4
		assert.True(t, isValidUUIDv4(traceIDHeader),
			"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)

		// Assert: Response status is 202 Accepted
		assert.Equal(t, http.StatusAccepted, resp.StatusCode,
			"expected 202 Accepted, got %d", resp.StatusCode)
	})

	// Test Case 3: Invalid request WITHOUT X-Trace-ID → error response includes generated X-Trace-ID
	t.Run("case_3_error_no_trace_id_generates_new", func(t *testing.T) {
		// Use invalid request (future timestamp)
		request := mothers.PositionMother{}.FutureTimestamp().Build()
		reqBody, err := json.Marshal(request)
		require.NoError(t, err, "failed to marshal request")

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "failed to create HTTP request")

		httpReq.Header.Set("Content-Type", "application/json")
		// DO NOT set X-Trace-ID header

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "failed to send HTTP request")
		defer resp.Body.Close()

		// Assert: Response includes X-Trace-ID header (even for error responses)
		traceIDHeader := resp.Header.Get("X-Trace-ID")
		assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty (even for error)")

		// Assert: X-Trace-ID is valid UUID v4
		assert.True(t, isValidUUIDv4(traceIDHeader),
			"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)

		// Assert: Response status is 422 (Unprocessable Content)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
			"expected 422 for invalid data, got %d", resp.StatusCode)

		// Assert: Response body is ProblemDetails with trace ID
		var problemDetails dto.ProblemDetails
		err = json.NewDecoder(resp.Body).Decode(&problemDetails)
		require.NoError(t, err, "failed to decode ProblemDetails")

		// ProblemDetails may include trace_id as optional extension (RFC 9457)
		// but it's not required; the header is the key verification
	})

	// Test Case 4: Invalid request WITH valid X-Trace-ID → error response includes same X-Trace-ID
	t.Run("case_4_error_with_trace_id_uses_it", func(t *testing.T) {
		// Use invalid request (out of range latitude)
		request := mothers.PositionMother{}.OutOfRangeLatitude().Build()
		reqBody, err := json.Marshal(request)
		require.NoError(t, err, "failed to marshal request")

		providedTraceID := "660e8400-e29b-41d4-a716-446655440001"

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "failed to create HTTP request")

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Trace-ID", providedTraceID)

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "failed to send HTTP request")
		defer resp.Body.Close()

		// Assert: Response includes X-Trace-ID header with SAME value (even for error)
		traceIDHeader := resp.Header.Get("X-Trace-ID")
		assert.Equal(t, providedTraceID, traceIDHeader,
			"X-Trace-ID header should match request value (even for error)")

		// Assert: X-Trace-ID is valid UUID v4
		assert.True(t, isValidUUIDv4(traceIDHeader),
			"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)

		// Assert: Response status is 422 (Unprocessable Content)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
			"expected 422 for invalid data, got %d", resp.StatusCode)
	})

	// Test Case 5: Rate limited request → 429 response includes X-Trace-ID
	t.Run("case_5_rate_limit_error_includes_trace_id", func(t *testing.T) {
		// Create a new server for this test to get a fresh rate limiter
		server2, err := support.NewTestHTTPServer()
		require.NoError(t, err, "failed to create test HTTP server")
		defer server2.Stop()

		err = server2.Start()
		require.NoError(t, err, "failed to start test HTTP server")

		// Send 13 requests to exceed default rate limit (12/60s per vehicle)
		vehicleID := "VH-RATE-TEST"

		for i := 0; i < 13; i++ {
			request := mothers.PositionMother{}.Valid().WithVehicleID(vehicleID).Build()
			reqBody, err := json.Marshal(request)
			require.NoError(t, err, "failed to marshal request")

			httpReq, err := http.NewRequest(
				http.MethodPost,
				server2.BaseURL()+"/api/v1/positions",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err, "failed to create HTTP request")

			httpReq.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(httpReq)
			require.NoError(t, err, "failed to send HTTP request")

			if i < 12 {
				// First 12 requests should succeed (202)
				assert.Equal(t, http.StatusAccepted, resp.StatusCode,
					"request %d should succeed with 202", i+1)

				// Each response should include X-Trace-ID
				traceIDHeader := resp.Header.Get("X-Trace-ID")
				assert.NotEmpty(t, traceIDHeader,
					"X-Trace-ID should be present in success response %d", i+1)
				assert.True(t, isValidUUIDv4(traceIDHeader),
					"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)
			} else {
				// 13th request should be rate limited (429)
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
					"request %d should fail with 429", i+1)

				// Rate limit error response MUST include X-Trace-ID header
				traceIDHeader := resp.Header.Get("X-Trace-ID")
				assert.NotEmpty(t, traceIDHeader,
					"X-Trace-ID should be present even in rate limit error (429)")
				assert.True(t, isValidUUIDv4(traceIDHeader),
					"X-Trace-ID should be valid UUID v4 even in error, got: %s", traceIDHeader)
			}

			resp.Body.Close()
		}
	})

	// Test Case 6: Invalid Content-Type → 415 response includes X-Trace-ID
	t.Run("case_6_unsupported_media_type_includes_trace_id", func(t *testing.T) {
		request := mothers.PositionMother{}.Valid().Build()
		reqBody, err := json.Marshal(request)
		require.NoError(t, err, "failed to marshal request")

		httpReq, err := http.NewRequest(
			http.MethodPost,
			server.BaseURL()+"/api/v1/positions",
			bytes.NewReader(reqBody),
		)
		require.NoError(t, err, "failed to create HTTP request")

		// Invalid Content-Type
		httpReq.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(httpReq)
		require.NoError(t, err, "failed to send HTTP request")
		defer resp.Body.Close()

		// Assert: Response includes X-Trace-ID header
		traceIDHeader := resp.Header.Get("X-Trace-ID")
		assert.NotEmpty(t, traceIDHeader, "X-Trace-ID header should not be empty (even for 415)")

		// Assert: X-Trace-ID is valid UUID v4
		assert.True(t, isValidUUIDv4(traceIDHeader),
			"X-Trace-ID should be valid UUID v4, got: %s", traceIDHeader)

		// Assert: Response status is 415 (Unsupported Media Type)
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode,
			"expected 415 for unsupported media type, got %d", resp.StatusCode)
	})
}

// isValidUUIDv4 checks if a string is a valid UUID v4 format.
// Returns true if the string is a valid UUID v4, false otherwise.
func isValidUUIDv4(uuidStr string) bool {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return false
	}
	return parsed.Version() == 4
}
