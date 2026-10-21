package domain

import (
	"github.com/google/uuid"
)

// EventID represents a unique identifier for a position event.
// It is a Value Object: immutable after creation, validated in the constructor.
type EventID string

// NewEventID creates and validates a new EventID from a UUID v4 string.
// Returns error if the string is not a valid UUID v4.
func NewEventID(id string) (EventID, error) {
	if id == "" {
		return "", NewAppError(
			"INVALID_EVENT_ID",
			"event_id cannot be empty",
			nil,
		)
	}

	// Validate that it's a valid UUID v4
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", NewAppError(
			"INVALID_EVENT_ID",
			"event_id is not a valid UUID v4",
			err,
		)
	}

	// Verify it's v4 (optional but good practice)
	if parsed.Version() != 4 {
		return "", NewAppError(
			"INVALID_EVENT_ID",
			"event_id must be a UUID v4",
			nil,
		)
	}

	return EventID(parsed.String()), nil
}

// String returns the EventID as a string.
func (e EventID) String() string {
	return string(e)
}
