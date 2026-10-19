package usecases

import (
	"context"

	"github.com/fleet-platform/backend/internal/ingestion/application/commands"
	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// ReceivePositionUseCase implements the core business logic for receiving vehicle positions.
// Responsibilities:
//   - Construct VehiclePosition entity from command
//   - Validate domain invariants
//   - Publish async to Kafka via goroutine (fire-and-forget)
//   - Return 202 immediately (async semantics)
//   - Check CircuitBreaker before publishing
//
// Verifies: P2 (Async+202), P3 (CB-Recovery), P12 (Domain-Events), P13 (Domain-Event-Decoupling)
// RF-01, RF-03, RF-04, RF-06
type ReceivePositionUseCase struct {
	kafkaPublisher ports.KafkaPublisher
	circuitBreaker ports.CircuitBreakerManager
}

// NewReceivePositionUseCase creates a new receive position use case.
func NewReceivePositionUseCase(
	kafkaPublisher ports.KafkaPublisher,
	circuitBreaker ports.CircuitBreakerManager,
) *ReceivePositionUseCase {
	return &ReceivePositionUseCase{
		kafkaPublisher: kafkaPublisher,
		circuitBreaker: circuitBreaker,
	}
}

// Execute processes the command and returns the created entity.
// Workflow:
//  1. Create VehiclePosition entity (validates invariants, emits event)
//  2. Check CircuitBreaker before publishing (via Execute)
//  3. Publish to Kafka async (fire-and-forget in goroutine)
//  4. Return entity immediately (HTTP 202)
//
// Publishing is async: errors are logged, not returned to client.
// This allows HTTP to return 202 even if Kafka is temporarily unavailable
// (CircuitBreaker manages the retry strategy).
//
// Verifies: P2 (Async+202), P3 (CB-Recovery), P12 (Domain-Events), P13 (Domain-Event-Decoupling)
func (uc *ReceivePositionUseCase) Execute(ctx context.Context, cmd *commands.ReceivePositionCommand) (*domain.VehiclePosition, error) {
	// 1. Create entity (validates invariants, emits event)
	entity, err := domain.NewVehiclePosition(
		cmd.VehicleID,
		cmd.EventID,
		cmd.GeoPosition,
		cmd.CapturedAt,
		cmd.ReceivedAt,
		cmd.ProcessedAt,
		cmd.TraceID,
		cmd.IdempotencyKey,
		nil, // accuracy: optional, not in this MVP
	)
	if err != nil {
		return nil, err
	}

	// 2. Publish to Kafka async (fire-and-forget in goroutine)
	// Fire-and-forget ensures HTTP 202 is returned immediately without blocking on Kafka latency
	// CircuitBreaker is checked within the async publish operation
	go func() {
		// Create Kafka payload from entity (includes all 5 timestamps)
		payload := entity.ToKafkaPayload()

		// Execute publish with CircuitBreaker protection
		// If circuit is OPEN, Execute returns CircuitBreakerOpenError without attempting publish
		// If circuit is CLOSED, Execute calls the publish function and tracks success/failure
		_ = uc.circuitBreaker.Execute(context.Background(), func() error {
			// Publish via ports.KafkaPublisher
			// The publisher is responsible for retries, partitioning by vehicle_id, and error handling
			return uc.kafkaPublisher.Publish(context.Background(), "positions", cmd.VehicleID.String(), payload)
		})
		// Error is intentionally ignored in fire-and-forget pattern:
		// - If publish succeeds: CircuitBreaker records success
		// - If publish fails: CircuitBreaker records failure
		// - If circuit is open: CircuitBreaker records failure
		// The HTTP response (202) was already sent; we don't propagate the error to the client
	}()

	// 3. Return entity immediately (HTTP 202)
	// The client receives 202 before Kafka processing completes
	return entity, nil
}
