/**
 * SQLiteSchema
 *
 * Defines the SQLite schema for storing GPS positions with backoff retry support.
 * Run migrations on app startup via MigrationRunner.
 */

export const POSITIONS_TABLE_SCHEMA = `
CREATE TABLE IF NOT EXISTS positions (
  -- Primary identification
  event_id TEXT PRIMARY KEY NOT NULL,
  vehicle_id TEXT NOT NULL,
  
  -- GPS coordinates
  latitude REAL NOT NULL,
  longitude REAL NOT NULL,
  accuracy REAL,
  
  -- Temporal tracking
  captured_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  
  -- Request tracking
  trace_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  
  -- Sync status and error handling
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'error', 'validation_error')),
  error_reason TEXT,
  is_validation_error INTEGER NOT NULL DEFAULT 0,
  
  -- Retry management with exponential backoff
  last_sync_attempt TIMESTAMP,
  sync_attempt_count INTEGER NOT NULL DEFAULT 0,
  next_retry_at TIMESTAMP
);
`;

/**
 * Indexes for query optimization
 */
export const POSITIONS_INDEXES = [
  // Index for filtering by vehicle
  `CREATE INDEX IF NOT EXISTS idx_positions_vehicle_id ON positions(vehicle_id);`,

  // Index for status-based queries
  `CREATE INDEX IF NOT EXISTS idx_positions_status ON positions(status);`,

  // Compound index for finding retryable positions
  `CREATE INDEX IF NOT EXISTS idx_positions_retry ON positions(status, next_retry_at);`,

  // Index for ordering by captured time
  `CREATE INDEX IF NOT EXISTS idx_positions_captured_at ON positions(captured_at);`,

  // Unique constraint on idempotency key per vehicle (prevents duplicates)
  `CREATE UNIQUE INDEX IF NOT EXISTS idx_positions_idempotency ON positions(vehicle_id, idempotency_key);`,
];

/**
 * Migration version for versioning schema changes
 */
export const MIGRATION_VERSION = 1;

/**
 * Schema validation queries
 */
export const SCHEMA_VALIDATION = {
  // Count total positions
  countPositions: 'SELECT COUNT(*) as count FROM positions;',

  // Get positions by status for validation
  getPositionsByStatus: 'SELECT status, COUNT(*) as count FROM positions GROUP BY status;',

  // Check if table exists
  tableExists: `
    SELECT name FROM sqlite_master 
    WHERE type='table' AND name='positions';
  `,
};
