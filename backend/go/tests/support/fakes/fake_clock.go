package fakes

import (
	"sync"
	"time"
)

// FakeClock allows controllable time for deterministic tests
// Implements Clock port
type FakeClock struct {
	mu       sync.RWMutex
	current  time.Time
}

// NewFakeClock creates a FakeClock initialized to now
func NewFakeClock() *FakeClock {
	return &FakeClock{
		current: time.Now().UTC(),
	}
}

// NewFakeClockAt creates a FakeClock initialized to a specific time
func NewFakeClockAt(t time.Time) *FakeClock {
	return &FakeClock{
		current: t.UTC(),
	}
}

// Now returns the current controlled time
func (fc *FakeClock) Now() time.Time {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.current
}

// Advance moves the clock forward by duration
func (fc *FakeClock) Advance(d time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.current = fc.current.Add(d)
}

// SetTime sets the clock to a specific time
func (fc *FakeClock) SetTime(t time.Time) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.current = t.UTC()
}

// GetTime returns a copy of the current time
func (fc *FakeClock) GetTime() time.Time {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.current
}
