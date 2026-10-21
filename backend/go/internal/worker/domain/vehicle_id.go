package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// VehicleID is a Value Object representing a vehicle identifier.
// It is immutable and validates invariants in the constructor.
//
// Validation rules:
// - Non-empty
// - Alphanumeric only (letters and digits, no spaces or special chars)
// - Maximum 50 characters
type VehicleID struct {
	value string
}

// NewVehicleID creates a new VehicleID with validation.
// Returns error if validation fails.
func NewVehicleID(id string) (*VehicleID, error) {
	// Trim whitespace
	id = strings.TrimSpace(id)

	// Check if empty
	if id == "" {
		return nil, NewAppError(
			"INVALID_VEHICLE_ID",
			"vehicle_id cannot be empty",
			nil,
		)
	}

	// Check max length
	if len(id) > 50 {
		return nil, NewAppError(
			"INVALID_VEHICLE_ID",
			fmt.Sprintf("vehicle_id exceeds maximum length of 50 characters (got %d)", len(id)),
			nil,
		)
	}

	// Check alphanumeric (letters and digits only)
	for _, char := range id {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return nil, NewAppError(
				"INVALID_VEHICLE_ID",
				fmt.Sprintf("vehicle_id contains invalid character '%c'; only alphanumeric characters are allowed", char),
				nil,
			)
		}
	}

	return &VehicleID{value: id}, nil
}

// String returns the string representation of the vehicle ID.
func (v *VehicleID) String() string {
	return v.value
}
