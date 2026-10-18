package infrastructure

import (
	"github.com/google/uuid"
)

// RealIDGenerator implements ports.IDGenerator using google/uuid for UUID v4 generation.
// Thread-safe: uuid.New() is safe for concurrent use.
type RealIDGenerator struct{}

// NewRealIDGenerator creates a new real ID generator.
func NewRealIDGenerator() *RealIDGenerator {
	return &RealIDGenerator{}
}

// Generate returns a new UUID v4 as a string in canonical format.
// Format: 8-4-4-4-12 hex digits separated by hyphens.
// Example: "550e8400-e29b-41d4-a716-446655440000"
func (g *RealIDGenerator) Generate() string {
	return uuid.New().String()
}
