package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// ResilientRepository decorates a PositionRepository with retry logic and circuit breaker protection.
// This is infrastructure logic - not business logic.
//
// Clean Architecture: Infrastructure concerns (retries, circuit breaker) belong in the adapter layer,
// not in the domain or application layers.
type ResilientRepository struct {
	inner              ports.PositionRepository
	circuitBreaker     ports.CircuitBreaker
	maxRetries         int
	baseBackoff        time.Duration
	backoffFactor      float64
	dbOperationTimeout time.Duration
	logger             ports.Logger
}

// ResilientConfig holds retry configuration.
type ResilientConfig struct {
	MaxRetries         int
	BaseBackoff        time.Duration
	BackoffFactor      float64
	DBOperationTimeout time.Duration
}

// NewResilientRepository creates a repository with retry and circuit breaker capabilities.
func NewResilientRepository(
	inner ports.PositionRepository,
	circuitBreaker ports.CircuitBreaker,
	logger ports.Logger,
	cfg ResilientConfig,
) ports.PositionRepository {
	return &ResilientRepository{
		inner:              inner,
		circuitBreaker:     circuitBreaker,
		maxRetries:         cfg.MaxRetries,
		baseBackoff:        cfg.BaseBackoff,
		backoffFactor:      cfg.BackoffFactor,
		dbOperationTimeout: cfg.DBOperationTimeout,
		logger:             logger,
	}
}

// Save persists a position with retry logic and circuit breaker protection.
func (r *ResilientRepository) Save(
	ctx context.Context,
	position *domain.VehiclePosition,
	storedAt time.Time,
) *domain.SaveOutcome {
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		// Create timeout context for this attempt
		opCtx, cancel := context.WithTimeout(ctx, r.dbOperationTimeout)
		
		// Execute with circuit breaker protection
		var outcome *domain.SaveOutcome
		err := r.circuitBreaker.Execute(opCtx, func() error {
			outcome = r.inner.Save(opCtx, position, storedAt)
			if !outcome.Success() {
				return outcome.Error
			}
			return nil
		})
		cancel()

		// Circuit breaker opened
		if err != nil {
			if _, isOpen := err.(ports.CircuitBreakerOpenError); isOpen {
				r.logger.Warn("circuit breaker opened during persistence",
					ports.Field{Key: "event_id", Value: position.EventID().String()},
					ports.Field{Key: "attempt", Value: fmt.Sprintf("%d", attempt+1)},
				)
				// Return error immediately - no retries when circuit is open
				return &domain.SaveOutcome{
					IsNew: false,
					Error: domain.NewAppError("CIRCUIT_BREAKER_OPEN", "circuit breaker is open", err),
				}
			}
		}

		// Success
		if err == nil && outcome.Success() {
			if outcome.WasInserted() {
				r.logger.Info("position persisted successfully",
					ports.Field{Key: "event_id", Value: position.EventID().String()},
				)
			} else if outcome.WasDuplicate() {
				r.logger.Info("position deduplicated",
					ports.Field{Key: "event_id", Value: position.EventID().String()},
				)
			}
			return outcome
		}

		// Error occurred
		if err == nil && !outcome.Success() {
			err = outcome.Error
		}

		// Check if error is recoverable
		appErr, isAppError := err.(*domain.AppError)
		if !isAppError {
			r.logger.Error("unexpected error type from repository",
				ports.Field{Key: "event_id", Value: position.EventID().String()},
				ports.Field{Key: "error", Value: err.Error()},
			)
			return &domain.SaveOutcome{
				IsNew: false,
				Error: domain.NewAppError("UNEXPECTED_ERROR", "unexpected error type", err),
			}
		}

		// Non-recoverable error - fail immediately
		if !appErr.IsRecoverable {
			r.logger.Error("non-recoverable persistence error",
				ports.Field{Key: "event_id", Value: position.EventID().String()},
				ports.Field{Key: "error_code", Value: appErr.Code},
			)
			return outcome
		}

		// Recoverable error - check if we should retry
		if attempt >= r.maxRetries {
			r.logger.Error("persistence failed, retries exhausted",
				ports.Field{Key: "event_id", Value: position.EventID().String()},
				ports.Field{Key: "attempts", Value: fmt.Sprintf("%d", attempt+1)},
				ports.Field{Key: "error_code", Value: appErr.Code},
			)
			return &domain.SaveOutcome{
				IsNew: false,
				Error: domain.NewAppError("RETRIES_EXHAUSTED", "max retries exhausted", appErr),
			}
		}

		// Wait before retrying (exponential backoff)
		backoff := r.calculateBackoff(attempt)
		r.logger.Warn("recoverable error, retrying after backoff",
			ports.Field{Key: "event_id", Value: position.EventID().String()},
			ports.Field{Key: "attempt", Value: fmt.Sprintf("%d/%d", attempt+1, r.maxRetries)},
			ports.Field{Key: "backoff_ms", Value: fmt.Sprintf("%.0f", backoff.Seconds()*1000)},
		)
		time.Sleep(backoff)
	}

	// Should never reach here
	r.logger.Error("unexpected state: exhausted retries without outcome",
		ports.Field{Key: "event_id", Value: position.EventID().String()},
	)
	return &domain.SaveOutcome{
		IsNew: false,
		Error: domain.NewAppError("UNEXPECTED_STATE", "unexpected retry state", nil),
	}
}

// calculateBackoff computes exponential backoff.
func (r *ResilientRepository) calculateBackoff(attempt int) time.Duration {
	multiplier := 1.0
	for i := 0; i < attempt; i++ {
		multiplier *= r.backoffFactor
	}
	return time.Duration(float64(r.baseBackoff) * multiplier)
}

// Close delegates to inner repository.
func (r *ResilientRepository) Close() error {
	return r.inner.Close()
}

var _ ports.PositionRepository = (*ResilientRepository)(nil)
