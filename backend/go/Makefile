.PHONY: build-ingestion build-worker run clean

# Build the ingestion API binary
build-ingestion:
	@echo "Building ingestion API..."
	@mkdir -p bin
	go build -mod=readonly -o bin/ingestion ./cmd/ingestion

# Build the worker binary
build-worker:
	@echo "Building worker..."
	@mkdir -p bin
	go build -mod=readonly -o bin/worker ./cmd/worker

# Start services with Docker Compose
run:
	@echo "Starting services with Docker Compose..."
	docker compose up -d --build --wait

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
