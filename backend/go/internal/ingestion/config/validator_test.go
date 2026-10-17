package config

import (
	"testing"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

func TestConfigValidator_Validate_ValidConfig(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 8080
	cfg.HTTP.ReadTimeout = 15 * time.Second
	cfg.HTTP.WriteTimeout = 15 * time.Second
	cfg.HTTP.ShutdownTimeout = 30 * time.Second

	errors := validator.Validate(cfg)

	if len(errors) != 0 {
		t.Errorf("expected no validation errors for valid config, got %d errors: %+v", len(errors), errors)
	}
}

func TestConfigValidator_Validate_HTTPPortOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port too low", 0},
		{"port negative", -1},
		{"port too high", 65536},
		{"port far too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			cfg := ports.Config{}
			cfg.HTTP.Port = tt.port
			cfg.HTTP.ReadTimeout = 15 * time.Second
			cfg.HTTP.WriteTimeout = 15 * time.Second
			cfg.HTTP.ShutdownTimeout = 30 * time.Second

			errors := validator.Validate(cfg)

			if len(errors) == 0 {
				t.Errorf("expected validation error for port %d, got no errors", tt.port)
			}

			foundPortError := false
			for _, err := range errors {
				if err.Field == "HTTP.Port" && err.Rule == "range_1_65535" {
					foundPortError = true
					if err.Message == "" {
						t.Error("expected non-empty error message")
					}
				}
			}

			if !foundPortError {
				t.Errorf("expected HTTP.Port validation error with rule 'range_1_65535', got errors: %+v", errors)
			}
		})
	}
}

func TestConfigValidator_Validate_HTTPPortBoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		valid    bool
	}{
		{"port minimum valid", 1, true},
		{"port maximum valid", 65535, true},
		{"port below minimum", 0, false},
		{"port above maximum", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			cfg := ports.Config{}
			cfg.HTTP.Port = tt.port
			cfg.HTTP.ReadTimeout = 1 * time.Second
			cfg.HTTP.WriteTimeout = 1 * time.Second
			cfg.HTTP.ShutdownTimeout = 1 * time.Second

			errors := validator.Validate(cfg)

			hasPortError := false
			for _, err := range errors {
				if err.Field == "HTTP.Port" {
					hasPortError = true
					break
				}
			}

			if tt.valid && hasPortError {
				t.Errorf("expected no HTTP.Port error for valid port %d, got errors: %+v", tt.port, errors)
			}
			if !tt.valid && !hasPortError {
				t.Errorf("expected HTTP.Port error for invalid port %d, got no error", tt.port)
			}
		})
	}
}

func TestConfigValidator_Validate_ReadTimeoutTooLow(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 8080
	cfg.HTTP.ReadTimeout = 999 * time.Millisecond // Less than 1 second
	cfg.HTTP.WriteTimeout = 15 * time.Second
	cfg.HTTP.ShutdownTimeout = 30 * time.Second

	errors := validator.Validate(cfg)

	foundError := false
	for _, err := range errors {
		if err.Field == "HTTP.ReadTimeout" && err.Rule == "min_1s" {
			foundError = true
			if err.Message == "" {
				t.Error("expected non-empty error message")
			}
		}
	}

	if !foundError {
		t.Errorf("expected HTTP.ReadTimeout validation error, got errors: %+v", errors)
	}
}

func TestConfigValidator_Validate_WriteTimeoutTooLow(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 8080
	cfg.HTTP.ReadTimeout = 15 * time.Second
	cfg.HTTP.WriteTimeout = 500 * time.Millisecond // Less than 1 second
	cfg.HTTP.ShutdownTimeout = 30 * time.Second

	errors := validator.Validate(cfg)

	foundError := false
	for _, err := range errors {
		if err.Field == "HTTP.WriteTimeout" && err.Rule == "min_1s" {
			foundError = true
			if err.Message == "" {
				t.Error("expected non-empty error message")
			}
		}
	}

	if !foundError {
		t.Errorf("expected HTTP.WriteTimeout validation error, got errors: %+v", errors)
	}
}

func TestConfigValidator_Validate_ShutdownTimeoutOutOfRange(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"timeout too low", 999 * time.Millisecond},
		{"timeout zero", 0},
		{"timeout negative", -1 * time.Second},
		{"timeout too high", 301 * time.Second},
		{"timeout far too high", 1000 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			cfg := ports.Config{}
			cfg.HTTP.Port = 8080
			cfg.HTTP.ReadTimeout = 15 * time.Second
			cfg.HTTP.WriteTimeout = 15 * time.Second
			cfg.HTTP.ShutdownTimeout = tt.timeout

			errors := validator.Validate(cfg)

			foundError := false
			for _, err := range errors {
				if err.Field == "HTTP.ShutdownTimeout" && err.Rule == "range_1s_300s" {
					foundError = true
					if err.Message == "" {
						t.Error("expected non-empty error message")
					}
				}
			}

			if !foundError {
				t.Errorf("expected HTTP.ShutdownTimeout validation error for timeout %v, got errors: %+v", tt.timeout, errors)
			}
		})
	}
}

func TestConfigValidator_Validate_ShutdownTimeoutBoundaryValues(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{"timeout minimum valid", 1 * time.Second, true},
		{"timeout maximum valid", 300 * time.Second, true},
		{"timeout below minimum", 999 * time.Millisecond, false},
		{"timeout above maximum", 301 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigValidator()
			cfg := ports.Config{}
			cfg.HTTP.Port = 8080
			cfg.HTTP.ReadTimeout = 1 * time.Second
			cfg.HTTP.WriteTimeout = 1 * time.Second
			cfg.HTTP.ShutdownTimeout = tt.timeout

			errors := validator.Validate(cfg)

			hasShutdownError := false
			for _, err := range errors {
				if err.Field == "HTTP.ShutdownTimeout" {
					hasShutdownError = true
					break
				}
			}

			if tt.valid && hasShutdownError {
				t.Errorf("expected no HTTP.ShutdownTimeout error for valid timeout %v, got errors: %+v", tt.timeout, errors)
			}
			if !tt.valid && !hasShutdownError {
				t.Errorf("expected HTTP.ShutdownTimeout error for invalid timeout %v, got no error", tt.timeout)
			}
		})
	}
}

func TestConfigValidator_Validate_MultipleErrors(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 0                              // Invalid
	cfg.HTTP.ReadTimeout = 500 * time.Millisecond  // Invalid
	cfg.HTTP.WriteTimeout = 100 * time.Millisecond // Invalid
	cfg.HTTP.ShutdownTimeout = 400 * time.Second   // Invalid

	errors := validator.Validate(cfg)

	if len(errors) != 4 {
		t.Errorf("expected 4 validation errors, got %d: %+v", len(errors), errors)
	}

	expectedFields := map[string]bool{
		"HTTP.Port":            false,
		"HTTP.ReadTimeout":     false,
		"HTTP.WriteTimeout":    false,
		"HTTP.ShutdownTimeout": false,
	}

	for _, err := range errors {
		if _, exists := expectedFields[err.Field]; exists {
			expectedFields[err.Field] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("expected validation error for field %s, but it was not found", field)
		}
	}
}

func TestConfigValidator_Validate_ErrorDoesNotIncludeFieldValue(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 99999 // Invalid value
	cfg.HTTP.ReadTimeout = 1 * time.Second
	cfg.HTTP.WriteTimeout = 1 * time.Second
	cfg.HTTP.ShutdownTimeout = 1 * time.Second

	errors := validator.Validate(cfg)

	for _, err := range errors {
		// Verify that the error message doesn't contain the actual port value
		if err.Field == "HTTP.Port" {
			// The message should not contain "99999"
			if containsDigitSequence(err.Message, "99999") {
				t.Errorf("error message should not include the actual field value, got: %s", err.Message)
			}
		}
	}
}

func TestConfigValidator_Validate_MinimumViableConfig(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 1
	cfg.HTTP.ReadTimeout = 1 * time.Second
	cfg.HTTP.WriteTimeout = 1 * time.Second
	cfg.HTTP.ShutdownTimeout = 1 * time.Second

	errors := validator.Validate(cfg)

	if len(errors) != 0 {
		t.Errorf("expected no errors for minimum viable config, got %d errors: %+v", len(errors), errors)
	}
}

func TestConfigValidator_Validate_MaximumViableConfig(t *testing.T) {
	validator := NewConfigValidator()
	cfg := ports.Config{}
	cfg.HTTP.Port = 65535
	cfg.HTTP.ReadTimeout = 1000 * time.Second
	cfg.HTTP.WriteTimeout = 1000 * time.Second
	cfg.HTTP.ShutdownTimeout = 300 * time.Second

	errors := validator.Validate(cfg)

	if len(errors) != 0 {
		t.Errorf("expected no errors for maximum viable config, got %d errors: %+v", len(errors), errors)
	}
}

// Helper function to check if a string contains a specific digit sequence
func containsDigitSequence(s, seq string) bool {
	return len(seq) > 0 && len(s) >= len(seq) && stringContains(s, seq)
}

func stringContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
