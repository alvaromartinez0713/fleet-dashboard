-- Create telemetry.positions aligned with the Telemetry_Worker contract (Req 12).
-- The table is created from scratch (no production data exists), so it already
-- includes event_id with its unique constraint, the contract timestamp columns,
-- and the hypertable time dimension on captured_at.
CREATE TABLE IF NOT EXISTS telemetry.positions (
    id               UUID NOT NULL DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL,
    vehicle_id       VARCHAR(50) NOT NULL,
    latitude         NUMERIC(9, 6) NOT NULL,
    longitude        NUMERIC(9, 6) NOT NULL,
    accuracy         NUMERIC(5, 2),
    captured_at      TIMESTAMPTZ NOT NULL,
    received_at      TIMESTAMPTZ,
    processed_at     TIMESTAMPTZ,
    trace_id         UUID NOT NULL,
    idempotency_key  UUID NOT NULL,
    stored_at        TIMESTAMPTZ NOT NULL
);

-- Unique constraint that enables deduplication via ON CONFLICT DO NOTHING (Req 4, Req 12).
-- TimescaleDB requires any UNIQUE index on a hypertable to include the time/partitioning
-- column, so the index is (captured_at, event_id) with the partition column first.
-- Because event_id is a globally unique UUID v4, this composite still guarantees exactly
-- one row per event_id (a second row with the same event_id cannot exist regardless of
-- captured_at).
CREATE UNIQUE INDEX IF NOT EXISTS positions_event_id_key
    ON telemetry.positions (captured_at, event_id);

-- Convert to hypertable using captured_at as the time dimension (Event_Time_Dimension, Req 12).
SELECT create_hypertable('telemetry.positions', 'captured_at', if_not_exists => TRUE);

-- Index for efficient per-vehicle time-range queries (Req 12).
CREATE INDEX IF NOT EXISTS positions_vehicle_captured_at_idx
    ON telemetry.positions (vehicle_id, captured_at DESC);
