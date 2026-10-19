//go:build integration
// +build integration

package suite_integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/tests/support/harness"
	"github.com/fleet-platform/backend/tests/support/fakes"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestIdempotencyWindow1hRetention verifies that X-Idempotency-Key deduplication
// works within 1 hour window. Same key sent twice → first succeeds, second rejected or cached.
// After 1 hour window expires, the same key is treated as new.
//
// This test validates:
// - X-Idempotency-Key is stored with 1h TTL (using FakeClock for time control)
// - First request with key K1 → published to Kafka
// - Second request with key K1 within window → NOT published again (deduplication)
// - Third request with key K1 after window expires → published as new event
// - Window expiration is controlled via FakeClock.Advance()
//
// Infrastructure:
// - TestHTTPServer with FakeClock (time is controllable)
// - FakeKafkaPublisher (tracks events)
// - In-memory idempotency store with TTL expiration
//
// Verifies: RNF-04 (Idempotency), RF-07 (Idempotency-Key in headers)
// RF: RF-03 (async messaging), RF-02 (position ingestion)
func TestIdempotencyWindow1hRetention(t *testing.T) {
	// Integration test - requires Docker and Testcontainers
	// Run with: go test -tags=integration ./tests/ingestion/suite-integration

	// ========== SETUP ==========
	// Create test HTTP server with all fakes injected
	server, err := harness.NewTestHTTPServer()
	require.NoError(t, err, "failed to create test HTTP server")
	require.NotNil(t, server, "test server should not be nil")

	err = server.Start()
	require.NoError(t, err, "failed to start test HTTP server")
	defer server.Stop()

	// Create in-memory idempotency store
	idempotencyStore := NewInMemoryIdempotencyStore(1 * time.Hour)

	// Get fakes for verification
	kafkaPub := server.GetKafkaPublisher()
	clock := server.GetClock()

	// ========== PHASE 1: Send first request with X-Idempotency-Key: K1 ==========
	idempotencyKeyK1 := uuid.New().String()
	t.Logf("Phase 1: First request with idempotency key K1 = %s", idempotencyKeyK1)

	// Build first request
	firstRequest := mothers.PositionMother{}.Valid().Build()

	// Send first request
	firstResp, firstEventID := sendIdempotencyPositionRequest(t, server, firstRequest, idempotencyKeyK1)
	defer firstResp.Body.Close()

	// Verify: 202 Accepted
	assert.Equal(t, http.StatusAccepted, firstResp.StatusCode, "first request should return 202 Accepted")

	// Verify: Event published to Kafka
	firstKafkaCount := kafkaPub.PublishedCount()
	assert.Equal(t, 1, firstKafkaCount, "first request should publish 1 event to Kafka")

	// Store in idempotency store (simulating server behavior)
	now := clock.Now()
	idempotencyStore.Store(idempotencyKeyK1, &IdempotencyEntry{
		EventID:   firstEventID,
		Timestamp: now,
	})

	t.Logf("Phase 1 complete: Event published (ID=%s), stored in idempotency store", firstEventID)

	// ========== PHASE 2: Send duplicate request with same X-Idempotency-Key: K1 within window ==========
	t.Logf("Phase 2: Duplicate request with idempotency key K1 (within 1h window)")

	// Send second request (identical)
	secondResp, secondEventID := sendIdempotencyPositionRequest(t, server, firstRequest, idempotencyKeyK1)
	defer secondResp.Body.Close()

	// Verify: 202 Accepted (or 200 if response caching is implemented)
	assert.Equal(t, http.StatusAccepted, secondResp.StatusCode, "duplicate request should return 202 Accepted")

	// Verify: Same event returned (if caching) OR deduplication occurred
	// Key assertion: Kafka should have exactly 1 event (no duplicate)
	secondKafkaCount := kafkaPub.PublishedCount()
	assert.Equal(t, 1, secondKafkaCount, "duplicate request should NOT publish a second event to Kafka (deduplication)")

	// Event ID should match (if server caches response) or differ (if server generates new ID but doesn't publish)
	// For idempotency-first design: EventID should be same (cached response)
	assert.Equal(t, firstEventID, secondEventID, "duplicate request should return same event ID (cached)")

	t.Logf("Phase 2 complete: Duplicate rejected (no new event), same event returned")

	// ========== PHASE 3: Advance time > 1 hour (use FakeClock) ==========
	t.Logf("Phase 3: Advancing FakeClock by 61 minutes (window expires)")

	// Advance clock by 61 minutes
	clock.Advance(61 * time.Minute)

	// Verify in idempotency store that entry is expired
	entry := idempotencyStore.Lookup(idempotencyKeyK1)
	assert.Nil(t, entry, "idempotency entry should be expired after 1h window")

	// ========== PHASE 4: Send request again with same K1 after window expired ==========
	t.Logf("Phase 4: Request with idempotency key K1 after window expiration")

	// Send third request (same key, but window has expired)
	thirdResp, thirdEventID := sendIdempotencyPositionRequest(t, server, firstRequest, idempotencyKeyK1)
	defer thirdResp.Body.Close()

	// Verify: 202 Accepted
	assert.Equal(t, http.StatusAccepted, thirdResp.StatusCode, "request after window expiration should return 202 Accepted")

	// Verify: NEW event generated and published
	thirdKafkaCount := kafkaPub.PublishedCount()
	assert.Equal(t, 2, thirdKafkaCount, "request after window expiration should publish a NEW event (total 2)")

	// Verify: Different event ID (new event)
	assert.NotEqual(t, firstEventID, thirdEventID, "new request after expiration should have a different event ID")

	t.Logf("Phase 4 complete: New event published (ID=%s), total events in Kafka: %d", thirdEventID, thirdKafkaCount)

	// ========== FINAL ASSERTIONS ==========
	t.Logf("Test complete: Idempotency window 1h retention verified")
	assert.Equal(t, 2, kafkaPub.PublishedCount(), "exactly 2 events should be published (1 original + 1 after expiration)")
}

// ============================================================================
// Helper: sendIdempotencyPositionRequest sends an HTTP POST to /api/v1/positions
// ============================================================================
func sendIdempotencyPositionRequest(t *testing.T, server *harness.TestHTTPServer, req *dto.PositionRequest, idempotencyKey string) (*http.Response, string) {
	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	require.NoError(t, err, "failed to marshal position request")

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", server.BaseURL()+"/api/v1/positions", bytes.NewReader(reqBody))
	require.NoError(t, err, "failed to create HTTP request")

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		httpReq.Header.Set("X-Idempotency-Key", idempotencyKey)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err, "failed to send HTTP request")

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	// Expect 202 response with PositionResponse DTO
	if resp.StatusCode != http.StatusAccepted {
		t.Logf("Response status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var respDto dto.PositionResponse
	err = json.Unmarshal(respBody, &respDto)
	require.NoError(t, err, "failed to parse response DTO: %s", string(respBody))

	return resp, respDto.EventID
}

// ============================================================================
// InMemoryIdempotencyStore: Simple in-memory store with TTL expiration
// ============================================================================

// IdempotencyEntry holds cached response data
type IdempotencyEntry struct {
	EventID   string
	Timestamp time.Time
}

// InMemoryIdempotencyStore stores idempotency keys with TTL
type InMemoryIdempotencyStore struct {
	entries map[string]*IdempotencyEntry
	ttl     time.Duration
	clock   *fakes.FakeClock
}

// NewInMemoryIdempotencyStore creates a new in-memory idempotency store
func NewInMemoryIdempotencyStore(ttl time.Duration) *InMemoryIdempotencyStore {
	return &InMemoryIdempotencyStore{
		entries: make(map[string]*IdempotencyEntry),
		ttl:     ttl,
		clock:   fakes.NewFakeClock(),
	}
}

// Store saves an entry in the idempotency store
func (s *InMemoryIdempotencyStore) Store(key string, entry *IdempotencyEntry) {
	s.entries[key] = entry
}

// Lookup retrieves an entry if it exists and hasn't expired
func (s *InMemoryIdempotencyStore) Lookup(key string) *IdempotencyEntry {
	entry, exists := s.entries[key]
	if !exists {
		return nil
	}

	// Check expiration using stored timestamp
	now := s.clock.Now()
	if now.Sub(entry.Timestamp) > s.ttl {
		// Entry has expired
		delete(s.entries, key)
		return nil
	}

	return entry
}

// SetClock sets the clock for this store (for testing with FakeClock)
func (s *InMemoryIdempotencyStore) SetClock(clock *fakes.FakeClock) {
	s.clock = clock
}

// Clear removes all entries
func (s *InMemoryIdempotencyStore) Clear() {
	s.entries = make(map[string]*IdempotencyEntry)
}
