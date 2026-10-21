package mothers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// ConsumedPositionEventMother provides predefined test scenarios for ports.ConsumedPositionEvent.
// This DTO represents a position event consumed from Kafka (already parsed from JSON).
//
// Used by worker tests (parser tests, integration tests with Kafka simulation, etc.)
// to construct Kafka message payloads without manual JSON marshaling.
//
// Example usage:
//
//	// Use default valid event
//	event := ConsumedPositionEventMother.Valid()
//
//	// Override specific field
//	event := ConsumedPositionEventMother.Valid()
//	    .WithVehicleID("VH-002")
//	    .ToJSON() // Returns []byte for Kafka
//
//	// Use predefined error scenario
//	event := ConsumedPositionEventMother.MissingEventID().ToJSON()
type ConsumedPositionEventMother struct{}

// ConsumedPositionEventBuilder holds mutable state for building a ConsumedPositionEvent.
type ConsumedPositionEventBuilder struct {
	eventID        string
	vehicleID      string
	latitude       float64
	longitude      float64
	accuracy       *float64
	capturedAt     string
	receivedAt     *string
	processedAt    *string
	traceID        *string
	idempotencyKey *string
}

// Valid returns a builder with all fields set to valid values.
// Scenario: VH-001 at Manhattan, captured 10 seconds ago, valid UUIDs.
func (ConsumedPositionEventMother) Valid() *ConsumedPositionEventBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	eventID := uuid.New().String()
	traceID := uuid.New().String()
	idempotencyKey := uuid.New().String()

	return &ConsumedPositionEventBuilder{
		eventID:        eventID,
		vehicleID:      "VH-001",
		latitude:       40.7128,
		longitude:      -74.0060,
		accuracy:       nil,
		capturedAt:     tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
		receivedAt:     nil,
		processedAt:    nil,
		traceID:        &traceID,
		idempotencyKey: &idempotencyKey,
	}
}

// MissingEventID returns a builder with event_id set to empty string.
// Scenario: Missing required field (should fail parser validation).
func (ConsumedPositionEventMother) MissingEventID() *ConsumedPositionEventBuilder {
	builder := ConsumedPositionEventMother.Valid()
	builder.eventID = ""
	return builder
}

// MissingVehicleID returns a builder with vehicle_id set to empty string.
// Scenario: Missing required field (should fail parser validation).
func (ConsumedPositionEventMother) MissingVehicleID() *ConsumedPositionEventBuilder {
	builder := ConsumedPositionEventMother.Valid()
	builder.vehicleID = ""
	return builder
}

// OutOfRangeLatitude returns a builder with latitude > 90.
// Scenario: Invalid coordinate (should fail domain validation in NewGeoPosition).
func (ConsumedPositionEventMother) OutOfRangeLatitude() *ConsumedPositionEventBuilder {
	builder := ConsumedPositionEventMother.Valid()
	builder.latitude = 95.0
	return builder
}

// WithEventID sets the event_id and returns the builder for chaining.
func (b *ConsumedPositionEventBuilder) WithEventID(eventID string) *ConsumedPositionEventBuilder {
	b.eventID = eventID
	return b
}

// WithVehicleID sets the vehicle_id and returns the builder for chaining.
func (b *ConsumedPositionEventBuilder) WithVehicleID(vehicleID string) *ConsumedPositionEventBuilder {
	b.vehicleID = vehicleID
	return b
}

// WithLatitude sets the latitude and returns the builder for chaining.
func (b *ConsumedPositionEventBuilder) WithLatitude(latitude float64) *ConsumedPositionEventBuilder {
	b.latitude = latitude
	return b
}

// WithLongitude sets the longitude and returns the builder for chaining.
func (b *ConsumedPositionEventBuilder) WithLongitude(longitude float64) *ConsumedPositionEventBuilder {
	b.longitude = longitude
	return b
}

// ToEvent converts the builder to a ConsumedPositionEvent struct.
func (b *ConsumedPositionEventBuilder) ToEvent() ports.ConsumedPositionEvent {
	return ports.ConsumedPositionEvent{
		EventID:        b.eventID,
		VehicleID:      b.vehicleID,
		Latitude:       b.latitude,
		Longitude:      b.longitude,
		Accuracy:       b.accuracy,
		CapturedAt:     b.capturedAt,
		ReceivedAt:     b.receivedAt,
		ProcessedAt:    b.processedAt,
		TraceID:        b.traceID,
		IdempotencyKey: b.idempotencyKey,
	}
}

// ToJSON converts the builder to JSON bytes suitable for Kafka message payload.
// Used to simulate raw Kafka message consumption in parser tests.
func (b *ConsumedPositionEventBuilder) ToJSON() []byte {
	event := b.ToEvent()
	data, _ := json.Marshal(event)
	return data
}
