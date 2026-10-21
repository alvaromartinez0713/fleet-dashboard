package domain

import (
	"fmt"
	"math"
)

// GeoPosition is a Value Object representing geographic coordinates.
// It is immutable and validates invariants in the constructor.
//
// Validation rules:
// - Latitude must be in range [-90.0, 90.0]
// - Longitude must be in range [-180.0, 180.0]
// - Both values are IEEE 754 64-bit floats
// - Neither latitude nor longitude can be NaN or Infinity
type GeoPosition struct {
	latitude  float64
	longitude float64
}

// NewGeoPosition creates a new GeoPosition with validation.
// Returns error if validation fails.
func NewGeoPosition(latitude, longitude float64) (*GeoPosition, error) {
	// Validate latitude: reject NaN and Infinity
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) {
		return nil, NewAppError(
			"INVALID_LATITUDE",
			"latitude must be a valid number (not NaN or Infinity)",
			nil,
		)
	}

	// Validate latitude range [-90, 90]
	if latitude < -90.0 || latitude > 90.0 {
		return nil, NewAppError(
			"INVALID_LATITUDE",
			fmt.Sprintf("latitude must be in range [-90.0, 90.0], got %.6f", latitude),
			nil,
		)
	}

	// Validate longitude: reject NaN and Infinity
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) {
		return nil, NewAppError(
			"INVALID_LONGITUDE",
			"longitude must be a valid number (not NaN or Infinity)",
			nil,
		)
	}

	// Validate longitude range [-180, 180]
	if longitude < -180.0 || longitude > 180.0 {
		return nil, NewAppError(
			"INVALID_LONGITUDE",
			fmt.Sprintf("longitude must be in range [-180.0, 180.0], got %.6f", longitude),
			nil,
		)
	}

	return &GeoPosition{
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

// Values returns the latitude and longitude as separate values.
func (g *GeoPosition) Values() (float64, float64) {
	return g.latitude, g.longitude
}
