package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// VehiclePosition is an Entity representing a received vehicle position.
// It is NOT anemic: it encapsulates behavior (latency calculations, payload generation)
// and enforces domain invariants (timestamp ordering).
//
// It encapsulates exactly 5 timestamps:
// 1. capturedAt (CapturedTimestamp) - client-provided, when location was captured
// 2. receivedAt (ReceivedTimestamp) - server received the HTTP request
// 3. processedAt (ProcessedTimestamp) - after validation, before Kafka publication
// 4. storedAt (optional, nil for HTTP layer) - when persisted in TimescaleDB (worker adds this)
// 5. processedAtHTTP (separate from storedAt) - explicitly for HTTP layer tracking
//
// The entity owns its lifecycle: no setters, immutable after creation.
// Methods express domain behavior, not just getters.
//
// Verifies: Requirement 7 (4-Timestamps), RF-07, RNF-04
type VehiclePosition struct {
	vehicleID      *VehicleID
	eventID        EventID
	geoPosition    *GeoPosition
	capturedAt     CapturedTimestamp
	receivedAt     ReceivedTimestamp
	processedAt    ProcessedTimestamp
	storedAt       *time.Time // optional, added by worker after TimescaleDB INSERT
	traceID        TraceID
	idempotencyKey *IdempotencyKey // optional
	accuracy       *float64        // optional accuracy in meters

	// Domain event emitted when entity is created
	positionReceivedEvent *PositionReceivedEvent
}

// NewVehiclePosition creates a new VehiclePosition Entity with validation.
// It validates timestamp ordering (capturedAt ≤ receivedAt ≤ processedAt)
// and emits a PositionReceivedEvent on success.
//
// Args:
//   vehicleID: Vehicle identifier (Value Object)
//   eventID: Unique event identifier (UUID v4)
//   geoPosition: Geographic coordinates (Value Object)
//   capturedAt: Client-provided capture time
//   receivedAt: Server reception time
//   processedAt: Server post-validation time
//   traceID: Trace ID for request tracking
//   idempotencyKey: Optional idempotency key (may be nil)
//   accuracy: Optional accuracy in meters (may be nil)
//
// Returns:
//   *VehiclePosition: Valid entity if all validations pass
//   error: If any validation fails (timestamp ordering, nil inputs)
//
// Verifies: Requirement 7 (4-Timestamps), P4, P5
func NewVehiclePosition(
	vehicleID *VehicleID,
	eventID EventID,
	geoPosition *GeoPosition,
	capturedAt CapturedTimestamp,
	receivedAt ReceivedTimestamp,
	processedAt ProcessedTimestamp,
	traceID TraceID,
	idempotencyKey *IdempotencyKey,
	accuracy *float64,
) (*VehiclePosition, error) {
	// Validate required inputs
	if vehicleID == nil {
		return nil, NewAppError("VEHICLE_POSITION_INVALID_VEHICLE_ID", "vehicleID is required", nil)
	}

	if eventID.IsEmpty() {
		return nil, NewAppError("VEHICLE_POSITION_INVALID_EVENT_ID", "eventID is required", nil)
	}

	if geoPosition == nil {
		return nil, NewAppError("VEHICLE_POSITION_INVALID_GEO_POSITION", "geoPosition is required", nil)
	}

	if traceID.IsEmpty() {
		return nil, NewAppError("VEHICLE_POSITION_INVALID_TRACE_ID", "traceID is required", nil)
	}

	// Validate timestamp ordering: capturedAt ≤ receivedAt ≤ processedAt
	if err := ValidateTimestampOrdering(capturedAt, receivedAt, processedAt); err != nil {
		return nil, NewAppError(
			"VEHICLE_POSITION_TIMESTAMP_ORDER_VIOLATION",
			fmt.Sprintf("timestamp ordering violation: %v", err),
			err,
		)
	}

	// Create the entity
	entity := &VehiclePosition{
		vehicleID:      vehicleID,
		eventID:        eventID,
		geoPosition:    geoPosition,
		capturedAt:     capturedAt,
		receivedAt:     receivedAt,
		processedAt:    processedAt,
		storedAt:       nil, // Will be set by worker after TimescaleDB INSERT
		traceID:        traceID,
		idempotencyKey: idempotencyKey,
		accuracy:       accuracy,
	}

	// Emit domain event: PositionReceivedEvent
	entity.positionReceivedEvent = NewPositionReceivedEvent(
		eventID,
		vehicleID,
		geoPosition,
		capturedAt,
		receivedAt,
		traceID,
		idempotencyKey,
		time.Now().UTC(),
	)

	return entity, nil
}

// --- Domain Behavior Methods (NOT getters) ---

// ReceptionLatency calculates the network + client processing time.
// Formula: receivedAt - capturedAt
// Represents time from client capture to server reception.
//
// Verifies: Requirement 7 (Latency Calculation)
func (vp *VehiclePosition) ReceptionLatency() time.Duration {
	return vp.receivedAt.Value().Sub(vp.capturedAt.Value())
}

// ProcessingLatency calculates the server validation time.
// Formula: processedAt - receivedAt
// Represents time from server reception to validation completion.
//
// Verifies: Requirement 7 (Latency Calculation)
func (vp *VehiclePosition) ProcessingLatency() time.Duration {
	return vp.processedAt.Value().Sub(vp.receivedAt.Value())
}

// TotalLatency calculates the total ingestion latency.
// Formula: processedAt - capturedAt
// Represents time from client capture to server validation completion.
//
// Verifies: Requirement 7 (Latency Calculation)
func (vp *VehiclePosition) TotalLatency() time.Duration {
	return vp.processedAt.Value().Sub(vp.capturedAt.Value())
}

// StorageLatency calculates the time from validation to database persistence.
// Only valid if storedAt is set (by worker after TimescaleDB INSERT).
// If storedAt is nil, returns 0 duration.
// Formula: storedAt - processedAt
//
// Verifies: Requirement 7 (Latency Calculation)
func (vp *VehiclePosition) StorageLatency() time.Duration {
	if vp.storedAt == nil {
		return 0
	}
	return vp.storedAt.Sub(vp.processedAt.Value())
}

// ToKafkaPayload generates a JSON-serializable representation for Kafka publishing.
// Includes all 5 timestamps (with storedAt as nil for HTTP layer).
// Used for publishing to Kafka and downstream processing.
//
// The returned map includes:
//   - event_id: UUID v4 string
//   - vehicle_id: Vehicle identifier string
//   - latitude: Geographic latitude
//   - longitude: Geographic longitude
//   - accuracy: Optional accuracy in meters (omitted if nil)
//   - captured_at: ISO 8601 UTC timestamp
//   - received_at: ISO 8601 UTC timestamp
//   - processed_at: ISO 8601 UTC timestamp
//   - stored_at: null (added by worker after TimescaleDB INSERT)
//
// Verifies: Requirement 7 (4-Timestamps), Requirement 3 (Kafka Payload)
func (vp *VehiclePosition) ToKafkaPayload() map[string]interface{} {
	payload := map[string]interface{}{
		"event_id":    vp.eventID.String(),
		"vehicle_id":  vp.vehicleID.String(),
		"latitude":    vp.geoPosition.Latitude(),
		"longitude":   vp.geoPosition.Longitude(),
		"captured_at": vp.capturedAt.String(),
		"received_at": vp.receivedAt.String(),
		"processed_at": vp.processedAt.String(),
		"stored_at":   nil, // HTTP layer does not set this; worker adds it
	}

	// Include accuracy if present
	if vp.accuracy != nil {
		payload["accuracy"] = *vp.accuracy
	}

	return payload
}

// --- Entity Accessors (for domain logic, not presentation) ---

// VehicleID returns the vehicle identifier.
func (vp *VehiclePosition) VehicleID() *VehicleID {
	return vp.vehicleID
}

// EventID returns the unique event identifier.
func (vp *VehiclePosition) EventID() EventID {
	return vp.eventID
}

// GeoPosition returns the geographic coordinates.
func (vp *VehiclePosition) GeoPosition() *GeoPosition {
	return vp.geoPosition
}

// CapturedAt returns the client-provided capture timestamp.
func (vp *VehiclePosition) CapturedAt() CapturedTimestamp {
	return vp.capturedAt
}

// ReceivedAt returns the server reception timestamp.
func (vp *VehiclePosition) ReceivedAt() ReceivedTimestamp {
	return vp.receivedAt
}

// ProcessedAt returns the server post-validation timestamp.
func (vp *VehiclePosition) ProcessedAt() ProcessedTimestamp {
	return vp.processedAt
}

// StoredAt returns the database persistence timestamp (may be nil for HTTP layer).
func (vp *VehiclePosition) StoredAt() *time.Time {
	return vp.storedAt
}

// TraceID returns the trace identifier for request tracking.
func (vp *VehiclePosition) TraceID() TraceID {
	return vp.traceID
}

// IdempotencyKey returns the optional idempotency key (may be nil).
func (vp *VehiclePosition) IdempotencyKey() *IdempotencyKey {
	return vp.idempotencyKey
}

// Accuracy returns the optional accuracy value in meters (may be nil).
func (vp *VehiclePosition) Accuracy() *float64 {
	return vp.accuracy
}

// --- Domain Events ---

// Events returns the domain events emitted by this entity.
// Currently emits only PositionReceivedEvent upon creation.
// The Application layer consumes this event to decide actions (e.g., publish to Kafka).
//
// Verifies: Requirement 11 (Domain Events), P12
func (vp *VehiclePosition) Events() []*PositionReceivedEvent {
	if vp.positionReceivedEvent == nil {
		return nil
	}
	return []*PositionReceivedEvent{vp.positionReceivedEvent}
}

// --- Serialization (for logging/debugging, not presentation) ---

// MarshalJSON implements json.Marshaler for serialization.
// Includes all 5 timestamps and relevant fields for auditing.
func (vp *VehiclePosition) MarshalJSON() ([]byte, error) {
	type Alias struct {
		VehicleID      string                 `json:"vehicle_id"`
		EventID        string                 `json:"event_id"`
		Latitude       float64                `json:"latitude"`
		Longitude      float64                `json:"longitude"`
		Accuracy       *float64               `json:"accuracy,omitempty"`
		CapturedAt     string                 `json:"captured_at"`
		ReceivedAt     string                 `json:"received_at"`
		ProcessedAt    string                 `json:"processed_at"`
		StoredAt       *string                `json:"stored_at"`
		TraceID        string                 `json:"trace_id"`
		IdempotencyKey *string                `json:"idempotency_key,omitempty"`
	}

	storedAtStr := ""
	var storedAtPtr *string
	if vp.storedAt != nil {
		storedAtStr = vp.storedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		storedAtPtr = &storedAtStr
	}

	idempotencyKeyStr := ""
	var idempotencyKeyPtr *string
	if vp.idempotencyKey != nil {
		idempotencyKeyStr = vp.idempotencyKey.String()
		idempotencyKeyPtr = &idempotencyKeyStr
	}

	return json.Marshal(&Alias{
		VehicleID:      vp.vehicleID.String(),
		EventID:        vp.eventID.String(),
		Latitude:       vp.geoPosition.Latitude(),
		Longitude:      vp.geoPosition.Longitude(),
		Accuracy:       vp.accuracy,
		CapturedAt:     vp.capturedAt.String(),
		ReceivedAt:     vp.receivedAt.String(),
		ProcessedAt:    vp.processedAt.String(),
		StoredAt:       storedAtPtr,
		TraceID:        vp.traceID.String(),
		IdempotencyKey: idempotencyKeyPtr,
	})
}

// SetStoredAt sets the database persistence timestamp.
// Called by the worker after successfully inserting into TimescaleDB.
// This is the only setter in the entity: it's used to complete the final timestamp.
//
// Note: This is NOT immutability violation because storedAt is explicitly
// designed to be set after entity creation, by the worker layer.
// The HTTP layer never calls this; it remains nil for HTTP responses.
func (vp *VehiclePosition) SetStoredAt(t time.Time) {
	if vp.storedAt == nil {
		vp.storedAt = &t
	}
}
