package ports

import "time"

// Config holds the configuration for the worker service.
// Configuration is loaded exclusively from environment variables.
type Config struct {
	Worker struct {
		ShutdownTimeout time.Duration // env: WORKER_SHUTDOWN_TIMEOUT, default: 30s, range: 1-300s
	}
	Logging struct {
		Level  string // env: LOG_LEVEL, default: "info"
		Format string // env: LOG_FORMAT, default: "json"
	}
}

// ConfigLoader defines the interface for loading configuration from environment variables.
type ConfigLoader interface {
	Load() (Config, error)
}

// Validator defines the interface for validating configuration values.
type Validator interface {
	Validate(cfg Config) []ValidationError
}

// ValidationError represents a single configuration validation error.
// The error identifies the field and provides a message without including the field value.
type ValidationError struct {
	Field   string // Name of the configuration field that failed validation
	Rule    string // The validation rule that was violated
	Message string // Human-readable error message without the field value
}
