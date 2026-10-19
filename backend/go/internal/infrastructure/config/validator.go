package config

// Validator validates migration service configuration.
type ConfigValidator struct{}

// NewConfigValidator creates a new configuration validator.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate checks that all required database configuration fields are present.
// Returns a slice of validation errors. If empty, configuration is valid.
func (v *ConfigValidator) Validate(cfg Config) []ValidationError {
	var errors []ValidationError

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

	if cfg.Database.Password == "" {
		errors = append(errors, ValidationError{
			Field:   "DB_PASSWORD",
			Message: "database password is required",
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
