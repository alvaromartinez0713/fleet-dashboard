package main

import (
	"fmt"
	"os"

	"github.com/fleet-platform/backend/internal/infrastructure/config"
	"github.com/fleet-platform/backend/internal/infrastructure/migrations"
)

// Run executes database migrations against TimescaleDB.
// Returns exit code: 0 for success, 1 for error.
func Run() int {
	// Load configuration from environment
	loader := config.NewEnvConfigLoader()
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		return 1
	}

	// Validate configuration
	validator := config.NewConfigValidator()
	validationErrors := validator.Validate(cfg)
	if len(validationErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Configuration validation failed:\n")
		for _, ve := range validationErrors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", ve.Field, ve.Message)
		}
		return 1
	}

	// Build connection string
	connStr := buildConnectionString(cfg)

	// Execute migrations
	fmt.Println("Starting database migrations")
	if err := migrations.RunMigrations(connStr, "/app/migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		return 1
	}

	fmt.Println("All migrations completed successfully")
	return 0
}

// buildConnectionString constructs a PostgreSQL connection string from configuration.
// Note: Password is included in the connection string; ensure this string is never logged.
func buildConnectionString(cfg config.Config) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}
