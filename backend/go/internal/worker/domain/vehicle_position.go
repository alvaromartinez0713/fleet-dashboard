package domain

import (
	"time"
)

// VehiclePosition is an Entity representing a position that has been validated
// and is ready for persistence in TimescaleDB.
//
// It encapsulates all validated data from a message. It contains ONLY
// domain concerns (value objects), not infrastructure concerns.
//
// VehiclePosition is constructed via NewVehiclePosition factory method which
// encapsulates all validation rules.
//
// Internal state is only accessible via the ToPersistenceSQL method,
// which converts the entity to SQL parameters.
//
// RF: RF-02 (Validation), RF-03 (Domain modeling), RNF-01 (Clean Architecture)
type VehiclePosition struct {
	eventID        EventID
	vehicleID      *VehicleID
	geoPosition    *GeoPosition
	accuracy       *Accuracy
	capturedAt     CapturedTimestamp
	receivedAt     *ReceivedTimestamp
	processedAt    *ProcessedTimestamp
	traceID        TraceID
	idempotencyKey IdempotencyKey
	storedAt       StoredTimestamp
}

// NewVehiclePosition creates and validates a VehiclePosition from primitive data.
//
// This is the domain's public factory - it receives primitives and constructs
// a fully validated entity with value objects.
//
// Clean Architecture principle: Domain does NOT depend on DTOs from outer layers.
// The adapter layer is responsible for extracting primitives and calling this factory.
//
// Args:
//   All primitive values extracted from external source (Kafka message)
//   validationTime: Current time used to validate temporal constraints (capturedAt, receivedAt, processedAt must not be in future)
//   persistenceTime: Time when this entity is being persisted to the database (generates storedAt)
//
// Returns a validated VehiclePosition entity or AppError.
func NewVehiclePosition(
	eventID string,
	vehicleID string,
	latitude float64,
	longitude float64,
	accuracy *float64,
	capturedAt string,
	receivedAt *string,
	processedAt *string,
	traceID *string,
	idempotencyKey *string,
	validationTime time.Time,
	persistenceTime time.Time,
) (*VehiclePosition, error) {
	// Construct value objects — each constructor validates its invariants
	evID, err := NewEventID(eventID)
	if err != nil {
		return nil, err
	}

	vID, err := NewVehicleID(vehicleID)
	if err != nil {
		return nil, err
	}

	geoPos, err := NewGeoPosition(latitude, longitude)
	if err != nil {
		return nil, err
	}

	capAt, err := NewCapturedTimestamp(capturedAt, validationTime)
	if err != nil {
		return nil, err
	}

	// ReceivedAt is optional
	var recAt *ReceivedTimestamp
	if receivedAt != nil && *receivedAt != "" {
		recAt, err = NewReceivedTimestampFromString(capAt.Value(), *receivedAt, validationTime)
		if err != nil {
			return nil, err
		}
	}

	// ProcessedAt is optional
	var procAt *ProcessedTimestamp
	if processedAt != nil && *processedAt != "" {
		procAt, err = NewProcessedTimestampFromString(capAt.Value(), *processedAt, validationTime)
		if err != nil {
			return nil, err
		}
	}

	// Accuracy is optional
	var acc *Accuracy
	if accuracy != nil {
		acc, err = NewAccuracy(*accuracy)
		if err != nil {
			return nil, err
		}
	}

	// TraceID - required, must come from client
	tID, err := NewTraceID(traceID)
	if err != nil {
		// Invalid trace_id → treat as non-recoverable
		return nil, err
	}

	// IdempotencyKey - required, must come from client
	idempKey, err := NewIdempotencyKey(idempotencyKey)
	if err != nil {
		// Invalid idempotency_key format → treat as non-recoverable
		return nil, err
	}

	// StoredAt - generated at persistence time
	storeAt := NewStoredTimestamp(persistenceTime)

	// Return domain entity (all fields are validated value objects or nil)
	return &VehiclePosition{
		eventID:        evID,
		vehicleID:      vID,
		geoPosition:    geoPos,
		accuracy:       acc,
		capturedAt:     capAt,
		receivedAt:     recAt,
		processedAt:    procAt,
		traceID:        tID,
		idempotencyKey: idempKey,
		storedAt:       storeAt,
	}, nil
}

// ToPersistenceSQL converts the entity to SQL parameters.
// This is the only way external code (adapters) accesses internal fields for persistence.
// Returns values in the order needed for the INSERT statement.
func (vp *VehiclePosition) ToPersistenceSQL() (
	eventID string,
	vehicleID *string,
	latitude *float64,
	longitude *float64,
	accuracy *float64,
	capturedAt string,
	receivedAt *string,
	processedAt *string,
	traceID *string,
	idempotencyKey *string,
	storedAt string,
) {
	// Build event ID (required)
	eventID = vp.eventID.String()

	// Build vehicle ID (optional)
	if vp.vehicleID != nil {
		vid := vp.vehicleID.String()
		vehicleID = &vid
	}

	// Build geo coordinates (optional)
	if vp.geoPosition != nil {
		lat, lon := vp.geoPosition.Values()
		latitude = &lat
		longitude = &lon
	}

	// Build accuracy (optional)
	if vp.accuracy != nil {
		acc := vp.accuracy.Value()
		accuracy = &acc
	}

	// Build captured at (required)
	capturedAt = vp.capturedAt.String()

	// Build received at (optional)
	if vp.receivedAt != nil {
		ras := vp.receivedAt.String()
		receivedAt = &ras
	}

	// Build processed at (optional)
	if vp.processedAt != nil {
		pas := vp.processedAt.String()
		processedAt = &pas
	}

	// Build trace ID (always present, required from client)
	tid := vp.traceID.String()
	traceID = &tid

	// Build idempotency key (always present, required from client)
	ik := vp.idempotencyKey.String()
	idempotencyKey = &ik

	// Build stored at (always present, generated at persistence time)
	storedAt = vp.storedAt.String()

	return
}

// Getters for external access (use case, services, adapters)

// EventID returns the unique event identifier.
func (vp *VehiclePosition) EventID() EventID {
	return vp.eventID
}

// VehicleID returns the vehicle identifier.
func (vp *VehiclePosition) VehicleID() *VehicleID {
	return vp.vehicleID
}

// TraceID returns the trace identifier for correlation.
func (vp *VehiclePosition) TraceID() TraceID {
	return vp.traceID
}
