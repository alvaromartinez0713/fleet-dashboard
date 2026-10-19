# Main Makefile for backend Go services

.PHONY: help build test test-ingestion test-ingestion-fast test-ingestion-integration test-worker verify clean clean-tests fmt vet

# Default target
help:
	@echo "Backend Go - Available Targets:"
	@echo ""
	@echo "Build:"
	@echo "  make build              - Build all services"
	@echo "  make build-api          - Build API service"
	@echo "  make build-worker       - Build Worker service"
	@echo ""
	@echo "Test:"
	@echo "  make test               - Run all tests (ingestion + worker)"
	@echo "  make test-ingestion     - Run all ingestion tests (fast + integration)"
	@echo "  make test-ingestion-fast - Run Suite Fast (fast tests, no Docker)"
	@echo "  make test-ingestion-integration - Run Suite Integration (Docker required)"
	@echo "  make test-worker        - Run worker tests"
	@echo ""
	@echo "Verify & Clean:"
	@echo "  make verify             - Verify Composer & DI implementation"
	@echo "  make clean              - Clean binaries and artifacts"
	@echo "  make clean-tests        - Clean test binaries"
	@echo ""
	@echo "Development:"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"

# Build targets
build: build-api build-worker

build-api:
	@echo "Building API service..."
	@go build -o bin/api ./cmd/api

build-worker:
	@echo "Building Worker service..."
	@go build -o bin/worker ./cmd/worker

# Test targets
test: test-ingestion test-worker

test-ingestion: test-ingestion-fast test-ingestion-integration

test-ingestion-fast:
	@echo "Running ingestion Suite Fast (fast tests)..."
	@cd ./tests/ingestion && go test -v -timeout 30s ./suite-fast/...

test-ingestion-integration:
	@echo "Running ingestion Suite Integration (integration tests)..."
	@cd ./tests/ingestion && go test -v -tags=integration -timeout 120s ./suite-integration/...

test-worker:
	@echo "Running worker tests..."
	@cd ./tests/worker && go test -v ./suite-fast/... ./suite-integration/...

# Code quality targets
fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

# Clean targets
clean: clean-tests
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@find . -name "*.test" -delete

clean-tests:
	@echo "Cleaning test binaries..."
	@find ./tests -name "*.test" -delete
	@find ./tests -type f -name "*_test" -delete
	@rm -f coverage.out coverage.html

# Verification target
verify:
	@echo "Verifying Composer implementation..."
	@./verify_composer.sh

