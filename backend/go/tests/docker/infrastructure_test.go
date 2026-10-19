//go:build docker

package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZookeeperHealthCheck verifies Zookeeper starts and health check passes
//
// This test validates:
// - Zookeeper container can be started via docker-compose
// - Health check passes within timeout
// - Container state is "healthy"
//
// Verifies: RF-01, RF-02, RF-03 (infrastructure foundation)
func TestZookeeperHealthCheck(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// SETUP: Start Zookeeper container
	err := dockerComposeUp(ctx, "zookeeper")
	require.NoError(t, err, "failed to start zookeeper via docker-compose")

	// CLEANUP: Always stop container
	defer func() {
		_ = dockerComposeDown(context.Background(), "zookeeper")
	}()

	// ACT: Wait for healthy status
	err = waitForHealthy(ctx, "fleet-zookeeper")
	require.NoError(t, err, "zookeeper failed to become healthy")

	// ASSERT: Verify container health state
	state, err := getContainerHealthState(ctx, "fleet-zookeeper")
	require.NoError(t, err, "failed to get container health state")
	assert.Equal(t, "healthy", state, "container should be in healthy state")
}

// TestKafkaHealthCheck verifies Kafka starts and health check passes
//
// This test validates:
// - Kafka container can be started via docker-compose
// - Health check passes within timeout (depends on zookeeper)
// - Container state is "healthy"
//
// Verifies: RF-04 (async messaging infrastructure)
func TestKafkaHealthCheck(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Start Kafka (also starts Zookeeper via depends_on)
	err := dockerComposeUp(ctx, "kafka")
	require.NoError(t, err, "failed to start kafka via docker-compose")

	// CLEANUP: Always stop container
	defer func() {
		_ = dockerComposeDown(context.Background(), "kafka")
	}()

	// ACT: Wait for healthy status (45s timeout)
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka failed to become healthy within 45s")

	// ASSERT: Verify container health state
	state, err := getContainerHealthState(ctx, "fleet-kafka")
	require.NoError(t, err, "failed to get container health state")
	assert.Equal(t, "healthy", state, "kafka container should be in healthy state")
}

// TestKafkaTopicExists verifies telemetry.positions topic exists with correct configuration
//
// This test validates:
// - Kafka container is healthy
// - telemetry.positions topic exists
// - Topic has 3 partitions
// - Topic has replication factor 1
//
// Verifies: RF-04 (async messaging), RF-03 (event schema)
func TestKafkaTopicExists(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Start Kafka
	err := dockerComposeUp(ctx, "kafka")
	require.NoError(t, err, "failed to start kafka via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "kafka")
	}()

	// Wait for Kafka to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka failed to become healthy")

	// ACT & ASSERT: List topics
	listCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-kafka",
		"kafka-topics", "--list", "--bootstrap-server", "localhost:9092",
	)
	output, err := listCmd.CombinedOutput()
	require.NoError(t, err, "failed to list kafka topics: %s", string(output))

	topicsList := string(output)
	assert.Contains(t, topicsList, "telemetry.positions",
		"telemetry.positions topic should exist in Kafka")

	// ACT & ASSERT: Describe topic to verify configuration
	describeCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-kafka",
		"kafka-topics", "--describe",
		"--topic", "telemetry.positions",
		"--bootstrap-server", "localhost:9092",
	)
	output, err = describeCmd.CombinedOutput()
	require.NoError(t, err, "failed to describe kafka topic: %s", string(output))

	topicDesc := string(output)

	// Verify partition count: should have 3 partitions
	// Output format includes "Partition: 0", "Partition: 1", "Partition: 2"
	partitionCount := strings.Count(topicDesc, "Partition:")
	assert.Equal(t, 3, partitionCount,
		"telemetry.positions should have exactly 3 partitions")

	// Verify replication factor is 1
	// Output format includes "Isr: 1" (In-Sync Replicas)
	assert.Contains(t, topicDesc, "Isr: 1",
		"telemetry.positions should have replication factor 1")
}

// TestTimescaleDBHealthCheck verifies TimescaleDB starts and health check passes
//
// This test validates:
// - TimescaleDB container can be started
// - Health check passes within timeout
// - Container state is "healthy"
//
// Verifies: RF-05 (data persistence infrastructure)
func TestTimescaleDBHealthCheck(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	// CLEANUP: Always stop container
	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// ACT: Wait for healthy status (30s timeout)
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy within 30s")

	// ASSERT: Verify container health state
	state, err := getContainerHealthState(ctx, "fleet-timescaledb")
	require.NoError(t, err, "failed to get container health state")
	assert.Equal(t, "healthy", state, "timescaledb container should be in healthy state")
}

// TestTimescaleDBSchema verifies schema and table initialization
//
// This test validates:
// - TimescaleDB is healthy
// - telemetry schema exists
// - telemetry.positions table exists
// - telemetry.positions is a hypertable
//
// Verifies: RF-05 (data persistence), RF-06 (vehicle state queries)
func TestTimescaleDBSchema(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// Wait for TimescaleDB to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// ACT & ASSERT: Query schemas
	schemasCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", "\\dn",
	)
	output, err := schemasCmd.CombinedOutput()
	require.NoError(t, err, "failed to list schemas: %s", string(output))

	schemasOutput := string(output)
	assert.Contains(t, schemasOutput, "telemetry",
		"telemetry schema should exist")

	// ACT & ASSERT: Query tables
	tablesCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", "\\dt telemetry.*",
	)
	output, err = tablesCmd.CombinedOutput()
	require.NoError(t, err, "failed to list tables: %s", string(output))

	tablesOutput := string(output)
	assert.Contains(t, tablesOutput, "positions",
		"telemetry.positions table should exist")

	// ACT & ASSERT: Verify hypertable
	hypertableCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT hypertable_name FROM timescaledb_information.hypertables WHERE hypertable_name='positions'",
	)
	output, err = hypertableCmd.CombinedOutput()
	require.NoError(t, err, "failed to query hypertables: %s", string(output))

	hypertableOutput := strings.TrimSpace(string(output))
	assert.Equal(t, "positions", hypertableOutput,
		"telemetry.positions should be a hypertable")
}

// TestDNSResolution verifies service-to-service DNS resolution works
//
// This test validates:
// - Services can resolve each other by DNS name within docker-compose network
// - Kafka can resolve zookeeper hostname
// - Network connectivity is functional
//
// Verifies: RNF-06 (resiliency), network infrastructure
func TestDNSResolution(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Start Kafka (which depends on Zookeeper)
	err := dockerComposeUp(ctx, "kafka")
	require.NoError(t, err, "failed to start kafka via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "kafka")
	}()

	// Wait for Kafka to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka failed to become healthy")

	// ACT & ASSERT: Ping zookeeper from kafka container
	pingCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-kafka",
		"ping", "-c", "1", "zookeeper",
	)
	output, err := pingCmd.CombinedOutput()
	require.NoError(t, err, "kafka should be able to ping zookeeper: %s", string(output))

	pingOutput := string(output)
	assert.Contains(t, pingOutput, "1 packets transmitted",
		"ping should succeed from kafka to zookeeper")
	assert.Contains(t, pingOutput, "1 received",
		"ping response should be received from zookeeper")
}

// TestVolumePersistence verifies data persists across stop/start
//
// This test validates:
// - Data inserted into TimescaleDB is persisted to volume
// - Data survives container stop/start cycle
// - Volume mounts are functional
//
// Verifies: RNF-04 (data integrity), RF-05 (persistence)
func TestVolumePersistence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// Wait for TimescaleDB to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// ACT: Insert test data
	insertCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", `INSERT INTO telemetry.positions 
			(captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) 
			VALUES (NOW(), 'VH-TEST', 40.0, -74.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())`,
	)
	output, err := insertCmd.CombinedOutput()
	require.NoError(t, err, "failed to insert test data: %s", string(output))

	// ACT: Stop container
	err = dockerComposeStop(ctx, "timescaledb")
	require.NoError(t, err, "failed to stop timescaledb")

	// Give it a moment to stop
	time.Sleep(2 * time.Second)

	// ACT: Start container again
	err = dockerComposeStart(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb after stop")

	// Wait for TimescaleDB to be healthy again
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy after restart")

	// ACT & ASSERT: Query data to verify persistence
	queryCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-TEST'",
	)
	output, err = queryCmd.CombinedOutput()
	require.NoError(t, err, "failed to query test data: %s", string(output))

	count := strings.TrimSpace(string(output))
	assert.Equal(t, "1", count,
		"data should persist across container stop/start cycle")
}

// TestTimescaleDBPortMapping verifies port mapping works from host
//
// This test validates:
// - TimescaleDB port is correctly mapped to host
// - PostgreSQL connection is possible from host system
// - Environment variable TIMESCALEDB_PORT is respected
//
// Verifies: RF-05 (data accessibility), infrastructure configuration
func TestTimescaleDBPortMapping(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// Wait for TimescaleDB to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// ACT & ASSERT: Connect from host using psql
	// Note: This assumes TIMESCALEDB_PORT=5433 from .env
	connectCmd := exec.CommandContext(ctx,
		"psql", "-h", "localhost", "-p", "5433",
		"-U", "fleet_user", "-d", "fleet_db",
		"-c", "SELECT 1",
	)
	// Set password for psql
	connectCmd.Env = append(os.Environ(), "PGPASSWORD=fleet_password")

	output, err := connectCmd.CombinedOutput()
	require.NoError(t, err, "failed to connect to timescaledb from host: %s", string(output))

	result := string(output)
	assert.Contains(t, result, "1",
		"psql query should return result from timescaledb")
}

// TestEnvVariablesApplied verifies environment variables from .env are applied
//
// This test validates:
// - Custom environment variables override defaults
// - Port mapping reflects environment variable configuration
// - docker-compose respects .env file configuration
//
// Verifies: infrastructure configuration management
func TestEnvVariablesApplied(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Create temporary .env.test file with custom TIMESCALEDB_PORT
	testEnvFile := ".env.test"
	testEnvContent := `POSTGRES_USER=fleet_user
POSTGRES_PASSWORD=fleet_password
POSTGRES_DB=fleet_db
TIMESCALEDB_PORT=5434
POSTGRES_HOST_AUTH_METHOD=scram-sha-256
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC=telemetry.positions
KAFKA_GROUP_ID=fleet-worker-group
DB_HOST=timescaledb
DB_PORT=5432
DB_USER=fleet_user
DB_PASSWORD=fleet_password
DB_NAME=fleet_db
DB_SSLMODE=disable
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=30s
WORKER_SHUTDOWN_TIMEOUT=30s
LOG_LEVEL=info
LOG_FORMAT=json
`

	err := os.WriteFile(testEnvFile, []byte(testEnvContent), 0644)
	require.NoError(t, err, "failed to create test .env file")

	defer func() {
		_ = os.Remove(testEnvFile)
	}()

	// ACT: Start TimescaleDB with custom env file
	upCmd := exec.CommandContext(ctx,
		"docker-compose", "--env-file", testEnvFile,
		"up", "-d", "timescaledb",
	)
	output, err := upCmd.CombinedOutput()
	require.NoError(t, err, "failed to start with custom env file: %s", string(output))

	// CLEANUP: Stop containers started with custom env
	defer func() {
		downCmd := exec.CommandContext(context.Background(),
			"docker-compose", "--env-file", testEnvFile,
			"down",
		)
		_ = downCmd.Run()
	}()

	// Wait for TimescaleDB to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// ACT: Get docker-compose ps output to verify port mapping
	psCmd := exec.CommandContext(ctx,
		"docker-compose", "--env-file", testEnvFile,
		"ps",
	)
	output, err = psCmd.CombinedOutput()
	require.NoError(t, err, "failed to get docker-compose ps: %s", string(output))

	psOutput := string(output)
	// Verify port mapping shows 5434 (custom port)
	assert.Contains(t, psOutput, "5434",
		"port mapping should reflect custom TIMESCALEDB_PORT=5434")
}

// ============================================================================
// Helper Functions
// ============================================================================

// waitForHealthy polls container health status until healthy or timeout
func waitForHealthy(ctx context.Context, containerName string) error {
	return waitForHealthyWithTimeout(ctx, containerName, 30*time.Second)
}

// waitForHealthyWithTimeout polls container health status with custom timeout
func waitForHealthyWithTimeout(ctx context.Context, containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		state, err := getContainerHealthState(ctx, containerName)
		if err == nil && state == "healthy" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue polling
		}
	}

	return fmt.Errorf("container %s failed to become healthy within %v", containerName, timeout)
}

// getContainerHealthState returns the health state of a container
func getContainerHealthState(ctx context.Context, containerName string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "inspect",
		"--format={{.State.Health.Status}}",
		containerName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	state := strings.TrimSpace(string(output))
	return state, nil
}

// dockerComposeUp starts services via docker-compose
func dockerComposeUp(ctx context.Context, services ...string) error {
	args := []string{"compose", "up", "-d"}
	args = append(args, services...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w: %s", err, string(output))
	}

	return nil
}

// dockerComposeDown stops and removes services
func dockerComposeDown(ctx context.Context, services ...string) error {
	args := []string{"compose", "down"}
	args = append(args, services...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w: %s", err, string(output))
	}

	return nil
}

// dockerComposeStop stops services without removing them
func dockerComposeStop(ctx context.Context, services ...string) error {
	args := []string{"compose", "stop"}
	args = append(args, services...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose stop failed: %w: %s", err, string(output))
	}

	return nil
}

// dockerComposeStart starts stopped services
func dockerComposeStart(ctx context.Context, services ...string) error {
	args := []string{"compose", "start"}
	args = append(args, services...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose start failed: %w: %s", err, string(output))
	}

	return nil
}
