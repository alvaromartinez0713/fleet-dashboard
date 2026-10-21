package ports

import (
	"context"
)

// DeadLetterRecord represents a structured record of an invalid or retry-exhausted event
// written to the Dead_Letter_Sink for recovery and diagnosis (Req 9.2, 9.4).
// It intentionally omits latitude, longitude, and the full payload to reduce noise
// and protect sensitive data in diagnostic logs.
type DeadLetterRecord struct {
	// TraceID is the resolved trace ID associated with the event, allowing correlation
	// of logs and requests across the system.
	TraceID string

	// EventID is the unique identifier of the event. May be empty if the payload was
	// malformed and did not contain a valid event_id field.
	EventID string

	// VehicleID is the identifier of the vehicle. May be empty if the payload was
	// malformed and did not contain a valid vehicle_id field.
	VehicleID string

	// Classification categorizes the nature of the dead-letter record.
	// Valid values: "recoverable" | "non_recoverable"
	// - "non_recoverable": event is permanently invalid (malformed JSON, missing required fields, invalid data)
	// - "recoverable": event processing failed after maxRetries exhausted (transient errors that persisted)
	Classification string

	// ReasonCategory is a stable, machine-readable category describing why the event was dead-lettered.
	// Examples: "invalid_event", "malformed_payload", "retry_exhausted", "schema_mismatch"
	// Used for classification and diagnostic filtering without parsing unstructured messages.
	ReasonCategory string

	// OccurredAt is the timestamp (ISO 8601 UTC, millisecond precision) when the event was
	// written to the Dead_Letter_Sink.
	OccurredAt string
}

// DeadLetterSink records invalid or retry-exhausted events for later diagnosis
// without blocking consumption (Req 5.7, 6.7).
//
// For the MVP, the sink is a structured JSON log (Dead_Letter_Sink). A Kafka
// dead-letter topic is future work documented in Requisito 5.7 and product.md.
//
// A non-nil error from Write means the sink write failed; the caller MUST NOT
// commit the offset in that case. This ensures dead-letter records are not lost
// and the event remains available for reprocessing after manual intervention or
// configuration change (Req 5.7, 6.7).
type DeadLetterSink interface {
	// Write persists a dead-letter record.
	//
	// On success, Write returns nil and the record is logged for recovery and
	// diagnosis. On failure, Write returns a non-nil error; the caller MUST NOT
	// commit the Kafka offset of the message that triggered this dead-letter write
	// (Req 6.7). The event remains unconsumed and is reprocesed by the worker,
	// relying on deduplication (Req 4) to avoid duplicates if the sink write partially
	// succeeded before the error.
	//
	// The record parameter must not be nil.
	Write(ctx context.Context, record DeadLetterRecord) error
}
