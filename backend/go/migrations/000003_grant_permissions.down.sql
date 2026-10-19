-- Revoke permissions from fleet_user role
-- Reverse the grants applied in the up migration

-- Revoke INSERT and SELECT on telemetry.positions table
REVOKE SELECT, INSERT ON telemetry.positions FROM fleet_user;

-- Revoke INSERT and SELECT on all tables in the telemetry schema
REVOKE SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry FROM fleet_user;

-- Revoke usage on the telemetry schema
REVOKE USAGE ON SCHEMA telemetry FROM fleet_user;
