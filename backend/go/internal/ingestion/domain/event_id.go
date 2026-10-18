package domain

import (
	"github.com/fleet-platform/backend/internal/ingestion/ports"
	"github.com/google/uuid"
)

// EventID represents a unique identifier for a position event.
// It is a Value Object: immutable after creation, validated in the constructor.
// EventID is generated as a UUID v4 by an injected IDGenerator dependency.
type EventID string

// NewEventID creates a new EventID using the provided IDGenerator.
// The generator is called to produce a UUID v4 identifier.
// Returns an error if the generator is nil or if the generated UUID is invalid.
//
// Verifies: Requirement 3, P1 (Validation)
func NewEventID(generator ports.IDGenerator) (EventID, error) {
	if generator == nil {
		return "", NewAppError("EVENT_ID_GENERATOR_NIL", "IDGenerator is required", nil)
	}

	// Generate UUID v4 string from the injected IDGenerator
	rawID := generator.Generate()

	// Validate that the generated string is a valid UUID v4
	// by attempting to parse it with google/uuid
	if _, err := uuid.Parse(rawID); err != nil {
		return "", NewAppError(
			"INVALID_EVENT_ID",
			"generated event_id is not a valid UUID v4",
			err,
		)
	}

	return EventID(rawID), nil
}

// String returns the EventID as a string (hex format UUID).
func (e EventID) String() string {
	return string(e)
}

// IsEmpty checks if the EventID is empty or unset.
func (e EventID) IsEmpty() bool {
	return string(e) == ""
}
