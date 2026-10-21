package infrastructure

import (
	"time"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// SystemClock implements ports.Clock with the real system clock.
// For production use; returns UTC timestamps via time.Now().UTC().
//
// Thread-safe: holds no mutable state beyond the empty struct.
//
// RF: RF-03 (Persistence with timestamps)
type SystemClock struct{}

// NewSystemClock creates a SystemClock.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// Now returns the current instant in UTC.
// Implements ports.Clock.Now.
func (c *SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// Ensure SystemClock implements ports.Clock at compile time.
var _ ports.Clock = (*SystemClock)(nil)
