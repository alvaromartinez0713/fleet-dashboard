package mothers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPositionMotherValid(t *testing.T) {
	req := PositionMother{}.Valid().Build()
	
	assert.Equal(t, "VH-001", req.VehicleID)
	assert.Equal(t, 40.7128, req.Latitude)
	assert.Equal(t, -74.0060, req.Longitude)
	assert.False(t, req.CapturedAt.IsZero())
	// Should be in past (approximately 10 seconds ago)
	assert.True(t, req.CapturedAt.Before(time.Now()))
}

func TestPositionMotherFutureTimestamp(t *testing.T) {
	req := PositionMother{}.FutureTimestamp().Build()
	
	assert.Equal(t, "VH-001", req.VehicleID)
	// Should be in future
	assert.True(t, req.CapturedAt.After(time.Now()))
}

func TestPositionMotherOutOfRangeLatitude(t *testing.T) {
	req := PositionMother{}.OutOfRangeLatitude().Build()
	
	assert.Equal(t, 95.0, req.Latitude)
	assert.Greater(t, req.Latitude, 90.0)
}

func TestPositionMotherOutOfRangeLongitude(t *testing.T) {
	req := PositionMother{}.OutOfRangeLongitude().Build()
	
	assert.Equal(t, 185.0, req.Longitude)
	assert.Greater(t, req.Longitude, 180.0)
}

func TestPositionMotherMissingVehicleID(t *testing.T) {
	req := PositionMother{}.MissingVehicleID().Build()
	
	assert.Empty(t, req.VehicleID)
}

func TestPositionMotherBuilderChaining(t *testing.T) {
	req := PositionMother{}.Valid().
		WithVehicleID("VH-999").
		WithLatitude(50.0).
		WithLongitude(-100.0).
		Build()
	
	assert.Equal(t, "VH-999", req.VehicleID)
	assert.Equal(t, 50.0, req.Latitude)
	assert.Equal(t, -100.0, req.Longitude)
}

func TestPositionMotherNewInstancesPerCall(t *testing.T) {
	req1 := PositionMother{}.Valid().Build()
	req2 := PositionMother{}.Valid().Build()
	
	// Different instances (not shared)
	assert.NotSame(t, req1, req2)
	// But same values
	assert.Equal(t, req1.VehicleID, req2.VehicleID)
}
