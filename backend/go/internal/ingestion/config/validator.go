package config

import (
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// ConfigValidator validates configuration fields against their defined rules.
// It implements the ports.Validator interface.
type ConfigValidator struct{}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate checks all configuration fields against their validation rules.
// It returns a slice of ValidationError for each field that fails validation.
// If the returned slice is empty, the configuration is valid.
//
// Validation rules:
//   - HTTP.Port must be in range 1-65535
//   - HTTP.ReadTimeout must be >= 1s
//   - HTTP.WriteTimeout must be >= 1s
//   - HTTP.ShutdownTimeout must be between 1s and 300s
//
// Each ValidationError includes the Field name and a Message describing the failure,
// but intentionally does not include the actual field value.
func (v *ConfigValidator) Validate(cfg ports.Config) []ports.ValidationError {
	var errors []ports.ValidationError

	// Validate HTTP.Port: must be in range 1-65535
	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		errors = append(errors, ports.ValidationError{
			Field:   "HTTP.Port",
			Rule:    "range_1_65535",
			Message: "HTTP port must be in range 1-65535",
		})
	}

	// Validate HTTP.ReadTimeout: must be >= 1s
	if cfg.HTTP.ReadTimeout < 1*time.Second {
		errors = append(errors, ports.ValidationError{
			Field:   "HTTP.ReadTimeout",
			Rule:    "min_1s",
			Message: "HTTP read timeout must be at least 1 second",
		})
	}

	// Validate HTTP.WriteTimeout: must be >= 1s
	if cfg.HTTP.WriteTimeout < 1*time.Second {
		errors = append(errors, ports.ValidationError{
			Field:   "HTTP.WriteTimeout",
			Rule:    "min_1s",
			Message: "HTTP write timeout must be at least 1 second",
		})
	}

	// Validate HTTP.ShutdownTimeout: must be between 1s and 300s
	if cfg.HTTP.ShutdownTimeout < 1*time.Second || cfg.HTTP.ShutdownTimeout > 300*time.Second {
		errors = append(errors, ports.ValidationError{
			Field:   "HTTP.ShutdownTimeout",
			Rule:    "range_1s_300s",
			Message: "HTTP shutdown timeout must be between 1 second and 300 seconds",
		})
	}

	return errors
}
