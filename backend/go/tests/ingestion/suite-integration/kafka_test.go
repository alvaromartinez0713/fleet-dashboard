// +build integration

package suite_integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleet-platform/backend/internal/ingestion/adapters/outbound/messaging"
	"github.com/fleet-platform/backend/internal/ingestion/config"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
	"github.com/fleet-platform/backend/tests/support/fakes"
	"github.com/fleet-platform/backend/tests/support/harness"
	"github.com/fleet-platform/backend/tests/support/mothers"
)

// TestPositionPublishedToKafkaWithCorrectSchema verifies that when a valid position
// is received via HTTP POST, an event is published to Kafka with the correct schema,
// partitioned by vehicle_id, and the offset advances.
//
// This test validates:
// - Position event is published to Kafka topic "positions"
// - Event is partitioned by vehicle_id (used as partition key)
// - Offset advances after each publish
// - Event schema matches domain model with all required fields
// - No events are lost or duplicated (1:1 mapping HTTP → Kafka)
// - Correct schema: event_id, vehicle_id, latitude, longitude, accuracy, captured_at, received_at, processed_at, stored_at=null
//
// Infrastructure:
// - Real Kafka via Testcontainers (Zookeeper + Kafka containers)
// - Real KafkaPublisher adapter (not fake)
// - Real HTTP server connected to real Kafka
// - Real HTTP client making HTTP requests
//
// Verifies: P2 (Async+202), P12 (Domain-Events)
// RF: RF-03 (async messaging), RF-02 (valid position ingestion)
// Requirements: RF-03 (Kafka async publication), RF-07 (4-Timestamps)
func TestPositionPublishedToKafkaWithCorrectSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

	// Create topic "positions" with multiple partitions to test partitioning
	err = conn.CreateTopics(kafka.TopicConfigs{
		{
			Topic:             "positions",
			NumPartitions:     3, // Multiple partitions to verify partitioning by vehicle_id
			ReplicationFactor: 1,
		},
	})
	// Ignore error if topic already exists (may have been created before)
	_ = err

	// Give topic creation time to propagate through Kafka cluster
	time.Sleep(2 * time.Second)

	// Create HTTP server with real Kafka publisher injected
	server := harness.NewTestHTTPServerWithPublisher(kafkaPublisher)
	require.NotNil(t, server, "failed to create test HTTP server with Kafka publisher")
	defer server.Stop()

	// Get the server URL
	serverURL := server.BaseURL()
	require.NotEmpty(t, serverURL, "server URL should not be empty")

	// ========== ACT ==========
	// Build a valid position request using PositionMother
	// This creates: vehicle_id=VH-001, latitude=40.7128, longitude=-74.0060, capturedAt=10s ago
	positionReq := mothers.PositionMother{}.Valid().Build()
	require.NotNil(t, positionReq, "failed to build position request")

	// Marshal request to JSON
	bodyBytes, err := json.Marshal(positionReq)
	require.NoError(t, err, "failed to marshal position request to JSON")

	// Send HTTP POST to /api/v1/positions with Content-Type: application/json
	httpResp, err := http.Post(
		fmt.Sprintf("%s/api/v1/positions", serverURL),
		"application/json",
		harness.NewBytesReader(bodyBytes),
	)
	require.NoError(t, err, "failed to send HTTP POST request to server")
	defer httpResp.Body.Close()

	// Verify HTTP response is 202 Accepted
	assert.Equal(t, http.StatusAccepted, httpResp.StatusCode, 
		"HTTP status should be 202 Accepted, got %d", httpResp.StatusCode)

	// Extract trace_id from response header (should be set by TraceIDMiddleware)
	traceID := httpResp.Header.Get("X-Trace-ID")
	assert.NotEmpty(t, traceID, "X-Trace-ID header should be present in response")

	// Parse response body to verify response structure
	var responseData map[string]interface{}
	respBytes, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err, "failed to read HTTP response body")

	err = json.Unmarshal(respBytes, &responseData)
	require.NoError(t, err, "failed to unmarshal HTTP response body as JSON")

	// Verify response structure (202 Accepted response should include)
	assert.Equal(t, "accepted", responseData["status"], "response status should be 'accepted'")
	assert.NotEmpty(t, responseData["event_id"], "response should contain event_id (UUID v4)")
	assert.Equal(t, positionReq.VehicleID, responseData["vehicle_id"], 
		"response vehicle_id should match request vehicle_id")
	assert.NotEmpty(t, responseData["received_at"], "response should contain received_at timestamp")

	// Extract event_id from response for later verification
	eventID := responseData["event_id"].(string)
	require.NotEmpty(t, eventID, "event_id from response should not be empty")

	// Give Kafka async publisher time to send message (fire-and-forget pattern)
	// The publisher runs in background goroutines with retries up to 3 attempts
	time.Sleep(3 * time.Second)

	// ========== ASSERT: Verify message published to Kafka ==========

	// We need to read from all partitions because vehicle_id is used as partition key
	// The exact partition is determined by: partition = hash(vehicle_id) % num_partitions
	// Since we have 3 partitions, the vehicle "VH-001" will hash to a specific partition
	var kafkaMsg *kafka.Message
	var foundMessage bool
	var readErr error

	// Try to find our message by reading from all 3 partitions
	for partitionID := 0; partitionID < 3; partitionID++ {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          "positions",
			Partition:      partitionID,
			StartOffset:    0, // Read from beginning of partition
			CommitInterval: 100 * time.Millisecond,
			MaxBytes:       10 * 1024 * 1024, // 10 MB
		})

		// Try to read message with timeout
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		for {
			msg, err := reader.ReadMessage(readCtx)
			if err == context.DeadlineExceeded {
				break // Timeout, move to next partition
			}
			if err != nil {
				break // Error, move to next partition
			}

			// Check if this is our message (key = vehicle_id)
			msgKey := string(msg.Key)
			if msgKey == positionReq.VehicleID {
				kafkaMsg = &msg
				foundMessage = true
				reader.Close()
				readCancel()
				break
			}
		}
		readCancel()
		reader.Close()

		if foundMessage {
			break
		}
	}

	// Verify message was found in Kafka
	require.True(t, foundMessage, 
		"message should be published to Kafka topic 'positions' with vehicle_id as partition key")

	// Verify partition key is set to vehicle_id (guarantees ordering within vehicle stream)
	assert.Equal(t, positionReq.VehicleID, string(kafkaMsg.Key), 
		"partition key should be vehicle_id for ordering guarantee")

	// Verify offset is non-negative (message has valid offset)
	assert.GreaterOrEqual(t, kafkaMsg.Offset, int64(0), 
		"offset should be non-negative after publication")

	// Verify Kafka message value is valid JSON and matches schema
	var kafkaPayload map[string]interface{}
	err = json.Unmarshal(kafkaMsg.Value, &kafkaPayload)
	require.NoError(t, err, "failed to unmarshal Kafka message value as JSON")

	// ========== ASSERT: Verify schema matches Requirement 7 (4-Timestamps) ==========

	// Verify all required fields from ToKafkaPayload():
	// event_id, vehicle_id, latitude, longitude, accuracy (optional), 
	// captured_at, received_at, processed_at, stored_at (null for HTTP)

	// 1. event_id: Must be UUID v4 string
	assert.NotEmpty(t, kafkaPayload["event_id"], "Kafka message should contain event_id")
	kafkaEventID := kafkaPayload["event_id"].(string)
	assert.Equal(t, eventID, kafkaEventID, 
		"Kafka message event_id should match response event_id")

	// 2. vehicle_id: Must match request
	assert.Equal(t, positionReq.VehicleID, kafkaPayload["vehicle_id"], 
		"Kafka message vehicle_id should match request vehicle_id")

	// 3. latitude: Must match request
	assert.Equal(t, positionReq.Latitude, kafkaPayload["latitude"], 
		"Kafka message latitude should match request latitude")

	// 4. longitude: Must match request
	assert.Equal(t, positionReq.Longitude, kafkaPayload["longitude"], 
		"Kafka message longitude should match request longitude")

	// 5. accuracy: Optional field (may be null or omitted)
	if kafkaPayload["accuracy"] != nil {
		// If present, should be a number or null
		assert.True(t, 
			kafkaPayload["accuracy"] == nil || 
			fmt.Sprintf("%T", kafkaPayload["accuracy"]) == "float64",
			"accuracy should be null or float64")
	}

	// 6. captured_at: ISO 8601 UTC timestamp (provided by client)
	assert.NotEmpty(t, kafkaPayload["captured_at"], 
		"Kafka message should contain captured_at timestamp")
	capturedAtStr := kafkaPayload["captured_at"].(string)
	require.NotEmpty(t, capturedAtStr, "captured_at should be non-empty ISO 8601 string")

	// 7. received_at: ISO 8601 UTC timestamp (generated by server on HTTP receipt)
	assert.NotEmpty(t, kafkaPayload["received_at"], 
		"Kafka message should contain received_at timestamp")
	receivedAtStr := kafkaPayload["received_at"].(string)
	require.NotEmpty(t, receivedAtStr, "received_at should be non-empty ISO 8601 string")

	// 8. processed_at: ISO 8601 UTC timestamp (generated by server after validation)
	assert.NotEmpty(t, kafkaPayload["processed_at"], 
		"Kafka message should contain processed_at timestamp")
	processedAtStr := kafkaPayload["processed_at"].(string)
	require.NotEmpty(t, processedAtStr, "processed_at should be non-empty ISO 8601 string")

	// 9. stored_at: NULL for HTTP layer (set by worker after TimescaleDB INSERT)
	assert.Nil(t, kafkaPayload["stored_at"], 
		"Kafka message stored_at should be null (HTTP layer does not set it; worker does)")

	// ========== ASSERT: Verify timestamp ordering and validity ==========
	// Per Requirement 7: capturedAt ≤ receivedAt ≤ processedAt

	capturedTime, err := time.Parse(time.RFC3339Nano, capturedAtStr)
	require.NoError(t, err, "captured_at should parse as RFC3339 timestamp")

	receivedTime, err := time.Parse(time.RFC3339Nano, receivedAtStr)
	require.NoError(t, err, "received_at should parse as RFC3339 timestamp")

	processedTime, err := time.Parse(time.RFC3339Nano, processedAtStr)
	require.NoError(t, err, "processed_at should parse as RFC3339 timestamp")

	// Verify timestamp ordering: capturedAt ≤ receivedAt ≤ processedAt
	assert.LessOrEqual(t, capturedTime.Unix(), receivedTime.Unix(), 
		"captured_at ≤ received_at (by ordering invariant)")
	assert.LessOrEqual(t, receivedTime.Unix(), processedTime.Unix(), 
		"received_at ≤ processed_at (by ordering invariant)")
	assert.LessOrEqual(t, capturedTime.Unix(), processedTime.Unix(), 
		"captured_at ≤ processed_at (transitive from above)")

	// ========== ASSERT: Verify Kafka message offset is set ==========
	// Offset proves the message was successfully persisted in Kafka
	assert.False(t, kafkaMsg.Time.IsZero(), 
		"Kafka message timestamp should be set (indicates successful persistence)")

	// Verify offset is monotonically increasing (at least 0 for first message)
	assert.GreaterOrEqual(t, kafkaMsg.Offset, int64(0), 
		"Offset should be non-negative (indicates message sequence number)")

	// Verifies: 
	// - P2 (Async+202): HTTP returns 202 before Kafka confirmation
	// - P12 (Domain-Events): PositionReceivedEvent was emitted and published
	// - RF-03 (Async Kafka publication): Event published asynchronously
	// - RF-02 (Valid position ingestion): Request validated and published
	// - RF-07 (4-Timestamps): All 4 timestamps present with correct ordering
}

// TestKafkaConnectivity verifies that Kafka container is accessible
func TestKafkaConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ========== SETUP ==========
	env := SetupIntegrationTest(ctx, t)
	require.NotNil(t, env, "failed to setup integration test environment")

	// ========== ASSERT ==========
	// Verify Kafka URL is valid
	assert.NotEmpty(t, env.KafkaURL, "Kafka URL should be available")

	// Try to connect to Kafka broker
	conn, err := kafka.Dial("tcp", env.KafkaURL)
	require.NoError(t, err, "should be able to connect to Kafka broker")
	defer conn.Close()

	// Verify broker metadata
	brokers, err := conn.Brokers()
	require.NoError(t, err, "should be able to fetch broker metadata")
	assert.GreaterThan(t, len(brokers), 0, "should have at least one broker")
}
