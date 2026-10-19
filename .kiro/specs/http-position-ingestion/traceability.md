# Matriz de Trazabilidad: Servicio de Ingesta de Posiciones HTTP

## Control del Documento

**Nombre de la Especificación:** http-position-ingestion  
**Estado:** VIGENTE  
**Última Actualización:** 2024  
**Documentos Relacionados:**
- `requirements.md` - 13 Requisitos Funcionales (RF-01 a RF-13)
- `design.md` - Arquitectura COMPLETO, 14 Propiedades de Corrección, DTOs, Mappers, Manejo de Errores
- `tasks.md` - 17 pruebas Suite Rápida + 8 pruebas Suite Integración

---

## 1. Trazabilidad Requisito → Diseño

Esta tabla mapea cada uno de los 13 Requisitos Funcionales a secciones correspondientes en el documento de diseño.

| # | Requirement | Description | Design Coverage | Architecture Layers | Key Concepts |
|---|---|---|---|---|---|
| RF-01 | HTTP POST Endpoint | Accept position HTTP POST requests at `/api/v1/positions` | Section: "HTTP Layer: DTOs & Mappers", "HTTP Adapter (Handler & Middleware)" | HTTP Adapter (Handler, Router), Application, Domain | PositionRequest DTO, PositionHandler, Router config |
| RF-02 | Contract Validation | Validate required fields (vehicle_id, latitude, longitude, captured_at), types, ranges, formats, sizes | Section: "Domain Layer Foundation", Domain errors: FutureTimestampError, InvalidVehicleIDError, InvalidGeoPositionError | Domain (9 Value Objects), HTTP Adapter (ContentType, validation middleware) | 8 Value Objects with constructor validation, RFC 9457 ProblemDetails |
| RF-03 | Async Kafka Publication | Publish positions to Kafka asynchronously without blocking HTTP response | Section: "Outbound Adapters", "Data Flow: Client to Domain", UseCase async pattern | Application (ReceivePositionUseCase), Outbound Adapter (KafkaPublisher), Ports | Fire-and-forget goroutine, KafkaPublisher interface, partition key = vehicle_id |
| RF-04 | HTTP 202 Response (Fast, Explicit) | Respond with HTTP 202 < 100ms p99; explicit distinction (acceptance ≠ persistence) | Section: "HTTP Layer: DTOs & Mappers" (PositionResponse DTO), "Data Flow", Correctness Property 2 | HTTP Adapter (Handler), Application, Ports | PositionResponse DTO, 202 status, no "stored_at" in response |
| RF-05 | Retries + Error Handling | 3 retries with backoff (100ms, 200ms, 400ms), dead-letter queue, Circuit Breaker | Section: "Outbound Adapters" (KafkaPublisher, CircuitBreaker), Correctness Property 3 | Outbound Adapter (KafkaPublisher, CircuitBreaker), Ports | Exponential backoff, retryable vs non-retryable errors, dead-letter log |
| RF-06 | Distinguish Acceptance from Persistence | 202 confirms acceptance (Kafka enqueue), NOT TimescaleDB persistence | Section: "HTTP Layer: DTOs & Mappers", Correctness Property 2 | HTTP Adapter (Response DTO), Application | PositionResponse no "stored_at", implicit contract clarity |
| RF-07 | 4 Timestamps + Logging + Trace ID | captured_at (client), received_at (HTTP), processed_at (post-validation), stored_at (worker) + trace_id consistency | Section: "Domain Layer Foundation" (Value Objects: CapturedTimestamp, ReceivedTimestamp, ProcessedTimestamp), Correctness Properties 4, 5, 9 | Domain (3 Timestamp VOs, EventID, TraceID), HTTP Adapter (Middleware trace_id), Application | 4 Timestamps in Kafka payload, trace_id constant throughout request lifecycle |
| RF-08 | Logging + Monitoring + No Secrets | JSON logs, trace_id in all logs, no secrets exposed | Section: "Error Handling", "HTTP Layer: DTOs & Mappers" (ErrorMapper), Correctness Properties 8, 9, 14 | HTTP Adapter (Middleware, ErrorHandler), Application, Ports (Logger) | JSON log format, trace_id in context, sanitized error messages |
| RF-09 | OpenAPI 3.1.0 Contract | Explicit OpenAPI specification documenting endpoint, schemas, responses, RFC 9457 | Section: "Optional Tasks" (11.1) | HTTP Adapter (Router documentation), HTTP Layer (DTOs) | openapi.yaml file (not auto-generated) |
| RF-10 | Clean Architecture COMPLETO | Complete separation: Domain, Application, HTTP Adapters, Outbound Adapters, Ports, DTOs, Mappers, Config | Section: "Architecture", "Estructura de Carpetas COMPLETO", "Data Flow" diagrams | All layers: Domain, Application, Adapters (Inbound/Outbound), Ports, HTTP Layer, Config | 10 subdirectories, 6 ports, 4 outbound adapters, 5 middleware, 3 mappers |
| RF-11 | Domain Events | VehiclePosition Entity emits PositionReceivedEvent (eventID, timestamps, traceID), Application consumes to decide Kafka publish | Section: "Domain Events in Design", Correctness Property 12, 13 | Domain (PositionReceivedEvent VO, VehiclePosition Entity), Application (UseCase consumption), Ports | PositionReceivedEvent in domain/events.go, Entity.Events() accessor, no Kafka coupling in Domain |
| RF-12 | RFC 9457 Problem Details | All error responses conform to RFC 9457 (type, title, status, detail, instance), no secrets exposed | Section: "Error Handling", "Error → HTTP Status Mapping Table", Correctness Properties 7, 10, 14 | HTTP Adapter (ErrorMapper, ErrorHandlerMiddleware), HTTP Layer (ProblemDetails DTO), Ports (Logger) | ErrorMapper.ToProblemDetails(), 8 error types mapped, sanitized detail field |
| RF-13 | Rate Limiting (in-memory, 12/min) | In-memory rolling window per vehicle_id, 12 requests/60s default, no Redis | Section: "Outbound Adapters" (RateLimiter), Correctness Property 6 | Outbound Adapter (RateLimiter), HTTP Adapter (RateLimitMiddleware), Ports | RateLimiter implementation (sliding window, sync.Mutex, per-vehicle timestamp lists) |

---

## 2. Trazabilidad Requisito → Tareas

Esta tabla mapea cada Requisito a las tareas que lo implementan.

| Requirement | Implementation Tasks | Test Tasks (Suite Rápida) | Integration Tests (Suite Integración) |
|---|---|---|---|
| RF-01 | 1.1-1.3 (Domain), 2.1-2.2 (Application), 3.1 (Ports), 4.1 (Kafka Adapter), 5.1-5.2 (HTTP DTOs/Mappers), 6.1-6.3 (Handler/Middleware/Router), 7.1 (Composer) | 9.1 (valid_position_returns_202), 9.6 (invalid_content_type_returns_415) | 10.1 (position_published_to_kafka) |
| RF-02 | 1.1-1.9 (9 Domain VOs with validation), 5.1 (PositionRequest DTO), 6.1 (Validation middleware) | 9.2 (future_captured_at_returns_422), 9.3 (missing_vehicle_id_returns_422), 9.4 (latitude_out_of_range_returns_422), 9.5 (longitude_out_of_range_returns_422), 9.16 (malformed_json_returns_400), 9.17 (request_exceeds_max_size_returns_413) | 10.1 (schema validation) |
| RF-03 | 2.2 (UseCase async publish), 4.1 (KafkaPublisher adapter with partition key) | 9.1 (valid_position_returns_202 implies async) | 10.1 (position_published_to_kafka_with_correct_schema) |
| RF-04 | 5.1 (PositionResponse DTO), 5.2 (PositionMapper.FromEntity), 6.2 (Handler returns 202), 6.1 (Middleware latency < 100ms) | 9.1 (valid_position_returns_202), 9.2 (future_captured_at_returns_422 validates 422 < 50ms) | None specific (latency measured in Suite Rápida) |
| RF-05 | 4.1 (KafkaPublisher: 3 retries, exponential backoff, dead-letter log), 4.2 (CircuitBreaker state machine), 6.1 (RateLimitMiddleware guards circuit breaker check) | 9.10 (circuit_breaker_opens_on_repeated_failures), 9.11 (circuit_breaker_half_opens_after_timeout), 9.12 (circuit_breaker_closes_after_successes) | 10.2 (kafka_unavailable_toxiproxy_returns_503), 10.3 (kafka_timeout_toxiproxy_returns_503) |
| RF-06 | 5.1 (PositionResponse DTO no "stored_at"), 2.2 (UseCase returns Entity before Kafka confirms) | N/A (implicit in RF-04 test) | 10.8 (kafka_consumer_confirmation_not_persisted_without_worker) |
| RF-07 | 1.3-1.6 (3 Timestamp VOs + EventID VO), 1.3 (VehiclePosition Entity validates ordering), 3.1 (TraceID interface), 6.1 (TraceIDMiddleware generates/validates trace_id), 8.1 (Object Mother) | 9.1 (response includes trace_id), 9.13 (response_includes_trace_id_header), 9.14 (invalid_trace_id_header_returns_400), 9.15 (latency_timestamps_correct) | 10.4 (idempotency_window_1h_retention) |
| RF-08 | 3.1 (Logger port), 6.1 (Middleware logging), 5.2 (ErrorMapper sanitizes logs) | 9.13 (trace_id in response), 9.14 (invalid_trace_id_header_returns_400) | N/A (logging covered by Suite Rápida) |
| RF-09 | 11.1 (Create openapi.yaml, OpenAPI 3.1.0) | N/A (documentation artifact) | N/A |
| RF-10 | 1.1-1.9 (Domain), 2.1-2.2 (Application), 3.1 (6 Ports), 4.1-4.4 (4 Outbound Adapters), 5.1-5.2 (HTTP DTOs/Mappers), 6.1-6.3 (Handler/5 Middleware/Router), 7.1 (Composer) | All tests implicitly verify architecture (black-box behavior) | All tests verify end-to-end architecture |
| RF-11 | 1.3 (VehiclePosition Entity emits PositionReceivedEvent), 2.2 (ReceivePositionUseCase consumes event, publishes to Kafka) | N/A (event emission implicit in RF-01/RF-03 tests) | N/A (event handling implicit in integration tests) |
| RF-12 | 5.2 (ErrorMapper maps domain errors to RFC 9457), 5.1 (ProblemDetails DTO), 6.1 (ErrorHandlerMiddleware returns RFC 9457 format) | 9.2-9.8 (all validation errors return RFC 9457 status codes), 9.14 (invalid_trace_id_header_returns_400) | N/A (error mapping covered by Suite Rápida) |
| RF-13 | 4.3 (RateLimiter adapter: in-memory sliding window, 12/60s per vehicle_id), 6.1 (RateLimitMiddleware applies rate limit check) | 9.8 (rate_limit_exceeded_returns_429), 9.9 (rate_limit_window_sliding) | 10.5 (rate_limiter_window_reset_real_clock), 10.6 (concurrent_requests_same_vehicle_rate_limit), 10.7 (concurrent_requests_different_vehicles_isolated) |

---

## 3. Mapeo Propiedades de Corrección → Implementación

Esta tabla mapea cada una de las 14 Propiedades de Corrección a:
- Qué requisitos valida
- Qué secciones de diseño la implementan
- Qué tareas la aseguran
- Qué pruebas la verifican

| # | Property | Requirement(s) | Design Section(s) | Implementation Task(s) | Verification Tests (Suite Rápida) | Verification Tests (Suite Integración) |
|---|---|---|---|---|---|---|
| P1 | Rejection of Contract Violations | RF-01.2, RF-02.1-RF-02.9, RF-10, RF-12 | Domain Layer (9 VOs with validation), Error Handling | 1.1-1.9 (VOs), 5.1 (PositionRequest DTO), 5.2 (ErrorMapper), 6.1 (Validation middleware) | 9.2-9.5 (range/format validation), 9.16-9.17 (JSON/size validation), 9.6 (Content-Type), 9.14 (Trace-ID) | 10.1 (schema validation end-to-end) |
| P2 | Asynchronous Publication + 202 Response | RF-01.5, RF-04.1, RF-04.2, RF-10, RF-11 | Data Flow diagram, HTTP Layer (PositionResponse DTO), UseCase (async pattern) | 2.2 (UseCase async publish), 5.1 (PositionResponse DTO), 6.2 (Handler returns 202 before Kafka) | 9.1 (valid_position_returns_202) | 10.8 (202 returned before Kafka confirmation) |
| P3 | Circuit Breaker Recovery | RF-05.4, RF-05.5, RF-05.6, RF-10, RF-12 | Outbound Adapters (CircuitBreaker state machine), Error Handling | 4.2 (CircuitBreaker CLOSED→OPEN→HALF_OPEN→CLOSED transitions), 4.1 (5 failures threshold, 30s recovery timeout) | 9.10 (CLOSED→OPEN after 5 failures), 9.11 (OPEN→HALF_OPEN after 30s), 9.12 (HALF_OPEN→CLOSED after 2 successes) | 10.2-10.3 (Toxiproxy failure injection) |
| P4 | 4 Timestamps Integrity | RF-07.1, RF-07.2, RF-10 | Domain Layer (3 Timestamp VOs), VehiclePosition Entity ordering validation | 1.3-1.6 (CapturedTimestamp, ReceivedTimestamp, ProcessedTimestamp VOs, EventID VO), 1.9 (VehiclePosition Entity), 2.2 (UseCase creates all 4 timestamps) | 9.1 (response includes received_at), 9.15 (latency_timestamps_correct, ordering validated) | 10.1 (Kafka payload includes all 4 timestamps) |
| P5 | Future Timestamp Validation | RF-07.3, RF-07.7, RF-10, RF-12 | Domain Layer (CapturedTimestamp VO rejects future), Error Handling | 1.4 (CapturedTimestamp constructor validates captured_at not in future), 5.2 (ErrorMapper maps FutureTimestampError → 422) | 9.2 (future_captured_at_returns_422), 9.15 (timestamps ordered correctly) | None specific (validation covered by Suite Rápida) |
| P6 | Rate Limiting per VehicleID | RF-10 (implicit), RF-04, RF-12 | Outbound Adapters (RateLimiter sliding window), HTTP Adapter (RateLimitMiddleware) | 4.3 (RateLimiter: 12/60s per vehicle, sliding window), 6.1 (RateLimitMiddleware applies check, extracts vehicle_id) | 9.8 (rate_limit_exceeded_returns_429), 9.9 (rate_limit_window_sliding) | 10.5 (real clock window reset), 10.6 (concurrent same vehicle), 10.7 (concurrent different vehicles isolated) |
| P7 | RFC 9457 Conformance | RF-09.3, RF-10, RF-12 | HTTP Layer (ProblemDetails DTO), Error Handling (ErrorMapper) | 5.1 (ProblemDetails DTO with type/title/status/detail/instance), 5.2 (ErrorMapper returns RFC 9457 format), 6.1 (ErrorHandlerMiddleware sends RFC 9457) | 9.2-9.8 (all error tests verify RFC 9457 format) | None specific (format verified by Suite Rápida) |
| P8 | No Sensitive Information in Errors | RF-04.5, RF-10, RF-12 | Error Handling, ErrorMapper implementation | 5.2 (ErrorMapper sanitizes all error details, no stack traces/SQL/paths), 6.1 (ErrorHandlerMiddleware ensures sanitization) | All error tests (9.2-9.8, 9.14) verify sanitized error messages | All integration tests verify no secrets in error responses |
| P9 | Trace ID Continuity | RF-08.1, RF-10 | HTTP Adapter (TraceIDMiddleware), Domain Events, Error Handling | 6.1 (TraceIDMiddleware generates/validates X-Trace-ID, stores in context), 1.7 (TraceID VO), 5.2 (ErrorMapper includes trace_id in instance URI) | 9.1, 9.13 (response includes trace_id), 9.14 (invalid trace_id returns 400), 9.15 (timestamps reference trace_id) | All integration tests include trace_id in responses |
| P10 | Domain Error Mapping | RF-12.1, RF-12.3, RF-12.4 | Error Handling (Error → HTTP Status Mapping Table), ErrorMapper implementation | 5.2 (ErrorMapper maps FutureTimestampError→422, InvalidVehicleIDError→422, InvalidGeoPositionError→422, InvalidTraceIDError→400, CircuitBreakerOpenError→503, Unknown→500) | All error tests (9.2-9.8, 9.14) verify correct HTTP status mapping | None specific (mapping verified by Suite Rápida) |
| P11 | Error Propagation Without Transformation | RF-12.2 | Application Layer (ReceivePositionUseCase), Error Handling | 2.2 (UseCase captures domain error, logs with trace_id, propagates to HTTP layer without modification) | All error tests (9.2-9.8) implicitly verify error propagation | None specific (error handling verified by Suite Rápida) |
| P12 | Domain Event Emission | RF-11.1, RF-11.2 | Domain Layer (PositionReceivedEvent VO), VehiclePosition Entity | 1.3 (VehiclePosition Entity emits PositionReceivedEvent on creation), 1.10 (PositionReceivedEvent VO in domain/events.go) | 9.1 (entity creation succeeds, event emitted) | 10.1 (event fields verified in Kafka payload) |
| P13 | Domain Event Decoupling | RF-11.5, RF-11.6, RF-11.7 | Domain Layer (PositionReceivedEvent in domain/events.go), Application Layer (UseCase consumption) | 1.10 (PositionReceivedEvent contains no Kafka/HTTP/DTO references), 2.2 (UseCase consumes event, decides to publish Kafka, not Domain decision) | 9.1 (event decoupling implicit in design) | 10.1 (event contents verified to have no infrastructure coupling) |
| P14 | Error Handling No Leaks | RF-12.5 | Error Handling, ErrorMapper implementation | 5.2 (ErrorMapper for 500 errors returns generic "internal-error" without details), 6.1 (ErrorHandlerMiddleware ensures no stack traces/SQL/paths in response) | All error tests verify sanitized 500 responses | None specific (error handling verified by Suite Rápida) |

---

## 4. Trazabilidad Prueba → Requisito

Esta sección lista cada caso de prueba y los requisitos/propiedades que valida.

### Suite Rápida (Pruebas de Comportamiento de Caja Negra, ~15 pruebas)

| Test ID | Test Name | HTTP Scenario | Validates Requirements | Validates Properties | Code Location | Execution Time |
|---|---|---|---|---|---|---|
| 9.1 | valid_position_returns_202 | POST valid position → 202 accepted | RF-01, RF-02, RF-04, RF-07 | P2 (async publication), P4 (timestamps), P9 (trace_id) | tests/ingestion/suite-fast/position_handler_test.go::TestValidPositionReturns202 | <50ms |
| 9.2 | future_captured_at_returns_422 | POST with future captured_at → 422 | RF-02, RF-04, RF-12 | P1 (validation rejection), P5 (future timestamp), P7 (RFC 9457), P10 (error mapping) | tests/ingestion/suite-fast/validation_test.go::TestFutureCapturedAtReturns422 | <50ms |
| 9.3 | missing_vehicle_id_returns_422 | POST without vehicle_id → 422 | RF-02, RF-12 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/validation_test.go::TestMissingVehicleIDReturns422 | <50ms |
| 9.4 | latitude_out_of_range_returns_422 | POST with latitude > 90 or < -90 → 422 | RF-02, RF-12 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/validation_test.go::TestLatitudeOutOfRangeReturns422 | <50ms |
| 9.5 | longitude_out_of_range_returns_422 | POST with longitude > 180 or < -180 → 422 | RF-02, RF-12 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/validation_test.go::TestLongitudeOutOfRangeReturns422 | <50ms |
| 9.6 | invalid_content_type_returns_415 | POST with Content-Type: text/plain → 415 | RF-01, RF-08 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/middleware_test.go::TestInvalidContentTypeReturns415 | <50ms |
| 9.7 | idempotency_key_validation | POST with invalid X-Idempotency-Key (not UUID) → 400 | RF-07 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/middleware_test.go::TestIdempotencyKeyValidation | <50ms |
| 9.8 | rate_limit_exceeded_returns_429 | POST 13 times per vehicle in 60s → 13th returns 429 | RF-13, RF-12 | P6 (rate limiting), P7 (RFC 9457) | tests/ingestion/suite-fast/rate_limit_test.go::TestRateLimitExceededReturns429 | <100ms |
| 9.9 | rate_limit_window_sliding | POST at T=0, T=30, T=70 (window slides) → all succeed | RF-13 | P6 (sliding window rate limiting) | tests/ingestion/suite-fast/rate_limit_test.go::TestRateLimitWindowSliding | <200ms |
| 9.10 | circuit_breaker_opens_on_repeated_failures | Kafka fails 5 times → 6th request returns 503 | RF-05, RF-12 | P3 (CB opens), P7 (RFC 9457) | tests/ingestion/suite-fast/circuit_breaker_test.go::TestCircuitBreakerOpensOnRepeatedFailures | <100ms |
| 9.11 | circuit_breaker_half_opens_after_timeout | CB OPEN → wait 30s → try request → HALF_OPEN → request proceeds | RF-05 | P3 (CB HALF_OPEN recovery), P2 (async publication resumes) | tests/ingestion/suite-fast/circuit_breaker_test.go::TestCircuitBreakerHalfOpensAfterTimeout | <150ms |
| 9.12 | circuit_breaker_closes_after_successes | CB HALF_OPEN → 2 successful requests → CLOSED | RF-05 | P3 (CB recovers to CLOSED) | tests/ingestion/suite-fast/circuit_breaker_test.go::TestCircuitBreakerClosesAfterSuccesses | <100ms |
| 9.13 | response_includes_trace_id_header | Valid POST → response includes X-Trace-ID header | RF-07, RF-08 | P9 (trace_id continuity), P4 (timestamps include trace_id) | tests/ingestion/suite-fast/trace_id_test.go::TestResponseIncludesTraceIDHeader | <50ms |
| 9.14 | invalid_trace_id_header_returns_400 | POST with invalid X-Trace-ID header → 400 | RF-08, RF-12 | P1 (validation rejection), P7 (RFC 9457), P9 (trace_id validation) | tests/ingestion/suite-fast/trace_id_test.go::TestInvalidTraceIDHeaderReturns400 | <50ms |
| 9.15 | latency_timestamps_correct | Valid POST → response received_at correct, order maintained | RF-07, RF-04 | P4 (4 timestamps), P5 (timestamp validation) | tests/ingestion/suite-fast/timestamp_test.go::TestLatencyTimestampsCorrect | <50ms |
| 9.16 | malformed_json_returns_400 | POST with malformed JSON → 400 | RF-02, RF-12 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/validation_test.go::TestMalformedJSONReturns400 | <50ms |
| 9.17 | request_exceeds_max_size_returns_413 | POST with body > 10 KB → 413 | RF-02, RF-12 | P1 (validation rejection), P7 (RFC 9457) | tests/ingestion/suite-fast/validation_test.go::TestRequestExceedsMaxSizeReturns413 | <50ms |

**Tiempo de Ejecución Total Suite Rápida:** <1 segundo (todas las 17 pruebas en paralelo o secuencial, según framework)

---

### Suite Integración (Pruebas de Integración con Testcontainers/Toxiproxy, ~8 pruebas, etiqueta de compilación: `// +build integration`)

| Test ID | Test Name | Scenario | External Dependency | Validates Requirements | Validates Properties | Code Location | Execution Time |
|---|---|---|---|---|---|---|
| 10.1 | position_published_to_kafka_with_correct_schema | Valid POST → event published to Kafka with all 4 timestamps + event_id + trace_id | Real Kafka (Testcontainers) | RF-03, RF-02, RF-07, RF-10 | P2 (async publication), P4 (4 timestamps in Kafka), P9 (trace_id in event) | tests/suite_integracao/kafka_test.go::TestPositionPublishedToKafkaWithCorrectSchema | 2-3s |
| 10.2 | kafka_unavailable_toxiproxy_returns_503 | Toxiproxy blocks Kafka → HTTP returns 503 | Real Kafka + Toxiproxy (block) | RF-05, RF-10, RF-12 | P3 (CB opens), P7 (RFC 9457), P8 (no secrets in error) | tests/suite_integracao/circuit_breaker_test.go::TestKafkaUnavailableToxiproxyReturns503 | 2-3s |
| 10.3 | kafka_timeout_toxiproxy_returns_503 | Toxiproxy adds 10s latency (exceeds timeout) → HTTP returns 503 | Real Kafka + Toxiproxy (latency) | RF-05, RF-10, RF-12 | P3 (CB opens on timeout), P7 (RFC 9457), P2 (async publication fails gracefully) | tests/suite_integracao/circuit_breaker_test.go::TestKafkaTimeoutToxiproxyReturns503 | 3-5s |
| 10.4 | idempotency_window_1h_retention | Send same X-Idempotency-Key twice within 1 hour → both accepted (no dedup in MVP) | Real clock (time.Now()) | RF-07 (idempotency key validation) | P9 (trace_id continuity) | tests/suite_integracao/idempotency_test.go::TestIdempotencyWindow1hRetention | 1-2s |
| 10.5 | rate_limiter_window_reset_real_clock | Rate limit window slides with real server clock | Real clock (time.Now()) | RF-13 | P6 (sliding window with real time) | tests/suite_integracao/rate_limit_test.go::TestRateLimiterWindowResetRealClock | 65-70s (waits for 60s window) |
| 10.6 | concurrent_requests_same_vehicle_rate_limit | 12 concurrent requests for same vehicle within 60s → 13th request returns 429 | Real clock, concurrent goroutines | RF-13 | P6 (rate limit enforced with concurrency) | tests/suite_integracao/rate_limit_test.go::TestConcurrentRequestsSameVehicleRateLimit | 2-3s |
| 10.7 | concurrent_requests_different_vehicles_isolated | 12 requests for vehicle A + 12 requests for vehicle B (concurrent) → all succeed (isolated per vehicle) | Real clock, concurrent goroutines | RF-13 | P6 (rate limit isolated per vehicle_id) | tests/suite_integracao/rate_limit_test.go::TestConcurrentRequestsDifferentVehiclesIsolated | 2-3s |
| 10.8 | kafka_consumer_confirmation_not_persisted_without_worker | Valid POST → 202 returned; event in Kafka; BUT TimescaleDB empty (worker not running) | Real Kafka (Testcontainers), NO worker process | RF-06, RF-05 | P2 (202 ≠ persisted), P6 (async nature verified) | tests/suite_integracao/persistence_test.go::TestKafkaConfirmationNotPersistedWithoutWorker | 2-3s |

**Tiempo de Ejecución Total Suite Integración:** ~80-95 segundos (incluye espera de 60s en 10.5; puede optimizarse con mock de reloj)

---

## 5. Fuera de Alcance (Explícitamente NO en Esta Especificación)

Esta sección documenta qué se implementa intencionalmente en el servicio de Ingesta de Posiciones HTTP:

| Elemento | Razón | Dónde se Implementa | Notas |
|---|---|---|---|
| Worker (consumidor de Kafka) | Servicio separado (responsabilidad de RF-05 para monitoreo/dead-letter, no ingesta) | `cmd/worker/`, especificación futura | El servicio HTTP publica; el worker maneja consumo, validación, persistencia |
| Persistencia en TimescaleDB | Responsabilidad del worker; el servicio HTTP solo encola en Kafka | `cmd/worker/`, especificación futura | El servicio HTTP explícitamente NO persiste; 202 significa "encolado", no "persistido" |
| Deduplicación (X-Idempotency-Key) | Alcance del MVP: solo validación, sin lógica real de dedup | Fase 2 Futura (dedup respaldada por Redis/BD) | Encabezado validado como UUID v4, pero duplicados pasan (2 eventos de Kafka para la misma clave) |
| Redis (Caché/Dedup/Estado) | No en el stack del MVP; limitación de velocidad hecha en memoria | N/A | El limitador de velocidad usa sync.Mutex + mapas por vehículo; funciona para MVP de proceso único |
| WebSockets (Actualizaciones en Tiempo Real) | Dashboard usa SSE (RNF-08); fuera del alcance de Ingesta HTTP | Especificación de Frontend | El servicio HTTP solo publica; sin push en tiempo real a clientes |
| Autenticación/Autorización | Límites de seguridad definidos en `security.md`; no en alcance del MVP | Por Definir en Fase 2 | El MVP asume que todas las solicitudes provienen de la aplicación móvil de confianza |
| Puerta de Enlace de API/Equilibrador de Carga | Preocupación de infraestructura (Terraform, AWS); no en código de servicio | `infrastructure/terraform/` | El servicio HTTP asume instancia única o detrás de LB transparente |
| Métricas/Observabilidad (Prometheus) | Logging implementado (logs JSON); recolección de métricas en Fase 2 | Fase 2 Futura | Los logs JSON pueden ser consumidos por agregador de métricas (CloudWatch, DataDog) |

---

## 6. Plan de Ejecución y Verificación de Pruebas

### Ejecutar Suite Rápida (Local, <1 segundo)

```bash
cd backend/go
go test -v -timeout 2s ./tests/ingestion/suite-fast/...
```

**Esperado:** Las 17 pruebas pasan sin dependencias externas requeridas.

---

### Ejecutar Suite Integración (Local, ~90 segundos con Docker)

```bash
cd backend/go
go test -v -tags=integration -timeout 120s ./tests/suite_integracao/...
```

**Requisitos Previos:**
- Docker ejecutándose (Testcontainers iniciará contenedores de Kafka + Toxiproxy)
- ~2GB de RAM disponible

**Esperado:** Las 8 pruebas pasan; Testcontainers limpia automáticamente después de las pruebas.

---

### Ejecutar Todas las Pruebas

```bash
cd backend/go
go test -v ./tests/... && go test -v -tags=integration -timeout 120s ./tests/suite_integracao/...
```

---

## 7. Lista de Verificación

Antes de marcar la especificación como "COMPLETA", verificar:

- [ ] Los 13 Requisitos (RF-01 a RF-13) tienen tareas de implementación en tasks.md
- [ ] Las 14 Propiedades de Corrección están mapeadas a pruebas de verificación (Suite Rápida + Suite Integración)
- [ ] Las 17 pruebas de Suite Rápida pasan localmente (<1 segundo)
- [ ] Las 8 pruebas de Suite Integración pasan con Testcontainers (docker-compose ejecutándose)
- [ ] Revisión de código confirma Clean Architecture COMPLETO (10 subdirectorios, 6 puertos, 4 adaptadores, 5 middleware)
- [ ] Especificación OpenAPI 3.1.0 existe y documenta RFC 9457 ProblemDetails
- [ ] Todas las respuestas de error desinfectadas (sin trazas de pila, SQL, secretos, rutas internas)
- [ ] Trace ID presente en todas las entradas de log y respuestas de error
- [ ] 4 Timestamps validados y ordenados (captured_at ≤ received_at ≤ processed_at)
- [ ] Transiciones de estado del Circuit Breaker verificadas (CLOSED → OPEN después de 5 fallos, HALF_OPEN después de 30s, CLOSED después de 2 éxitos)
- [ ] Deslizamiento de ventana del Limitador de Velocidad verificado (12/60s por vehicle_id, aislado por vehículo)
- [ ] Respuesta HTTP 202 < 100ms p99 (medida en pruebas de Suite Rápida)
- [ ] Publicación en Kafka asíncrona (HTTP retorna antes de confirmación de Kafka)

---

## 8. Documentos Relacionados

- **product.md** - Visión del producto, RF-04 (Ingesta asincrónica de telemetría)
- **tech.md** - Selección de stack (Go, Kafka, TimescaleDB)
- **structure.md** - Límites de Clean Architecture
- **testing.md** - Metodología de pruebas con 4 Pilares (Protección, Resistencia, Feedback, Mantenibilidad)
- **api-standards.md** - Estándar RFC 9457 Problem Details
- **security.md** - Manejo de errores, sin exposición de secretos
- **lenguaje.md** - Convenciones de lenguaje (English en código/API, Spanish en docs)

---

## 9. Registro de Cambios

| Fecha | Versión | Cambio | Autor |
|---|---|---|---|
| 2024 | 1.0 | Matriz de trazabilidad inicial creada; 13 RF → Diseño, 14 Propiedades → Implementación, 25 Pruebas creadas | Kiro |
| Por Definir | 1.1 | Actualizaciones posteriores a la implementación después de ejecución de pruebas | Por Definir |

---

**FIN DE LA MATRIZ DE TRAZABILIDAD**
