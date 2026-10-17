package ports

import "time"

// Config holds the configuration for the ingestion service.
// This configuration is loaded from environment variables at startup
// and validated before the service begins accepting requests.
type Config struct {
	HTTP struct {
		// Port is the HTTP server port (env: HTTP_PORT)
		// Valid range: 1-65535, default: 8080
		Port int

		// ReadTimeout is the maximum duration for reading the entire request (env: HTTP_READ_TIMEOUT)
		// Minimum: 1s, default: 15s
		ReadTimeout time.Duration

		// WriteTimeout is the maximum duration before timing out writes of the response (env: HTTP_WRITE_TIMEOUT)
		// Minimum: 1s, default: 15s
		WriteTimeout time.Duration

		// ShutdownTimeout is the maximum duration to wait for graceful shutdown (env: HTTP_SHUTDOWN_TIMEOUT)
		// Valid range: 1-300s, default: 30s
		ShutdownTimeout time.Duration
	}
	Logging struct {
		// Level is the log level (env: LOG_LEVEL)
		// Valid values: "debug", "info", "warn", "error"
		// Default: "info"
		Level string

		// Format is the log format (env: LOG_FORMAT)
		// Valid values: "json", "console"
		// Default: "json"
		Format string
	}
	// Note: Kafka and PostgreSQL configuration fields are not included in this bootstrap spec.
	// They will be added in the ingestion spec when the adapters are implemented.
}

// ConfigLoader loads configuration from environment variables.
// It reads all required configuration values and returns them as a structured Config object.
type ConfigLoader interface {
	Load() (Config, error)
}

// Validator validates a Config structure.
// It checks all fields against their validation rules and returns a list of validation errors.
// If the returned slice is empty, the configuration is valid.
type Validator interface {
	Validate(cfg Config) []ValidationError
}

// ValidationError represents a single configuration validation error.
// It contains information about which field failed validation and why,
// but intentionally does not include the actual field value.
type ValidationError struct {
	// Field is the name of the configuration field that failed validation
	Field string

	// Rule is the validation rule that was violated
	Rule string

	// Message is a human-readable description of the validation failure
	// It does not include the actual field value to avoid logging sensitive data
	Message string
}
