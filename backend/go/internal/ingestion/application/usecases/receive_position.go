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
//   1. Create VehiclePosition entity (validates invariants, emits event)
//   2. Check CircuitBreaker (if open, return error without publishing)
//   3. Publish to Kafka async (fire-and-forget in goroutine)
//   4. Return entity immediately (HTTP 202)
//
// Publishing is async: errors are logged, not returned to client.
// This allows HTTP to return 202 even if Kafka is temporarily unavailable
// (CircuitBreaker manages the retry strategy).
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

	// 2. Check CircuitBreaker before publishing
	if uc.circuitBreaker.IsOpen("kafka-positions") {
		return nil, domain.NewCircuitBreakerOpenError("kafka-positions")
	}

	// 3. Publish to Kafka async (fire-and-forget)
	go func() {
		// Create Kafka payload
		payload := entity.ToKafkaPayload()

		// Publish via ports.KafkaPublisher (includes retry logic)
		err := uc.kafkaPublisher.Publish(context.Background(), "positions", cmd.VehicleID.String(), payload)
		if err != nil {
			// Record failure with CircuitBreaker
			uc.circuitBreaker.RecordFailure("kafka-positions")
		} else {
			// Record success with CircuitBreaker
			uc.circuitBreaker.RecordSuccess("kafka-positions")
		}
	}()

	// 4. Return entity immediately (HTTP 202)
	return entity, nil
}
