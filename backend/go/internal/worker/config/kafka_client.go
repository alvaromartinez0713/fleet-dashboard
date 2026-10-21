package config

import (
	"context"
	"errors"

	kafka "github.com/segmentio/kafka-go"

	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/messaging"
)

// KafkaClient implements messaging.MessageQueue ACL using segmentio/kafka-go.
type KafkaClient struct {
	reader *kafka.Reader
}

// NewKafkaClient wraps a segmentio kafka.Reader with the ACL interface.
func NewKafkaClient(reader *kafka.Reader) (messaging.MessageQueue, error) {
	if reader == nil {
		return nil, errors.New("reader is nil")
	}
	return &KafkaClient{
		reader: reader,
	}, nil
}

// FetchMessage implements messaging.MessageQueue.FetchMessage.
func (k *KafkaClient) FetchMessage(ctx context.Context) (messaging.Message, error) {
	msg, err := k.reader.FetchMessage(ctx)
	if err != nil {
		return messaging.Message{}, err
	}

	return messaging.Message{
		Key:       msg.Key,
		Value:     msg.Value,
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}, nil
}

// CommitOffset implements messaging.MessageQueue.CommitOffset.
func (k *KafkaClient) CommitOffset(ctx context.Context, msg messaging.Message) error {
	kafkaMsg := kafka.Message{
		Key:       msg.Key,
		Value:     msg.Value,
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}
	return k.reader.CommitMessages(ctx, kafkaMsg)
}

// Close closes the underlying Kafka reader.
func (k *KafkaClient) Close() error {
	return k.reader.Close()
}

var _ messaging.MessageQueue = (*KafkaClient)(nil)
