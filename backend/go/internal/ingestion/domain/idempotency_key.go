package domain

import (
	"github.com/google/uuid"
)

// IdempotencyKey is a Value Object representing an optional idempotency key.
// It is immutable after creation and validated in the constructor.
// IdempotencyKey can be nil (when not provided by the client).
// When present, it must be a valid UUID v4 format.
//
// RFC Reference: X-Idempotency-Key header (optional, from client)
// Validation: If present, must be valid UUID v4
// Note: Deduplication is NOT implemented in this spec (future work).
//
// Verifies: Requirement 2.7 (X-Idempotency-Key validation), P1 (Validation)
type IdempotencyKey struct {
	value *string // nil = not provided; non-nil = UUID v4 string
}

// NewIdempotencyKey creates a new IdempotencyKey from an optional string.
// If the input is nil or empty string, returns a nil IdempotencyKey (permitted).
// If the input is a non-empty string, validates it as UUID v4 format.
// Returns InvalidIdempotencyKeyError if validation fails (non-empty, invalid UUID v4).
func NewIdempotencyKey(keyStr *string) (*IdempotencyKey, error) {
	// If nil or empty, return nil IdempotencyKey (permitted, optional)
	if keyStr == nil || *keyStr == "" {
		return &IdempotencyKey{value: nil}, nil
	}

	// Validate that the provided string is a valid UUID v4
	if _, err := uuid.Parse(*keyStr); err != nil {
		return nil, &InvalidIdempotencyKeyError{
			Value: *keyStr,
		}
	}

	// Store the validated UUID string
	return &IdempotencyKey{value: keyStr}, nil
}

// String returns the IdempotencyKey as a string.
// Returns empty string if the key is nil (not provided).
func (i *IdempotencyKey) String() string {
	if i == nil || i.value == nil {
		return ""
	}
	return *i.value
}

// IsPresent checks if the IdempotencyKey was provided (not nil).
func (i *IdempotencyKey) IsPresent() bool {
	return i != nil && i.value != nil
}

// Value returns the underlying string pointer.
// Returns nil if the key is not present.
func (i *IdempotencyKey) Value() *string {
	if i == nil {
		return nil
	}
	return i.value
}
