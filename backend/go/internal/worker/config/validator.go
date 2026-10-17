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

	return errors
}
