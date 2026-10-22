package main

import (
	"fmt"
	"os"
	"time"

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

	// Build connection string using postgresql:// URL format (required by golang-migrate)
	connStr := buildConnectionString(cfg)

	// Execute migrations with retry logic
	fmt.Println("Starting database migrations")
	fmt.Printf("Connecting to database at %s:%s\n", cfg.Database.Host, cfg.Database.Port)

	// Retry 3 times with 2 second delay
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := migrations.RunMigrations(connStr, "/app/migrations"); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "Migration attempt %d failed: %v\n", attempt, err)
			if attempt < 3 {
				fmt.Printf("Retrying in 2 seconds...\n")
				time.Sleep(2 * time.Second)
			}
		} else {
			fmt.Println("All migrations completed successfully")
			return 0
		}
	}

	fmt.Fprintf(os.Stderr, "Migration failed after 3 attempts: %v\n", lastErr)
	return 1
}

// buildConnectionString constructs a PostgreSQL DSN for golang-migrate.
// golang-migrate requires a URL with a scheme (postgresql:// or postgres://)
// Format: postgresql://user:password@host:port/dbname?sslmode=prefer
func buildConnectionString(cfg config.Config) string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}
