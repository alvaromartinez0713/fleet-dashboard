# Implementation Plan: Telemetry Worker

## Overview

Este plan cubre la implementación del Telemetry Worker, un servicio Go independiente que consume eventos `PositionReceived` desde Kafka, valida datos requeridos, deduplica por `event_id` y persiste las posiciones en TimescaleDB con garantías de idempotencia y sin pérdida silenciosa.

El worker cierra el flujo de ingesta asincrónica definido en `product.md` (RF-04, RF-05, RF-10) implementando Clean Architecture, desacoplamiento (RNF-03), integridad e idempotencia (RNF-04), persistencia especializada (RNF-05) y resiliencia (RNF-06).

El plan se organiza en **8 ondas de dependencia**: dominio → puertos → aplicación → adaptadores → consumer Kafka → configuración → bootstrap → verificación. Cada onda respeta la regla: `domain/` nunca importa capas superiores; `application/` solo importa `domain/` y `ports/`; adaptadores implementan puertos; configuración conecta piezas. Las pruebas siguen `testing.md`: entrada únicamente por casos de uso con fakes inyectados, sin tests unitarios de domain/adapters.

## Tasks

### Wave 0: Migración y Setup

- [x] 1. Migración 000002 - Schema de positions_table
  - [x] 1.1 Editar `backend/go/migrations/000002_create_positions_table.up.sql` — eliminar columnas legadas (`timestamp`, `altitude`, `speed`, `heading`); agregar `event_id UUID NOT NULL`, `received_at TIMESTAMPTZ NULL`, `processed_at TIMESTAMPTZ NULL`, `stored_at TIMESTAMPTZ NOT NULL`; crear UNIQUE INDEX `(captured_at, event_id)` con `captured_at` primero (requisito TimescaleDB); crear INDEX `(vehicle_id, captured_at DESC)`; Hypertable dimension: `captured_at`; editar también `down.sql` con `DROP TABLE IF EXISTS telemetry.positions CASCADE`
    - _Verificación_: Schema exacto con 10 columnas (id, event_id, vehicle_id, latitude, longitude, accuracy, captured_at, received_at, processed_at, stored_at)
    - _RF: RF-10_

### Wave 1: Domain y Config

- [x] 2. Domain - Tipos y validación
  - [x] 2.1 Crear `internal/worker/domain/consumed_event.go` — `package domain`; `ConsumedPositionEvent` struct con campos: `EventID` (uuid.UUID), `VehicleID` (string), `Latitude` (float64), `Longitude` (float64), `CapturedAt` (time.Time), `ReceivedAt` (*time.Time), `ProcessedAt` (*time.Time), `Accuracy` (*float64), `TraceID` (*string); reutilizar value objects de `internal/ingestion/domain` (EventID, VehicleID, GeoPosition, CapturedTimestamp, ReceivedTimestamp) o reconvertir al dominio del worker; constructor valida invariantes sin Kafka ni SQL
    - _RF: RF-02_
  - [x] 2.2 Crear `internal/worker/domain/validation.go` — `package domain`; función `ValidateConsumedPositionEvent(e ConsumedPositionEvent) []error`; valida 10 criterios (RF-02): EventID UUID v4, VehicleID alfanumérico 1-50, Latitude [-90,90], Longitude [-180,180], CapturedAt ISO 8601 UTC válido, CapturedAt no en futuro, ReceivedAt (si presente) ISO 8601 UTC válido, ReceivedAt >= CapturedAt (si presente), ProcessedAt (si presente) ISO 8601 UTC válido, campos desconocidos ignorados; sin imports de Kafka, SQL, HTTP
    - _RF: RF-02_
  - [x] 2.3 Crear `internal/worker/domain/dead_letter.go` — `package domain`; `DeadLetterRecord` struct con: `TraceID string`, `EventID string` (puede ser vacío si payload malformado), `VehicleID string` (puede ser vacío), `Classification string` ("recoverable"|"non_recoverable"), `ReasonCategory string` (stable, ej. "invalid_event", "malformed_payload"), `OccurredAt time.Time` (UTC ms); sin latitude, longitude, payload completo
    - _RF: RF-08_
  - [x] 2.4 Crear `internal/worker/domain/errors.go` — `package domain`; `AppError` con campos `Code string`, `Message string`, `Inner error`, `IsRecoverable bool`; funciones `NewAppError` y `NewRecoverableError`; implementar `Error()` y `Unwrap()`; sin imports de Kafka, SQL, HTTP
    - _RF: RF-01 a RF-08_

- [x] 3. Config - Estructura y validación
  - [x] 3.1 Crear `internal/worker/ports/config.go` — `package ports`; `Config` struct con secciones: `Kafka{Brokers []string, Topic string, ConsumerGroup string, BatchMaxSize int}`, `Retry{MaxRetries int, BaseBackoff time.Duration, BackoffFactor float64}`, `CircuitBreaker{FailureThreshold int, RecoveryInterval time.Duration}`, `Database{DSN string, OperationTimeout time.Duration, MaxOpenConns int, MaxIdleConns int}`, `Worker{ShutdownTimeout time.Duration}`, `Logging{Level, Format string}`; `ConfigLoader` interface con `Load() (Config, error)`; `Validator` interface con `Validate(Config) error`
    - _RF: RF-09_
  - [x] 3.2 Crear `internal/worker/config/loader.go` — `package config`; `EnvConfigLoader`; lee variables `WORKER_KAFKA_BROKERS` (CSV), `WORKER_KAFKA_TOPIC`, `WORKER_KAFKA_CONSUMER_GROUP`, `WORKER_RETRY_MAX`, `WORKER_RETRY_BASE_BACKOFF`, `WORKER_CB_FAILURE_THRESHOLD`, `WORKER_CB_RECOVERY_INTERVAL`, `WORKER_DB_DSN`, `WORKER_DB_OPERATION_TIMEOUT`, `WORKER_SHUTDOWN_TIMEOUT`, `LOG_LEVEL`, `LOG_FORMAT`; `getEnvRequired` falla con nombre del campo si la variable no está definida; `parseInt`, `parseDuration` fallan con nombre del campo si el formato es incorrecto; el error identifica el campo sin incluir el valor
    - _RF: RF-09_
  - [x] 3.3 Crear `internal/worker/config/validator.go` — `package config`; `ConfigValidator`; valida: `Kafka.Brokers` no vacío, `Retry.MaxRetries` en rango 0-10, `CircuitBreaker.FailureThreshold` > 0, `Database.DSN` no vacío, `Worker.ShutdownTimeout` en rango 1-300s, `Database.OperationTimeout` > 0; cada `ValidationError` incluye `Field` y `Message` sin el valor del campo; NO expone DSN en mensajes de error
    - _RF: RF-09_

### Wave 2: Puertos

- [x] 4. Puertos - Repository, Clock, CircuitBreaker, DeadLetterSink
  - [x] 4.1 Crear `internal/worker/ports/repository.go` — `package ports`; interfaz `PositionRepository` con `Save(ctx context.Context, p PersistablePosition) (SaveResult, error)` y `Close() error`; tipos: `SaveResult` enum (SaveResultInserted=0, SaveResultDuplicate=1), `PersistablePosition` struct con campos del contrato (event_id, vehicle_id, latitude, longitude, captured_at, received_at, processed_at, stored_at, accuracy)
    - _RF: RF-03_
  - [x] 4.2 Crear `internal/worker/ports/clock.go` — `package ports`; interfaz `Clock` con `Now() time.Time`; usado para generar `stored_at` en UTC ms, inyectable para determinismo en pruebas
    - _RF: RF-03_
  - [x] 4.3 Crear `internal/worker/ports/circuit_breaker.go` — `package ports`; interfaz `CircuitBreaker` con `Execute(ctx context.Context, fn func() error) error` y `State() CircuitBreakerState`; tipos: `CircuitBreakerState` enum (CLOSED, OPEN, HALF_OPEN), `CircuitBreakerOpenError` error type
    - _RF: RF-07_
  - [x] 4.4 Crear `internal/worker/ports/dead_letter_sink.go` — `package ports`; interfaz `DeadLetterSink` con `Write(ctx context.Context, record domain.DeadLetterRecord) error`; sin salida a Kafka en MVP (solo log JSON)
    - _RF: RF-08_

### Wave 3: Application - Use Case

- [x] 5. Application - ProcessPositionEvent
  - [x] 5.1 Crear `internal/worker/application/usecases/process_position_event.go` — `package usecases`; `ProcessPositionEvent` struct que recibe 6 puertos inyectados: `repo ports.PositionRepository`, `clock ports.Clock`, `breaker ports.CircuitBreaker`, `sink ports.DeadLetterSink`, `logger ports.Logger`, `config ports.Config`; tipo `ProcessResult{Commit bool, Paused bool}`; método `Process(ctx context.Context, rawPayload []byte) ProcessResult` que implementa el pipeline (Requisito 1-6): 1) Parsear JSON tolerante (malformado → no recuperable → dead-letter → commit), 2) Validar (inválido → dead-letter → commit), 3) Construir PersistablePosition con stored_at del Clock en UTC ms, 4) Bajo CircuitBreaker + timeout: persistir con reintentos backoff 100/200/400ms determinista, 5) Mapear resultado a ProcessResult según reglas de offset commit (Requisito 6), 6) Emitir logs JSON INFO/ERROR sin lat/lon/SQL; propagar/generar trace_id (UUID v4 válido → propagar; ausente o inválido → generar)
    - _RF: RF-01 a RF-08_
  - [x] 5.2 Crear `tests/support/mothers/worker_position_mother.go` — `package mothers`; `WorkerPositionMother` con métodos `Valid() *WorkerPositionBuilder`, `OutOfRange() *WorkerPositionBuilder`, `MissingReceivedAt() *WorkerPositionBuilder`, `MalformedPayload() *WorkerPositionBuilder`; builder pattern con `WithVehicleID(...)`, `Build()` que retorna `[]byte` (JSON); generación determinista de UUID y timestamps
    - _Tests Support_
  - [x] 5.3 Crear `tests/support/fakes/fake_repository.go`, `fake_clock.go`, `fake_dead_letter_sink.go`, `fake_logger.go`, `fake_circuit_breaker.go` — fakes con estado, semántica ON CONFLICT DO NOTHING, fallos configurables; Clock devuelve time.Time fijo; Logger captura logs para verificación; CircuitBreaker transiciones simulables
    - _Tests Support_

### Wave 4: Adaptadores de Salida

- [x] 6. Adaptador Clock
  - [x] 6.1 Crear `internal/worker/adapters/outbound/infrastructure/clock.go` — `package infrastructure`; `SystemClock` struct; implementar `Clock` retornando `time.Now()` en UTC con precisión de milisegundos
    - _RF: RF-03_

- [x] 7. Adaptador Circuit Breaker
  - [x] 7.1 Crear `internal/worker/adapters/outbound/resilience/circuit_breaker.go` — `package resilience`; implementar `CircuitBreaker` con 3 estados (CLOSED, OPEN, HALF_OPEN); contar solo `Error_Recuperable` (not `IsRecoverable=false`) para umbral (WORKER_CB_FAILURE_THRESHOLD); pausa consumo mientras OPEN (devuelve `CircuitBreakerOpenError`); transiciones automáticas según Requisito 7; thread-safe con `sync.RWMutex`; parámetros desde Config
    - _RF: RF-07_

- [x] 8. Adaptador Dead-Letter Sink
  - [x] 8.1 Crear `internal/worker/adapters/outbound/deadletter/log_sink.go` — `package deadletter`; implementar `DeadLetterSink` como log JSON estructurado sobre Logger existente; escribe `DeadLetterRecord` sin lat/lon, payload completo; logs JSON con trace_id, event_id, vehicle_id, classification, reason
    - _RF: RF-08_

- [x] 9. Adaptador Repository TimescaleDB
  - [x] 9.1 Crear `internal/worker/adapters/outbound/persistence/timescale_repository.go` — `package persistence`; implementar `PositionRepository` con `database/sql` + pgx driver; `Save(ctx, p)` ejecuta INSERT parametrizado único: `INSERT INTO telemetry.positions (...) VALUES (...) ON CONFLICT (captured_at, event_id) DO NOTHING`; distinguir insertado vs duplicado por `RowsAffected()`; timeout per operazione via `context.WithTimeout(ctx, WORKER_DB_OPERATION_TIMEOUT)`; clasificar errores driver en `AppError` (recuperable/no recuperable — ej. `UNIQUE VIOLATION` es no recuperable para validación); método `Close()` cierra pool
    - _RF: RF-03, RF-04_

### Wave 5: Adaptador Entrada Kafka

- [x] 10. Consumer Kafka y Runner
  - [x] 10.1 Crear `internal/worker/consumer/consumer.go` — `package consumer`; envoltorio de `segmentio/kafka-go`; `Consumer` struct con `client *kafka.Reader`; `CommitInterval: 0` (auto-commit OFF); recibe `ConsumerConfig` desde `ports.Config`; método `FetchMessage() ([]byte, error)` retorna mensaje sin cambios de offset; método `CommitOffset(ctx, offset int64) error` confirma explícitamente
    - _RF: RF-01_
  - [x] 10.2 Crear `internal/worker/consumer/runner.go` — `package consumer`; `Runner` struct que orquesta el loop de consumo; procesamiento secuencial por partición (orden de offset ascendente); pausa fetch mientras CircuitBreaker OPEN; rebalanceo: completa trabajo en curso dentro de WORKER_SHUTDOWN_TIMEOUT; reconexión con Backoff_Exponencial ante Kafka no disponible (Requisito 1.7); commit explícito solo cuando `ProcessResult.Commit == true`; reintentos de commit (3x backoff); log WARN al reanudar con offset por partición; graceful shutdown: SIGTERM/SIGINT, libera conexiones
    - _RF: RF-01, RF-05, RF-06_

### Wave 6: Wiring y Bootstrap

- [x] 11. Config - Loader, Validator, Composer
  - [x] 11.1 Crear `internal/worker/config/composer.go` — `package config`; `DependencyComposer` con `ComposeWorker() (func(), error)` que devuelve función de limpieza; instancia: Clock → CircuitBreaker → Repository → DeadLetterSink → ProcessPositionEvent → Consumer → Runner; inyecta todos los puertos correctamente; SIN lógica de dominio; si una instanciación falla libera recursos ya inicializados
    - _RF: RF-09_

- [x] 12. Bootstrap - cmd/worker
  - [x] 12.1 Crear `backend/go/cmd/worker/main.go` — import solo `internal/worker/...`; delgado, llama `bootstrap.Run()` y maneja código de salida
    - _RF: RF-01_
  - [x] 12.2 Crear `backend/go/cmd/worker/bootstrap.go` — coordina: load config → validate → NewLogger → ComposeWorker → start Runner; maneja errores con contexto
    - _RF: RF-09_
  - [x] 12.3 Crear `backend/go/cmd/worker/lifecycle.go` — maneja: worker running → SIGTERM → cleanup con timeout de 10s → segunda señal fuerza `os.Exit(1)`
    - _RF: RF-05, RF-06_
  - [x] 12.4 Crear `backend/go/cmd/worker/signals.go` — encapsula: `setupSignalHandler()`, `waitForSignal()`, `waitForSecondSignal()`; usa `signal.Notify()` y `os.Signal`
    - _RF: RF-05, RF-06_

### Wave 7: Pruebas

- [x] 13. Pruebas Suite Fast - ProcessPositionEvent
  - [x] 13.1 Prueba: `tests/suite_fast/process_position_event_test.go` — `camino_feliz_insertado_retorna_commit_true` (evento válido insertado → ProcessResult.Commit=true); `duplicado_retorna_commit_true` (SaveResultDuplicate → commit); `evento_invalido_retorna_commit_true` (dead-letter OK); `payload_malformado_retorna_commit_true` (dead-letter OK); `dead_letter_falla_retorna_commit_false` (no commit); `propagar_trace_id` (UUID v4 válido → propagar); `generar_trace_id` (ausente → generar)
    - _Correctness Properties: P1 (Validation), P2 (Async+Commit), P3 (Dedup), P4 (Error-Classification)_
    - _RF: RF-01 a RF-08_
  - [x] 13.2 Prueba: `tests/suite_fast/circuit_breaker_test.go` — `abre_tras_5_fallos` (5 errores recuperables → OPEN); `pausa_consumo_cuando_abierto` (OPEN → bloquea Execute); `half_open_tras_timeout_recuperacion` (OPEN + 30s → HALF_OPEN); `cierra_tras_exitos_en_half_open` (2 éxitos HALF_OPEN → CLOSED)
    - _Correctness Properties: P5 (Resilience)_
    - _RF: RF-07_

- [x] 14. Pruebas Suite Integration (tags)
  - [x] 14.1 Prueba: `tests/suite_integration/persistence_integration_test.go` (build tags `+build integration`) — `publica_a_kafka_y_persiste` (evento → Kafka → Worker → fila en BD); `mismo_evento_dos_veces_una_fila` (deduplicación por event_id); `evento_invalido_zero_filas` (event inválido → 0 filas); `Toxiproxy_corte_DB_recuperacion_sin_perdida` (corte DB → reintentos → recuperación)
    - _Correctness Properties: P3 (Dedup), P6 (No-Loss)_
    - _RF: RF-03, RF-04, RF-05_
  - [x] 14.2 Prueba: `tests/suite_integration/idempotencia_property_test.go` (rapid property-based) — Property: `procesar_N_veces_1_fila` (N ≥ 1, mismo evento); `lote_con_duplicados_1_fila_per_event_id`; `orden_irrelevante` (confluencia); `sin_perdida_offset_solo_cuando_garantia`; `no_recuperable_cero_reintentos`; `timestamps_presentes_sin_modificar` (captured_at, received_at, processed_at preservados); `stored_at_generado_del_clock` (stored_at generado por Clock, no del evento)
    - _Correctness Properties: P7 (Idempotence), P8 (Timestamps)_
    - _RF: RF-02, RF-04, RF-05_

### Wave 8: Verificación y Limpieza

- [x] 15. Verificación Arquitectura
  - [x] 15.1 Test: `internal/worker/architecture_test.go` — falla si `domain/` importa: Kafka, `database/sql`, HTTP, `adapters/`, `consumer/`; falla si `application/usecases/` importa: `adapters/`, `consumer/`, implementaciones concretas
    - _RF: RNF-01_

- [x] 16. Compilación y Tests
  - [x] 16.1 Verificar que `go build ./cmd/worker` compila sin errores
  - [x] 16.2 Verificar que `go test ./...` pasa (< 60s)
  - [x] 16.3 Verificar que `go test -tags=integration ./...` pasa (opcional, con Testcontainers)

## Notes

### Correctness Properties

- **P1 (Validation)**: Todos los campos validados en entrada, rechazo de valores inválidos → dead-letter
- **P2 (Async+Commit)**: HTTP 202 retornado antes de confirmación de Kafka; offset commit solo después de garantía
- **P3 (Dedup)**: Unique constraint `(captured_at, event_id)` + `ON CONFLICT DO NOTHING`
- **P4 (Error-Classification)**: Recuperable vs no recuperable correctamente clasificados
- **P5 (Resilience)**: Circuit Breaker transiciones CLOSED → OPEN → HALF_OPEN → CLOSED
- **P6 (No-Loss)**: Offset nunca confirmado antes de garantía de persistencia o dead-letter
- **P7 (Idempotence)**: Procesar N veces = procesar 1 vez (mismo estado final)
- **P8 (Timestamps)**: captured_at ≤ received_at ≤ processed_at, stored_at generado por Clock

### Offset Commit Rules

| Resultado | ¿Commit? | Razón |
|-----------|----------|-------|
| INSERT exitoso | Sí | Persistencia confirmada |
| Duplicado (ON CONFLICT) | Sí | Garantía ya satisfecha |
| Erro no recuperable + dead-letter OK | Sí | Registrado para diagnóstico |
| Error recuperable agotado + dead-letter OK | Sí | Registrado |
| Error recuperable agotado + dead-letter FALLA | No | Sin pérdida silenciosa |
| Circuit OPEN | No | Pausa consumo |

### Test Data

WorkerPositionMother genera datos determinísticos:
- `Valid()` → entrada válida con todos los campos correctos
- `OutOfRange()` → latitud/longitud fuera de rango
- `MissingReceivedAt()` → received_at ausente
- `MalformedPayload()` → JSON inválido

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"], "label": "Migration" },
    { "id": 1, "tasks": ["2.1", "2.2", "2.3", "2.4", "3.1", "3.2", "3.3"], "label": "Domain + Config" },
    { "id": 2, "tasks": ["4.1", "4.2", "4.3", "4.4"], "label": "Ports" },
    { "id": 3, "tasks": ["5.1", "5.2", "5.3"], "label": "Application + Test Support" },
    { "id": 4, "tasks": ["6.1", "7.1", "8.1", "9.1"], "label": "Adapters Outbound" },
    { "id": 5, "tasks": ["10.1", "10.2"], "label": "Kafka Consumer" },
    { "id": 6, "tasks": ["11.1", "12.1", "12.2", "12.3", "12.4"], "label": "Config + Bootstrap" },
    { "id": 7, "tasks": ["13.1", "13.2", "14.1", "14.2", "15.1", "16.1", "16.2", "16.3"], "label": "Tests + Verification" }
  ]
}
```
