//go:build docker

package docker

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComposeDownPreservesVolumes verifies docker compose down preserves volumes
//
// This test validates:
// - docker compose down removes containers but preserves volumes
// - Data in volumes persists after down
// - Containers can be recreated and access previous data
//
// Verifies: RNF-04 (data integrity), RF-05 (persistence)
func TestComposeDownPreservesVolumes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// Insert test data
	insertCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", `INSERT INTO telemetry.positions 
			(captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) 
			VALUES (NOW(), 'VH-PRESERVE', 40.0, -74.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())`,
	)
	_, err = insertCmd.CombinedOutput()
	require.NoError(t, err, "failed to insert test data")

	// ACT: Run docker compose down (without -v flag, preserves volumes)
	downCmd := exec.CommandContext(ctx,
		"docker-compose", "down",
	)
	output, err := downCmd.CombinedOutput()
	require.NoError(t, err, "docker compose down failed: %s", string(output))

	// CLEANUP: Ensure containers are removed
	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// ACT: Start TimescaleDB again
	err = dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to restart timescaledb after down")

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy after restart")

	// ACT & ASSERT: Verify data persists
	queryCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-PRESERVE'",
	)
	output, err = queryCmd.CombinedOutput()
	require.NoError(t, err, "failed to query data after down/up cycle")

	count := string(output)
	assert.Contains(t, count, "1",
		"data should persist after docker compose down (volumes preserved)")
}

// TestComposeDownRemovesVolumes verifies docker compose down -v removes volumes
//
// This test validates:
// - docker compose down -v removes both containers and volumes
// - Data is lost after volume removal
// - Fresh database state when containers restart
//
// Verifies: infrastructure lifecycle management
func TestComposeDownRemovesVolumes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// Insert test data
	insertCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", `INSERT INTO telemetry.positions 
			(captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) 
			VALUES (NOW(), 'VH-REMOVE', 40.0, -74.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())`,
	)
	_, err = insertCmd.CombinedOutput()
	require.NoError(t, err, "failed to insert test data")

	// ACT: Run docker compose down -v (removes volumes)
	downCmd := exec.CommandContext(ctx,
		"docker-compose", "down", "-v",
	)
	output, err := downCmd.CombinedOutput()
	require.NoError(t, err, "docker compose down -v failed: %s", string(output))

	// ACT: Start TimescaleDB again
	err = dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to restart timescaledb after down -v")

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy after restart")

	// ACT & ASSERT: Verify data is gone (fresh database)
	queryCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-REMOVE'",
	)
	output, err = queryCmd.CombinedOutput()
	require.NoError(t, err, "failed to query data after down -v")

	count := string(output)
	assert.Contains(t, count, "0",
		"data should be removed after docker compose down -v")

	// CLEANUP
	_ = dockerComposeDown(context.Background(), "timescaledb")
}

// TestVolumePreservationAfterStop verifies volumes persist through stop → start cycles
//
// This test validates:
// - docker compose stop preserves volumes
// - Data remains accessible after stop
// - Data persists through multiple stop/start cycles
//
// Verifies: RNF-04 (data integrity), container lifecycle management
func TestVolumePreservationAfterStop(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping docker test in short mode (Docker required)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// SETUP: Start TimescaleDB
	err := dockerComposeUp(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb via docker-compose")

	defer func() {
		_ = dockerComposeDown(context.Background(), "timescaledb")
	}()

	// Wait for healthy
	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy")

	// Insert test data
	insertCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", `INSERT INTO telemetry.positions 
			(captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) 
			VALUES (NOW(), 'VH-CYCLE1', 40.0, -74.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())`,
	)
	_, err = insertCmd.CombinedOutput()
	require.NoError(t, err, "failed to insert test data for cycle 1")

	// Cycle 1: Stop and start
	err = dockerComposeStop(ctx, "timescaledb")
	require.NoError(t, err, "failed to stop timescaledb")

	time.Sleep(1 * time.Second)

	err = dockerComposeStart(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb after stop")

	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy after cycle 1")

	// Verify data persists after cycle 1
	queryCmd := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-CYCLE1'",
	)
	output, err := queryCmd.CombinedOutput()
	require.NoError(t, err, "failed to query after cycle 1")
	assert.Contains(t, string(output), "1", "data should persist after cycle 1")

	// Insert more data
	insertCmd2 := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db",
		"-c", `INSERT INTO telemetry.positions 
			(captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) 
			VALUES (NOW(), 'VH-CYCLE2', 41.0, -75.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())`,
	)
	_, err = insertCmd2.CombinedOutput()
	require.NoError(t, err, "failed to insert test data for cycle 2")

	// Cycle 2: Stop and start again
	err = dockerComposeStop(ctx, "timescaledb")
	require.NoError(t, err, "failed to stop timescaledb for cycle 2")

	time.Sleep(1 * time.Second)

	err = dockerComposeStart(ctx, "timescaledb")
	require.NoError(t, err, "failed to start timescaledb after cycle 2 stop")

	err = waitForHealthyWithTimeout(ctx, "fleet-timescaledb", 30*time.Second)
	require.NoError(t, err, "timescaledb failed to become healthy after cycle 2")

	// Verify both datasets persist
	queryCmd2 := exec.CommandContext(ctx,
		"docker", "exec", "fleet-timescaledb",
		"psql", "-U", "fleet_user", "-d", "fleet_db", "-t",
		"-c", "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id IN ('VH-CYCLE1', 'VH-CYCLE2')",
	)
	output, err = queryCmd2.CombinedOutput()
	require.NoError(t, err, "failed to query after cycle 2")
	assert.Contains(t, string(output), "2", "both datasets should persist after cycle 2")
}
