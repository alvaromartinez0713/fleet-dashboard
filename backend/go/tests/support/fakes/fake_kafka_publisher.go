package fakes

import (
	"context"
	"sync"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// FakeKafkaPublisher is a test implementation of ports.KafkaPublisher.
// It records all published events in memory without connecting to a real Kafka broker.
//
// Thread-safe: all operations are protected by sync.Mutex.
type FakeKafkaPublisher struct {
	publishedEvents []*domain.PositionReceivedEvent
	mutex           sync.Mutex
}

// Verify that FakeKafkaPublisher implements ports.KafkaPublisher.
var _ ports.KafkaPublisher = (*FakeKafkaPublisher)(nil)

// Publish records the event in the publishedEvents slice.
// Always returns nil (no errors simulated).
func (f *FakeKafkaPublisher) Publish(ctx context.Context, topic string, key string, value interface{}) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	event, ok := value.(*domain.PositionReceivedEvent)
	if ok {
		f.publishedEvents = append(f.publishedEvents, event)
	}

	return nil
}

// PublishedEvents returns a copy of all published events.
func (f *FakeKafkaPublisher) PublishedEvents() []*domain.PositionReceivedEvent {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	result := make([]*domain.PositionReceivedEvent, len(f.publishedEvents))
	copy(result, f.publishedEvents)
	return result
}

// LastPublishedEvent returns the most recently published event, or nil if no events.
func (f *FakeKafkaPublisher) LastPublishedEvent() *domain.PositionReceivedEvent {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.publishedEvents) == 0 {
		return nil
	}
	return f.publishedEvents[len(f.publishedEvents)-1]
}

// PublishedCount returns the total number of published events.
func (f *FakeKafkaPublisher) PublishedCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return len(f.publishedEvents)
}

// Reset clears all recorded events.
func (f *FakeKafkaPublisher) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.publishedEvents = []*domain.PositionReceivedEvent{}
}
