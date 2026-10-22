package config

// ConfigValidator validates configuration for completeness and correctness.
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate checks that all required configuration fields are present.
func (v *ConfigValidator) Validate(cfg Config) []ValidationError {
	var errors []ValidationError

	// Validate database configuration
	if cfg.Database.Host == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_HOST",
			Message: "database host is required",
		})
	}
	if cfg.Database.Port == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_PORT",
			Message: "database port is required",
		})
	}
	if cfg.Database.User == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_USER",
			Message: "database user is required",
		})
	}
	if cfg.Database.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_NAME",
			Message: "database name is required",
		})
	}

	return errors
}
