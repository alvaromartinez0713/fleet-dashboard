-- Create telemetry.positions table for storing vehicle coordinate events
CREATE TABLE IF NOT EXISTS telemetry.positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id VARCHAR(50) NOT NULL,
    latitude NUMERIC(9, 6) NOT NULL,
    longitude NUMERIC(9, 6) NOT NULL,
    altitude NUMERIC(8, 2),
    speed NUMERIC(7, 2),
    heading NUMERIC(5, 2),
    accuracy NUMERIC(5, 2),
    timestamp TIMESTAMPTZ NOT NULL
);

-- Convert positions table to hypertable on timestamp dimension
-- TimescaleDB optimizes storage and query performance for time-series data
SELECT create_hypertable('telemetry.positions', 'timestamp', if_not_exists => TRUE);

-- Create index on (vehicle_id, timestamp DESC) for efficient vehicle tracking queries
CREATE INDEX ON telemetry.positions (vehicle_id, timestamp DESC);

-- Create index on (timestamp DESC) for efficient time-range queries
CREATE INDEX ON telemetry.positions (timestamp DESC);
