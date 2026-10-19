# Docker Infrastructure Tests

This directory contains comprehensive tests for the fleet platform infrastructure services running in Docker containers.

## Test Files

### infrastructure_test.go

Comprehensive infrastructure validation tests for Docker Compose services:

#### Test Cases (10 total)

1. **TestZookeeperHealthCheck** - Verifies Zookeeper starts and health check passes
   - Run: `docker compose up -d zookeeper`
   - Wait for healthy status (max 30s)
   - Assert: container state is "healthy"

2. **TestKafkaHealthCheck** - Verifies Kafka starts and health check passes
   - Run: `docker compose up -d kafka`
   - Wait for healthy status (max 45s)
   - Assert: container state is "healthy"

3. **TestKafkaTopicExists** - Verifies telemetry.positions topic exists with correct configuration
   - Run: `docker compose up -d kafka`
   - Wait kafka healthy
   - List topics: `docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092`
   - Assert: output contains "telemetry.positions"
   - Describe topic: `docker exec fleet-kafka kafka-topics --describe --topic telemetry.positions --bootstrap-server localhost:9092`
   - Assert: 3 partitions, replication factor 1

4. **TestTimescaleDBHealthCheck** - Verifies TimescaleDB starts and health check passes
   - Run: `docker compose up -d timescaledb`
   - Wait for healthy status (max 30s)
   - Assert: container state is "healthy"

5. **TestTimescaleDBSchema** - Verifies schema and table initialization
   - Run: `docker compose up -d timescaledb`
   - Wait healthy
   - Query schemas: `docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "\dn"`
   - Assert: output contains schema "telemetry"
   - Query tables: `docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "\dt telemetry.*"`
   - Assert: output contains table "telemetry.positions"
   - Query hypertables: `docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "SELECT * FROM timescaledb_information.hypertables WHERE hypertable_name='positions'"`
   - Assert: query returns 1 row

6. **TestDNSResolution** - Verifies service-to-service DNS resolution works
   - Run: `docker compose up -d kafka`
   - Wait healthy
   - Ping zookeeper: `docker exec fleet-kafka ping -c 1 zookeeper`
   - Assert: ping succeeds

7. **TestVolumePersistence** - Verifies data persists across stop/start
   - Run: `docker compose up -d timescaledb`
   - Wait healthy
   - Insert test data: `docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "INSERT INTO telemetry.positions ..."`
   - Run: `docker compose stop`
   - Run: `docker compose start`
   - Wait healthy
   - Query: `docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -t -c "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-TEST'"`
   - Assert: count = 1 (data persisted)

8. **TestTimescaleDBPortMapping** - Verifies port mapping works from host
   - Run: `docker compose up -d timescaledb`
   - Wait healthy
   - Connect from host: `psql -h localhost -p 5433 -U fleet_user -d fleet_db -c "SELECT 1"`
   - Assert: query succeeds

9. **TestEnvVariablesApplied** - Verifies environment variables from .env are applied
   - Create temporary .env.test with TIMESCALEDB_PORT=5433
   - Run: `docker compose --env-file .env.test up -d timescaledb`
   - Run: `docker compose --env-file .env.test ps`
   - Assert: port mapping shows "0.0.0.0:5433->5432/tcp"
   - Cleanup: `docker compose --env-file .env.test down`
   - Delete .env.test

### cleanup_test.go

Docker Compose lifecycle management tests:

- **TestComposeDownPreservesVolumes** - Verifies docker compose down preserves volumes
- **TestComposeDownRemovesVolumes** - Verifies docker compose down -v removes volumes
- **TestVolumePreservationAfterStop** - Verifies volumes persist through stop → start cycles

### startup_order_test.go

Service startup and dependency management tests:

- **TestStartupOrderRespectsDependencies** - Verifies depends_on configuration
- **TestStartupOrderDependencyValidation** - Verifies dependencies are resolved correctly
- **TestStartupTimeoutScenario** - Verifies startup timeout handling
- **TestServiceStartupSequence** - Verifies predictable service startup order
- **TestConcurrentServiceStartup** - Verifies parallel startup of independent services
- **TestServiceHealthCheckConfiguration** - Verifies health check configuration
- **TestStartupFailureRecovery** - Verifies service recovery after restart

## Running Tests

### Run all Docker infrastructure tests
```bash
cd backend/go
make test-infrastructure
```

### Run specific test
```bash
cd backend/go
go test -tags=docker -v -run TestZookeeperHealthCheck ./tests/docker/...
```

### Run with short timeout (for CI)
```bash
cd backend/go
go test -tags=docker -v -timeout 180s ./tests/docker/...
```

## Requirements

- Docker installed and running
- Docker Compose installed
- `.env` file configured with infrastructure variables
- See `docker-compose.yml` for service configuration

## Helper Functions

### waitForHealthy(ctx, containerName)
Polls container health status until healthy or timeout

### getContainerHealthState(ctx, containerName)
Returns the health state of a container

### dockerComposeUp(ctx, services...)
Starts services via docker-compose

### dockerComposeDown(ctx, services...)
Stops and removes services

### dockerComposeStop(ctx, services...)
Stops services without removing them

### dockerComposeStart(ctx, services...)
Starts stopped services

## Test Characteristics

- **Parallel execution**: Tests use `t.Parallel()` for independent tests
- **Proper cleanup**: All tests use `defer` to ensure cleanup
- **Deterministic**: Tests use fixed timeouts and health checks
- **Docker-aware**: Tests skip gracefully if Docker is unavailable
- **Isolated**: Each test manages its own service lifecycle

## Verifies

- **RF-01, RF-02, RF-03**: Infrastructure foundation
- **RF-04**: Async messaging infrastructure (Kafka)
- **RF-05, RF-06**: Data persistence infrastructure (TimescaleDB)
- **RNF-03**: Startup performance verification
- **RNF-04**: Data integrity and idempotence
- **RNF-06**: Resiliency (DNS resolution, port mapping)

## Notes

- Tests with `testing.Short()` skip gracefully when `-short` flag is used
- Port mapping test uses TIMESCALEDB_PORT from .env (default 5433)
- Environment variable test creates temporary .env.test file
- All tests clean up via `defer` or explicit cleanup commands
