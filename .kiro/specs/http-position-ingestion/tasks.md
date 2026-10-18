# Implementation Plan: HTTP Position Ingestion Service

## Overview

Este plan cubre la implementación del servicio HTTP de ingesta de posiciones siguiendo **Clean Architecture COMPLETO** con 14 Correctness Properties validadas mediante pruebas de comportamiento (black-box).

Las tareas se organizan en **15 ondas de dependencia** (máx 4 tareas/onda):
- **Ondas 0-2:** Domain layer (9 Value Objects + 2 errores + 1 Entity + 1 Event)
- **Ondas 3-6:** Application + Ports + Adapters (6 puertos + 4 adaptadores outbound + 5 middleware)
- **Ondas 7-14:** Test infrastructure + 25 pruebas (17 Suite Fast + 8 Suite Integration)

La implementación se verifica mediante:
- Suite Fast: 17 pruebas de comportamiento de caja negra, <1s total, con fakes inyectados
- Suite Integration: 8 pruebas con Testcontainers (Kafka real) y Toxiproxy (inyección de fallos)

### ⚠️ ESTRATEGIA DE TESTING: Crítico

**NO crear archivos `*_test.go` en las Ondas 0-6 (domain, ports, adapters, middleware)**

Según `testing.md` (secciones 3, 5, 10):
- Las pruebas entran **únicamente** por casos de uso (`internal/application/usecases/`)
- Los Value Objects, entidades, puertos y adaptadores NO se testean de forma aislada
- La validación de VOs se verifica **indirectamente** a través de tests de comportamiento de Suite Fast que ejercen los casos de uso
- Crear tests unitarios por archivo viola "resistencia al refactoring": las pruebas fallan sin cambio de comportamiento

**Estructura correcta de tests:**
- `tests/suite_fast/` → 17 pruebas que entran por casos de uso con fakes
- `tests/suite_integration/` → 8 pruebas con Kafka real + Toxiproxy
- `tests/support/mothers/` → Object Mothers (datos de prueba)
- `tests/support/fakes/` → Fakes inyectables (KafkaPublisher, Clock, IDGenerator, RateLimiter)

**Ondas 0-2:** Solo código de dominio, SIN pruebas
**Ondas 3-6:** Solo adaptadores/middleware/handler, SIN pruebas
**Ondas 7-11:** TODAS las pruebas (Suite Fast), entrando por casos de uso
**Ondas 12-13:** TODAS las pruebas (Suite Integration), con contenedores reales

---

## Tasks

### Onda 0: Domain Value Objects part 1

⚠️ **Nota:** Estas tareas crean solo archivos `.go` (sin `_test.go`). Las VOs se testean indirectamente en Suite Fast (Onda 7+).

- [x] 1.1 Implementar VehicleID VO: no vacío, alphanumeric, máx 50 caracteres, constructor valida invariantes
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Location: internal/ingestion/domain/vehicle_id.go_
  - **Nota:** No crear vehicle_id_test.go. Validación se verifica en Suite Fast.

- [x] 1.2 Implementar GeoPosition VO: latitud -90 a 90, longitud -180 a 180, constructor valida invariantes
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Location: internal/ingestion/domain/geo_position.go_
  - **Nota:** No crear geo_position_test.go. Validación se verifica en Suite Fast.

- [x] 1.3 Implementar Timestamp VOs: CapturedTimestamp, ReceivedTimestamp, ProcessedTimestamp con validación de ordenamiento capturedAt ≤ receivedAt ≤ processedAt
  - _Correctness Property: P1 (Validation), P5 (Future-Validation)_
  - _RF: RF-02, RF-07_
  - _Location: internal/ingestion/domain/timestamps.go_
  - **Nota:** No crear timestamps_test.go. Validación se verifica en Suite Fast.

- [x] 1.4 Implementar EventID VO: UUID v4 generado por IDGenerator, inmodificable tras creación
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-03_
  - _Location: internal/ingestion/domain/event_id.go_
  - **Nota:** No crear event_id_test.go. Validación se verifica en Suite Fast.

### Onda 1: Domain Value Objects part 2

⚠️ **Nota:** Continuación de Onda 0. Solo `.go`, sin `_test.go`. Validación en Suite Fast.

- [x] 1.5 Implementar TraceID VO: UUID v4 validado de encabezado X-Trace-ID, generado si ausente
  - _Correctness Property: P1 (Validation), P9 (Trace-ID)_
  - _RF: RF-08_
  - _Location: internal/ingestion/domain/trace_id.go_
  - **Nota:** No crear trace_id_test.go.

- [x] 1.6 Implementar IdempotencyKey VO: UUID v4 validado de encabezado X-Idempotency-Key, formato obligatorio si presente
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-07_
  - _Location: internal/ingestion/domain/idempotency_key.go_
  - **Nota:** No crear idempotency_key_test.go.

- [x] 1.7 Implementar RateLimitConfig VO: requestsPerWindow (12), windowSize (60s), inmodificable
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: internal/ingestion/domain/rate_limit_config.go_
  - **Nota:** No crear rate_limit_config_test.go.

- [x] 1.8 Implementar Clock interface + RealClock adapter: inyectable para determinismo de pruebas, RealClock retorna time.Now()
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: internal/ingestion/ports/clock.go, adapters/outbound/infrastructure/clock.go_
  - **Nota:** No crear clock_test.go. Comportamiento se verifica en Suite Fast.

### Onda 2: Domain errors + Entity + Event + Ports

⚠️ **Nota:** Última onda de dominio. Solo `.go`. Tests en Ondas 7+.

- [x] 1.9 Implementar dominio errors: FutureTimestampError, InvalidVehicleIDError, InvalidGeoPositionError, InvalidEventIDError, InvalidTraceIDError, RateLimitExceededError, CircuitBreakerOpenError
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02, RF-12_
  - _Location: internal/ingestion/domain/errors.go_
  - **Nota:** No crear errors_test.go.

- [x] 2.1 Implementar PositionReceivedEvent VO: 7 campos (eventID, vehicleID, geoPosition, capturedAt, receivedAt, traceID, idempotencyKey), Value Object immutable
  - _Correctness Property: P12 (Domain-Events)_
  - _RF: RF-03_
  - _Location: internal/ingestion/domain/position_received_event.go_
  - **Nota:** No crear position_received_event_test.go.

- [x] 2.2 Implementar VehiclePosition Entity: encapsula 5 timestamps + métodos de dominio (ReceptionLatency, ProcessingLatency, TotalLatency, ToKafkaPayload), NO anémica
  - _Correctness Property: P4 (4-Timestamps), P7 (RFC9457)_
  - _RF: RF-07_
  - _Location: internal/ingestion/domain/vehicle_position.go_
  - **Nota:** No crear vehicle_position_test.go.

- [x] 3.1 Definir 6 puertos: KafkaPublisher, CircuitBreakerManager, RateLimiter, Logger, Clock, IDGenerator como interfaces sin dependencias de frameworks
  - _Correctness Property: P13 (Domain-Event-Decoupling)_
  - _RF: RF-03, RF-10, RF-13_
  - _Location: internal/ingestion/ports/*.go_
  - **Nota:** No crear tests en ports/. Solo interfaces.

### Onda 3: Clock port + Outbound Adapters part 1

⚠️ **Nota:** Adaptadores. Solo `.go`, sin `_test.go`. Tests en Ondas 7-13.

- [x] 3.2 Definir Clock puerto: interfaz Now() time.Time inyectable
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: internal/ingestion/ports/clock.go_
  - **Nota:** No crear puerto_test.go.

- [x] 4.1 Implementar KafkaPublisher adapter: 3 reintentos con backoff exponencial, partición por vehicle_id, dead-letter queue para fallos definitivos
  - _Correctness Property: P2 (Async+202), P3 (CB-Recovery)_
  - _RF: RF-03, RF-05_
  - _Location: adapters/outbound/messaging/kafka_publisher.go_
  - **Nota:** No crear kafka_publisher_test.go. Comportamiento se verifica en Suite Integration (Onda 12).

- [x] 4.2 Implementar CircuitBreaker adaptador: máquina de estados CLOSED → OPEN (5 fallos) → HALF_OPEN (30s timeout) → CLOSED (2 éxitos), thread-safe con sync.RWMutex
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-10, RF-11_
  - _Location: adapters/outbound/resilience/circuit_breaker.go_
  - **Nota:** No crear circuit_breaker_test.go. Comportamiento se verifica en Suite Fast (Onda 10).

- [x] 4.3 Implementar RateLimiter adapter: ventana deslizante en memoria, 12/60s por defecto, aislamiento por vehicle_id, thread-safe
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: adapters/outbound/resilience/rate_limiter.go_
  - **Nota:** No crear rate_limiter_test.go. Comportamiento se verifica en Suite Fast (Onda 9).

### Onda 4: Adapters part 2 + DTOs + Middleware part 1

⚠️ **Nota:** Adaptadores, DTOs, middleware. Solo `.go`. Tests en Ondas 7-13.

- [x] 4.4 Implementar Clock e IDGenerator adapters (reales para producción, inyectables para pruebas): RealClock retorna time.Now(), RealIDGenerator retorna UUID v4
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: adapters/outbound/infrastructure/clock.go, adapters/outbound/infrastructure/id_generator.go_
  - **Nota:** No crear adapters_test.go.

- [x] 5.1 Implementar DTOs: PositionRequest (vehicle_id, latitude, longitude, captured_at), PositionResponse (event_id, status, timestamp), ProblemDetails (RFC 9457)
  - _Correctness Property: P7 (RFC9457)_
  - _RF: RF-01, RF-04, RF-12_
  - _Location: adapters/inbound/http/dto/*.go_
  - **Nota:** No crear dto_test.go. DTOs se verifican indirectamente en tests de respuesta.

- [x] 5.2 Implementar mappers: PositionMapper (FromRequest → ReceivePositionCommand, FromEntity → PositionResponse), ErrorMapper (DomainError → ProblemDetails)
  - _Correctness Property: P7 (RFC9457), P10 (Error-Mapping)_
  - _RF: RF-04, RF-12_
  - _Location: adapters/inbound/http/mapper/*.go_
  - **Nota:** No crear mapper_test.go. Mapeos se verifican en Suite Fast (Ondas 8-11).

- [x] 6.1 Implementar TraceIDMiddleware: generar o validar UUID v4 de X-Trace-ID, almacenar en gin.Context, escribir respuesta header X-Trace-ID antes de c.Next()
  - _Correctness Property: P9 (Trace-ID)_
  - _RF: RF-08_
  - _Location: adapters/inbound/http/middleware/trace_id.go_
  - **Nota:** No crear middleware_test.go. Comportamiento se verifica en Suite Fast (Onda 9).

### Onda 5: Middleware part 2 + part 3

⚠️ **Nota:** Middleware. Solo `.go`. Tests en Ondas 9-11.

- [x] 6.2 Implementar IdempotencyKeyMiddleware: validar X-Idempotency-Key si presente (UUID v4), almacenar en Context, no blockear si ausente
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-07_
  - _Location: adapters/inbound/http/middleware/idempotency_key.go_

- [x] 6.3 Implementar ContentTypeMiddleware: validar Content-Type: application/json, rechazar con 415 si diferente
  - _Correctness Property: P7 (RFC9457)_
  - _RF: RF-08_
  - _Location: adapters/inbound/http/middleware/content_type.go_

- [x] 6.4 Implementar RateLimitMiddleware: extraer vehicle_id del cuerpo vía gin.GetRawData(), verificar RateLimiter, rechazar con 429 si límite excedido
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Location: adapters/inbound/http/middleware/rate_limit.go_

- [x] 6.5 Implementar ErrorHandlerMiddleware: capturar c.Error() y panic, convertir dominio errors a ProblemDetails con trace_id, responder RFC 9457
  - _Correctness Property: P7 (RFC9457), P11 (Error-Propagation)_
  - _RF: RF-12_
  - _Location: adapters/inbound/http/middleware/error_handler.go_

### Onda 6: Handler + Composer + Test Support part 1

⚠️ **Nota:** Onda de transición. Handler y Composer son `.go`. Test support son herramientas (sin `_test.go`).

- [x] 6.6 Implementar PositionHandler: ~15 líneas, deserializar request → mapper.FromRequest → useCase.Execute → mapper.FromEntity → responder 202
  - _Correctness Property: P2 (Async+202)_
  - _RF: RF-01, RF-04_
  - _Location: adapters/inbound/http/handler/position_handler.go_

- [x] 7.1 Implementar Composer: inyección de dependencias, instanciar config → adaptadores → useCase → middleware → Router Gin con pila en orden correcto
  - _Correctness Property: (all)_
  - _RF: (all)_
  - _Location: internal/ingestion/config/composer.go_

- [x] 8.1 Implementar PositionMother: Object Mother con escenarios predefinidos (Valid, FutureTimestamp, OutOfRangeLatitude, OutOfRangeLongitude, MissingVehicleID)
  - _Test Coverage: Suite Fast + Suite Integration_
  - _RF: RF-02_
  - _Location: tests/support/mothers/position_mother.go_
  - **Nota:** PositionMother es herramienta de prueba, no test. No es `*_test.go`.

- [x] 8.2 Implementar Fakes: FakeKafkaPublisher (state tracking), FakeClock (controllable time), FakeIDGenerator (deterministic UUIDs), FakeRateLimiter (configurable limits)
  - _Test Coverage: Suite Fast_
  - _RF: (all)_
  - _Location: tests/support/fakes/*.go_
  - **Nota:** Fakes son herramientas de prueba, no tests. No son `*_test.go`.

### Onda 7: Test Support part 2 + UseCase + Suite Fast part 1

✅ **Nota:** PRIMERA onda con `_test.go`. Entradas: Solo a través de casos de uso. Fakes inyectados.

- [-] 8.3 Implementar TestHTTPServer: harness Gin con fakes inyectados, expone puerto para pruebas, detiene limpiamente
  - _Test Coverage: Suite Fast_
  - _RF: (all)_
  - _Location: tests/support/http_server.go_
  - **Nota:** Herramienta, no test. No es `_test.go`.

- [-] 8.4 Implementar Router HTTP: pila de middleware en orden correcto (TraceID → Idempotency → ContentType → RateLimit → ErrorHandler, evaluados en reverso)
  - _Correctness Property: (all)_
  - _RF: (all)_
  - _Location: adapters/inbound/http/router.go_

- [-] 8.5 Implementar ReceivePositionUseCase: crear VehiclePosition → validar → publicar async a Kafka vía goroutine (fire-and-forget) → devolver 202 inmediatamente
  - _Correctness Property: P2 (Async+202), P3 (CB-Recovery), P12 (Domain-Events), P13 (Domain-Event-Decoupling)_
  - _RF: RF-01, RF-03, RF-04, RF-06_
  - _Location: application/usecases/receive_position.go_
  - **Nota:** UseCase es código de producción, no test.

- [-] 9.1 Prueba: valid_position_returns_202 (verifica: entrada válida retorna 202, estructura response correcta, trace_id header presente)
  - _Correctness Property: P2 (Async+202), P4 (4-Timestamps), P9 (Trace-ID)_
  - _RF: RF-01, RF-02, RF-04, RF-07_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/valid_position_test.go_
  - **✅ Primer archivo `_test.go` legítimo aquí.**

### Onda 8: Suite Fast part 2

✅ **Nota:** Tests Suite Fast. Entradas por casos de uso. Fakes inyectados.

- [x] 9.2 Prueba: future_captured_at_returns_422 (verifica: captured_at > now rechaza con 422, ProblemDetails con tipo correcto)
  - _Correctness Property: P5 (Future-Validation)_
  - _RF: RF-02, RF-04, RF-12_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

- [x] 9.3 Prueba: missing_vehicle_id_returns_422 (verifica: vehicle_id vacío rechaza con 422, ProblemDetails con detail)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

- [x] 9.4 Prueba: latitude_out_of_range_returns_422 (verifica: latitud > 90 o < -90 rechaza con 422)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

- [x] 9.5 Prueba: longitude_out_of_range_returns_422 (verifica: longitud > 180 o < -180 rechaza con 422)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

### Onda 9: Suite Fast part 3

✅ **Nota:** Tests Suite Fast. Middleware + Rate Limit.

- [x] 9.6 Prueba: invalid_content_type_returns_415 (verifica: Content-Type != application/json rechaza con 415)
  - _Correctness Property: P7 (RFC9457)_
  - _RF: RF-01, RF-08_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/middleware_test.go_

- [x] 9.7 Prueba: idempotency_key_validation (verifica: X-Idempotency-Key inválido rechaza con 400, ausente es permitido)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-07_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/middleware_test.go_

- [x] 9.8 Prueba: rate_limit_exceeded_returns_429 (verifica: 13ª solicitud del mismo vehicle_id retorna 429)
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/rate_limit_test.go_

- [x] 9.9 Prueba: rate_limit_window_sliding (verifica: cuota se regenera después de 60s de ventana)
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/rate_limit_test.go_

### Onda 10: Suite Fast part 4

✅ **Nota:** Tests Suite Fast. Circuit Breaker.

- [x] 9.10 Prueba: circuit_breaker_opens_on_repeated_failures (verifica: 5 fallos consecutivos → CircuitBreaker abre, 6ª solicitud retorna 503)
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-10, RF-11_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/circuit_breaker_test.go_

- [x] 9.11 Prueba: circuit_breaker_half_opens_after_timeout (verifica: OPEN → HALF_OPEN tras 30s timeout, próxima solicitud se intenta)
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-10, RF-11_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/circuit_breaker_test.go_

- [x] 9.12 Prueba: circuit_breaker_closes_after_successes (verifica: 2 éxitos en HALF_OPEN → transiciona a CLOSED)
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-10, RF-11_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/circuit_breaker_test.go_

- [x] 9.13 Prueba: response_includes_trace_id_header (verifica: respuesta incluye X-Trace-ID header con UUID v4)
  - _Correctness Property: P9 (Trace-ID)_
  - _RF: RF-08, RF-07_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/middleware_test.go_

### Onda 11: Suite Fast part 5

✅ **Nota:** Tests Suite Fast. Últimas pruebas.

- [~] 9.14 Prueba: invalid_trace_id_header_returns_400 (verifica: X-Trace-ID inválido rechaza con 400)
  - _Correctness Property: P9 (Trace-ID)_
  - _RF: RF-08_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/middleware_test.go_

- [~] 9.15 Prueba: latency_timestamps_correct (verifica: procesamiento y recepción latency calculadas correctamente, capturedAt ≤ receivedAt ≤ processedAt)
  - _Correctness Property: P4 (4-Timestamps)_
  - _RF: RF-07_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/timestamp_test.go_

- [~] 9.16 Prueba: malformed_json_returns_400 (verifica: JSON inválido rechaza con 400, ProblemDetails)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

- [~] 9.17 Prueba: request_exceeds_max_size_returns_413 (verifica: solicitud > 1MB rechaza con 413)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-02_
  - _Test Suite: Suite Fast_
  - _Test File: tests/suite_fast/validation_test.go_

### Onda 12: Suite Integration part 1

✅ **Nota:** Tests Suite Integration. Kafka real + Testcontainers.

- [~] 10.1 Prueba: position_published_to_kafka_with_correct_schema (verifica: evento persistido en Kafka con schema correcto: partición=vehicle_id, offset avanzado)
  - _Correctness Property: P2 (Async+202), P12 (Domain-Events)_
  - _RF: RF-03, RF-02_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/kafka_test.go_

- [~] 10.2 Prueba: kafka_unavailable_toxiproxy_returns_503 (verifica: Toxiproxy bloquea Kafka, CircuitBreaker abre, retorna 503 con ProblemDetails)
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-05, RF-10, RF-11_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/resilience_test.go_

- [~] 10.3 Prueba: kafka_timeout_toxiproxy_returns_503 (verifica: Toxiproxy injeta 2s latencia, timeout respetado, retorna 503)
  - _Correctness Property: P3 (CB-Recovery)_
  - _RF: RF-05, RF-10, RF-11_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/resilience_test.go_

- [~] 10.4 Prueba: idempotency_window_1h_retention (verifica: X-Idempotency-Key almacenado con ventana de 1h, mismo key en ventana se detecta)
  - _Correctness Property: P1 (Validation)_
  - _RF: RF-07_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/idempotency_test.go_

### Onda 13: Suite Integration part 2

✅ **Nota:** Tests Suite Integration. Concurrencia + Async.

- [~] 10.5 Prueba: rate_limiter_window_reset_real_clock (verifica: reloj real, ventana se desliza tras 60s, cuota se regenera)
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/rate_limit_test.go_

- [~] 10.6 Prueba: concurrent_requests_same_vehicle_rate_limit (verifica: 12 solicitudes concurrentes del mismo vehicle_id, 13ª rechazada con 429)
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/rate_limit_test.go_

- [~] 10.7 Prueba: concurrent_requests_different_vehicles_isolated (verifica: rate limit de VH-A no afecta a VH-B, aislamiento total)
  - _Correctness Property: P6 (Rate-Limit)_
  - _RF: RF-13_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/rate_limit_test.go_

- [~] 10.8 Prueba: kafka_consumer_confirmation_not_persisted_without_worker (verifica: 202 retornado antes de confirmación de Kafka, sin worker no hay persistencia final)
  - _Correctness Property: P2 (Async+202)_
  - _RF: RF-05, RF-06_
  - _Test Suite: Suite Integration_
  - _Test File: tests/suite_integration/async_test.go_

### Onda 14: Tareas Opcionales

- [~] 11.1 Crear especificación OpenAPI 3.1.0 (endpoint POST /positions, schema request/response, ProblemDetails, ejemplos)
  - _RF: RF-09_
  - _Location: openapi.yaml_

- [~] 11.2 Implementar pruebas basadas en propiedades: latencia monótona (processingLatency ≥ receptionLatency), deslizamiento de ventana de RateLimiter
  - _Correctness Property: P4 (4-Timestamps), P6 (Rate-Limit)_
  - _Location: tests/suite_fast/property_based_test.go_

## Notes

### Correctness Properties

- **P1 (Validation):** Todas las entradas validadas en constructor, rechazo de valores inválidos
- **P2 (Async+202):** HTTP retorna 202 antes de confirmación de Kafka
- **P3 (CB-Recovery):** Circuit Breaker transiciona correctamente CLOSED → OPEN → HALF_OPEN → CLOSED
- **P4 (4-Timestamps):** capturedAt ≤ receivedAt ≤ processedAt, latencias calculadas correctamente
- **P5 (Future-Validation):** capturedAt en futuro rechazado con 422
- **P6 (Rate-Limit):** Ventana deslizante 60s, 12/60s por vehicle_id, aislamiento total
- **P7 (RFC9457):** RFC 9457 Problem Details en todas las respuestas de error
- **P8 (No-Secrets):** Logs y respuestas no contienen tokens, passwords, URLs secretas
- **P9 (Trace-ID):** X-Trace-ID presente en todas las respuestas, generado o validado
- **P10 (Error-Mapping):** Errores de dominio mapeados a HTTP status + ProblemDetails correctos
- **P11 (Error-Propagation):** Errores en middleware no silenciados, propagados a ErrorHandlerMiddleware
- **P12 (Domain-Events):** PositionReceivedEvent emitido, desacoplado de persistencia
- **P13 (Domain-Event-Decoupling):** UseCase no conoce detalles de KafkaPublisher, solo puerto
- **P14 (Error-No-Leaks):** Stack traces, SQL, paths no expuestos en respuestas HTTP

### Middleware Execution Order

Orden de ejecución: **TraceID (primero) → Idempotency → ContentType → RateLimit → ErrorHandler (último)**

ErrorHandler captura todos los errores y pánico de middleware internos, convierte a ProblemDetails con trace_id.

### Test Data

PositionMother genera datos válidos/inválidos determinísticos:
- `Valid()` → entrada válida con todos los campos correctos
- `FutureTimestamp()` → captured_at en futuro
- `OutOfRangeLatitude()` → latitud > 90
- `OutOfRangeLongitude()` → longitud > 180
- `MissingVehicleID()` → vehicle_id vacío

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3", "1.4"], "label": "Domain VOs part 1 (no tests)" },
    { "id": 1, "tasks": ["1.5", "1.6", "1.7", "1.8"], "label": "Domain VOs part 2 (no tests)" },
    { "id": 2, "tasks": ["1.9", "2.1", "2.2", "3.1"], "label": "Domain errors/entity/event + Ports (no tests)" },
    { "id": 3, "tasks": ["3.2", "4.1", "4.2", "4.3"], "label": "Clock port + Adapters part 1 (no tests)" },
    { "id": 4, "tasks": ["4.4", "5.1", "5.2", "6.1"], "label": "Adapters part 2 + DTOs + Middleware part 1 (no tests)" },
    { "id": 5, "tasks": ["6.2", "6.3", "6.4", "6.5"], "label": "Middleware part 2-3 (no tests)" },
    { "id": 6, "tasks": ["6.6", "7.1", "8.1", "8.2"], "label": "Handler + Composer + Test support part 1 (no tests)" },
    { "id": 7, "tasks": ["8.3", "8.4", "8.5", "9.1"], "label": "Test support part 2 + UseCase + Suite Fast part 1 (FIRST *_test.go)" },
    { "id": 8, "tasks": ["9.2", "9.3", "9.4", "9.5"], "label": "Suite Fast part 2" },
    { "id": 9, "tasks": ["9.6", "9.7", "9.8", "9.9"], "label": "Suite Fast part 3" },
    { "id": 10, "tasks": ["9.10", "9.11", "9.12", "9.13"], "label": "Suite Fast part 4" },
    { "id": 11, "tasks": ["9.14", "9.15", "9.16", "9.17"], "label": "Suite Fast part 5" },
    { "id": 12, "tasks": ["10.1", "10.2", "10.3", "10.4"], "label": "Suite Integration part 1" },
    { "id": 13, "tasks": ["10.5", "10.6", "10.7", "10.8"], "label": "Suite Integration part 2" },
    { "id": 14, "tasks": ["11.1", "11.2"], "label": "Tareas opcionales" }
  ]
}
```
