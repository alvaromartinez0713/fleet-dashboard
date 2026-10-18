package infrastructure

import (
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// Ensure RealClock implements the Clock port.
var _ ports.Clock = (*RealClock)(nil)

// RealClock is the production implementation of the Clock port.
// It wraps time.Now() to provide the current UTC time.
//
// RealClock has no state and is safe for concurrent use.
// It can be reused across multiple goroutines without synchronization.
type RealClock struct{}

// NewRealClock creates a new RealClock instance.
// This constructor is provided for consistency with adapter instantiation patterns,
// though RealClock carries no state.
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current time in UTC.
// This method wraps time.Now().UTC() directly.
func (c *RealClock) Now() time.Time {
	return time.Now().UTC()
}
