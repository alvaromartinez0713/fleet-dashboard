package infrastructure

import (
	"testing"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

func TestRealClockImplementsClockPort(t *testing.T) {
	// Compile-time check: RealClock must implement ports.Clock
	var _ ports.Clock = (*RealClock)(nil)
}

func TestRealClockReturnsUTCTime(t *testing.T) {
	clock := NewRealClock()
	now := clock.Now()

	if now.IsZero() {
		t.Fatal("Clock.Now() returned zero time")
	}

	if now.Location() != time.UTC {
		t.Fatalf("Clock.Now() did not return UTC time, got location: %v", now.Location())
	}
}

func TestRealClockReturnsCurrentTime(t *testing.T) {
	clock := NewRealClock()
	before := time.Now().UTC()
	now := clock.Now()
	after := time.Now().UTC()

	// now should be between before and after (with reasonable margin)
	if now.Before(before) || now.After(after.Add(1*time.Second)) {
		t.Fatalf("Clock.Now() did not return current time: before=%v, clock=%v, after=%v", before, now, after)
	}
}

func TestNewRealClockReturnsValidInstance(t *testing.T) {
	clock := NewRealClock()
	if clock == nil {
		t.Fatal("NewRealClock() returned nil")
	}

	// Verify the instance can be used
	now := clock.Now()
	if now.IsZero() {
		t.Fatal("Created clock instance could not call Now()")
	}
}
