package domain

import (
	"github.com/fleet-platform/backend/internal/ingestion/ports"
	"github.com/google/uuid"
)

// TraceID represents a unique identifier for tracing requests across the system.
// It is a Value Object: immutable after creation, validated in the constructor.
// TraceID is either provided as a UUID v4 string from the X-Trace-ID header,
// or generated automatically if absent.
// This enables complete traceability of requests from client to backend.
type TraceID string

// NewTraceID creates a new TraceID from an optional header value or generates a new one.
// If headerValue is nil or empty, generates a new UUID v4 using the provided IDGenerator.
// If headerValue is provided, validates it as a UUID v4 format before accepting it.
// Returns an error if the provided header value is not a valid UUID v4.
//
// Verifies: Requirement 8 (Trace-ID), P1 (Validation), P9 (Trace-ID)
//
// Args:
//   headerValue: Optional string from X-Trace-ID HTTP header (may be nil or empty)
//   generator: IDGenerator dependency for generating new UUID v4 when headerValue is absent
//
// Returns:
//   TraceID: The validated or generated TraceID
//   error: InvalidTraceIDError if headerValue is provided but invalid, or if generator fails
func NewTraceID(headerValue *string, generator ports.IDGenerator) (TraceID, error) {
	// Case 1: Header is provided and non-empty
	if headerValue != nil && *headerValue != "" {
		return validateProvidedTraceID(*headerValue)
	}

	// Case 2: Header is absent or empty - generate a new UUID v4
	return generateNewTraceID(generator)
}

// validateProvidedTraceID validates a header-provided trace ID.
// The header value must be a valid UUID v4 format.
// Returns InvalidTraceIDError if validation fails.
func validateProvidedTraceID(headerValue string) (TraceID, error) {
	// Attempt to parse as UUID v4
	parsedUUID, err := uuid.Parse(headerValue)
	if err != nil {
		return "", InvalidTraceIDError{Value: headerValue}
	}

	// Verify that the parsed UUID is actually v4
	if parsedUUID.Version() != 4 {
		return "", InvalidTraceIDError{Value: headerValue}
	}

	return TraceID(parsedUUID.String()), nil
}

// generateNewTraceID generates a new UUID v4 trace ID using the injected IDGenerator.
// Returns an error if the generator is nil or produces an invalid UUID.
func generateNewTraceID(generator ports.IDGenerator) (TraceID, error) {
	if generator == nil {
		return "", NewAppError("TRACE_ID_GENERATOR_NIL", "IDGenerator is required", nil)
	}

	// Generate UUID v4 string from the injected IDGenerator
	rawID := generator.Generate()

	// Validate that the generated string is a valid UUID v4
	if _, err := uuid.Parse(rawID); err != nil {
		return "", NewAppError(
			"INVALID_GENERATED_TRACE_ID",
			"generated trace_id is not a valid UUID v4",
			err,
		)
	}

	return TraceID(rawID), nil
}

// String returns the TraceID as a string (hex format UUID).
func (t TraceID) String() string {
	return string(t)
}

// IsEmpty checks if the TraceID is empty or unset.
func (t TraceID) IsEmpty() bool {
	return string(t) == ""
}
