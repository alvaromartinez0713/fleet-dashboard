package main

import (
	"fmt"
	"os"

	"github.com/fleet-platform/backend/internal/worker/config"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// Run bootstraps the worker service and blocks until termination.
// Returns the exit code (0 for success, 1 for failure).
func Run() int {
	// Load and validate configuration
	cfg, err := loadAndValidateConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	// Set up logger
	logger, loggerCleanup, err := setupLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 1
	}
	defer loggerCleanup()

	// Compose dependencies
	cleanup, err := composeDependencies(cfg, logger)
	if err != nil {
		logger.Error("failed to compose worker dependencies",
			ports.Field{Key: "error", Value: err.Error()},
		)
		return 1
	}

	// Delegate to worker lifecycle
	return runWorker(cleanup, cfg, logger)
}

// loadAndValidateConfig loads configuration from environment and validates it.
func loadAndValidateConfig() (ports.Config, error) {
	// Load configuration from environment variables
	loader := config.NewEnvConfigLoader()
	cfg, err := loader.Load()
	if err != nil {
		return ports.Config{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	validator := config.NewConfigValidator()
	validationErrors := validator.Validate(cfg)
	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "configuration validation failed:\n")
		for _, ve := range validationErrors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", ve.Field, ve.Message)
		}
		return ports.Config{}, fmt.Errorf("configuration validation failed with %d error(s)", len(validationErrors))
	}

	return cfg, nil
}

// setupLogger creates and initializes the logger.
func setupLogger(cfg ports.Config) (ports.Logger, func(), error) {
	logger, cleanup, err := config.NewLogger(cfg)
	if err != nil {
		return nil, nil, err
	}
	return logger, cleanup, nil
}

// composeDependencies creates and wires all service dependencies.
func composeDependencies(cfg ports.Config, logger ports.Logger) (func(), error) {
	composer := config.NewDependencyComposer(cfg, logger)
	cleanup, err := composer.ComposeWorker()
	if err != nil {
		return nil, err
	}
	return cleanup, nil
}
