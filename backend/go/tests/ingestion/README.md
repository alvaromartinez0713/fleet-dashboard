# Módulo Ingestion - Documentación de Pruebas

Pruebas para el módulo **HTTP Position Ingestion**, que maneja la recepción de coordenadas de vehículos mediante HTTP API y las publica de forma asíncrona a Kafka.

Ver [../README.md](../README.md) para la organización general de pruebas y comandos de ejecución.

## Visión General del Módulo

**Responsabilidad:** Aceptar telemetría de posiciones mediante HTTP POST, validarla, aplicar rate limiting, publicar a Kafka, retornar HTTP 202 Accepted.

**Comportamientos clave probados:**
- Validación de entrada (coordenadas, timestamps, vehicle ID)
- Respuesta asíncrona (202 antes de confirmación de Kafka)
- Rate limiting por vehículo (12 requests/60s ventana deslizante)
- Protección Circuit Breaker contra fallos de Kafka
- RFC 9457 Problem Details para todos los errores
- Ventana de idempotencia (deduplicación de 1 hora)

## Estructura de Directorios

```
ingestion/
├── README.md                  # Este archivo
├── suite-fast/                # Pruebas rápidas (17 pruebas, <1s)
│   ├── validation_test.go     # Validación de entrada (errores 422)
│   ├── valid_position_test.go # Caso feliz (respuesta 202)
│   ├── async_test.go          # Comportamiento asíncrono (202 inmediato)
│   ├── rate_limit_test.go     # Rate limiting (errores 429)
│   ├── circuit_breaker_test.go # Circuit Breaker (errores 503)
│   ├── middleware_test.go     # Stack de middleware (headers, trace ID)
│   ├── timestamp_test.go      # Modelo de 4 timestamps
│   ├── properties_latency_test.go # Property-based: monotonía de latencia
│   └── properties_rate_limit_test.go # Property-based: ventana deslizante
└── suite-integration/         # Pruebas de integración (8 pruebas, <120s)
    ├── kafka_test.go          # Schema y particionamiento de Kafka
    ├── resilience_test.go     # Inyección de fallos con Toxiproxy
    ├── idempotency_test.go    # Ventana de deduplicación de 1 hora
    ├── rate_limit_test.go     # Requests concurrentes, aislamiento de cuota
    └── async_test.go          # 202 sin persistencia del worker
```

## Suite Fast (17 pruebas, <1s)

### Pruebas de Validación (validation_test.go)
- Campos requeridos faltantes (vehicle_id, latitude, longitude, captured_at) → 422
- Coordenadas fuera de rango (lat > 90, long < -180) → 422
- Timestamps futuros (captured_at > hora del servidor) → 422
- Formato de timestamp inválido (no ISO 8601) → 422
- JSON malformado → 400
- Tamaño de request > 10 KB → 413

### Comportamiento Asíncrono (async_test.go)
- Posición válida retorna 202 inmediatamente (<100ms) sin esperar a Kafka
- Respuesta incluye event_id, trace_id, captured_at, received_at, processed_at

### Rate Limiting (rate_limit_test.go)
- 12 requests/60s por vehículo → todas aceptadas (202)
- 13º request dentro de 60s → rechazado (429) con headers X-RateLimit-*
- Vehículos diferentes tienen cuotas independientes

### Circuit Breaker (circuit_breaker_test.go)
- 5 fallos consecutivos de Kafka → Circuit Breaker OPEN → respuesta 503
- Después de 30s → HALF_OPEN → request de prueba
- 2 éxitos → CLOSED → operación normal se reanuda

### Pruebas Property-Based (properties_*.go)
- **Monotonía de latencia:** processingLatency ≥ receptionLatency (100+ iteraciones)
- **Ventana deslizante:** Cualquier ventana de 60s acepta ≤ 12 requests por vehículo (100+ iteraciones)

## Suite Integration (8 pruebas, <120s)

### Contrato Kafka (kafka_test.go)
- Posición válida publicada a Kafka real (Testcontainers)
- Verifica: schema, partition key (vehicle_id), avance de offset, timestamps

### Resiliencia (resilience_test.go)
- **Kafka no disponible:** Toxiproxy corta la conexión → 503, Circuit Breaker OPEN
- **Timeout de Kafka:** Toxiproxy 2s de latencia + 1s timeout → 503, respuesta <1.5s

### Idempotencia (idempotency_test.go)
- Duplicado dentro de ventana de 1h → segundo request rechazado
- Después de 61 minutos → duplicado aceptado (ventana expirada)

### Rate Limiting (rate_limit_test.go)
- **Concurrente mismo vehículo:** 13 requests paralelos → 12 aceptados, 1 rechazado (429)
- **Concurrente vehículos diferentes:** VH-A y VH-B tienen cuotas independientes
- **Reset de ventana:** Después de 61 segundos, la cuota se regenera

### Persistencia Asíncrona (async_test.go)
- Respuesta 202 <500ms, evento en Kafka
- SIN worker: sin persistencia en TimescaleDB (confirma async)

## Datos de Prueba (Object Mothers)

Ubicados en `../support/mothers/position_mother.go`.

**Escenarios:**
```go
// Posición válida (coordenadas por defecto, timestamp válido)
position := mothers.PositionMother{}.Valid().Build()

// Latitud fuera de rango
position := mothers.PositionMother{}.OutOfRangeLatitude().Build()

// Timestamp futuro
position := mothers.PositionMother{}.FutureTimestamp().Build()

// Sobrescrituras personalizadas
position := mothers.PositionMother{}.Valid()
    .WithVehicleID("VH-001")
    .WithLatitude(40.7128)
    .WithLongitude(-74.0060)
    .Build()
```

## Ejecución de Pruebas

```bash
# Solo pruebas rápidas (desde raíz del backend)
make test-ingestion-fast

# Pruebas de integración (requiere Docker)
make test-ingestion-integration

# Archivo de prueba específico
cd tests/ingestion && go test -v ./suite-fast/validation_test.go

# Función de prueba específica
cd tests/ingestion && go test -v -run TestValidPositionReturns202 ./suite-fast/

# Pruebas property-based con salida verbose de rapid
cd tests/ingestion && go test -v -run TestLatencyMonotonicity ./suite-fast/
```

## Cobertura de Criterios de Aceptación

Todos los criterios de aceptación de `requirements.md` están verificados por pruebas:

| Criterio | Prueba(s) | Suite |
|-----------|---------|-------|
| P1: Validación entrada | validation_test.go (9 pruebas) | Fast |
| P2: Async + 202 | async_test.go, valid_position_test.go | Fast |
| P3: Circuit Breaker | circuit_breaker_test.go, resilience_test.go | Fast + Integration |
| P4: 4-Timestamps | timestamp_test.go, kafka_test.go | Fast + Integration |
| P5: Errores RFC 9457 | validation_test.go, rate_limit_test.go | Fast |
| P6: Rate limit | rate_limit_test.go (ambas suites) | Fast + Integration |
| P7: Schema Kafka | kafka_test.go | Integration |
| P8: Idempotencia | idempotency_test.go | Integration |
| P9: Trace-ID | middleware_test.go | Fast |

## Limitaciones Conocidas

- La deduplicación de idempotencia NO está implementada (X-Idempotency-Key validado pero ignorado)
- El timestamp stored_at NO está en la respuesta HTTP 202 (agregado por el worker)
- La persistencia en TimescaleDB se prueba solo mediante las pruebas del módulo worker

## Referencias

- [requirements.md](../../../.kiro/specs/http-position-ingestion/requirements.md) - Criterios de aceptación
- [design.md](../../../.kiro/specs/http-position-ingestion/design.md) - Arquitectura y dependencias
- [api-standards.md](../../../.kiro/steering/api-standards.md) - RFC 9457 Problem Details
