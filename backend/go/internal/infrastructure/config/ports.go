package config

import "time"

// Config holds the configuration for the migrate service.
// This configuration is loaded from environment variables at startup
// and validated before migrations are executed.
type Config struct {
	Database struct {
		// Host is the database host (env: DB_HOST)
		Host string

		// Port is the database port (env: DB_PORT)
		Port string

		// User is the database user (env: DB_USER)
		User string

		// Password is the database password (env: DB_PASSWORD)
		Password string

		// Name is the database name (env: DB_NAME)
		Name string
	}
}

// ConfigLoader loads configuration from environment variables.
type ConfigLoader interface {
	Load() (Config, error)
}

// Validator validates a Config structure.
type Validator interface {
	Validate(cfg Config) []ValidationError
}

// ValidationError represents a single configuration validation error.
type ValidationError struct {
	// Field is the name of the configuration field that failed validation
	Field string

	// Message is a human-readable error message explaining the validation failure
	Message string
}
