package config

import (
	"fmt"
	"time"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// ConfigValidator validates configuration values against defined rules.
// Each validation error includes the field name and a message without the field value.
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate checks all configuration fields against their validation rules.
// Returns a slice of ValidationError, one for each field that fails validation.
// Returns an empty slice if all fields are valid.
// The validator does not expose sensitive values (e.g., DSN) in error messages (security.md).
func (v *ConfigValidator) Validate(cfg ports.Config) []ports.ValidationError {
	var errors []ports.ValidationError

	// Validate Worker.ShutdownTimeout: must be between 1s and 300s
	if cfg.Worker.ShutdownTimeout < 1*time.Second {
		errors = append(errors, ports.ValidationError{
			Field:   "Worker.ShutdownTimeout",
			Rule:    "min_value",
			Message: fmt.Sprintf("must be at least %s", 1*time.Second),
		})
	}
	if cfg.Worker.ShutdownTimeout > 300*time.Second {
		errors = append(errors, ports.ValidationError{
			Field:   "Worker.ShutdownTimeout",
			Rule:    "max_value",
			Message: fmt.Sprintf("must not exceed %s", 300*time.Second),
		})
	}

	// Validate Kafka.Brokers: required
	if len(cfg.Kafka.Brokers) == 0 {
		errors = append(errors, ports.ValidationError{
			Field:   "Kafka.Brokers",
			Rule:    "required",
			Message: "WORKER_KAFKA_BROKERS is required and must not be empty",
		})
	}

	// Validate Kafka.BatchMaxSize: must be in range 1-5000
	if cfg.Kafka.BatchMaxSize < 1 {
		errors = append(errors, ports.ValidationError{
			Field:   "Kafka.BatchMaxSize",
			Rule:    "min_value",
			Message: "must be at least 1",
		})
	}
	if cfg.Kafka.BatchMaxSize > 5000 {
		errors = append(errors, ports.ValidationError{
			Field:   "Kafka.BatchMaxSize",
			Rule:    "max_value",
			Message: "must not exceed 5000",
		})
	}

	// Validate Retry.MaxRetries: must be in range 0-10
	if cfg.Retry.MaxRetries < 0 {
		errors = append(errors, ports.ValidationError{
			Field:   "Retry.MaxRetries",
			Rule:    "min_value",
			Message: "must be at least 0",
		})
	}
	if cfg.Retry.MaxRetries > 10 {
		errors = append(errors, ports.ValidationError{
			Field:   "Retry.MaxRetries",
			Rule:    "max_value",
			Message: "must not exceed 10",
		})
	}

	// Validate CircuitBreaker.FailureThreshold: must be >= 1
	if cfg.CircuitBreaker.FailureThreshold < 1 {
		errors = append(errors, ports.ValidationError{
			Field:   "CircuitBreaker.FailureThreshold",
			Rule:    "min_value",
			Message: "must be at least 1",
		})
	}

	// Validate Database.DSN: required
	if cfg.Database.DSN == "" {
		errors = append(errors, ports.ValidationError{
			Field:   "Database.DSN",
			Rule:    "required",
			Message: "WORKER_DB_DSN is required and must not be empty",
		})
	}

	return errors
}
