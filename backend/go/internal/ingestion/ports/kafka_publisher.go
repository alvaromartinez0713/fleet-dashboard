package ports

import "context"

// KafkaPublisher defines the contract for publishing events to Kafka.
// Implementations are responsible for:
// - Publishing event data asynchronously to a Kafka topic
// - Retrying on transient failures (timeout, network errors)
// - Handling permanent failures gracefully (logging, dead-letter queue if applicable)
// - Supporting idempotent publishing when possible
//
// The publisher must not block the calling function waiting for Kafka acknowledgment.
// It may use background goroutines or buffered channels for async operations.
//
// Published messages must include:
// - topic: identifies which Kafka topic receives the message
// - key: partition key (e.g., vehicle_id for ordering guarantees)
// - value: serialized event data (typically JSON)
//
// Implementations are injectable to enable testing with FakeKafkaPublisher
// that records published events without connecting to a real broker.
type KafkaPublisher interface {
	// Publish sends an event to a Kafka topic asynchronously.
	// If the context is cancelled before publishing completes,
	// the context cancellation takes precedence over the publish operation.
	//
	// Parameters:
	//   ctx    - context for cancellation and timeout (inherited from HTTP request)
	//   topic  - Kafka topic name (e.g., "telemetry-events")
	//   key    - partition key for message routing (e.g., vehicle_id)
	//   value  - event data to publish (typically a struct that will be JSON-encoded)
	//
	// Returns error if:
	//   - ctx is cancelled or times out before publishing starts
	//   - message validation fails
	//   - all retries exhausted (permanent failure)
	//
	// Does NOT return error if:
	//   - message is queued locally but Kafka acknowledgment not yet received
	//     (acknowledgment may arrive asynchronously)
	//   - broker ack is pending (client considers this success)
	//
	// Example:
	//   err := pub.Publish(ctx, "positions", "VH-001", positionEvent)
	//   if err != nil {
	//     return CircuitBreakerOpenError or PublishError
	//   }
	//   // Message queued, HTTP response sent immediately
	Publish(ctx context.Context, topic string, key string, value interface{}) error
}
