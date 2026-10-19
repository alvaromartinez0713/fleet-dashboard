package ports

// IDGenerator defines the contract for generating unique identifiers.
// Implementations must generate UUID v4 identifiers suitable for event IDs,
// trace IDs, and other unique identification purposes.
// Implementations are expected to be injectable to enable deterministic testing.
type IDGenerator interface {
	// Generate returns a new UUID v4 identifier as a string in hex format (36 characters).
	// The returned value is guaranteed to be a valid UUID v4.
	// Example: "550e8400-e29b-41d4-a716-446655440000"
	Generate() string
}
