# ============================================================================
# Fleet Platform - Local Development Orchestration
# ============================================================================
# Makefile provides convenient commands for managing Docker Compose stack
# from project root. All targets are PHONY (not file-based).
#
# Usage: make [target]
#        make help          (show all targets)
#        make up            (start stack in background)
#        make down          (stop stack)
#        make logs          (stream logs)
# ============================================================================

.PHONY: help up down logs logs-timescaledb logs-kafka status clean clean-hard restart build

# Default target when 'make' is called without args
.DEFAULT_GOAL := help

# ============================================================================
# HELP
# ============================================================================
help:
	@echo "Fleet Platform - Docker Compose Commands"
	@echo "=========================================="
	@echo ""
	@echo "Startup & Management:"
	@echo "  make up              - Start infrastructure in background"
	@echo "  make down            - Stop all services"
	@echo "  make restart         - Restart all services (no recreation)"
	@echo "  make build           - Rebuild Docker images"
	@echo ""
	@echo "Inspection:"
	@echo "  make status          - Show status of all services"
	@echo "  make logs            - Stream logs from all services"
	@echo "  make logs-kafka      - Stream logs from Kafka only"
	@echo "  make logs-timescaledb - Stream logs from TimescaleDB only"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean           - Stop and remove containers (preserve volumes)"
	@echo "  make clean-hard      - Remove everything including volumes and prune unused"
	@echo ""

# ============================================================================
# STARTUP & MANAGEMENT
# ============================================================================

up:
	@echo "Starting Fleet Platform infrastructure..."
	docker compose up -d
	@echo "✓ Stack started. Services will be healthy in 20-30 seconds."
	@echo "✓ Ingestion API: http://localhost:8080"
	@echo "✓ TimescaleDB: localhost:$$(grep TIMESCALEDB_PORT .env 2>/dev/null | cut -d= -f2 || echo 5432)"

down:
	@echo "Stopping Fleet Platform infrastructure..."
	docker compose down
	@echo "✓ Services stopped. Volumes preserved for next start."

restart:
	@echo "Restarting all services..."
	docker compose restart
	@echo "✓ Services restarted."

build:
	@echo "Building Docker images..."
	docker compose build
	@echo "✓ Build complete."

# ============================================================================
# INSPECTION
# ============================================================================

status:
	@echo "Service Status:"
	@echo "==============="
	docker compose ps

logs:
	@echo "Streaming logs from all services (Ctrl+C to exit)..."
	docker compose logs -f

logs-kafka:
	@echo "Streaming logs from Kafka (Ctrl+C to exit)..."
	docker compose logs -f kafka

logs-timescaledb:
	@echo "Streaming logs from TimescaleDB (Ctrl+C to exit)..."
	docker compose logs -f timescaledb

# ============================================================================
# CLEANUP
# ============================================================================

clean:
	@echo "Stopping and removing containers (volumes preserved)..."
	docker compose down
	@echo "✓ Cleanup complete. Use 'make clean-hard' to remove volumes."

clean-hard:
	@echo "WARNING: This will remove all containers, volumes, and prune unused resources."
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Removing everything..."; \
		docker compose down -v; \
		docker system prune -f; \
		echo "✓ Hard cleanup complete. All data removed."; \
	else \
		echo "Aborted."; \
	fi
