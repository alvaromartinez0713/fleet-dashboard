package config

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/persistence"
	"github.com/fleet-platform/backend/internal/worker/domain"
)

// PostgresqlClient implements persistence.SQLConnection ACL using stdlib database/sql with PostgreSQL driver.
// This translates stdlib errors to domain.AppError with recovery classification.
type PostgresqlClient struct {
	db               *sql.DB
	operationTimeout time.Duration
}

// NewPostgresqlClient wraps a stdlib *sql.DB with timeout and error classification.
// Called by config/ to create the ACL implementation.
func NewPostgresqlClient(db *sql.DB, operationTimeout time.Duration) (persistence.SQLConnection, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if operationTimeout <= 0 {
		return nil, errors.New("operationTimeout must be positive")
	}
	return &PostgresqlClient{
		db:               db,
		operationTimeout: operationTimeout,
	}, nil
}

// ExecInsert implements persistence.SQLConnection.ExecInsert.
func (c *PostgresqlClient) ExecInsert(ctx context.Context, query string, args ...interface{}) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
	defer cancel()

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, c.classifyError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, domain.NewRecoverableError("DB_ROWS_AFFECTED", "failed to get rows affected", err)
	}

	return rowsAffected, nil
}

// Close closes the underlying database connection pool.
func (c *PostgresqlClient) Close() error {
	return c.db.Close()
}

// classifyError translates PostgreSQL errors to domain.AppError.
func (c *PostgresqlClient) classifyError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return domain.NewRecoverableError("DB_TIMEOUT", "database operation timeout", err)
	}
	if errors.Is(err, context.Canceled) {
		return domain.NewRecoverableError("DB_CANCELLED", "context cancelled", err)
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") {
		return domain.NewRecoverableError("DB_CONNECTION_REFUSED", "connection refused", err)
	}
	if strings.Contains(errMsg, "connection reset") {
		return domain.NewRecoverableError("DB_CONNECTION_RESET", "connection reset by peer", err)
	}
	if strings.Contains(errMsg, "i/o timeout") {
		return domain.NewRecoverableError("DB_IO_TIMEOUT", "I/O timeout", err)
	}
	if strings.Contains(errMsg, "broken pipe") {
		return domain.NewRecoverableError("DB_BROKEN_PIPE", "broken pipe", err)
	}
	if strings.Contains(errMsg, "too many connections") {
		return domain.NewRecoverableError("DB_POOL_EXHAUSTED", "connection pool exhausted", err)
	}
	if strings.Contains(errMsg, "server closed the connection") {
		return domain.NewRecoverableError("DB_UNAVAILABLE", "database unavailable", err)
	}
	if strings.Contains(errMsg, "deadlock") || strings.Contains(errMsg, "40P01") {
		return domain.NewRecoverableError("DB_DEADLOCK", "deadlock detected", err)
	}
	if strings.Contains(errMsg, "constraint") {
		return domain.NewAppError("DB_CONSTRAINT", "constraint violation", err)
	}
	if strings.Contains(errMsg, "syntax") {
		return domain.NewAppError("DB_SYNTAX", "SQL syntax error", err)
	}

	return domain.NewAppError("DB_ERROR", "database error", err)
}

var _ persistence.SQLConnection = (*PostgresqlClient)(nil)
