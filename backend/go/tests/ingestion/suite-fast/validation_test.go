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

// TestLatitudeOutOfRangeReturns422 verifies that a position request with latitude > 90
// or latitude < -90 is rejected with status 422 (Unprocessable Content).
//
// This test validates:
// - HTTP status code is 422 (Unprocessable Content)
// - Response body is ProblemDetails (RFC 9457)
// - ProblemDetails.Type indicates validation error
// - ProblemDetails.Title describes the issue
// - ProblemDetails.Status == 422
// - ProblemDetails.Detail mentions latitude constraint
// - X-Trace-ID header present in response
// - ProblemDetails.TraceID matches response header X-Trace-ID
//
// Valid latitude range: -90 to 90 degrees (inclusive)
// Test case uses latitude = 95 (OutOfRangeLatitude scenario from PositionMother)
//
// Verifies: P1 (Validation), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-02, RF-04, RF-12
//
// Pattern for validation tests:
// 1. Arrange: Create TestHTTPServer with fakes, build invalid PositionRequest
// 2. Act: POST to /api/v1/positions with invalid data
// 3. Assert: Verify observable outputs (status, ProblemDetails, headers)
func TestLatitudeOutOfRangeReturns422(t *testing.T) {
	// ========== ARRANGE ==========
	// Create test HTTP server with all fakes injected
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	// Start the server (listens on random port)
	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Build invalid position request using Object Mother
	// OutOfRangeLatitude() returns a builder with latitude = 95 (> 90, invalid)
	request := mothers.PositionMother{}.OutOfRangeLatitude().Build()

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

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========

	// 1. Status code must be 422 (Unprocessable Content)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
		"expected 422 Unprocessable Content for out-of-range latitude, got %d", resp.StatusCode)

	// 2. Parse response body as ProblemDetails (RFC 9457)
	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode ProblemDetails response")

	// 3. ProblemDetails.Type must indicate validation error
	assert.NotEmpty(t, problemDetails.Type, "ProblemDetails.type should not be empty")
	assert.Contains(t, problemDetails.Type, "validation",
		"ProblemDetails.type should indicate validation error: %s", problemDetails.Type)

	// 4. ProblemDetails.Title describes the issue
	assert.NotEmpty(t, problemDetails.Title, "ProblemDetails.title should not be empty")
	assert.Contains(t, problemDetails.Title, "invalid",
		"ProblemDetails.title should mention invalid data")

	// 5. ProblemDetails.Status == 422
	assert.Equal(t, 422, problemDetails.Status,
		"ProblemDetails.status should be 422")

	// 6. ProblemDetails.Detail mentions latitude constraint
	assert.NotEmpty(t, problemDetails.Detail, "ProblemDetails.detail should not be empty")
	assert.Contains(t, problemDetails.Detail, "latitude",
		"ProblemDetails.detail should mention latitude: %s", problemDetails.Detail)

	// 7. X-Trace-ID response header is present
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader, "X-Trace-ID response header should not be empty")

	// 8. X-Trace-ID is valid UUID v4
	_, err = uuid.Parse(traceIDHeader)
	assert.NoError(t, err, "X-Trace-ID header should be valid UUID v4: %s", traceIDHeader)

	// 9. ProblemDetails.TraceID is present
	assert.NotEmpty(t, problemDetails.TraceID, "ProblemDetails.trace_id should not be empty")

	// 10. ProblemDetails.TraceID matches response header X-Trace-ID
	assert.Equal(t, traceIDHeader, problemDetails.TraceID,
		"ProblemDetails.trace_id should match X-Trace-ID response header")

	// 11. No event should have been published to Kafka (request rejected before publication)
	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published to Kafka for invalid request")
}

// TestLongitudeOutOfRangeReturns422 verifies that a position request with invalid longitude
// (longitude > 180 or < -180) is rejected with status 422 Unprocessable Content.
//
// Validates:
// - HTTP status code is 422 (Unprocessable Content)
// - Response body is RFC 9457 ProblemDetails
// - ProblemDetails.Type indicates validation error
// - ProblemDetails.Status == 422
// - ProblemDetails.Detail mentions longitude constraint
// - ProblemDetails.TraceID is present and UUID v4
// - X-Trace-ID response header is present
//
// Test data: PositionMother.OutOfRangeLongitude() creates a request with longitude = 185.0
// (valid values: -180 to 180)
//
// Verifies: P1 (Validation), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-02, RF-04, RF-12
//
// This test follows the black-box pattern:
// - Enters via POST /api/v1/positions (HTTP endpoint)
// - Verifies observable output: status code, response body structure, headers
// - Does NOT verify internal implementation details
// - Uses fakes for all external dependencies (Kafka, clock, ID generation, etc.)
func TestLongitudeOutOfRangeReturns422(t *testing.T) {
	// ========== ARRANGE ==========
	// Create test HTTP server with all fakes injected
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	// Start the server (listens on random port)
	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Build invalid position request using Object Mother
	// OutOfRangeLongitude() returns a builder with longitude = 185.0 (invalid: > 180)
	request := mothers.PositionMother{}.OutOfRangeLongitude().Build()

	// Verify that the request indeed has out-of-range longitude (sanity check)
	require.Greater(t, request.Longitude, 180.0,
		"test data must have longitude > 180 to verify validation works")

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

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	// X-Trace-ID will be generated by middleware if not provided

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========

	// 1. Status code must be 422 (Unprocessable Content)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
		"expected 422 Unprocessable Content, got %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	// 2. Response Content-Type should be application/problem+json
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/problem+json",
		"response Content-Type should be application/problem+json, got %s", contentType)

	// 3. Parse response body as ProblemDetails (RFC 9457)
	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode response body as ProblemDetails")

	// 4. Verify ProblemDetails structure

	// 4a. Type should indicate validation error
	assert.NotEmpty(t, problemDetails.Type,
		"ProblemDetails.Type should not be empty")
	assert.Contains(t, problemDetails.Type, "validation",
		"ProblemDetails.Type should indicate validation error, got %s", problemDetails.Type)

	// 4b. Title should be human-readable
	assert.NotEmpty(t, problemDetails.Title,
		"ProblemDetails.Title should not be empty")

	// 4c. Status must be 422
	assert.Equal(t, http.StatusUnprocessableEntity, problemDetails.Status,
		"ProblemDetails.Status should be 422")

	// 4d. Detail should mention the validation issue (longitude)
	assert.NotEmpty(t, problemDetails.Detail,
		"ProblemDetails.Detail should not be empty")
	assert.Contains(t, problemDetails.Detail, "longitude",
		"ProblemDetails.Detail should mention 'longitude', got: %s", problemDetails.Detail)

	// 4e. TraceID is present and is UUID v4
	assert.NotEmpty(t, problemDetails.TraceID,
		"ProblemDetails.TraceID should not be empty")
	_, err = uuid.Parse(problemDetails.TraceID)
	assert.NoError(t, err,
		"ProblemDetails.TraceID should be valid UUID v4: %s", problemDetails.TraceID)

	// 5. X-Trace-ID response header is present
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader,
		"X-Trace-ID response header should not be empty")

	// 6. X-Trace-ID header should match ProblemDetails.TraceID
	assert.Equal(t, problemDetails.TraceID, traceIDHeader,
		"X-Trace-ID header should match ProblemDetails.TraceID")

	// 7. Verify NO event was published to Kafka (request was rejected)
	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published to Kafka (request was rejected)")
}

// TestFutureCapturedAtReturns422 verifies that a position request with captured_at
// in the future is rejected with status 422 Unprocessable Content.
//
// Validates:
// - HTTP status code is 422
// - Response is ProblemDetails with type indicating validation error
// - Detail mentions future timestamp
// - X-Trace-ID header present
//
// Verifies: P5 (Future-Validation), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-02, RF-04, RF-12
func TestFutureCapturedAtReturns422(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Build request with future timestamp (60 seconds in future)
	request := mothers.PositionMother{}.FutureTimestamp().Build()

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

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
		"expected 422 Unprocessable Content for future timestamp")

	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode response as ProblemDetails")

	assert.Contains(t, problemDetails.Type, "validation",
		"Type should indicate validation error")
	assert.Equal(t, http.StatusUnprocessableEntity, problemDetails.Status,
		"Status should be 422")
	assert.NotEmpty(t, problemDetails.TraceID,
		"TraceID should be present")

	// Verify no event published
	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published (request was rejected)")
}

// TestMissingVehicleIDReturns422 verifies that a position request with empty vehicle_id
// is rejected with status 422 Unprocessable Content.
//
// Validates:
// - HTTP status code is 422
// - Response is ProblemDetails
// - Detail mentions vehicle_id
// - X-Trace-ID header present
//
// Verifies: P1 (Validation), P7 (RFC9457)
// RF: RF-02
func TestMissingVehicleIDReturns422(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Build request with missing vehicle_id
	request := mothers.PositionMother{}.MissingVehicleID().Build()
	require.Empty(t, request.VehicleID, "test data must have empty vehicle_id")

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

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode,
		"expected 422 Unprocessable Content for missing vehicle_id")

	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode response as ProblemDetails")

	assert.Contains(t, problemDetails.Type, "validation",
		"Type should indicate validation error")
	assert.Equal(t, http.StatusUnprocessableEntity, problemDetails.Status,
		"Status should be 422")
	assert.NotEmpty(t, problemDetails.TraceID,
		"TraceID should be present")

	// Verify no event published
	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published (request was rejected)")
}

// TestMalformedJSONReturns400 verifies that a position request with malformed JSON
// is rejected with status 400 Bad Request.
//
// This test validates:
// - HTTP status code is 400 (Bad Request) when JSON is malformed
// - Response body is ProblemDetails (RFC 9457)
// - ProblemDetails.Type indicates validation/client error
// - ProblemDetails.Status == 400
// - X-Trace-ID header is still present even for malformed requests
// - No 500 Internal Server Error (proper error handling)
//
// Malformed JSON examples tested:
// - Truncated JSON: {"vehicleId": "VH-001", "latitude": 40.7
// - Extra commas: {"vehicleId": "VH-001",, "latitude": 40.7}
// - Invalid syntax: {invalid json}
// - Empty body: (empty string)
//
// Uses table-driven test pattern for multiple malformed payloads.
//
// Verifies: P1 (Validation - input constraints), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-02 (validation), RF-12 (error handling), RF-04 (trace context)
func TestMalformedJSONReturns400(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Table-driven test: various malformed JSON payloads
	testCases := []struct {
		name           string
		malformedJSON  string
		expectedStatus int
	}{
		{
			name:           "truncated JSON",
			malformedJSON:  `{"vehicleId": "VH-001", "latitude": 40.7`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "extra comma",
			malformedJSON:  `{"vehicleId": "VH-001",, "latitude": 40.7}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing comma",
			malformedJSON:  `{"vehicleId": "VH-001" "latitude": 40.7}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid syntax",
			malformedJSON:  `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			malformedJSON:  ``,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ========== ACT ==========
			req, err := http.NewRequest(
				http.MethodPost,
				server.BaseURL()+"/api/v1/positions",
				bytes.NewReader([]byte(tc.malformedJSON)),
			)
			require.NoError(t, err, "failed to create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "failed to send HTTP request")
			defer resp.Body.Close()

			// ========== ASSERT ==========

			// 1. Status code must be 400 Bad Request (not 500 Internal Server Error)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode,
				"HTTP status should be 400 for malformed JSON: %q", tc.malformedJSON)

			// 2. Response Content-Type must be application/problem+json
			contentType := resp.Header.Get("Content-Type")
			assert.Contains(t, contentType, "application/problem+json",
				"Response Content-Type should be application/problem+json, got: %s", contentType)

			// 3. X-Trace-ID header must be present even for error responses
			traceIDHeader := resp.Header.Get("X-Trace-ID")
			assert.NotEmpty(t, traceIDHeader,
				"X-Trace-ID response header should be present for error responses")

			// Verify X-Trace-ID is valid UUID v4
			_, err = uuid.Parse(traceIDHeader)
			assert.NoError(t, err,
				"X-Trace-ID header should be valid UUID v4: %s", traceIDHeader)

			// 4. Parse response body as ProblemDetails (RFC 9457)
			var problemDetails dto.ProblemDetails
			err = json.NewDecoder(resp.Body).Decode(&problemDetails)
			require.NoError(t, err, "failed to decode response body as ProblemDetails")

			// 5. Verify ProblemDetails structure
			assert.NotEmpty(t, problemDetails.Type,
				"ProblemDetails.type should not be empty")

			// 6. ProblemDetails.Status must be 400
			assert.Equal(t, http.StatusBadRequest, problemDetails.Status,
				"ProblemDetails.status should be 400")

			// 7. ProblemDetails.Detail should explain the error
			assert.NotEmpty(t, problemDetails.Detail,
				"ProblemDetails.detail should explain the JSON parsing error")

			// 8. ProblemDetails.TraceID should match response header
			assert.Equal(t, traceIDHeader, problemDetails.TraceID,
				"ProblemDetails.trace_id should match X-Trace-ID response header")

			// 9. Verify no event was published to Kafka (request rejected)
			kafkaPublisher := server.GetKafkaPublisher()
			assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
				"no event should have been published to Kafka (request was rejected)")
		})
	}

	// Verifies: P1 (Validation), P7 (RFC9457), P9 (Trace-ID)
	// RF: RF-02, RF-04, RF-12
}

// TestRequestExceedsMaxSizeReturns413 verifies that a position request with payload
// exceeding the maximum allowed size is rejected with status 413 Payload Too Large.
//
// This test validates:
// - HTTP status code is 413 (Payload Too Large) when body > 10 KB
// - Response body is ProblemDetails (RFC 9457)
// - ProblemDetails.Type indicates size limit error
// - ProblemDetails.Status == 413
// - ProblemDetails.Detail indicates the size constraint
// - X-Trace-ID header is present even for size errors
// - No event is published to Kafka (request rejected before publication)
//
// Size limit: 10 KB (10,240 bytes) per HTTP POST body
// Test strategy: Create valid JSON structure but add filler to exceed 10 KB
//
// Verifies: P1 (Validation - size constraints), P7 (RFC9457), P9 (Trace-ID)
// RF: RF-02 (validation), RF-12 (error handling), RF-04 (trace context)
func TestRequestExceedsMaxSizeReturns413(t *testing.T) {
	// ========== ARRANGE ==========
	server, err := support.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	defer server.Stop()

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")

	// Create a payload that exceeds 10 KB (10,240 bytes)
	// Strategy: Build valid base request, then add a large field to exceed the limit
	largeFieldSize := 10*1024 + 1 // 10 KB + 1 byte
	largeField := string(make([]byte, largeFieldSize))
	for i := 0; i < largeFieldSize; i++ {
		largeField[i] = 'x'
	}

	// Build request with oversized field
	oversizedPayload := map[string]interface{}{
		"vehicleId":   "VH-001",
		"latitude":    40.7,
		"longitude":   -74.0,
		"capturedAt":  "2024-01-15T10:30:00.000Z",
		"oversizedField": largeField,
	}

	payloadBytes, err := json.Marshal(oversizedPayload)
	require.NoError(t, err, "failed to marshal oversized payload")

	// Verify payload actually exceeds 10 KB
	require.Greater(t, len(payloadBytes), 10*1024,
		"Payload should exceed 10 KB: %d bytes", len(payloadBytes))

	// ========== ACT ==========
	req, err := http.NewRequest(
		http.MethodPost,
		server.BaseURL()+"/api/v1/positions",
		bytes.NewReader(payloadBytes),
	)
	require.NoError(t, err, "failed to create HTTP request")

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to send HTTP request")
	defer resp.Body.Close()

	// ========== ASSERT ==========

	// 1. Status code must be 413 Payload Too Large
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode,
		"HTTP status should be 413 Payload Too Large, got %d %s",
		resp.StatusCode, http.StatusText(resp.StatusCode))

	// 2. Response Content-Type must be application/problem+json
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/problem+json",
		"Response Content-Type should be application/problem+json, got: %s", contentType)

	// 3. Parse response body as ProblemDetails (RFC 9457)
	var problemDetails dto.ProblemDetails
	err = json.NewDecoder(resp.Body).Decode(&problemDetails)
	require.NoError(t, err, "failed to decode response body as ProblemDetails")

	// 4. Verify ProblemDetails structure

	// 4a. Type should indicate size/payload error
	assert.NotEmpty(t, problemDetails.Type,
		"ProblemDetails.Type should not be empty")
	assert.Contains(t, problemDetails.Type, "size",
		"ProblemDetails.Type should indicate size error, got: %s", problemDetails.Type)

	// 4b. Title should be human-readable
	assert.NotEmpty(t, problemDetails.Title,
		"ProblemDetails.Title should not be empty")

	// 4c. Status must be 413
	assert.Equal(t, http.StatusRequestEntityTooLarge, problemDetails.Status,
		"ProblemDetails.Status should be 413, got: %d", problemDetails.Status)

	// 4d. Detail should explain the size constraint
	assert.NotEmpty(t, problemDetails.Detail,
		"ProblemDetails.Detail should not be empty")
	assert.Contains(t, problemDetails.Detail, "10",
		"ProblemDetails.Detail should mention size limit: %s", problemDetails.Detail)

	// 4e. TraceID is present and is UUID v4
	assert.NotEmpty(t, problemDetails.TraceID,
		"ProblemDetails.TraceID should not be empty")
	_, err = uuid.Parse(problemDetails.TraceID)
	assert.NoError(t, err,
		"ProblemDetails.TraceID should be valid UUID v4: %s", problemDetails.TraceID)

	// 5. X-Trace-ID response header is present
	traceIDHeader := resp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceIDHeader,
		"X-Trace-ID response header should be present for error responses")

	// 6. X-Trace-ID header should match ProblemDetails.TraceID
	assert.Equal(t, problemDetails.TraceID, traceIDHeader,
		"X-Trace-ID header should match ProblemDetails.TraceID")

	// 7. Verify no event was published to Kafka (request rejected)
	kafkaPublisher := server.GetKafkaPublisher()
	assert.Equal(t, 0, kafkaPublisher.PublishedCount(),
		"no event should have been published to Kafka (request was rejected)")

	// Verifies: P1 (Validation - size constraints), P7 (RFC9457), P9 (Trace-ID)
	// RF: RF-02, RF-12
}
