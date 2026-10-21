package messaging

import (
	"context"
)

// Message represents a message consumed from the message queue.
type Message struct {
	Key       []byte
	Value     []byte
	Topic     string
	Partition int
	Offset    int64
}

// MessageQueue is a private anti-corruption layer interface.
// It abstracts the underlying message queue client (segmentio/kafka-go, confluent-kafka, etc.)
// and is only used internally by this adapter package.
//
// This is NOT a public port - it's an implementation detail.
type MessageQueue interface {
	// FetchMessage reads the next message from the queue.
	FetchMessage(ctx context.Context) (Message, error)

	// CommitOffset marks the offset as committed.
	CommitOffset(ctx context.Context, msg Message) error

	// Close releases queue resources.
	Close() error
}
