package ports

// ConsumedPositionEvent is the domain model of a position event consumed from Kafka.
// It represents the PositionReceived_Event JSON published by the ingestion service.
//
// Validation of individual fields is the responsibility of validation.go, which
// reuses value objects from ingestion/domain without redefining constraints.
//
// Pointers for optional fields (received_at, processed_at, accuracy, trace_id, idempotency_key)
// enable NULL persistence in the database when those fields are absent.
//
// RF: RF-02, RF-03, RF-08
type ConsumedPositionEvent struct {
	// EventID is the unique identifier (UUID v4) of the event.
	// Required field. Must be a valid UUID v4.
	EventID string

	// VehicleID is the vehicle identifier.
	// Required field. Must be alphanumeric, 1–50 characters.
	VehicleID string

	// Latitude is the geographic latitude.
	// Required field. Must be in range [-90.0, 90.0], not NaN or Infinity.
	Latitude float64

	// Longitude is the geographic longitude.
	// Required field. Must be in range [-180.0, 180.0], not NaN or Infinity.
	Longitude float64

	// Accuracy is the optional accuracy/uncertainty of the position.
	// Nil means absent; when present, persists as a numeric value in the database.
	// Optional field (RF-02).
	Accuracy *float64

	// CapturedAt is the ISO 8601 UTC timestamp when the device captured the position.
	// Required field. Must be a valid ISO 8601 UTC string.
	CapturedAt string

	// ReceivedAt is the ISO 8601 UTC timestamp when the ingestion service received the position.
	// Optional field (RF-02): if not provided, Nil means NULL persistence.
	// When present, must be a valid ISO 8601 UTC timestamp.
	ReceivedAt *string

	// ProcessedAt is the ISO 8601 UTC timestamp when the ingestion service completed validation.
	// Optional field (RF-02): if not provided, Nil means NULL persistence.
	// When present, must be a valid ISO 8601 UTC timestamp.
	ProcessedAt *string

	// TraceID is the optional trace identifier (UUID v4) for request correlation.
	// Nil means absent or will be generated (RF-08).
	// When present, must be a valid UUID v4.
	TraceID *string

	// IdempotencyKey is the optional idempotency key from the X-Idempotency-Key header
	// passed through the Kafka event. If present, must be a valid UUID v4.
	// Nil means the header was not provided in the original HTTP request.
	// The worker tolerates this field-if-absent but does NOT use it for deduplication;
	// deduplication is performed only via (captured_at, event_id) in the database.
	IdempotencyKey *string
}
