# Documento de Diseño: Telemetry Worker

## Overview

El Telemetry Worker es un servicio Go independiente (proceso `cmd/worker`) que cierra el flujo de ingesta asincrónica de telemetría. Consume eventos `PositionReceived_Event` desde `positions_topic` de Kafka, valida los datos requeridos de cada evento, deduplica por `event_id` y persiste las posiciones válidas en la hypertable `telemetry.positions` de TimescaleDB. El offset de Kafka se confirma únicamente después de que la garantía de persistencia (INSERT confirmado, deduplicación resuelta o derivación exitosa al Dead_Letter_Sink) quede satisfecha.

El diseño prioriza la **integridad de los datos** sobre el rendimiento máximo. Las propiedades centrales son:

- **Idempotencia**: procesar el mismo evento N veces deja el mismo estado final que procesarlo una vez.
- **Deduplicación por `event_id`**: unique constraint en la base de datos sobre `(captured_at, event_id)` + `ON CONFLICT (captured_at, event_id) DO NOTHING`.
- **Sin pérdida silenciosa**: el `Offset_Commit` nunca ocurre antes de satisfacer la garantía de persistencia.
- **Resiliencia**: clasificación de errores recuperables/no recuperables, reintentos con backoff determinista y Circuit Breaker sobre TimescaleDB.

### Posición en el flujo de ingesta asincrónica

```
[App móvil] --HTTP batch--> [Ingestion service] --publish PositionReceived--> [Kafka: positions_topic]
                                                                                      |
                                                                                      v
                                                          [Telemetry_Worker: consume -> valida -> dedup/persist -> commit]
                                                                                      |
                                                                                      v
                                                                        [TimescaleDB: telemetry.positions]
```

## Arquitectura Clean Architecture

El worker sigue Clean Architecture (RNF-01). Todo el código vive bajo `backend/go/internal/worker/` organizado en capas:

```
backend/go/internal/worker/
├── domain/                      # Reglas, invariantes, value objects
│   ├── errors.go                # AppError con clasificación
│   ├── consumed_event.go        # ConsumedPositionEvent
│   ├── validation.go            # Validación de campos
│   ├── dead_letter.go           # DeadLetterRecord
│   └── trace.go                 # TraceID (existente)
│
├── ports/                       # Interfaces de puertos (nivel superior)
│   ├── config.go                # Config, ConfigLoader
│   ├── logging.go               # Logger
│   ├── repository.go            # PositionRepository
│   ├── clock.go                 # Clock
│   ├── circuit_breaker.go       # CircuitBreaker
│   └── dead_letter_sink.go      # DeadLetterSink
│
├── application/
│   └── usecases/
│       └── process_position_event.go # ProcessPositionEvent
│
├── consumer/                    # Adaptador entrada (nivel superior)
│   ├── consumer.go              # Consumer Kafka
│   └── runner.go                # Runner con lógica de offset
│
├── adapters/outbound/
│   ├── persistence/
│   │   └── timescale_repository.go
│   ├── resilience/
│   │   └── circuit_breaker.go
│   ├── deadletter/
│   │   └── log_sink.go
│   └── infrastructure/
│       └── clock.go
│
└── config/                      # Wiring (sin dominio)
    ├── loader.go
    ├── validator.go
    └── composer.go

cmd/worker/
└── main.go                      # Bootstrap
```

### Reglas de dependencia

- `domain/` no importa Kafka, SQL, HTTP ni `adapters/`
- `application/usecases` solo importa `domain/` y `ports/`
- `consumer/` (adaptador entrada) importa `ports/` y `application/`
- `adapters/` implementan los puertos
- `config/` solo instancia e inyecta

## Componentes

### Puertos de salida (interfaces)

**PositionRepository**: persistencia idempotente
- `Save(ctx context.Context, p PersistablePosition) (SaveResult, error)`
- `Close() error`
- `SaveResult`: SaveResultInserted | SaveResultDuplicate

**Clock**: reloj inyectable
- `Now() time.Time`

**CircuitBreaker**: resiliencia
- `Execute(ctx context.Context, fn func() error) error`
- `State() CircuitBreakerState`
- Estados: CLOSED, OPEN, HALF_OPEN

**DeadLetterSink**: registro de descartes
- `Write(ctx context.Context, record domain.DeadLetterRecord) error`

**Logger**: (existente)
- Logs JSON estructurados sin secretos ni lat/lon

### Caso de uso: ProcessPositionEvent

```go
type ProcessPositionEvent struct {
    repo   ports.PositionRepository
    cb     ports.CircuitBreaker
    clock  ports.Clock
    sink   ports.DeadLetterSink
    logger ports.Logger
}

type ProcessResult struct {
    Commit bool
    Paused bool
}

func (uc *ProcessPositionEvent) Process(ctx context.Context, rawPayload []byte) ProcessResult
```

Pipeline:
1. Parsear JSON (malformado → no recuperable → dead-letter → commit)
2. Validar (Requisito 2)
3. Construir PersistablePosition con `stored_at` del Clock en UTC ms
4. Persistir bajo Circuit Breaker + timeout, reintentos con backoff 100/200/400ms determinista
5. Mapear resultado a ProcessResult según reglas de offset commit

### Adaptador entrada: Kafka Consumer

**Consumer**: envoltorio de `segmentio/kafka-go`
- Auto-commit OFF
- Commit explícito solo cuando `ProcessResult.Commit == true`

**Runner**: bucle de consumo
- Procesamiento secuencial por partición (preserva orden por vehicle_id)
- Rebalanceo: completa trabajo en curso dentro de WORKER_SHUTDOWN_TIMEOUT
- Reconexión con Backoff_Exponencial
- Pausa fetch mientras Circuit Breaker OPEN
- Log INFO al reanudar con el offset

## Modelo de Dominio

### ConsumedPositionEvent
- `EventID` (UUID v4, requerido)
- `VehicleID` (alfanumérico 1-50, requerido)
- `Latitude` ([-90,90], requerido)
- `Longitude` ([-180,180], requerido)
- `CapturedAt` (ISO 8601 UTC, requerido)
- `ReceivedAt` (*time.Time, opcional → NULL)
- `ProcessedAt` (*time.Time, opcional → NULL)
- `Accuracy` (*float64, opcional → NULL)
- `TraceID` (*string, opcional, propagar o generar)

### DeadLetterRecord
- TraceID, EventID, VehicleID (puede estar vacío si payload malformado)
- Classification: "recoverable" | "non_recoverable"
- ReasonCategory: stable, p.ej. "invalid_event", "malformed_payload"
- OccurredAt: ISO 8601 UTC ms

## Schema de TimescaleDB

La migración `000002` editada crea `telemetry.positions`:

```sql
CREATE TABLE telemetry.positions (
    id               UUID NOT NULL DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL,
    vehicle_id       VARCHAR(50) NOT NULL,
    latitude         NUMERIC(9, 6) NOT NULL,
    longitude        NUMERIC(9, 6) NOT NULL,
    accuracy         NUMERIC(5, 2),
    captured_at      TIMESTAMPTZ NOT NULL,
    received_at      TIMESTAMPTZ,
    processed_at     TIMESTAMPTZ,
    stored_at        TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX positions_event_id_key
    ON telemetry.positions (captured_at, event_id);

SELECT create_hypertable('telemetry.positions', 'captured_at', if_not_exists => TRUE);

CREATE INDEX positions_vehicle_captured_at_idx
    ON telemetry.positions (vehicle_id, captured_at DESC);
```

**Decisión**: Editar `000002` en lugar de crear `000004` porque:
- Sin producción ni datos que preservar
- Bases efímeras arrancan de cero
- Esquema final más limpio (sin columnas legadas)

**Caveat operativo**: desarrolladores con base local preexistente deben recrearla.

## Configuración

Claves `WORKER_*`:

| Clave | Defecto | Requerido | Descripción |
|-------|---------|-----------|-------------|
| `WORKER_KAFKA_BROKERS` | - | Sí | Host1:Puerto,Host2:Puerto |
| `WORKER_KAFKA_TOPIC` | positions | No | Nombre del topic |
| `WORKER_KAFKA_CONSUMER_GROUP` | telemetry-worker | No | Consumer group |
| `WORKER_RETRY_MAX` | 3 | No | Máximo de reintentos (0-10) |
| `WORKER_RETRY_BASE_BACKOFF` | 100ms | No | Backoff inicial |
| `WORKER_CB_FAILURE_THRESHOLD` | 5 | No | Umbral para abrir breaker |
| `WORKER_CB_RECOVERY_INTERVAL` | 30s | No | Intervalo de recuperación |
| `WORKER_DB_DSN` | - | Sí | PostgreSQL connection string |
| `WORKER_DB_OPERATION_TIMEOUT` | 5s | No | Timeout de operación |
| `WORKER_SHUTDOWN_TIMEOUT` | 30s | No | Timeout de shutdown |

## Error Handling

**Errores Recuperables**: timeout DB, conexión rechazada, dependencia no disponible
- Acción: reintentar hasta WORKER_RETRY_MAX con backoff determinista
- Si agotados: dead-letter y commit offset

**Errores No Recuperables**: evento inválido, payload malformado
- Acción: zero reintentos, dead-letter inmediato

**Circuit Breaker**:
- CLOSED → OPEN: ≥ WORKER_CB_FAILURE_THRESHOLD fallos recuperables consecutivos
- OPEN: pausa fetch Kafka, no confirma offsets
- OPEN → HALF_OPEN: tras WORKER_CB_RECOVERY_INTERVAL
- HALF_OPEN → CLOSED: prueba exitosa
- HALF_OPEN → OPEN: prueba fallida

## Offset Commit Rules

| Resultado | ¿Commit? | Razón |
|-----------|----------|-------|
| INSERT exitoso | Sí | Persistencia confirmada |
| Duplicado (ON CONFLICT) | Sí | Garantía ya satisfecha |
| Error no recuperable + dead-letter OK | Sí | Registrado para diagnóstico |
| Error recuperable agotado + dead-letter OK | Sí | Registrado |
| Error recuperable agotado + dead-letter FALLA | No | Sin pérdida silenciosa |
| Circuit OPEN | No | Pausa consumo |

## Observabilidad

**Logs JSON estructurados**:
- SUCCESS: INFO con trace_id, event_id, vehicle_id
- ERROR: ERROR/WARN con trace_id, event_id, vehicle_id, clasificación

**NO registra**: latitude, longitude, SQL, stack traces, secretos, DSN

**Trace ID**: propagar si UUID v4 válido; generar si ausente o inválido

## Decisiones y Tradeoffs

- **Dead_Letter_Sink como log JSON**: simple, MVP; un dead-letter topic de Kafka es trabajo futuro
- **`stored_at` generado por Clock**: habilita determinismo en pruebas
- **At-least-once + persistencia idempotente = effectively-once**: no se afirma exactly-once
- **Reutilización de value objects de ingestion**: evita reglas contradictorias sin duplicación
- **Edición de `000002`**: esquema final alineado sin migración hacia adelante
- **Unique constraint `(captured_at, event_id)`**: requisito de TimescaleDB; `event_id` sigue siendo clave conceptual

