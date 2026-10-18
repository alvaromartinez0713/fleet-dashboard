package fakes

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// FakeIDGenerator is a test implementation of ports.IDGenerator.
// It returns deterministic UUIDs from a predefined sequence instead of random ones.
//
// Thread-safe: all operations are protected by sync.Mutex.
type FakeIDGenerator struct {
	sequentialIDs []uuid.UUID
	currentIndex  int
	mutex         sync.Mutex
}

// Verify that FakeIDGenerator implements ports.IDGenerator.
var _ ports.IDGenerator = (*FakeIDGenerator)(nil)

// NewFakeIDGenerator creates a new FakeIDGenerator with a predefined sequence of UUIDs.
func NewFakeIDGenerator(ids ...uuid.UUID) *FakeIDGenerator {
	return &FakeIDGenerator{
		sequentialIDs: ids,
		currentIndex:  0,
	}
}

// NewFakeIDGeneratorWithDefaults creates a new FakeIDGenerator with deterministic default UUIDs.
func NewFakeIDGeneratorWithDefaults() *FakeIDGenerator {
	ids := make([]uuid.UUID, 10)
	for i := 1; i <= 10; i++ {
		ids[i-1] = uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i))
	}
	return NewFakeIDGenerator(ids...)
}

// Generate returns the next UUID from the sequence.
// If all UUIDs have been consumed, wraps around to the beginning.
func (f *FakeIDGenerator) Generate() string {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.sequentialIDs) == 0 {
		return uuid.Nil.String()
	}

	id := f.sequentialIDs[f.currentIndex%len(f.sequentialIDs)]
	f.currentIndex++
	return id.String()
}

// ResetSequence resets the generator back to the first UUID.
func (f *FakeIDGenerator) ResetSequence() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.currentIndex = 0
}

// CurrentIndex returns the current position in the sequence.
func (f *FakeIDGenerator) CurrentIndex() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.currentIndex
}
