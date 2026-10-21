package domain

import (
	"math"
)

// Accuracy is a Value Object representing the uncertainty/accuracy of a geographic position.
// It is optional - a position may not have an accuracy value.
// When present, it must be a valid positive number (not NaN or Infinity).
//
// Accuracy is typically measured in meters.
type Accuracy struct {
	value float64
}

// NewAccuracy creates a new Accuracy value object with validation.
// Returns error if the value is NaN or Infinity.
func NewAccuracy(val float64) (*Accuracy, error) {
	// Check for NaN
	if math.IsNaN(val) {
		return nil, NewAppError(
			"INVALID_ACCURACY",
			"accuracy must be a valid number (not NaN)",
			nil,
		)
	}

	// Check for Infinity
	if math.IsInf(val, 0) {
		return nil, NewAppError(
			"INVALID_ACCURACY",
			"accuracy must be a valid number (not Infinity)",
			nil,
		)
	}

	return &Accuracy{value: val}, nil
}

// Value returns the underlying float64 value.
func (a *Accuracy) Value() float64 {
	if a == nil {
		return 0
	}
	return a.value
}
