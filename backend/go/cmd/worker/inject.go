package main

import (
	"time"

	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/messaging"
	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/parser"
	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/deadletter"
	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/infrastructure"
	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/persistence"
	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/resilience"
	"github.com/fleet-platform/backend/internal/worker/application/usecases"
	"github.com/fleet-platform/backend/internal/worker/config"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// WorkerStack holds all components.
type WorkerStack struct {
	Runner  *messaging.Runner
	Cleanup func()
}

// InjectDependencies creates and wires all components.
//
// Clean Architecture - Dependency Injection at composition root.
// Separation of concerns: Runner (loop) + ErrorStrategy (decisions).
func InjectDependencies(cfg ports.Config, logger ports.Logger) (*WorkerStack, error) {
	// Infrastructure: clock, dead-letter sink
	clock := infrastructure.NewSystemClock()
	deadLetterSink := deadletter.NewLogSink(logger)

	// Infrastructure: external connections
	dbConn, err := config.NewPostgresqlConnection(
		cfg.Database.DSN,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		time.Duration(cfg.Database.OperationTimeout)*time.Millisecond,
	)
	if err != nil {
		return nil, err
	}

	kafkaQueue, err := config.NewKafkaMessageReader(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.ConsumerGroup,
		cfg.Kafka.BatchMaxSize,
	)
	if err != nil {
		dbConn.Close()
		return nil, err
	}

	// Infrastructure: circuit breaker
	circuitBreaker := resilience.NewCircuitBreaker(
		uint32(cfg.CircuitBreaker.FailureThreshold),
		time.Duration(cfg.CircuitBreaker.RecoveryInterval)*time.Millisecond,
		2, 1, clock,
	)

	// Infrastructure: repository (base + decorated with retries)
	baseRepo := dbConn
	
	repository := persistence.NewResilientRepository(
		baseRepo,
		circuitBreaker,
		logger,
		persistence.ResilientConfig{
			MaxRetries:         cfg.Retry.MaxRetries,
			BaseBackoff:        time.Duration(cfg.Retry.BaseBackoff) * time.Millisecond,
			BackoffFactor:      cfg.Retry.BackoffFactor,
			DBOperationTimeout: time.Duration(cfg.Database.OperationTimeout) * time.Millisecond,
		},
	)

	// Infrastructure: parser
	eventParser := parser.NewPositionEventParser()

	// Infrastructure: error strategy (encapsulates error handling decisions)
	errorStrategy := messaging.NewErrorStrategy(eventParser, deadLetterSink, clock, logger)

	// Application: use case (implements ports.PositionProcessor)
	var processor ports.PositionProcessor = usecases.NewProcessPositionEvent(repository, clock)

	// Infrastructure: runner (orchestrates loop, delegates error handling)
	runner := messaging.NewRunner(
		kafkaQueue,
		eventParser,
		processor,
		circuitBreaker,
		errorStrategy,
		clock,
		logger,
		time.Duration(cfg.Worker.ShutdownTimeout)*time.Millisecond,
		5*time.Second,
	)

	return &WorkerStack{
		Runner: runner,
		Cleanup: func() {
			logger.Info("cleanup", ports.BuildFields()...)
			runner.Stop()
			kafkaQueue.Close()
			repository.Close()
		},
	}, nil
}
