# ============================================================================
# Multi-stage Dockerfile for Fleet Platform Go Services
# ============================================================================
# This Dockerfile builds both the ingestion and worker services as separate
# targets, allowing independent deployment while sharing the base Go image
# and dependency layer.
# ============================================================================

# ----------------------------------------------------------------------------
# Stage 1: Base builder with dependencies
# ----------------------------------------------------------------------------
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files and download dependencies (cached layer)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# ----------------------------------------------------------------------------
# Stage 2: Build ingestion service
# ----------------------------------------------------------------------------
FROM builder AS ingestion-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=readonly -ldflags="-w -s" -o /app/ingestion ./cmd/ingestion

# ----------------------------------------------------------------------------
# Stage 3: Build worker service
# ----------------------------------------------------------------------------
FROM builder AS worker-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=readonly -ldflags="-w -s" -o /app/worker ./cmd/worker

# ----------------------------------------------------------------------------
# Stage 4: Ingestion runtime image
# ----------------------------------------------------------------------------
FROM alpine:3.19 AS ingestion

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S fleet && adduser -S fleet -G fleet

# Copy binary from builder
COPY --from=ingestion-builder /app/ingestion /usr/local/bin/ingestion

# Set user
USER fleet

# Health check (used by docker-compose)
HEALTHCHECK --interval=10s --timeout=3s --retries=3 --start-period=5s \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Expose HTTP port
EXPOSE 8080

# Run the ingestion service
ENTRYPOINT ["/usr/local/bin/ingestion"]

# ----------------------------------------------------------------------------
# Stage 5: Worker runtime image
# ----------------------------------------------------------------------------
FROM alpine:3.19 AS worker

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S fleet && adduser -S fleet -G fleet

# Copy binary from builder
COPY --from=worker-builder /app/worker /usr/local/bin/worker

# Set user
USER fleet

# Run the worker service
ENTRYPOINT ["/usr/local/bin/worker"]
