package mothers

import (
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
)

// PositionMother provides predefined test scenarios for PositionRequest objects.
// It implements the Object Mother pattern with builder support for flexible test data creation.
// All values are deterministic (no randomness) to ensure test reproducibility.
//
// Example usage:
//
//	// Use default valid request
//	req := PositionMother.Valid().Build()
//
//	// Override specific field
//	req := PositionMother.Valid().WithVehicleID("VH-002").Build()
//
//	// Use predefined scenario
//	req := PositionMother.FutureTimestamp().Build()
type PositionMother struct{}

// PositionRequestBuilder holds mutable state for building a PositionRequest.
// Each field can be independently overridden before calling Build().
type PositionRequestBuilder struct {
	vehicleID  string
	latitude   float64
	longitude  float64
	capturedAt string // ISO 8601 UTC format
}

// Valid returns a builder with all fields set to valid values.
// Scenario: VH-001 at Manhattan (40.7128, -74.0060) captured 10 seconds ago.
func (PositionMother) Valid() *PositionRequestBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	return &PositionRequestBuilder{
		vehicleID:  "VH-001",
		latitude:   40.7128,
		longitude:  -74.0060,
		capturedAt: tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// FutureTimestamp returns a builder with captured_at set to 60 seconds in the future.
// Scenario: Request with future timestamp (should be rejected with 422).
func (PositionMother) FutureTimestamp() *PositionRequestBuilder {
	now := time.Now().UTC()
	futureTime := now.Add(60 * time.Second)

	return &PositionRequestBuilder{
		vehicleID:  "VH-001",
		latitude:   40.7128,
		longitude:  -74.0060,
		capturedAt: futureTime.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// OutOfRangeLatitude returns a builder with latitude > 90.
// Scenario: Invalid latitude (should be rejected with 422).
func (PositionMother) OutOfRangeLatitude() *PositionRequestBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	return &PositionRequestBuilder{
		vehicleID:  "VH-001",
		latitude:   95.0, // Invalid: > 90
		longitude:  -74.0060,
		capturedAt: tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// OutOfRangeLongitude returns a builder with longitude > 180.
// Scenario: Invalid longitude (should be rejected with 422).
func (PositionMother) OutOfRangeLongitude() *PositionRequestBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	return &PositionRequestBuilder{
		vehicleID:  "VH-001",
		latitude:   40.7128,
		longitude:  185.0, // Invalid: > 180
		capturedAt: tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// MissingVehicleID returns a builder with empty vehicle_id.
// Scenario: Missing required field (should be rejected with 422).
func (PositionMother) MissingVehicleID() *PositionRequestBuilder {
	now := time.Now().UTC()
	tenSecondsAgo := now.Add(-10 * time.Second)

	return &PositionRequestBuilder{
		vehicleID:  "", // Invalid: empty
		latitude:   40.7128,
		longitude:  -74.0060,
		capturedAt: tenSecondsAgo.Format("2006-01-02T15:04:05.000Z07:00"),
	}
}

// --- Builder Methods (Chainable) ---

// WithVehicleID sets the vehicle_id and returns the builder for chaining.
func (b *PositionRequestBuilder) WithVehicleID(vehicleID string) *PositionRequestBuilder {
	b.vehicleID = vehicleID
	return b
}

// WithLatitude sets the latitude and returns the builder for chaining.
func (b *PositionRequestBuilder) WithLatitude(latitude float64) *PositionRequestBuilder {
	b.latitude = latitude
	return b
}

// WithLongitude sets the longitude and returns the builder for chaining.
func (b *PositionRequestBuilder) WithLongitude(longitude float64) *PositionRequestBuilder {
	b.longitude = longitude
	return b
}

// WithCapturedAt sets the captured_at timestamp and returns the builder for chaining.
func (b *PositionRequestBuilder) WithCapturedAt(capturedAt string) *PositionRequestBuilder {
	b.capturedAt = capturedAt
	return b
}

// Build constructs and returns a new PositionRequest instance.
// Each call creates a new instance (no sharing of mutable state).
func (b *PositionRequestBuilder) Build() *dto.PositionRequest {
	return &dto.PositionRequest{
		VehicleID:  b.vehicleID,
		Latitude:   b.latitude,
		Longitude:  b.longitude,
		CapturedAt: b.capturedAt,
	}
}
