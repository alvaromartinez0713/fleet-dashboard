-- Grant permissions to fleet_user role
-- Principle of least privilege: fleet_user can only SELECT and INSERT
-- No DELETE, UPDATE, ALTER, or administrative privileges

-- Grant usage on the telemetry schema
GRANT USAGE ON SCHEMA telemetry TO fleet_user;

-- Grant SELECT and INSERT on all existing tables in the telemetry schema
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry TO fleet_user;

-- Grant SELECT and INSERT on telemetry.positions table
GRANT SELECT, INSERT ON telemetry.positions TO fleet_user;
