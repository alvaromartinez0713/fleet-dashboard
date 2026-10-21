package mothers

import (
	"time"

	"github.com/google/uuid"

	"github.com/fleet-platform/backend/internal/worker/domain"
)

// WorkerVehiclePositionMother provides predefined test scenarios for domain.VehiclePosition entities.
// It implements the Object Mother pattern for testing the worker's domain layer.
// All values are deterministic and valid by default.
//
// This is used by worker tests (Suite Fast, property-based tests, etc.)
// to construct validated position entities without calling the parser.
//
// Example usage:
//
//	// Use default valid position
//	position, _ := WorkerVehiclePositionMother.Valid().Build()
//
//	// Override specific field
//	position, _ := WorkerVehiclePositionMother.Valid().
//	    WithVehicleID("VH-002").
//	    Build()
//
//	// Use predefined error scenario
//	position, err := WorkerVehiclePositionMother.FutureTimestamp().Build()
//	// err != nil (expected validation failure)
type WorkerVehiclePositionMother struct{}

// WorkerVehiclePositionBuilder holds mutable state for building a domain.VehiclePosition.
type WorkerVehiclePositionBuilder struct {
	eventID        string
	vehicleID      string
	latitude       float64
	longitude      float64
	accuracy       *float64
	capturedAt     string // ISO 8601 UTC
	receivedAt     *string
	processedAt    *string
	traceID        *string
	idempotencyKey *string
	validationTime time.Time
	persistenceTime time.Time
}

// Valid returns a builder with all fields set to valid values.
// Scenario: VH-001 at Manhattan, captured 10 seconds ago, valid UUIDs generated.
func (WorkerVehiclePositionMother) Valid() *WorkerVehiclePositionBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	eventID := uuid.New().String()
	traceID := uuid.New().String()
	idempotencyKey := uuid.New().String()

	return &WorkerVehiclePositionBuilder{
		eventID:        eventID,
		vehicleID:      "VH-001",
		latitude:       40.7128,
		longitude:      -74.0060,
		accuracy:       nil, // Optional field
		capturedAt:     tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
		receivedAt:     nil, // Optional field
		processedAt:    nil, // Optional field
		traceID:        &traceID,
		idempotencyKey: &idempotencyKey,
		validationTime: now,
		persistenceTime: now,
	}
}

// FutureTimestamp returns a builder with captured_at set to 60 seconds in the future.
// Scenario: Future timestamp (should fail NewCapturedTimestamp validation).
func (WorkerVehiclePositionMother) FutureTimestamp() *WorkerVehiclePositionBuilder {
	now := time.Now().UTC()
	futureTime := now.Add(60 * time.Second)

	builder := WorkerVehiclePositionMother.Valid()
	builder.capturedAt = futureTime.Format("2006-01-02T15:04:05.000Z07:00")
	return builder
}

// OutOfRangeLatitude returns a builder with latitude > 90.
// Scenario: Invalid latitude (should fail NewGeoPosition validation).
func (WorkerVehiclePositionMother) OutOfRangeLatitude() *WorkerVehiclePositionBuilder {
	builder := WorkerVehiclePositionMother.Valid()
	builder.latitude = 95.0 // Invalid: > 90
	return builder
}

// OutOfRangeLongitude returns a builder with longitude > 180.
// Scenario: Invalid longitude (should fail NewGeoPosition validation).
func (WorkerVehiclePositionMother) OutOfRangeLongitude() *WorkerVehiclePositionBuilder {
	builder := WorkerVehiclePositionMother.Valid()
	builder.longitude = 185.0 // Invalid: > 180
	return builder
}

// InvalidTraceID returns a builder with invalid trace_id format.
// Scenario: Non-UUID trace ID (should fail NewTraceID validation).
func (WorkerVehiclePositionMother) InvalidTraceID() *WorkerVehiclePositionBuilder {
	builder := WorkerVehiclePositionMother.Valid()
	invalidID := "not-a-uuid"
	builder.traceID = &invalidID
	return builder
}

// InvalidIdempotencyKey returns a builder with invalid idempotency_key format.
// Scenario: Non-UUID idempotency key (should fail NewIdempotencyKey validation).
func (WorkerVehiclePositionMother) InvalidIdempotencyKey() *WorkerVehiclePositionBuilder {
	builder := WorkerVehiclePositionMother.Valid()
	invalidKey := "not-a-uuid"
	builder.idempotencyKey = &invalidKey
	return builder
}

// WithEventID sets the event_id and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithEventID(eventID string) *WorkerVehiclePositionBuilder {
	b.eventID = eventID
	return b
}

// WithVehicleID sets the vehicle_id and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithVehicleID(vehicleID string) *WorkerVehiclePositionBuilder {
	b.vehicleID = vehicleID
	return b
}

// WithLatitude sets the latitude and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithLatitude(latitude float64) *WorkerVehiclePositionBuilder {
	b.latitude = latitude
	return b
}

// WithLongitude sets the longitude and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithLongitude(longitude float64) *WorkerVehiclePositionBuilder {
	b.longitude = longitude
	return b
}

// WithAccuracy sets the accuracy and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithAccuracy(accuracy float64) *WorkerVehiclePositionBuilder {
	b.accuracy = &accuracy
	return b
}

// WithTraceID sets the trace_id and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithTraceID(traceID string) *WorkerVehiclePositionBuilder {
	b.traceID = &traceID
	return b
}

// WithIdempotencyKey sets the idempotency_key and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithIdempotencyKey(key string) *WorkerVehiclePositionBuilder {
	b.idempotencyKey = &key
	return b
}

// WithValidationTime sets the validation time and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithValidationTime(t time.Time) *WorkerVehiclePositionBuilder {
	b.validationTime = t
	return b
}

// WithPersistenceTime sets the persistence time and returns the builder for chaining.
func (b *WorkerVehiclePositionBuilder) WithPersistenceTime(t time.Time) *WorkerVehiclePositionBuilder {
	b.persistenceTime = t
	return b
}

// Build constructs and returns a new domain.VehiclePosition instance.
// Returns error if any domain validation fails (as expected for error scenarios).
// Each call creates a new instance (no sharing of mutable state).
func (b *WorkerVehiclePositionBuilder) Build() (*domain.VehiclePosition, error) {
	return domain.NewVehiclePosition(
		b.eventID,
		b.vehicleID,
		b.latitude,
		b.longitude,
		b.accuracy,
		b.capturedAt,
		b.receivedAt,
		b.processedAt,
		b.traceID,
		b.idempotencyKey,
		b.validationTime,
		b.persistenceTime,
	)
}
