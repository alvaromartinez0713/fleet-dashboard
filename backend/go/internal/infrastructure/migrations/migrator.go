package migrations

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes pending database migrations using golang-migrate.
// It loads migrations from the specified directory, connects to PostgreSQL,
// and applies all pending migrations idempotently.
// ErrNoChange is treated as success (idempotence guarantee).
func RunMigrations(connectionString, migrationsPath string) error {
	log.Printf("Starting database migrations from %s", migrationsPath)

	// Create migrate instance with file source and postgres driver
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connectionString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Execute Up() to apply all pending migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	// ErrNoChange is treated as success (idempotence)
	if err == migrate.ErrNoChange {
		log.Print("No pending migrations to apply")
	} else {
		log.Print("Database migrations completed successfully")
	}

	return nil
}
