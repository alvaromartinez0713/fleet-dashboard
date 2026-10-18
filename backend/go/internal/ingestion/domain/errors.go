package domain

import "fmt"

// AppError is the base error type for the api domain layer.
type AppError struct {
	Code          string
	Message       string
	Inner         error
	IsRecoverable bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Inner)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Inner
}

// NewAppError creates a new non-recoverable AppError.
func NewAppError(code, message string, inner error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		Inner:         inner,
		IsRecoverable: false,
	}
}

// NewRecoverableError creates a new recoverable AppError.
func NewRecoverableError(code, message string, inner error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		Inner:         inner,
		IsRecoverable: true,
	}
}

// InvalidGeoPositionError indicates that latitude or longitude values are out of valid bounds.
type InvalidGeoPositionError struct {
	*AppError
}

// NewInvalidGeoPositionError creates a new InvalidGeoPositionError.
func NewInvalidGeoPositionError(message string) *InvalidGeoPositionError {
	return &InvalidGeoPositionError{
		AppError: NewAppError("INVALID_GEO_POSITION", message, nil),
	}
}
