package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// TimescaleRepository implements ports.PositionRepository.
// It persists VehiclePosition entities to TimescaleDB with idempotent semantics.
type TimescaleRepository struct {
	db SQLConnection
}

// NewTimescaleRepository creates a repository.
func NewTimescaleRepository(db SQLConnection) (ports.PositionRepository, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &TimescaleRepository{
		db: db,
	}, nil
}

// Save implements ports.PositionRepository.Save
// Receives a domain.VehiclePosition entity and persists it to TimescaleDB.
// Uses the entity's ToPersistenceSQL method to extract SQL parameters.
// Returns a SaveOutcome distinguishing insert from duplicate.
func (r *TimescaleRepository) Save(
	ctx context.Context,
	position *domain.VehiclePosition,
	storedAt time.Time,
) *domain.SaveOutcome {
	const query = `
		INSERT INTO telemetry.positions (
			event_id, vehicle_id, latitude, longitude, accuracy,
			captured_at, received_at, processed_at, trace_id, idempotency_key, stored_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (captured_at, event_id) DO NOTHING
	`

	// Convert entity to SQL parameters
	eventID, vehicleID, latitude, longitude, accuracy, capturedAt, receivedAt, processedAt, traceID, idempotencyKey, storedAtStr := position.ToPersistenceSQL()

	rowsAffected, err := r.db.ExecInsert(ctx, query,
		eventID, vehicleID, latitude, longitude, accuracy,
		capturedAt, receivedAt, processedAt, traceID, idempotencyKey, storedAtStr,
	)
	if err != nil {
		// Wrap in AppError for domain consistency
		appErr := domain.NewAppError(
			"PERSISTENCE_ERROR",
			"failed to insert position",
			err,
		)
		return &domain.SaveOutcome{
			IsNew: false,
			Error: appErr,
		}
	}

	// rowsAffected indicates outcome
	if rowsAffected == 0 {
		// ON CONFLICT triggered - this was a duplicate
		return &domain.SaveOutcome{
			IsNew: false,
			Error: nil,
		}
	}

	// rowsAffected == 1 - new insertion
	return &domain.SaveOutcome{
		IsNew: true,
		Error: nil,
	}
}

// Close closes the underlying database connection.
func (r *TimescaleRepository) Close() error {
	return r.db.Close()
}

var _ ports.PositionRepository = (*TimescaleRepository)(nil)
