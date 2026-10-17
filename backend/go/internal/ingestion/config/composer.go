package config

import (
	"context"

	ingestionhttp "github.com/fleet-platform/backend/internal/ingestion/http"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// DependencyComposer wires together all dependencies for the ingestion service.
// It is responsible for instantiating and connecting adapters, handlers, and infrastructure
// components according to the provided configuration.
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

// ComposeAPI wires together the HTTP server and all its dependencies.
// It creates the HealthChecker, HealthHandler, and Router, connecting them together.
// If the construction of any dependency fails, it releases already initialized resources
// and returns an error.
//
// Returns:
//   - ports.HTTPServer: The configured and ready-to-start HTTP server
//   - error: Any error encountered during composition
func (c *DependencyComposer) ComposeAPI() (ports.HTTPServer, error) {
	// Create the health checker
	// For this bootstrap spec, we use a simple always-healthy checker
	// Future specs will add dependency checks (Kafka, PostgreSQL, etc.)
	healthChecker := NewSimpleHealthChecker()

	// Create the HTTP router with all dependencies
	// The router constructor handles middleware registration internally
	router := ingestionhttp.NewRouter(c.cfg, c.logger, healthChecker)

	// No cleanup needed in this bootstrap spec since all dependencies are stateless
	// Future specs with database connections, Kafka clients, etc. will need cleanup logic here

	c.logger.Info("API dependencies composed successfully")

	return router, nil
}

// SimpleHealthChecker is a basic implementation of ports.HealthChecker
// that always reports the service as healthy.
// In future specs, this will be replaced with a checker that verifies
// the availability of Kafka, PostgreSQL, and other required dependencies.
type SimpleHealthChecker struct{}

// NewSimpleHealthChecker creates a new SimpleHealthChecker.
func NewSimpleHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{}
}

// Check always returns (true, nil) indicating the service is healthy.
// This is appropriate for the bootstrap spec where no external dependencies exist yet.
func (s *SimpleHealthChecker) Check(ctx context.Context) (bool, error) {
	// For this bootstrap spec, the service is always healthy
	// Future specs will add checks for:
	// - Kafka connectivity
	// - PostgreSQL connectivity
	// - Any other required dependencies
	return true, nil
}
