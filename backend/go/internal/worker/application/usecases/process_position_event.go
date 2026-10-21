package usecases

import (
	"context"

	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// ProcessPositionEvent orchestrates the business logic for processing a position event.
//
// Responsibilities (Clean Architecture - Application Layer):
// - Coordinate domain operations
// - Delegate persistence to repository
//
// Returns the persisted entity or domain error.
// Infrastructure concerns (parsing, retries, circuit breaker, dead-letter) are handled by adapters.
type ProcessPositionEvent struct {
	repository ports.PositionRepository
	clock      ports.Clock
}

// NewProcessPositionEvent constructs the use case.
func NewProcessPositionEvent(
	repository ports.PositionRepository,
	clock ports.Clock,
) *ProcessPositionEvent {
	return &ProcessPositionEvent{
		repository: repository,
		clock:      clock,
	}
}

// Execute processes a validated position entity.
//
// Flow:
// 1. Receive validated entity (parsing/validation done by adapter)
// 2. Persist entity via repository port
// 3. Return persisted entity
//
// Infrastructure concerns (Kafka offset commit, dead-letter, circuit breaker)
// are NOT the responsibility of this use case - they belong to the adapter layer.
func (uc *ProcessPositionEvent) Execute(
	ctx context.Context,
	position *domain.VehiclePosition,
) (*domain.VehiclePosition, error) {
	// Persist via repository (infrastructure)
	// Repository handles retries, circuit breaker, idempotency
	storedAt := uc.clock.Now()
	outcome := uc.repository.Save(ctx, position, storedAt)
	if !outcome.Success() {
		// Persistence error - return it
		// The adapter layer will decide what to do (retry, dead-letter, etc.)
		return nil, outcome.Error
	}

	// Return persisted entity
	return position, nil
}

// Verify that ProcessPositionEvent implements ports.PositionProcessor
var _ ports.PositionProcessor = (*ProcessPositionEvent)(nil)
