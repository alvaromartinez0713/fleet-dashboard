package config

import (
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// DependencyComposer wires together all dependencies for the worker service.
// It is responsible for instantiating and connecting adapters required for
// message consumption and processing.
type DependencyComposer struct {
	cfg    ports.Config
	logger ports.Logger
}

// NewDependencyComposer creates a new DependencyComposer with the provided configuration and logger.
func NewDependencyComposer(cfg ports.Config, logger ports.Logger) *DependencyComposer {
	return &DependencyComposer{
		cfg:    cfg,
		logger: logger,
	}
}

// ComposeWorker wires together the worker dependencies and returns a cleanup function.
//
// NOTE: This is a skeleton implementation for the bootstrap spec.
// The message consumption logic (Kafka consumer, message handlers, processing pipeline)
// will be implemented in the ingestion spec when the worker is connected to Kafka
// and begins processing telemetry events.
//
// Returns:
//   - func(): A cleanup function that releases all acquired resources
//   - error: Any error encountered during composition
func (c *DependencyComposer) ComposeWorker() (func(), error) {
	// For the bootstrap spec, there are no external dependencies to initialize
	// (no Kafka consumer, no database connection pool, etc.)
	// Future specs will add:
	// - Kafka consumer client
	// - PostgreSQL connection pool
	// - Message processing pipeline
	// - Resource cleanup for all of the above

	c.logger.Info("Worker dependencies composed successfully (bootstrap skeleton)")

	// Return a no-op cleanup function for now
	// Future specs will return a function that:
	// - Closes the Kafka consumer
	// - Closes database connections
	// - Releases any other acquired resources
	cleanup := func() {
		c.logger.Info("Worker cleanup called (no resources to release in bootstrap spec)")
	}

	return cleanup, nil
}
