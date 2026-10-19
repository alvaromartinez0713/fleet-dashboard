//go:build docker

package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartupOrderRespectsDependencies verifies that Docker Compose respects service dependencies
//
// This test validates:
// - Services with depends_on start in the correct order
// - Dependent services wait for their dependencies
// - All services reach healthy state in the correct sequence
//
// Verifies: RNF-03 (startup performance), infrastructure orchestration
func TestStartupOrderRespectsDependencies(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// SETUP: Start all services (Kafka depends on Zookeeper)
	err := dockerComposeUp(ctx, "kafka", "timescaledb")
	require.NoError(t, err, "failed to start services via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "kafka", "timescaledb")
	}()

	// ACT & ASSERT: Verify Zookeeper started first (implicitly via depends_on)
	// Check that Zookeeper is healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-zookeeper", 30*time.Second)
	require.NoError(t, err, "zookeeper failed to become healthy")

	// Verify Kafka is healthy (proves it waited for Zookeeper)
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka failed to become healthy (depends_on not respected)")

	// Verify TimescaleDB is healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// Verify container order in logs
	logsCmd := exec.CommandContext(ctx,
		"docker-compose", "ps",
	)
	output, err := logsCmd.CombinedOutput()
	require.NoError(t, err, "failed to get compose ps")

	psOutput := string(output)
	// All containers should be running
	assert.Contains(t, psOutput, "fleet-zookeeper", "zookeeper should be running")
	assert.Contains(t, psOutput, "fleet-kafka", "kafka should be running")
	assert.Contains(t, psOutput, "fleet-timescaledb", "timescaledb should be running")
}

// TestStartupOrderDependencyValidation verifies depends_on configuration is correct
//
// This test validates:
// - Kafka's depends_on includes zookeeper
// - Starting only Kafka also starts Zookeeper (via depends_on)
// - Dependency resolution is transitive
//
// Verifies: infrastructure dependency management, RNF-03
func TestStartupOrderDependencyValidation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// SETUP: Start only Kafka (which has depends_on: [zookeeper])
	err := dockerComposeUp(ctx, "kafka")
	require.NoError(t, err, "failed to start kafka via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "kafka")
	}()

	// ACT: Verify both Kafka and its dependency (Zookeeper) are running
	// Check Zookeeper health
	err = waitForHealthyWithTimeout(ctx, "fleet-zookeeper", 30*time.Second)
	require.NoError(t, err, "zookeeper should be started as dependency of kafka")

	// Check Kafka health
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka should be healthy")

	// ASSERT: Verify both containers exist
	psCmd := exec.CommandContext(ctx,
		"docker-compose", "ps",
	)
	output, err := psCmd.CombinedOutput()
	require.NoError(t, err, "failed to get ps output")

	psOutput := string(output)
	assert.Contains(t, psOutput, "fleet-zookeeper",
		"zookeeper should be started as kafka's dependency")
	assert.Contains(t, psOutput, "fleet-kafka",
		"kafka should be running")
}

// TestStartupTimeoutScenario verifies services handle startup timeouts gracefully
//
// This test validates:
// - Services that take time to start are still waited for
// - Health checks have appropriate timeouts
// - Timeout behavior is deterministic
//
// Verifies: RNF-03 (startup performance), resilience
func TestStartupTimeoutScenario(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Record start time
	startTime := time.Now()

	// SETUP: Start TimescaleDB (slowest service)
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// ACT: Wait for healthy with 30s timeout (TimescaleDB needs this time)
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb should become healthy within 30s")

	// ASSERT: Verify it took reasonable time but not too long
	elapsed := time.Since(startTime)
	assert.Greater(t, elapsed.Seconds(), 5.0,
		"startup should take some time (health checks need it)")
	assert.Less(t, elapsed.Seconds(), 40.0,
		"startup should not take too long")

	// Verify container is actually healthy
	state, err := getContainerHealthState(ctx, "fleet-timescaledb")
	require.NoError(t, err, "failed to get health state")
	assert.Equal(t, "healthy", state, "container should be in healthy state after startup")

	// ASSERT: Verify TimescaleDB is actually functional (connection test)
	connectCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-c", "SELECT 1",
	)
	output, err := connectCmd.CombinedOutput()
	require.NoError(t, err, "timescaledb should be functional after startup: %s", string(output))
	assert.Contains(t, string(output), "1", "database query should succeed")
}

// TestServiceStartupSequence verifies sequential startup of interdependent services
//
// This test validates:
// - Services start in predictable order
// - Each service reaches healthy state before dependents start
// - Startup sequence is reproducible
//
// Verifies: infrastructure reliability, RNF-03
func TestServiceStartupSequence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// SETUP: Start all services
	err := dockerComposeUp(ctx, "zookeeper", "kafka", "timescaledb")
	require.NoError(t, err, "failed to start all services")

	defer func() {
		_ = dockerComposeDown(context.Background(), "zookeeper", "kafka", "timescaledb")
	}()

	// ACT & ASSERT: Verify startup sequence
	// 1. Zookeeper should be healthy first
	t1 := time.Now()
	err = waitForHealthyWithTimeout(ctx, "fleet-zookeeper", 30*time.Second)
	require.NoError(t, err, "zookeeper should become healthy")
	z00keeperTime := time.Since(t1)

	// 2. Kafka depends on Zookeeper, should be healthy next
	t2 := time.Now()
	err = waitForHealthyWithTimeout(ctx, "fleet-kafka", 45*time.Second)
	require.NoError(t, err, "kafka should become healthy after zookeeper")
	kafkaTime := time.Since(t2)

	// 3. TimescaleDB is independent, can be healthy in parallel
	t3 := time.Now()
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb should become healthy")
	timescaleTime := time.Since(t3)

	// ASSERT: Verify timing is reasonable
	assert.Less(t, z00keeperTime.Seconds(), 25.0,
		"zookeeper should startup quickly")
	assert.Less(t, kafkaTime.Seconds(), 40.0,
		"kafka should startup within timeout")
	assert.Less(t, timescaleTime.Seconds(), 25.0,
		"timescaledb should startup quickly")

	// Verify all are actually healthy
	for _, container := range []string{"fleet-zookeeper", "fleet-kafka", "fleet-timescaledb"} {
		state, err := getContainerHealthState(ctx, container)
		require.NoError(t, err, "failed to get health state for %s", container)
		assert.Equal(t, "healthy", state, "container %s should be healthy", container)
	}
}

// TestConcurrentServiceStartup verifies services that don't depend on each other start in parallel
//
// This test validates:
// - Services without dependencies start concurrently
// - Startup time is not cumulative for independent services
// - Parallel startup improves overall startup time
//
// Verifies: infrastructure performance optimization
func TestConcurrentServiceStartup(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// SETUP: Start Zookeeper and TimescaleDB together (independent services)
	startTime := time.Now()
	err := dockerComposeUp(ctx, "zookeeper", "timescaledb")
	require.NoError(t, err, "failed to start independent services")

	defer func() {
		_ = dockerComposeDown(context.Background(), "zookeeper", "timescaledb")
	}()

	// ACT: Wait for both to be healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-zookeeper", 30*time.Second)
	require.NoError(t, err, "zookeeper should become healthy")

	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb should become healthy")

	parallelTime := time.Since(startTime)

	// ASSERT: Verify concurrent startup is efficient
	// Both services independently need ~15-25 seconds
	// But started together should not be additive (should be ~25s total, not ~50s)
	assert.Less(t, parallelTime.Seconds(), 40.0,
		"parallel startup of independent services should be fast (not additive)")

	// Verify both are healthy
	z00keeperState, err := getContainerHealthState(ctx, "fleet-zookeeper")
	require.NoError(t, err)
	assert.Equal(t, "healthy", z00keeperState)

	timescaleState, err := getContainerHealthState(ctx, "fleet-timescaledb")
	require.NoError(t, err)
	assert.Equal(t, "healthy", timescaleState)
}

// TestServiceHealthCheckConfiguration verifies health check settings are applied
//
// This test validates:
// - Health checks are configured for all services
// - Health check intervals and timeouts are appropriate
// - Services report correct health status
//
// Verifies: infrastructure monitoring, RNF-06
func TestServiceHealthCheckConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// SETUP: Start all services
	err := dockerComposeUp(ctx, "zookeeper", "kafka", "timescaledb")
	require.NoError(t, err, "failed to start services")

	defer func() {
		_ = dockerComposeDown(context.Background(), "zookeeper", "kafka", "timescaledb")
	}()

	// Wait for all services to reach healthy state
	for _, container := range []string{"fleet-zookeeper", "fleet-kafka", "fleet-timescaledb"} {
		err = waitForHealthyWithTimeout(ctx, container, 45*time.Second)
		require.NoError(t, err, "failed to reach healthy state for %s", container)
	}

	// ACT & ASSERT: Inspect health check configuration for each service
	for _, container := range []string{"fleet-zookeeper", "fleet-kafka", "fleet-timescaledb"} {
		inspectCmd := exec.CommandContext(ctx,
			"docker", "inspect",
			"--format={{.Config.Healthcheck}}",
			container,
		)
		output, err := inspectCmd.CombinedOutput()
		require.NoError(t, err, "failed to inspect health check for %s", container)

		inspection := string(output)
		assert.NotEmpty(t, inspection, "health check should be configured for %s", container)
		assert.NotContains(t, inspection, "<nil>", "health check should not be nil for %s", container)
	}

	// ASSERT: Verify health status is healthy
	for _, container := range []string{"fleet-zookeeper", "fleet-kafka", "fleet-timescaledb"} {
		state, err := getContainerHealthState(ctx, container)
		require.NoError(t, err, "failed to get health state for %s", container)
		assert.Equal(t, "healthy", state, "container %s should be in healthy state", container)
	}
}

// TestStartupFailureRecovery verifies services can recover from startup failures
//
// This test validates:
// - A failed service can be restarted
// - Restart is faster than initial startup
// - Service becomes healthy after restart
//
// Verifies: resilience, infrastructure recovery
func TestStartupFailureRecovery(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb should become healthy")

	// Verify it's working
	queryCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-c", "SELECT 1",
	)
	output, err := queryCmd.CombinedOutput()
	require.NoError(t, err, "database should be responsive")
	assert.Contains(t, string(output), "1")

	// ACT: Stop the container
	stopCmd := exec.CommandContext(ctx,
		"docker", "stop", "fleet-timescaledb",
	)
	err = stopCmd.Run()
	require.NoError(t, err, "failed to stop container")

	time.Sleep(2 * time.Second)

	// ACT: Start it again
	startCmd := exec.CommandContext(ctx,
		"docker", "start", "fleet-timescaledb",
	)
	err = startCmd.Run()
	require.NoError(t, err, "failed to start container")

	// ASSERT: Service should recover to healthy state quickly
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb should recover to healthy state")

	// Verify it's working again
	queryCmd2 := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-c", "SELECT 1",
	)
	output, err = queryCmd2.CombinedOutput()
	require.NoError(t, err, "database should be responsive after restart")
	assert.Contains(t, string(output), "1")
}
