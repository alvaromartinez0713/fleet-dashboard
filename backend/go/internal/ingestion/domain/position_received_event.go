package domain

import (
	"encoding/json"
	"time"
)

// PositionReceivedEvent is a Domain Event emitted when a VehiclePosition Entity
// is successfully created.
//
// It is a Value Object: immutable after creation, contains no references
// to HTTP, Kafka, or other infrastructure.
//
// The event is consumed by the Application layer to make decisions
// (e.g., publish to Kafka). The Domain does NOT publish events itself.
//
// **7 Fields:**
// 1. eventID (EventID) - unique event identifier
// 2. vehicleID (VehicleID) - vehicle identifier
// 3. geoPosition (GeoPosition) - lat/long
// 4. capturedAt (CapturedTimestamp) - when client captured
// 5. receivedAt (ReceivedTimestamp) - when server received
// 6. traceID (TraceID) - for traceability
// 7. idempotencyKey (IdempotencyKey) - optional, for deduplication
//
// Verifies: Requirement 11 (Domain Events), P12 (Domain-Event-Decoupling)
type PositionReceivedEvent struct {
	eventID        EventID
	vehicleID      *VehicleID
	geoPosition    *GeoPosition
	capturedAt     CapturedTimestamp
	receivedAt     ReceivedTimestamp
	traceID        TraceID
	idempotencyKey *IdempotencyKey
	occurredAt     time.Time
}

// NewPositionReceivedEvent creates a new PositionReceivedEvent with the provided fields.
// No validation is performed (all inputs are already-validated Value Objects).
// All parameters are required (geoPosition and idempotencyKey may be nil but are included).
//
// Args:
//   eventID: EventID - unique event identifier (UUID v4)
//   vehicleID: *VehicleID - vehicle identifier
//   geoPosition: *GeoPosition - lat/long coordinates
//   capturedAt: CapturedTimestamp - when client captured
//   receivedAt: ReceivedTimestamp - when server received
//   traceID: TraceID - for traceability
//   idempotencyKey: *IdempotencyKey - optional, for deduplication (may be nil)
//   occurredAt: time.Time - when this event was emitted
//
// Returns:
//   *PositionReceivedEvent: Immutable domain event
//
// Verifies: Requirement 11 (Domain Events)
func NewPositionReceivedEvent(
	eventID EventID,
	vehicleID *VehicleID,
	geoPosition *GeoPosition,
	capturedAt CapturedTimestamp,
	receivedAt ReceivedTimestamp,
	traceID TraceID,
	idempotencyKey *IdempotencyKey,
	occurredAt time.Time,
) *PositionReceivedEvent {
	return &PositionReceivedEvent{
		eventID:        eventID,
		vehicleID:      vehicleID,
		geoPosition:    geoPosition,
		capturedAt:     capturedAt,
		receivedAt:     receivedAt,
		traceID:        traceID,
		idempotencyKey: idempotencyKey,
		occurredAt:     occurredAt,
	}
}

// --- Domain Event Behavior ---

// AggregateID returns the aggregate ID (vehicleID) for tracking purposes.
// This identifies the root aggregate that emitted the event.
//
// Verifies: Requirement 11 (Domain Events)
func (e *PositionReceivedEvent) AggregateID() string {
	if e.vehicleID == nil {
		return ""
	}
	return e.vehicleID.String()
}

// EventType returns the domain event type identifier.
// Used for event serialization and routing.
//
// Verifies: Requirement 11 (Domain Events)
func (e *PositionReceivedEvent) EventType() string {
	return "PositionReceived"
}

// OccurredAt returns the time when the event was emitted.
//
// Verifies: Requirement 11 (Domain Events)
func (e *PositionReceivedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// --- Event Data Accessors ---

// EventID returns the unique event identifier.
func (e *PositionReceivedEvent) EventID() EventID {
	return e.eventID
}

// VehicleID returns the vehicle identifier from the event.
func (e *PositionReceivedEvent) VehicleID() *VehicleID {
	return e.vehicleID
}

// GeoPosition returns the geographic position (coordinates).
func (e *PositionReceivedEvent) GeoPosition() *GeoPosition {
	return e.geoPosition
}

// CapturedAt returns the client-provided capture timestamp.
func (e *PositionReceivedEvent) CapturedAt() CapturedTimestamp {
	return e.capturedAt
}

// ReceivedAt returns the server reception timestamp.
func (e *PositionReceivedEvent) ReceivedAt() ReceivedTimestamp {
	return e.receivedAt
}

// TraceID returns the trace identifier for request tracking.
func (e *PositionReceivedEvent) TraceID() TraceID {
	return e.traceID
}

// IdempotencyKey returns the idempotency key if present, nil otherwise.
func (e *PositionReceivedEvent) IdempotencyKey() *IdempotencyKey {
	return e.idempotencyKey
}

// --- Serialization ---

// MarshalJSON implements json.Marshaler for serialization.
// Includes all event data for logging and auditing.
func (e *PositionReceivedEvent) MarshalJSON() ([]byte, error) {
	type Alias struct {
		EventType      string  `json:"event_type"`
		EventID        string  `json:"event_id"`
		VehicleID      string  `json:"vehicle_id"`
		Latitude       float64 `json:"latitude"`
		Longitude      float64 `json:"longitude"`
		CapturedAt     string  `json:"captured_at"`
		ReceivedAt     string  `json:"received_at"`
		TraceID        string  `json:"trace_id"`
		IdempotencyKey *string `json:"idempotency_key,omitempty"`
		OccurredAt     string  `json:"occurred_at"`
	}

	var idempotencyKeyStr *string
	if e.idempotencyKey != nil && e.idempotencyKey.IsPresent() {
		keyVal := e.idempotencyKey.String()
		idempotencyKeyStr = &keyVal
	}

	var latitude, longitude float64
	if e.geoPosition != nil {
		latitude = e.geoPosition.Latitude()
		longitude = e.geoPosition.Longitude()
	}

	return json.Marshal(&Alias{
		EventType:      e.EventType(),
		EventID:        e.eventID.String(),
		VehicleID:      e.AggregateID(),
		Latitude:       latitude,
		Longitude:      longitude,
		CapturedAt:     e.capturedAt.String(),
		ReceivedAt:     e.receivedAt.String(),
		TraceID:        e.traceID.String(),
		IdempotencyKey: idempotencyKeyStr,
		OccurredAt:     e.occurredAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	})
}
