package ports

import (
	"context"

	"github.com/fleet-platform/backend/internal/worker/domain"
)

// PositionProcessor defines the contract for processing position events.
//
// This is a port (interface) that allows the infrastructure layer (Runner)
// to depend on a contract rather than a concrete implementation.
//
// The use case (application layer) implements this interface.
// The adapter (Runner) depends on this interface.
//
// This respects the Dependency Rule of Clean Architecture:
// Infrastructure → Ports ← Application
type PositionProcessor interface {
	// Execute processes a validated position entity and persists it.
	//
	// Args:
	//   ctx: Context for cancellation and deadlines
	//   position: Validated domain entity (parsing/validation done by caller)
	//
	// Returns:
	//   - Persisted VehiclePosition entity on success
	//   - Domain error (AppError) on persistence failure
	//
	// The caller (infrastructure) is responsible for:
	//   - Parsing raw payload
	//   - Validating and constructing the entity
	//   - Handling errors (dead-letter, commit offset, pause)
	Execute(ctx context.Context, position *domain.VehiclePosition) (*domain.VehiclePosition, error)
}
