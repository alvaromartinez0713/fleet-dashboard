package config

import (
	kafka "github.com/segmentio/kafka-go"

	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/messaging"
)

// NewKafkaMessageReader creates a Kafka reader and wraps it in the ACL.
// Returns: messaging.MessageQueue (private ACL)
func NewKafkaMessageReader(brokers []string, topic, group string, batchSize int) (messaging.MessageQueue, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        group,
		CommitInterval: 0,
		MinBytes:       1,
		MaxBytes:       10e6,
		QueueCapacity:  batchSize,
	})
	return NewKafkaClient(reader)
}
