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
// Returns InvalidGeoPositionError if validation fails.
func NewGeoPosition(latitude, longitude float64) (*GeoPosition, error) {
	// Validate latitude: reject NaN and Infinity
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) {
		return nil, &InvalidGeoPositionError{
			Latitude:  &latitude,
			Longitude: &longitude,
			Reason:    "latitude must be a valid number (not NaN or Infinity)",
		}
	}

	// Validate latitude range [-90, 90]
	if latitude < -90.0 || latitude > 90.0 {
		return nil, &InvalidGeoPositionError{
			Latitude:  &latitude,
			Longitude: &longitude,
			Reason:    fmt.Sprintf("latitude must be in range [-90.0, 90.0], got %.6f", latitude),
		}
	}

	// Validate longitude: reject NaN and Infinity
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) {
		return nil, &InvalidGeoPositionError{
			Latitude:  &latitude,
			Longitude: &longitude,
			Reason:    "longitude must be a valid number (not NaN or Infinity)",
		}
	}

	// Validate longitude range [-180, 180]
	if longitude < -180.0 || longitude > 180.0 {
		return nil, &InvalidGeoPositionError{
			Latitude:  &latitude,
			Longitude: &longitude,
			Reason:    fmt.Sprintf("longitude must be in range [-180.0, 180.0], got %.6f", longitude),
		}
	}

	return &GeoPosition{
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

// Latitude returns the latitude value.
func (g *GeoPosition) Latitude() float64 {
	return g.latitude
}

// Longitude returns the longitude value.
func (g *GeoPosition) Longitude() float64 {
	return g.longitude
}

// Equals checks if two GeoPosition values are equal.
// Note: Uses exact floating-point comparison; for approximate equality,
// use a custom comparison function with epsilon tolerance.
func (g *GeoPosition) Equals(other *GeoPosition) bool {
	if g == nil || other == nil {
		return g == other
	}
	return g.latitude == other.latitude && g.longitude == other.longitude
}

// String returns the string representation of the position.
func (g *GeoPosition) String() string {
	return fmt.Sprintf("%.6f,%.6f", g.latitude, g.longitude)
}
