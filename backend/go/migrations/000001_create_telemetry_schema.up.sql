-- Create TimescaleDB extension for time-series data handling
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Create telemetry schema to organize telemetry-related objects
CREATE SCHEMA IF NOT EXISTS telemetry;
