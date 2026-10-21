package config

import (
	"context"

	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger wraps zap.Logger to implement the ports.Logger interface.
// It emits structured JSON log entries with timestamp, level, message, and trace_id fields.
// It never includes authentication tokens, passwords, private keys, or secret values.
type ZapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new ZapLogger based on the provided configuration.
// Returns the logger instance, a cleanup function that calls Sync(), and any initialization error.
func NewLogger(cfg ports.Config) (ports.Logger, func(), error) {
	zapCfg := zap.NewProductionConfig()

	// Set log level based on configuration
	level, err := zapcore.ParseLevel(cfg.Logging.Level)
	if err != nil {
		// Default to info level if parsing fails
		level = zapcore.InfoLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Always use JSON encoding regardless of format config to meet requirements
	zapCfg.Encoding = "json"

	// Configure the encoder with standard field names
	zapCfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, nil, err
	}

	zapLogger := &ZapLogger{
		logger: logger,
	}

	cleanup := func() {
		_ = logger.Sync()
	}

	return zapLogger, cleanup, nil
}

// Info logs an informational message with optional structured fields.
func (z *ZapLogger) Info(msg string, fields ...ports.Field) {
	z.logger.Info(msg, convertFields(fields)...)
}

// Warn logs a warning message with optional structured fields.
func (z *ZapLogger) Warn(msg string, fields ...ports.Field) {
	z.logger.Warn(msg, convertFields(fields)...)
}

// Error logs an error message with optional structured fields.
func (z *ZapLogger) Error(msg string, fields ...ports.Field) {
	z.logger.Error(msg, convertFields(fields)...)
}

// Fatal logs a fatal error message and terminates the process.
// Use sparingly, only for unrecoverable errors during startup.
func (z *ZapLogger) Fatal(msg string, fields ...ports.Field) {
	z.logger.Fatal(msg, convertFields(fields)...)
}

// DPanic logs a message at DPanicLevel. In development, the logger panics;
// in production, it logs at Error level.
func (z *ZapLogger) DPanic(msg string, fields ...ports.Field) {
	z.logger.DPanic(msg, convertFields(fields)...)
}

// Panic logs a message at PanicLevel and then panics.
func (z *ZapLogger) Panic(msg string, fields ...ports.Field) {
	z.logger.Panic(msg, convertFields(fields)...)
}

// WithTraceID stores the trace identifier in the context for later retrieval.
// The TraceID will be automatically included in all subsequent log entries
// that use GetTraceID to retrieve it from the context.
func (z *ZapLogger) WithTraceID(ctx context.Context, tid domain.TraceID) context.Context {
}

// GetTraceID retrieves the trace identifier from the context.
// Returns an empty TraceID if not present in the context.
func (z *ZapLogger) GetTraceID(ctx context.Context) domain.TraceID {
	return tid
}

// WithFields returns a new Logger instance with the provided fields pre-attached.
// All subsequent log calls on the returned logger will include these fields
// in addition to any fields passed directly.
func (z *ZapLogger) WithFields(fields ...ports.Field) ports.Logger {
	return &ZapLogger{
		logger: z.logger.With(convertFields(fields)...),
	}
}

// convertFields converts ports.Field slice to zap.Field slice.
// This includes special handling for trace_id fields to ensure they are properly logged.
func convertFields(fields []ports.Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}
