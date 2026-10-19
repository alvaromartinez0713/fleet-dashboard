# Suite Integration - Ingestion Tests

End-to-end integration tests with **real Kafka** (Testcontainers) and **Toxiproxy** for fault injection.

## Status

✅ **Structure Created** — 8 tests stubbed, compilable
⚠️ **Implementation Pending** — Testcontainers integration, Kafka clients, Toxiproxy setup

## Execution

```bash
# Once Testcontainers is added to go.mod:
make test-ingestion-integration    # Run all 8 tests (requires Docker)

# Or directly:
cd backend/go && go test -v -tags=integration -timeout 120s ./tests/ingestion/suite-integration/...

# Skip in CI if Docker not available:
go test -short ./tests/ingestion/suite-integration/...
```

## Build Tag

```go
// +build integration
```

Tests are compiled only with `-tags=integration` flag. Default build excludes them.

## Infrastructure

### Required Containers

- **Zookeeper:** Message broker coordination
- **Kafka:** Real message broker
- **Toxiproxy:** (Optional) Fault injection for resilience tests

### Test Lifecycle

1. **Setup:** Start Zookeeper + Kafka containers, get connection URLs
2. **Test:** Send HTTP requests to server connected to real Kafka
3. **Verify:** Read events from Kafka topic, check deduplication, idempotency
4. **Teardown:** Stop containers, cleanup resources

## Tests (8 total)

| ID | File | Test | Property | Purpose |
|----|------|------|----------|---------|
| 10.1 | kafka_test.go | position_published_to_kafka_with_correct_schema | P2, P12 | Event schema + partitioning |
| 10.2 | resilience_test.go | kafka_unavailable_toxiproxy_returns_503 | P3 | Circuit Breaker opens on Kafka failure |
| 10.3 | resilience_test.go | kafka_timeout_toxiproxy_returns_503 | P3 | Timeout enforced (1s configured) |
| 10.4 | idempotency_test.go | idempotency_window_1h_retention | P1, RNF-04 | Idempotency deduplication |
| 10.5 | rate_limit_test.go | rate_limiter_window_reset_real_clock | P6 | Rate limit window sliding (12/60s) |
| 10.6 | resilience_test.go | circuit_breaker_half_open_after_timeout | P3 | CB recovery: OPEN → HALF_OPEN → CLOSED |
| 10.7 | concurrent_test.go | concurrent_requests_no_race_conditions | RNF-04 | 100 concurrent requests, no races |
| 10.8 | deduplication_test.go | duplicate_events_deduplicated | P1, RNF-04 | Same EventID sent 5x → 1 stored |

## Files

| File | Lines | Status | Purpose |
|------|-------|--------|---------|
| `setup.go` | ~100 | ✅ Created | Testcontainers helpers + stubs |
| `kafka_test.go` | ~75 | ⚠️ Stubbed | Kafka schema + partitioning |
| `resilience_test.go` | ~65 | ⚠️ Stubbed | CB + Toxiproxy fault injection |
| `idempotency_test.go` | ~30 | ⚠️ Stubbed | Idempotency key deduplication |
| `rate_limit_test.go` | ~35 | ⚠️ Stubbed | Rate limiting with real clock |
| `concurrent_test.go` | ~30 | ⚠️ Stubbed | Concurrent request safety |
| `deduplication_test.go` | ~30 | ⚠️ Stubbed | Event deduplication |
| `README.md` | - | ✅ This file | Documentation |

## Implementation TODO

### 1. Add Testcontainers to go.mod

```bash
cd backend/go
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/wait@latest
```

### 2. Implement Kafka Client

Choose one:
- **Shopify/sarama** (pure Go, slow)
- **confluentinc/confluent-kafka-go** (C bindings, fast)
- **segmentio/kafka-go** (pure Go, modern)

Add to go.mod and implement in `setup.go`:
- `CreateKafkaProducer()`
- `CreateKafkaConsumer()`

### 3. Implement Toxiproxy Integration (Optional)

For resilience tests, configure Toxiproxy via REST API:
- Proxy: `http://127.0.0.1:8474/proxies`
- Toxic: `http://127.0.0.1:8474/proxies/<proxy>/toxics`

### 4. Implement Test Helpers

In `setup.go`:
- `SetupIntegrationTest()` — Start containers, return env
- `SendHTTPPost()` — Send request to server, return response
- `ReadEventFromKafka()` — Read + parse Kafka message
- `VerifyKafkaSchema()` — Validate event schema

### 5. Complete Individual Tests

Each test file has TODO comments for:
- Setup: Start containers, create clients
- Act: Send HTTP requests or publish events
- Assert: Verify Kafka state, status codes, etc.

## Constraints

- ⏱️ **Timeout:** 120 seconds total for all 8 tests
- 🐳 **Docker Required:** Testcontainers must have Docker daemon access
- 🚫 **No Mocks:** Real Kafka, real HTTP, real adapters (not fakes)
- ✅ **Black-box:** Enter via HTTP, verify Kafka + response status
- 🔄 **Deterministic:** Use FakeClock where possible; real clock for rate limiter

## Compilation

```bash
# Compiles with integration tests included
go build -tags=integration ./tests/ingestion/suite-integration/...

# Compiles without integration tests (default)
go build ./tests/ingestion/suite-fast/...
```

## Next Steps

1. Add testcontainers to go.mod
2. Implement Kafka producer/consumer in setup.go
3. Implement test helper functions
4. Run: `make test-ingestion-integration`
5. Verify all 8 tests pass in <120 seconds

## Parallel Execution

Testcontainers can run in parallel if each test starts its own containers:

```bash
go test -v -tags=integration -timeout 120s -parallel 4 ./tests/ingestion/suite-integration/...
```

However, for MVP, sequential execution (1 test at a time) is acceptable.

## References

- `testing.md` §5: Suite Integración definition
- `testing.md` §14: Toxiproxy + Resilience testing
- `testing.md` §16: k6 load testing (separate from unit/integration)
- Testcontainers Go: https://golang.testcontainers.org/
- Kafka Client Libraries: https://kafka.apache.org/clients
- Toxiproxy: https://github.com/Shopify/toxiproxy

---

**Status:** ✅ Ready for Testcontainers integration and implementation.
