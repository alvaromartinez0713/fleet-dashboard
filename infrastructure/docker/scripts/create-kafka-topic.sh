#!/bin/bash
# ============================================================================
# Kafka Topic Creation Script
# ============================================================================
# This script creates the telemetry.positions topic in Kafka with the required
# configuration for the Fleet Platform.
#
# Configuration:
#   - Topic: telemetry.positions
#   - Partitions: 3 (for parallel processing by vehicle_id)
#   - Replication Factor: 1 (single broker setup)
#   - Retention: 24 hours (86400000 ms)
#   - Cleanup Policy: delete
#
# Usage:
#   docker compose exec kafka bash /app/create-kafka-topic.sh
#
# Note: This script is idempotent - it will not fail if topic already exists
# ============================================================================

# Wait for Kafka to be fully ready
# The healthcheck ensures the broker is up, but we wait a bit more for stability
echo "Waiting for Kafka to be fully ready..."
sleep 15

# Create topic with specified configuration
echo "Creating topic telemetry.positions..."
kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic telemetry.positions \
  --partitions 3 \
  --replication-factor 1 \
  --config cleanup.policy=delete \
  --config retention.ms=86400000 \
  --if-not-exists 2>/dev/null

if [ $? -eq 0 ]; then
  echo "✓ Topic telemetry.positions created or already exists"
else
  echo "✗ Failed to create topic telemetry.positions"
  exit 1
fi

# Verify topic was created successfully
echo "Verifying topic configuration..."
kafka-topics --describe \
  --bootstrap-server localhost:9092 \
  --topic telemetry.positions

echo "✓ Topic creation complete"
