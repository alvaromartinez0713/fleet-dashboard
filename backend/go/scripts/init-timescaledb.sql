-- ============================================================================
-- TimescaleDB Initialization for Fleet Telemetry
-- ============================================================================
-- This script runs once when the container starts for the first time.
-- Subsequent starts use existing data from timescaledb-data volume.
--
-- Schema Design:
-- - Single table `telemetry.positions` with all fields (no split into metadata)
-- - Partitioned by `captured_at` (device timestamp, not server timestamp)
-- - Includes 4 timestamps for latency tracking (captured, received, processed, stored)
-- - Includes traceability fields (event_id, trace_id, idempotency_key)
-- ============================================================================

-- Create TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- ============================================================================
-- Schema: telemetry (bounded context organization)
-- ============================================================================
CREATE SCHEMA IF NOT EXISTS telemetry;

-- ============================================================================
-- Table: telemetry.positions
-- ============================================================================
-- Single table design with all position data and traceability metadata.
-- Partitioned by `captured_at` (device timestamp).
--
CREATE TABLE IF NOT EXISTS telemetry.positions (
  -- Timestamp (partitioning column, device time)
  captured_at TIMESTAMPTZ NOT NULL,
  
  -- Vehicle identification
  vehicle_id VARCHAR(50) NOT NULL,
  
  -- Geographic coordinates
  latitude NUMERIC(10, 8) NOT NULL,
  longitude NUMERIC(11, 8) NOT NULL,
  accuracy NUMERIC(10, 2),
  
  -- Latency tracking timestamps
  received_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL,
  stored_at TIMESTAMPTZ DEFAULT NOW(),
  
  -- Traceability and idempotency
  event_id UUID NOT NULL UNIQUE,
  trace_id UUID NOT NULL,
  idempotency_key UUID,
  
  -- Primary key: composite (captured_at, vehicle_id) for partitioning
  PRIMARY KEY (captured_at, vehicle_id)
);

-- ============================================================================
-- Create Hypertable (TimescaleDB partitioning)
-- ============================================================================
-- Partition by `captured_at` with 1-hour chunks
-- This optimizes queries like "positions in last hour" and automatic retention
SELECT create_hypertable(
  'telemetry.positions',
  by_range('captured_at', INTERVAL '1 hour'),
  if_not_exists => TRUE
);

-- ============================================================================
-- Indexes
-- ============================================================================
-- Index for queries by vehicle_id (most common access pattern)
CREATE INDEX IF NOT EXISTS idx_positions_vehicle_id
  ON telemetry.positions (vehicle_id, captured_at DESC);

-- Index for idempotency checks (duplicate detection)
CREATE INDEX IF NOT EXISTS idx_positions_event_id
  ON telemetry.positions (event_id);

-- Index for distributed tracing queries
CREATE INDEX IF NOT EXISTS idx_positions_trace_id
  ON telemetry.positions (trace_id);

-- ============================================================================
-- Compression Policy (Optional, for production)
-- ============================================================================
-- Compress chunks older than 7 days to save storage
-- Uncomment when needed:
-- ALTER TABLE telemetry.positions SET (
--   timescaledb.compress,
--   timescaledb.compress_segmentby = 'vehicle_id',
--   timescaledb.compress_orderby = 'captured_at DESC'
-- );
--
-- SELECT add_compression_policy('telemetry.positions', INTERVAL '7 days');

-- ============================================================================
-- Retention Policy (Optional, for production)
-- ============================================================================
-- Drop chunks older than 90 days automatically
-- Uncomment when needed:
-- SELECT add_retention_policy('telemetry.positions', INTERVAL '90 days');

-- ============================================================================
-- Grant Permissions
-- ============================================================================
-- Grant usage on schema and table to fleet_user
GRANT USAGE ON SCHEMA telemetry TO fleet_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON telemetry.positions TO fleet_user;

-- Grant privileges on future tables created in telemetry schema
ALTER DEFAULT PRIVILEGES IN SCHEMA telemetry 
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO fleet_user;

-- ============================================================================
-- Initialization Complete
-- ============================================================================
DO $$
BEGIN
  RAISE NOTICE '✓ TimescaleDB schema initialized successfully';
  RAISE NOTICE '  - Schema: telemetry';
  RAISE NOTICE '  - Table: telemetry.positions (hypertable)';
  RAISE NOTICE '  - Partitioning: captured_at (1-hour chunks)';
  RAISE NOTICE '  - Indexes: vehicle_id, event_id, trace_id';
END $$;
