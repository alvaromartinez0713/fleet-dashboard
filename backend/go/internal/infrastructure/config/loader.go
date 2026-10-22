package config

import "os"

// EnvConfigLoader loads configuration from environment variables.
type EnvConfigLoader struct{}

// NewEnvConfigLoader creates a new environment configuration loader.
func NewEnvConfigLoader() *EnvConfigLoader {
	return &EnvConfigLoader{}
}

// Load reads configuration from environment variables.
func (e *EnvConfigLoader) Load() (Config, error) {
	return Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "timescaledb"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "fleet_user"),
			Password: getEnv("DB_PASSWORD", "fleet_password"),
			Name:     getEnv("DB_NAME", "fleet_db"),
			SSLMode:  getEnv("DB_SSLMODE", "prefer"),
		},
	}, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}
