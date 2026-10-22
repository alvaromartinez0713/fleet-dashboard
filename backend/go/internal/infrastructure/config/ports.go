package config

// ConfigLoader loads configuration from environment variables.
type ConfigLoader interface {
	Load() (Config, error)
}

// Validator validates a Config structure.
type Validator interface {
	Validate(cfg Config) []ValidationError
}
