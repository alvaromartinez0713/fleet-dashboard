# Design Document: HTTP Position Ingestion Service (COMPLETO)

## Executive Summary

Este documento especifica el diseño del **HTTP Position Ingestion Service**, el componente responsable de recibir, validar y publicar posiciones de vehículos en Kafka de forma asincrónica. 

El servicio implementa una arquitectura de **Clean Architecture COMPLETO** con separación explícita de capas (Domain, Application, HTTP Adapters, Outbound Adapters, Ports, DTOs, Mappers), con énfasis en:
- **Rapidez**: respuesta HTTP 202 en < 100ms sin esperar Kafka
- **Resiliencia**: reintentos limitados, Circuit Breaker, dead-letter logging
- **Auditoría**: 4 timestamps por evento, trazabilidad completa con trace_id
- **Validación explícita**: contrato HTTP riguroso, RFC 9457 Problem Details
- **Rate Limiting**: protección por vehicle_id sin Redis (in-memory, rolling window 60s, 12/min default)
- **Separación Clara**: cada carpeta una responsabilidad, DTOs+Mappers centralizados, dominio protegido

Requisitos cubiertos: **RF-01 a RF-13** (requirements.md), incluyendo **RF-10 Arquitectura COMPLETO** y **RF-13 Rate Limiting in-memory**.

---

## Overview

### Propósito

El servicio HTTP de ingesta es el **punto de entrada único** para posiciones de vehículos en el sistema. Su responsabilidad es:

1. **Aceptar** solicitudes HTTP POST de clientes (móvil, simulador, externos)
2. **Validar** contrato explícito (tipos, formatos, límites)
3. **Publicar asincrónicamenteEnqueue** en Kafka sin bloquear respuesta
4. **Responder rápidamente** con HTTP 202 (Aceptado)
5. **Rastrear completamente** cada solicitud con trace_id y 4 timestamps
6. **Proteger** el sistema con Circuit Breaker, reintentos y rate limiting
7. **Separar claramente** domain, application, adapters, ports y DTOs en capas independientes

### Responsabilidades por Capa

**Domain (`domain/`):**
- Lógica pura de negocio (Value Objects, Entity, Errors)
- Sin dependencias a HTTP, Kafka, DTOs, frameworks
- Validaciones de invariantes
- Métodos expresivos del dominio

**Application (`application/`):**
- Orquestación de casos de uso (Commands, UseCases)
- Construcción de entidades a partir de comandos
- Coordinación de puertos (interfaces)
- Sin HTTP, sin DTOs, sin persistencia directa

**HTTP Adapter (`adapters/inbound/http/`):**
- Handler HTTP (thin, ~15 líneas)
- 5 Middleware (TraceID, Idempotency, ContentType, RateLimit, ErrorHandler)
- Responsabilidad: HTTP → Application

**Outbound Adapters (`adapters/outbound/`):**
- Implementaciones de puertos (KafkaPublisher, CircuitBreaker, RateLimiter, Clock)
- Responsabilidad: Application → externos

**Ports (`ports/`):**
- 6 interfaces (contratos)
- Definen qué necesita la Application de componentes externos

**HTTP Layer (`http/`):**
- DTOs: PositionRequest, PositionResponse, ProblemDetails
- Mappers: PositionMapper, ErrorMapper
- Centralizados y reutilizables
- Protegen el dominio de HTTP

**Config (`config/`):**
- Composición de dependencias
- Instancia todas las capas
- Conecta interfaces (puertos) con implementaciones (adapters)

---

## Architecture

### Estructura de Carpetas COMPLETO

```
backend/go/internal/ingestion/
│
├── domain/                           # DOMINIO: Lógica pura de negocio
│   ├── vehicle_id.go                # VehicleID (Value Object)
│   ├── geo_position.go              # GeoPosition (Value Object)
│   ├── captured_timestamp.go        # CapturedTimestamp (Value Object)
│   ├── received_timestamp.go        # ReceivedTimestamp (Value Object)
│   ├── processed_timestamp.go       # ProcessedTimestamp (Value Object)
│   ├── event_id.go                  # EventID (Value Object)
│   ├── trace_id.go                  # TraceID (Value Object)
│   ├── idempotency_key.go           # IdempotencyKey (Value Object)
│   ├── rate_limit_config.go         # RateLimitConfig (Value Object)
│   ├── vehicle_position.go          # VehiclePosition (Entity)
│   └── errors.go                    # Domain errors
│
├── application/                      # APPLICATION: Orquestación
│   ├── commands/
│   │   └── receive_position.go      # ReceivePositionCommand (DTO interno)
│   └── usecases/
│       └── receive_position.go      # ReceivePositionUseCase
│
├── adapters/                         # ADAPTERS: Conversión entre capas
│   ├── inbound/
│   │   └── http/
│   │       ├── handler.go           # PositionHandler (~15 líneas)
│   │       ├── router.go            # Rutas Gin
│   │       └── middleware/
│   │           ├── trace_id.go      # TraceIDMiddleware
│   │           ├── idempotency.go   # IdempotencyKeyMiddleware
│   │           ├── content_type.go  # ContentTypeMiddleware
│   │           ├── rate_limit.go    # RateLimitMiddleware
│   │           └── error_handler.go # ErrorHandlerMiddleware
│   └── outbound/
│       ├── messaging/
│       │   └── kafka_publisher.go   # KafkaPublisherImpl (publishes, retries, dead-letter)
│       ├── resilience/
│       │   ├── circuit_breaker.go   # CircuitBreakerImpl (CLOSED/OPEN/HALF_OPEN)
│       │   └── rate_limiter.go      # RateLimiterImpl (rolling window, 12/min)
│       └── time/
│           └── clock.go             # ClockImpl (wraps time.Now())
│
├── ports/                            # PORTS: Interfaces (contratos)
│   ├── kafka_publisher.go           # KafkaPublisher interface
│   ├── circuit_breaker_manager.go   # CircuitBreakerManager interface
│   ├── rate_limiter.go              # RateLimiter interface
│   ├── logger.go                    # Logger interface
│   ├── clock.go                     # Clock interface
│   └── id_generator.go              # IDGenerator interface
│
├── http/                             # HTTP LAYER: DTOs + Mappers
│   ├── dto/
│   │   ├── request.go               # PositionRequest DTO
│   │   └── response.go              # PositionResponse DTO, ProblemDetails DTO
│   └── mapper/
│       ├── position_mapper.go       # PositionMapper (ToCommand, FromEntity)
│       └── error_mapper.go          # ErrorMapper (ToProblemDetails)
│
└── config/                           # CONFIG: Composición de dependencias
    └── composer.go                  # Instancia todo, inyecta dependencias
```

### Diagrama: Clean Architecture COMPLETO

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ HTTP TRANSPORT LAYER (adapters/inbound/http/)                              │
│                                                                             │
│ ┌──────────────────────────────────────────────────────────────────────┐   │
│ │ MIDDLEWARE PIPELINE (5 Middleware)                                  │   │
│ │                                                                      │   │
│ │ 1. TraceIDMiddleware:        Valida/genera X-Trace-ID (UUID v4)    │   │
│ │ 2. IdempotencyKeyMiddleware: Valida X-Idempotency-Key (UUID v4)    │   │
│ │ 3. ContentTypeMiddleware:    Valida Content-Type: application/json  │   │
│ │ 4. RateLimitMiddleware:      Verifica límite por vehicle_id (12/min)│   │
│ │ 5. ErrorHandlerMiddleware:   Mapea errores a RFC 9457 Problem Details
│ │                                                                      │   │
│ │ Result: HTTP request → Contexto enriquecido (trace_id, headers)    │   │
│ └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│ ┌──────────────────────────────────────────────────────────────────────┐   │
│ │ PositionHandler (handler.go, ~15 líneas)                            │   │
│ │                                                                      │   │
│ │ 1. Gin.ShouldBindJSON(request) → PositionRequest DTO (untrusted)  │   │
│ │ 2. Extract headers from context (set by middleware)                 │   │
│ │ 3. PositionMapper.ToCommand() → ReceivePositionCommand (internal) │   │
│ │ 4. useCase.Execute(command)                                         │   │
│ │ 5. PositionMapper.FromEntity(entity) → PositionResponse DTO        │   │
│ │ 6. Gin.JSON(202, response)                                          │   │
│ └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓ PositionRequest DTO                          ↑ PositionResponse DTO

┌─────────────────────────────────────────────────────────────────────────────┐
│ HTTP LAYER: DTOs & MAPPERS (http/dto/ + http/mapper/)                      │
│                                                                             │
│ DTOs: PositionRequest, PositionResponse, ProblemDetails                    │
│ Mappers: PositionMapper.ToCommand(), .FromEntity()                         │
│          ErrorMapper.ToProblemDetails()                                    │
│                                                                             │
│ Función: Protegen dominio, traducen entre contextos, reutilizables        │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓ ReceivePositionCommand (interno)              ↑ Entity (dominio)

┌─────────────────────────────────────────────────────────────────────────────┐
│ APPLICATION LAYER (application/usecases/ + application/commands/)           │
│                                                                             │
│ ReceivePositionCommand (DTO de aplicación)                                 │
│ ReceivePositionUseCase (orquesta, valida, construye Entity)                │
│                                                                             │
│ Workflow: Command → VO → Entity → Valida CB → Publica Async → Retorna    │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓ 8 Value Objects + 1 Entity                   ↑ Entity (dominio)

┌─────────────────────────────────────────────────────────────────────────────┐
│ DOMAIN LAYER (domain/)                                                      │
│                                                                             │
│ 8 Value Objects: VehicleID, GeoPosition, CapturedTimestamp,               │
│                  ReceivedTimestamp, ProcessedTimestamp, EventID,           │
│                  TraceID, IdempotencyKey, RateLimitConfig                 │
│                                                                             │
│ 1 Entity: VehiclePosition (orquesta VO, garantiza invariantes)            │
│                                                                             │
│ Errors: Domain errors (no HTTP, sin DTOs)                                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ PORTS LAYER (ports/ — interfaces públicas)                                 │
│                                                                             │
│ 6 Interfaces: KafkaPublisher, CircuitBreakerManager, RateLimiter,          │
│               Logger, Clock, IDGenerator                                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ OUTBOUND ADAPTERS (adapters/outbound/ — implementan puertos)               │
│                                                                             │
│ KafkaPublisherImpl, CircuitBreakerImpl, RateLimiterImpl, ClockImpl            │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ CONFIG LAYER (config/composer.go)                                           │
│                                                                             │
│ Instancia dependencias, conecta puertos con adapters                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow: Del Cliente al Dominio y de Vuelta

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ CLIENT REQUEST (HTTP POST /api/v1/positions)                                │
│ Headers: Content-Type: application/json, X-Trace-ID (opcional)              │
│ Body: { "vehicle_id": "VH-001", "latitude": 40.7128, ... }                 │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓

┌─────────────────────────────────────────────────────────────────────────────┐
│ MIDDLEWARE PIPELINE (5 layers)                                              │
│                                                                             │
│ 1. TraceIDMiddleware         → Genera/valida X-Trace-ID (UUID v4)          │
│ 2. IdempotencyKeyMiddleware  → Valida X-Idempotency-Key (UUID v4) si existe
│ 3. ContentTypeMiddleware     → Valida Content-Type: application/json        │
│ 4. RateLimitMiddleware       → Verifica límite por vehicle_id (12/min)     │
│ 5. ErrorHandlerMiddleware    → Captura errores, mapea a RFC 9457           │
│                                                                             │
│ Resultado: Contexto enriquecido (trace_id, headers, límites)               │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓

┌─────────────────────────────────────────────────────────────────────────────┐
│ POSITION HANDLER (~15 líneas)                                               │
│                                                                             │
│ 1. Gin.ShouldBindJSON(request) → PositionRequest DTO (untrusted)          │
│ 2. Extract headers from context (set by middleware)                         │
│ 3. PositionMapper.ToCommand() → ReceivePositionCommand (internal DTO)      │
│ 4. useCase.Execute(command)                                                 │
│    ├─ [Inside UseCase...]                                                  │
│    ├─ Crea 8 Value Objects (validación dominio)                            │
│    ├─ Crea VehiclePosition Entity                                          │
│    ├─ Entity emite PositionReceivedEvent (Domain Event) ← DOMAIN EVENT    │
│    ├─ UseCase captura event de Entity                                      │
│    ├─ UseCase verifica Circuit Breaker                                     │
│    └─ UseCase publica async a Kafka                                        │
│    └─ Retorna Entity (si éxito) o error (si fallo)                        │
│    └─ [End UseCase...]                                                     │
│ 5. Si error en Domain/Application:                                         │
│    └─ PositionMapper NUNCA se llama                                        │
│    └─ HTTP Layer ErrorMapper.ToProblemDetails(error)                       │
│    └─ Retorna RFC 9457 ProblemDetails + status (400/422/429/503/500)      │
│ 6. Si éxito:                                                                │
│    ├─ PositionMapper.FromEntity(entity) → PositionResponse DTO            │
│    └─ Gin.JSON(202, response)                                              │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓

┌──────────────────────────────────────────────────────────────────────────────┐
│ RESPONSE TO CLIENT                                                           │
│                                                                              │
│ If Error (400/422/429/503/500):                                             │
│ {                                                                            │
│   "type": "urn:fleet:error:...",                                            │
│   "title": "...",                                                           │
│   "status": 400/422/429/503/500,                                            │
│   "detail": "Human-readable message (sin detalles técnicos)",               │
│   "instance": "/problems/{type}/{trace_id}"                                 │
│ }                                                                            │
│                                                                              │
│ If Success (202):                                                           │
│ {                                                                            │
│   "status": "accepted",                                                     │
│   "trace_id": "550e8400-e29b-41d4-a716-446655440000",                      │
│   "vehicle_id": "VH-001",                                                   │
│   "event_id": "12345678-1234-5678-1234-567812345678",                      │
│   "received_at": "2024-10-17T14:30:45.456Z"                                │
│ }                                                                            │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Flujo Lineal (Éxito):**
```
HTTP POST → Middleware (5 pasos) → Handler → Mapper.ToCommand()
  → UseCase.Execute()
    → Crea VO → Crea Entity → Emite Event → Verifica CB → Publica Kafka (async)
  → Mapper.FromEntity() → Response JSON 202
```

**Flujo Lineal (Error):**
```
HTTP POST → Middleware (5 pasos) → Handler → Mapper.ToCommand()
  → UseCase.Execute()
    → Error en VO creation → Propaga error hacia HTTP
  → ErrorMapper.ToProblemDetails(error)
  → Response JSON 400/422/429/503/500 + RFC 9457
```

---

## HTTP Layer: DTOs & Mappers

Esta capa **protege el dominio** y **traduce entre contextos** (HTTP ↔ Application ↔ Domain).

### 1. Domain Events in Design

#### PositionReceivedEvent

Domain Event emitido cuando VehiclePosition Entity es creada exitosamente.

```go
// domain/events.go

type PositionReceivedEvent struct {
    vehicleID    VehicleID
    eventID      EventID
    capturedAt   CapturedTimestamp
    receivedAt   ReceivedTimestamp
    processedAt  ProcessedTimestamp
    traceID      TraceID
    occurredAt   time.Time
}

// Value Object methods
func (e *PositionReceivedEvent) AggregateID() string {
    return e.vehicleID.String()
}

func (e *PositionReceivedEvent) EventType() string {
    return "PositionReceived"
}

func (e *PositionReceivedEvent) OccurredAt() time.Time {
    return e.occurredAt
}
```

**Responsabilidad:**
- Emitido por `VehiclePosition` Entity cuando se crea exitosamente
- Desacoplado de Kafka, HTTP, base de datos
- Contiene datos necesarios para tomar decisiones en Application Layer
- Application consume el event (ej: decide publicar a Kafka)

**Ubicación:** `domain/events.go`

**Entity Method:**

```go
type VehiclePosition struct {
    // ... fields ...
    events []PositionReceivedEvent
}

func (v *VehiclePosition) Events() []PositionReceivedEvent {
    return v.events
}
```

---

### 2. Domain Errors

Domain errors son específicos, expresivos del dominio, sin detalles técnicos.

```go
// domain/errors.go

type FutureTimestampError struct {
    Value time.Time
}

func (e FutureTimestampError) Error() string {
    return fmt.Sprintf("captured_at cannot be in the future: %v", e.Value)
}

type InvalidVehicleIDError struct {
    Value string
    Reason string
}

func (e InvalidVehicleIDError) Error() string {
    return fmt.Sprintf("invalid vehicle_id: %s", e.Reason)
}

type InvalidGeoPositionError struct {
    Latitude  *float64
    Longitude *float64
    Reason    string
}

func (e InvalidGeoPositionError) Error() string {
    return fmt.Sprintf("invalid geolocation: %s", e.Reason)
}

type InvalidEventIDError struct {
    Value string
}

func (e InvalidEventIDError) Error() string {
    return "event_id must be valid UUID v4"
}

type InvalidTraceIDError struct {
    Value string
}

func (e InvalidTraceIDError) Error() string {
    return "X-Trace-ID header must be valid UUID v4"
}
```

---

### 3. Error Handling Flow

#### Flujo de Errores: Domain → Application → HTTP

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ CLIENT REQUEST (HTTP POST)                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ HTTP LAYER (adapters/inbound/http/)                                         │
│                                                                             │
│ 1. Middleware pipeline (TraceID, ContentType, RateLimit, etc)              │
│ 2. PositionHandler deserialized JSON → PositionRequest DTO                 │
│ 3. PositionMapper.ToCommand() → ReceivePositionCommand                     │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ APPLICATION LAYER (application/usecases/)                                   │
│                                                                             │
│ ReceivePositionUseCase.Execute(command)                                    │
│   ├─ Crea Value Objects                                                    │
│   │   ├─ VehicleID     ← puede fallar → FutureTimestampError              │
│   │   ├─ GeoPosition   ← puede fallar → InvalidGeoPositionError           │
│   │   ├─ Timestamps    ← puede fallar → FutureTimestampError              │
│   │   ├─ EventID       ← puede fallar → InvalidEventIDError               │
│   │   └─ TraceID       ← puede fallar → InvalidTraceIDError               │
│   │                                                                         │
│   ├─ Crea VehiclePosition Entity (factory)                                 │
│   │   ├─ Entity emite PositionReceivedEvent (Domain Event)                 │
│   │   └─ Retorna Entity                                                    │
│   │                                                                         │
│   └─ Verifica Circuit Breaker                                              │
│       ├─ CB está OPEN → CircuitBreakerOpenError                            │
│       └─ CB está CLOSED → Procede                                          │
│                                                                             │
│ CATCH Domain Error:                                                         │
│   ├─ Log error (level ERROR, con trace_id)                                 │
│   ├─ Propaga error hacia HTTP Layer SIN transformar                        │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓ (si error) o Entity (si éxito)
┌─────────────────────────────────────────────────────────────────────────────┐
│ HTTP ERROR MAPPER (http/mapper/error_mapper.go)                             │
│                                                                             │
│ ErrorMapper.ToProblemDetails(error)                                        │
│   ├─ FutureTimestampError      → (422, future-timestamp, ...)            │
│   ├─ InvalidVehicleIDError     → (422, invalid-vehicle-id, ...)          │
│   ├─ InvalidGeoPositionError   → (422, invalid-geo-position, ...)        │
│   ├─ InvalidEventIDError       → (422, invalid-event-id, ...)            │
│   ├─ InvalidTraceIDError       → (400, invalid-trace-id, ...)            │
│   ├─ CircuitBreakerOpenError   → (503, kafka-unavailable, ...)           │
│   └─ Unknown                   → (500, internal-error, genérico)           │
│                                                                             │
│ Retorna: RFC 9457 ProblemDetails (type, title, status, detail, instance) │
│          SIN: stack traces, SQL, rutas, secretos                           │
└─────────────────────────────────────────────────────────────────────────────┘
        ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ CLIENT RESPONSE (HTTP + JSON)                                               │
│                                                                             │
│ Error: 400/422/429/503/500 + RFC 9457 ProblemDetails (JSON)               │
│ Success: 202 Accepted + PositionResponse (JSON)                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

#### Error → HTTP Status Mapping Table

| Domain Error | HTTP Status | Type | Title | Detail |
|---|---|---|---|---|
| FutureTimestampError | 422 | `urn:fleet:error:future-timestamp` | Future Timestamp | Captured timestamp cannot be in the future |
| InvalidVehicleIDError | 422 | `urn:fleet:error:invalid-vehicle-id` | Invalid Vehicle ID | Vehicle ID must be non-empty, max 255 chars |
| InvalidGeoPositionError | 422 | `urn:fleet:error:invalid-geo-position` | Invalid Position | Latitude must be in [-90, 90], longitude in [-180, 180] |
| InvalidEventIDError | 422 | `urn:fleet:error:invalid-event-id` | Invalid Event ID | Event ID must be valid UUID v4 |
| InvalidTraceIDError | 400 | `urn:fleet:error:invalid-trace-id` | Invalid Trace ID | X-Trace-ID header must be valid UUID v4 |
| RateLimitExceededError | 429 | `urn:fleet:error:rate-limit-exceeded` | Rate Limit Exceeded | Too many requests for this vehicle (12/min) |
| CircuitBreakerOpenError | 503 | `urn:fleet:error:kafka-unavailable` | Service Unavailable | Message broker is temporarily unavailable |
| Unexpected/Unknown | 500 | `urn:fleet:error:internal-error` | Internal Error | An unexpected error occurred |

---

#### ErrorMapper Implementation Pattern

```go
// http/mapper/error_mapper.go

type ErrorMapper struct{}

func (m *ErrorMapper) ToProblemDetails(
    err error,
    traceID string,
) ProblemDetails {
    
    switch e := err.(type) {
    
    case FutureTimestampError:
        return ProblemDetails{
            Type:     "urn:fleet:error:future-timestamp",
            Title:    "Future Timestamp",
            Status:   422,
            Detail:   "Captured timestamp cannot be in the future",
            Instance: "/problems/validation/" + traceID,
        }
    
    case InvalidVehicleIDError:
        return ProblemDetails{
            Type:     "urn:fleet:error:invalid-vehicle-id",
            Title:    "Invalid Vehicle ID",
            Status:   422,
            Detail:   "Vehicle ID must be non-empty, max 255 characters",
            Instance: "/problems/validation/" + traceID,
        }
    
    case InvalidGeoPositionError:
        return ProblemDetails{
            Type:     "urn:fleet:error:invalid-geo-position",
            Title:    "Invalid Position",
            Status:   422,
            Detail:   "Latitude in [-90, 90], longitude in [-180, 180]",
            Instance: "/problems/validation/" + traceID,
        }
    
    case InvalidEventIDError:
        return ProblemDetails{
            Type:     "urn:fleet:error:invalid-event-id",
            Title:    "Invalid Event ID",
            Status:   422,
            Detail:   "Event ID must be valid UUID v4",
            Instance: "/problems/validation/" + traceID,
        }
    
    case InvalidTraceIDError:
        return ProblemDetails{
            Type:     "urn:fleet:error:invalid-trace-id",
            Title:    "Invalid Trace ID",
            Status:   400,
            Detail:   "X-Trace-ID header must be valid UUID v4",
            Instance: "/problems/header/" + traceID,
        }
    
    case CircuitBreakerOpenError:
        return ProblemDetails{
            Type:     "urn:fleet:error:kafka-unavailable",
            Title:    "Service Unavailable",
            Status:   503,
            Detail:   "Message broker is temporarily unavailable",
            Instance: "/problems/dependency/" + traceID,
        }
    
    default:
        return ProblemDetails{
            Type:     "urn:fleet:error:internal-error",
            Title:    "Internal Error",
            Status:   500,
            Detail:   "An unexpected error occurred",
            Instance: "/problems/error/" + traceID,
        }
    }
}
```

---



### DTOs (Data Transfer Objects)

Los DTOs son **tipos simples** (strings, floats, bool) sin lógica de dominio.

#### PositionRequest DTO

Representa la entrada HTTP cruda, **no validada**.

```go
// http/dto/request.go

type PositionRequest struct {
    VehicleID   string   `json:"vehicle_id"`
    Latitude    float64  `json:"latitude"`
    Longitude   float64  `json:"longitude"`
    Accuracy    *float64 `json:"accuracy,omitempty"`
    CapturedAt  string   `json:"captured_at"`  // ISO 8601 UTC
}

// Sin métodos de validación (validación ocurre en dominio)
// Sin referencias a Value Objects
// Tipos simples solamente
```

**Responsabilidad:** Recibir JSON deserializado por Gin, nada más.

---

#### PositionResponse DTO

Representa la salida HTTP para códigos 202.

```go
// http/dto/response.go

type PositionResponse struct {
    Status       string `json:"status"`       // "accepted"
    TraceID      string `json:"trace_id"`     // UUID v4
    VehicleID    string `json:"vehicle_id"`
    EventID      string `json:"event_id"`     // UUID v4
    ReceivedAt   string `json:"received_at"`  // ISO 8601 UTC
}

// Sin métodos de lógica
// Tipos simples solamente
// Construido por PositionMapper.FromEntity()
```

**Responsabilidad:** Enviar JSON al cliente.

---

#### ProblemDetails DTO

Respuestas de error conforme RFC 9457.

```go
// http/dto/response.go

type ProblemDetails struct {
    Type     string `json:"type"`      // URI identifying problem type
    Title    string `json:"title"`     // Brief summary
    Status   int    `json:"status"`    // HTTP status code
    Detail   string `json:"detail"`    // Human-readable explanation
    Instance string `json:"instance"`  // URI of this occurrence
}

// Sin métodos de validación
// Tipos simples solamente
// Construido por ErrorMapper.ToProblemDetails()
```

**Responsabilidad:** Enviar errores en formato estándar RFC 9457.

---

### Mappers (Traductores)

Los Mappers **convierten entre capas** y **centralizan transformaciones**.

#### PositionMapper

Bidireccional:

```go
// http/mapper/position_mapper.go

type PositionMapper struct{}

// DTO → Command (HTTP → Application)
func (m *PositionMapper) ToCommand(
    request PositionRequest,
    traceID string,              // desde context (middleware)
    idempotencyKey *string,      // desde context (middleware)
) ReceivePositionCommand {
    
    return ReceivePositionCommand{
        VehicleID:       request.VehicleID,
        Latitude:        request.Latitude,
        Longitude:       request.Longitude,
        Accuracy:        request.Accuracy,
        CapturedAt:      request.CapturedAt,
        TraceID:         traceID,
        IdempotencyKey:  idempotencyKey,
    }
}

// Entity → Response (Domain → HTTP)
func (m *PositionMapper) FromEntity(
    entity *VehiclePosition,  // desde useCase
) PositionResponse {
    
    return PositionResponse{
        Status:       "accepted",
        TraceID:      entity.TraceID().String(),
        VehicleID:    entity.VehicleID().String(),
        EventID:      entity.EventID().String(),
        ReceivedAt:   entity.ReceivedTimestamp().String(),  // ISO 8601
    }
}

// Nota: Mappers son stateless, reutilizables, testeable
```

**Responsabilidad:**
- Centraliza conversiones (fácil de cambiar)
- Reutilizable (usado en handler + tests)
- Testeable aisladamente

---

#### ErrorMapper

Mapea errores de dominio a RFC 9457.

```go
// http/mapper/error_mapper.go

type ErrorMapper struct{}

func (m *ErrorMapper) ToProblemDetails(
    err error,
    traceID string,
) ProblemDetails {
    
    // Identifica tipo de error dominio
    switch err {
    
    case ErrEmptyVehicleID:
        return ProblemDetails{
            Type:     "urn:fleet:error:validation_error",
            Title:    "Validation Error",
            Status:   422,
            Detail:   "Field 'vehicle_id' is required and non-empty",
            Instance: "/problems/validation/" + traceID,
        }
    
    case ErrInvalidLatitude:
        return ProblemDetails{
            Type:     "urn:fleet:error:validation_error",
            Title:    "Validation Error",
            Status:   422,
            Detail:   "Latitude must be in range [-90.0, 90.0]",
            Instance: "/problems/validation/" + traceID,
        }
    
    case ErrFutureCapturedTimestamp:
        return ProblemDetails{
            Type:     "urn:fleet:error:validation_error",
            Title:    "Validation Error",
            Status:   422,
            Detail:   "captured_at cannot be in the future",
            Instance: "/problems/validation/" + traceID,
        }
    
    case ErrCircuitBreakerOpen:
        return ProblemDetails{
            Type:     "urn:fleet:error:service_unavailable",
            Title:    "Service Unavailable",
            Status:   503,
            Detail:   "Kafka dependency is temporarily unavailable",
            Instance: "/problems/circuit_breaker/" + traceID,
        }
    
    case ErrRateLimitExceeded:
        return ProblemDetails{
            Type:     "urn:fleet:error:rate_limit_exceeded",
            Title:    "Rate Limit Exceeded",
            Status:   429,
            Detail:   "Too many requests for this vehicle",
            Instance: "/problems/rate_limit/" + traceID,
        }
    
    // ... más errores ...
    
    default:
        return ProblemDetails{
            Type:     "urn:fleet:error:internal_server_error",
            Title:    "Internal Server Error",
            Status:   500,
            Detail:   "An unexpected error occurred",  // Sin detalles
            Instance: "/problems/error/" + traceID,
        }
    }
}

// Nota: Centraliza mapeo, facilita cambios, seguridad
```

**Responsabilidad:**
- Centraliza mapeo de errores a HTTP
- Evita exponer detalles internos
- Reutilizable en múltiples handlers (futuro)

---

### Beneficios de Centralizar DTOs & Mappers

1. **Reutilizables:** Mismo Mapper en handler + tests
2. **Mantenibles:** Un lugar para cambiar conversión
3. **Seguros:** Errores exponen solo lo necesario
4. **Testeable:** Mappers se prueban aisladamente
5. **Escalable:** Agregar nuevo endpoint = reutilizar Mappers

---

## Domain Layer

### 9 Value Objects + 1 Entity

#### 1. VehicleID

Identificador único del vehículo.

```go
type VehicleID struct {
    value string
}

func NewVehicleID(raw string) (VehicleID, error) {
    // Validación: no vacío, max 255 chars
}

func (v VehicleID) String() string
func (v VehicleID) Equals(other VehicleID) bool
```

---

#### 2. GeoPosition

Coordenadas de ubicación con precision.

```go
type GeoPosition struct {
    latitude  float64
    longitude float64
    accuracy  *float64
}

func NewGeoPosition(lat, lon float64, acc *float64) (GeoPosition, error) {
    // Validación: lat ∈ [-90, 90], lon ∈ [-180, 180], acc ≥ 0
}
```

---

#### 3. CapturedTimestamp

Timestamp de captura (cliente).

```go
type CapturedTimestamp struct {
    value time.Time
}

func NewCapturedTimestamp(raw string, clock Clock) (CapturedTimestamp, error) {
    // Validación: ISO 8601 UTC, no futuro
}
```

---

#### 4. ReceivedTimestamp

Timestamp de recepción (servidor).

```go
type ReceivedTimestamp struct {
    value time.Time
}

func NewReceivedTimestamp(clock Clock) ReceivedTimestamp {
    // Generado como time.Now()
}
```

---

#### 5. ProcessedTimestamp

Timestamp post-validación (servidor).

```go
type ProcessedTimestamp struct {
    value time.Time
}

func NewProcessedTimestamp(clock Clock) ProcessedTimestamp {
    // Generado como time.Now() post-validación
}
```

---

#### 6. EventID

Identificador único del evento (UUID v4).

```go
type EventID struct {
    value uuid.UUID
}

func NewEventID(generator IDGenerator) EventID {
    // Genera UUID v4
}
```

---

#### 7. TraceID

Identificador de trazabilidad (UUID v4).

```go
type TraceID struct {
    value uuid.UUID
}

func NewTraceID(raw *string, generator IDGenerator) (TraceID, error) {
    // Valida si presente, genera si no
}
```

---

#### 8. IdempotencyKey

Identificador de idempotencia (UUID v4, solo validado).

```go
type IdempotencyKey struct {
    value *uuid.UUID
}

func NewIdempotencyKey(raw *string) (IdempotencyKey, error) {
    // Valida si presente, nil si no
}
```

---

#### 9. RateLimitConfig

Configuración de rate limiting.

```go
type RateLimitConfig struct {
    frequencySeconds int
}

func NewRateLimitConfig(freqSecs int) RateLimitConfig {
    // max = (60 / freq) * 1.2
}
```

---

#### Entity: VehiclePosition

Orquesta los 9 Value Objects.

```go
type VehiclePosition struct {
    eventID            EventID
    vehicleID          VehicleID
    location           GeoPosition
    capturedTimestamp  CapturedTimestamp
    receivedTimestamp  ReceivedTimestamp
    processedTimestamp ProcessedTimestamp
    traceID            TraceID
}

func NewVehiclePosition(...) (*VehiclePosition, error) {
    // Factory: valida invariantes, garantiza orden timestamps
}

func (v *VehiclePosition) ToKafkaPayload() map[string]interface{}
func (v *VehiclePosition) ReceptionLatency() time.Duration
```

---

### Domain Errors

```go
var (
    ErrEmptyVehicleID               = errors.New("vehicle_id cannot be empty")
    ErrInvalidLatitude              = errors.New("latitude must be in range [-90, 90]")
    ErrInvalidLongitude             = errors.New("longitude must be in range [-180, 180]")
    ErrFutureCapturedTimestamp      = errors.New("captured_at cannot be in future")
    ErrInvalidCapturedTimestampFormat = errors.New("captured_at must be ISO 8601 UTC")
    ErrInvalidTraceID               = errors.New("X-Trace-ID must be valid UUID v4")
    ErrInvalidIdempotencyKey        = errors.New("X-Idempotency-Key must be valid UUID v4")
    ErrCircuitBreakerOpen           = errors.New("kafka dependency unavailable")
    ErrRateLimitExceeded            = errors.New("too many requests")
    ErrQueueFull                    = errors.New("internal queue capacity exceeded")
)
```

---

## Application Layer

### ReceivePositionCommand

DTO interno de aplicación.

```go
type ReceivePositionCommand struct {
    VehicleID       string
    Latitude        float64
    Longitude       float64
    Accuracy        *float64
    CapturedAt      string
    TraceID         string
    IdempotencyKey  *string
}
```

---

### ReceivePositionUseCase

Orquesta la lógica.

```go
type ReceivePositionUseCase struct {
    kafkaPublisher   KafkaPublisher
    circuitBreaker   CircuitBreakerManager
    logger           Logger
    clock            Clock
    idGenerator      IDGenerator
    rateLimiter      RateLimiter
}

func (u *ReceivePositionUseCase) Execute(
    ctx context.Context,
    command ReceivePositionCommand,
) (*VehiclePosition, error) {
    // 1. Crear Value Objects (validación)
    // 2. Construir Entity (factory)
    // 3. Verificar Circuit Breaker
    // 4. Publicar async
    // 5. Retornar Entity
}
```

---

## Ports Layer

### 6 Interfaces

```go
type KafkaPublisher interface {
    PublishAsync(ctx context.Context, position *VehiclePosition, callback func(error)) error
}

type CircuitBreakerManager interface {
    Check() bool
    RecordSuccess()
    RecordFailure()
    State() CircuitBreakerState
}

type RateLimiter interface {
    CheckLimit(ctx context.Context, vehicleID VehicleID) error
    GetLimit(ctx context.Context, vehicleID VehicleID) (limit, remaining int, resetAt time.Time, err error)
}

type Logger interface {
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}

type Clock interface {
    Now() time.Time
}

type IDGenerator interface {
    Generate() EventID
}
```

---

## Outbound Adapters

### KafkaPublisherImpl

Implementa `KafkaPublisher` con cola interna, reintentos y Circuit Breaker.

```go
type KafkaPublisherImpl struct {
    producer       sarama.AsyncProducer
    topic          string
    queue          chan *VehiclePosition  // max 10,000
    circuitBreaker CircuitBreakerManager
    logger         Logger
    backoffDelays  []time.Duration  // [100ms, 200ms, 400ms]
}

func (p *KafkaPublisherImpl) PublishAsync(
    ctx context.Context,
    position *VehiclePosition,
    callback func(error),
) error {
    // Enqueue, retry on failure, dead-letter on exhaust
}
```

---

### CircuitBreakerImpl

Implementa `CircuitBreakerManager` con estados.

```go
type CircuitBreakerImpl struct {
    state            CircuitBreakerState
    failureCount     int
    failureThreshold int      // 5
    failureWindow    time.Duration  // 60s
    halfOpenTimeout  time.Duration  // 30s
    mu               sync.RWMutex
}

func (cb *CircuitBreakerImpl) Check() bool
func (cb *CircuitBreakerImpl) RecordSuccess()
func (cb *CircuitBreakerImpl) RecordFailure()
```

---

### RateLimiterImpl

Implementa `RateLimiter` con rolling window en memoria.

```go
type RateLimiterImpl struct {
    config   RateLimitConfig
    counters map[string]*RateLimitCounter
    mu       sync.RWMutex
}

func (r *RateLimiterImpl) CheckLimit(ctx context.Context, vehicleID VehicleID) error
func (r *RateLimiterImpl) GetLimit(ctx context.Context, vehicleID VehicleID) (int, int, time.Time, error)
```

---

### ClockImpl

Implementa `Clock`.

```go
type ClockImpl struct{}

func (c *ClockImpl) Now() time.Time {
    return time.Now().UTC()
}
```

---

## HTTP Adapter: Handler & Middleware

### PositionHandler

Thin, ~15 líneas.

```go
type PositionHandler struct {
    useCase ReceivePositionUseCase
    mapper  PositionMapper
}

func (h *PositionHandler) Handle(c *gin.Context) {
    var request PositionRequest
    c.ShouldBindJSON(&request)
    
    command := h.mapper.ToCommand(
        request,
        c.GetString("trace_id"),
        c.GetString("idempotency_key"),
    )
    
    entity, err := h.useCase.Execute(c.Request.Context(), command)
    if err != nil {
        // ErrorHandler middleware takes care
        c.AbortWithError(http.StatusInternalServerError, err)
        return
    }
    
    response := h.mapper.FromEntity(entity)
    c.JSON(http.StatusAccepted, response)
}
```

---

### 5 Middleware

#### TraceIDMiddleware

```go
func TraceIDMiddleware(idGenerator IDGenerator) gin.HandlerFunc {
    // Valida/genera X-Trace-ID, almacena en context
}
```

#### IdempotencyKeyMiddleware

```go
func IdempotencyKeyMiddleware() gin.HandlerFunc {
    // Valida X-Idempotency-Key (si presente), almacena en context
}
```

#### ContentTypeMiddleware

```go
func ContentTypeMiddleware() gin.HandlerFunc {
    // Valida Content-Type: application/json
}
```

#### RateLimitMiddleware

```go
func RateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
    // Verifica límite por vehicle_id, retorna 429 si excede
}
```

#### ErrorHandlerMiddleware

```go
func ErrorHandlerMiddleware(mapper ErrorMapper) gin.HandlerFunc {
    // Mapea errores a Problem Details (RFC 9457)
}
```

---

## Config Layer

### Composer

Composición de dependencias.

```go
func NewServer() (*gin.Engine, error) {
    // Instancia todas las capas
    // Conecta puertos con adapters
    // Construye middleware pipeline
    // Retorna router listo para usar
}
```

---

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do.*

### Property 1: Rechazo de Validación Contractual

**For any** solicitud HTTP que viole los criterios de RF-02 (campos obligatorios, tipos, formatos, límites), **the service SHALL reject** with el código de estado correcto (422 para semántica, 415 para media type, 413 para tamaño).

**Validates: Requirements RF-01.2, RF-02.1 through RF-02.9, RF-10, RF-12**

---

### Property 2: Publicación Asincrónica y Respuesta 202

**For any** solicitud válida, **the HTTP 202 response SHALL be sent** antes de que el servicio intente publicar en Kafka. Si Kafka falla posteriormente, el cliente NUNCA recibe notificación de ese fallo.

**Validates: Requirements RF-01.5, RF-04.1, RF-04.2, RF-10, RF-11**

---

### Property 3: Recuperación del Circuit Breaker

**For any** secuencia de 5 fallos de Kafka en ventana de 60 segundos, **the Circuit Breaker SHALL transition to OPEN state**. Después de esto, nuevas solicitudes **SHALL receive HTTP 503**. Cada 30 segundos, **the service SHALL attempt recovery** sin bloquear solicitudes nuevas.

**Validates: Requirements RF-05.4, RF-05.5, RF-05.6, RF-10, RF-12**

---

### Property 4: Integridad de Cuatro Timestamps

**For any** evento encolado para Kafka, **all four timestamps SHALL be present** en el payload, en orden cronológico correcto: captured_at ≤ received_at ≤ processed_at ≤ stored_at=null.

**Validates: Requirement RF-07.1, RF-07.2, RF-10**

---

### Property 5: Validación de Timestamp Futuro

**For any** captured_at (ISO 8601 UTC enviado por cliente), **if the timestamp is in the future** (posterior a la hora actual del servidor), **the service SHALL reject** con 422. **If the timestamp is in the past**, sin importar su antigüedad, **the service SHALL accept** it.

**Validates: Requirement RF-07.3, RF-07.7, RF-10, RF-12**

---

### Property 6: Rate Limiting por VehicleID

**For any** vehicle_id, **the service SHALL allow** 12 solicitudes por minuto máximo (default). **If the limit is exceeded**, **the service SHALL respond with HTTP 429**, incluyendo headers X-RateLimit-*.

**Validates: Requirements RF-10 (implícito), RF-04, RF-12**

---

### Property 7: RFC 9457 Conformance

**For any** respuesta de error (códigos 400, 413, 415, 422, 503, 429), **the JSON body SHALL conform to RFC 9457 Problem Details** con campos obligatorios: type, title, status, detail, instance.

**Validates: Requirement RF-09.3, RF-10, RF-12**

---

### Property 8: Ausencia de Información Sensible en Errores

**For any** error interno (código 500 o fallos de componentes), **the Problem Details response SHALL NOT expose** stack traces, consultas SQL, rutas internas, tokens ni detalles técnicos específicos.

**Validates: Requirement RF-04.5, RF-10, RF-12**

---

### Property 9: Trazabilidad del Trace ID

**For any** solicitud, **every log entry y response body SHALL include el mismo trace_id** durante todo el ciclo de vida de la solicitud.

**Validates: Requirement RF-08.1, RF-10**

---

### Property 10: Domain Error Mapping

**For any** error emitido por el Domain Layer (FutureTimestampError, InvalidVehicleIDError, InvalidGeoPositionError, InvalidEventIDError, InvalidTraceIDError), **the ErrorMapper SHALL correctly map** it to RFC 9457 ProblemDetails con el HTTP status correcto (422 para validación, 400 para headers, 503 para CB, 500 para unexpected).

**Validates: Requirement RF-12.1, RF-12.3, RF-12.4**

---

### Property 11: Error Propagation Sin Transformación

**When** el Domain Layer emite un error, **the Application Layer SHALL capture** ese error SIN transformarlo. **The Application SHALL log** el error con trace_id. **The error SHALL propagate** íntegro hacia el HTTP Layer para mapeo por ErrorMapper.

**Validates: Requirement RF-12.2**

---

### Property 12: Domain Event Emission

**When** VehiclePosition Entity es creada exitosamente, **the Entity SHALL emit** exactamente UNO PositionReceivedEvent. **The event SHALL contain** todos los campos requeridos: vehicleID, eventID, capturedAt, receivedAt, processedAt, traceID, occurredAt.

**Validates: Requirement RF-11.1, RF-11.2**

---

### Property 13: Domain Event Decoupling

**For any** PositionReceivedEvent emitido por el Domain, **the event SHALL NOT reference** Kafka, HTTP, DTOs, o cualquier detalle de infraestructura. **The Application Layer SHALL consume** el event y decida qué hacer (ej: publicar a Kafka), sin que el Domain imponga esa decisión.

**Validates: Requirement RF-11.5, RF-11.6, RF-11.7**

---

### Property 14: Error Handling No Leaks

**For any** error que resulte en respuesta HTTP 500, **the Problem Details SHALL NOT expose** stack traces, SQL queries, rutas internas, tokens, o detalles técnicos específicos del framework. **The detail field SHALL contain** solo una descripción genérica ("An unexpected error occurred").

**Validates: Requirement RF-12.5**

---

## Error Handling

| Domain Error | HTTP Status | Problem Type | Title |
|---|---|---|---|
| ErrEmptyVehicleID | 422 | validation_error | Validation Error |
| ErrInvalidLatitude | 422 | validation_error | Validation Error |
| ErrInvalidLongitude | 422 | validation_error | Validation Error |
| ErrFutureCapturedTimestamp | 422 | validation_error | Validation Error |
| ErrInvalidTraceID | 422 | validation_error | Validation Error |
| ErrInvalidIdempotencyKey | 422 | validation_error | Validation Error |
| ErrCircuitBreakerOpen | 503 | service_unavailable | Service Unavailable |
| ErrRateLimitExceeded | 429 | rate_limit_exceeded | Rate Limit Exceeded |
| Unexpected | 500 | internal_server_error | Internal Server Error |

---

## Testing Strategy

### Enfoque: Black Box por Comportamiento Observable

Según `testing.md`, el objetivo es **pocas pruebas valiosas y difíciles de romper** que verifiquen comportamiento importante como caja negra, **NO cubrir capa por capa**.

Cada prueba se evalúa con 4 pilares:

1. **Protección contra regresiones**: ¿Detectaría un bug real en una regla importante?
2. **Resistencia al refactoring**: ¿Sigue pasando si cambio la implementación sin cambiar comportamiento?
3. **Feedback rápido**: ¿Se ejecuta en segundos?
4. **Mantenibilidad**: ¿Se entiende y prepara con poco código?

---

### Cobertura Multi-Capa mediante Comportamiento Observable

Una **sola prueba de comportamiento cubre naturalmente múltiples capas** (testing.md sección 3):

**Ejemplo: `test_rejects_position_with_future_timestamp`**

```go
func TestRejectsPositionWithFutureTimestamp(t *testing.T) {
    // ARRANGE
    request := PositionRequestMother.valid().
        withCapturedAt(time.Now().Add(1*time.Hour)).build()
    
    // ACT
    entity, err := useCase.Execute(ctx, request)
    
    // ASSERT
    assert.Error(t, err)
    assert.IsType(t, &domain.FutureTimestampError{}, err)
    // Verifies: RF-02.6, RF-07.3
}
```

**Capas probadas en este ÚNICO test:**
- ✅ **Domain:** Value Object `CapturedTimestamp` (validación de rango)
- ✅ **Domain:** Entity factory (garantiza invariantes)
- ✅ **Domain:** Error `FutureTimestampError` (corrección)
- ✅ **Application:** UseCase orchestration (propagación de error)
- ✅ **HTTP:** Mapper.ToCommand() (conversión DTO → Command)
- ✅ **HTTP:** Error mapping hacia RFC 9457 (Observable en integración)

**Resultado:** Sin agregar 6 pruebas separadas (una por capa), una sola prueba verifica que **todos los componentes trabajan juntos correctamente** para rechazar una regla de negocio.

---

### Suite Rápida (go test ./...)

**Caracterización:**
- Herramientas: Go testing, table-driven tests, Object Mother
- Dependencias: Fakes (Clock, IDGenerator, Logger, KafkaPublisher)
- Ejecución: < 1 segundo total
- Enfoque: Entrada por UseCase/Mapper, salida observable (Entity/Error)
- Build tag: ninguno (default)

**Comportamientos verificados (~12 pruebas, ~200 líneas total):**

| Descripción | Entrada | Salida Observable | Capas Cubiertas | Pilares |
|---|---|---|---|---|
| `test_accepts_valid_position` | PositionRequest válido | Entity con 9 VO + EventID | VO + Entity + UseCase + Mapper | Protección + Resistencia |
| `test_rejects_empty_vehicle_id` | vehicle_id = "" | Error FutureTimestampError | VO + Entity + Error + Mapper | Protección + Resistencia |
| `test_rejects_future_timestamp` | captured_at = ahora + 1h | Error FutureTimestampError | VO + Entity + Error | Protección + Resistencia |
| `test_rejects_out_of_range_latitude` | latitude = 91.0 | Error InvalidGeoPositionError | VO + Entity + Error | Protección + Resistencia |
| `test_rejects_out_of_range_longitude` | longitude = 181.0 | Error InvalidGeoPositionError | VO + Entity + Error | Protección + Resistencia |
| `test_rejects_invalid_captured_at_format` | captured_at = "invalid" | Error InvalidCapturedTimestampError | VO + Error | Protección + Resistencia |
| `test_rejects_invalid_event_id_format` | event_id = "not-uuid" | Error InvalidEventIDError | VO + Error | Protección + Resistencia |
| `test_rejects_invalid_trace_id_format` | X-Trace-ID = "invalid" | Error InvalidTraceIDError | VO + Error + Middleware | Protección + Resistencia |
| `test_emits_domain_event_on_success` | PositionRequest válido | Entity.Events() contiene PositionReceivedEvent | Entity + Domain Event | Protección + Resistencia |
| `test_error_mapper_formats_rfc9457` | Domain error | ProblemDetails con type/title/status/detail/instance | Error + Mapper | Protección + Resistencia |
| `test_rate_limit_config_calculates_max_per_minute` | frequency = 5s | max = 14 solicitudes/min | VO + Domain rule | Protección |
| `test_accepts_timestamp_in_any_past` | captured_at = 2020-01-01 | Entity creado, no rechazado | VO + Entity | Protección + Resistencia |

**Estructura Table-Driven (ejemplo):**

```go
func TestPositionValidation(t *testing.T) {
    testCases := []struct {
        name        string
        request     PositionRequest
        expectError bool
        errorType   error
    }{
        {
            name:        "valid position accepted",
            request:     PositionRequestMother.valid().build(),
            expectError: false,
        },
        {
            name:        "future timestamp rejected",
            request:     PositionRequestMother.valid().withCapturedAt(futureTime).build(),
            expectError: true,
            errorType:   &domain.FutureTimestampError{},
        },
        // ... más casos ...
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            entity, err := useCase.Execute(ctx, tc.request)
            
            if tc.expectError {
                assert.Error(t, err)
                assert.IsType(t, tc.errorType, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, entity)
            }
            // Verifies: RF-02.X, RF-07.X
        })
    }
}
```

---

### Suite Integración (go test -tags=integration ./...)

**Caracterización:**
- Herramientas: Testcontainers (Kafka), Toxiproxy, Go testing
- Infraestructura: Contenedor Kafka efímero, Toxiproxy para inyectar fallos
- Ejecución: < 10 segundos por test
- Enfoque: HTTP real, Kafka real, estado observable final
- Build tag: `integration`

**Comportamientos verificados (~8 pruebas, ~300 líneas total):**

| Descripción | Setup | Comportamiento | Capas Cubiertas | Validaciones |
|---|---|---|---|---|
| `test_http_202_sent_before_kafka_attempt` | Kafka + network interceptor | 202 inmediato, luego Kafka attempt async | HTTP + KafkaPublisher | Protección (RNF-03) |
| `test_kafka_receives_correct_payload` | Kafka real | Evento en topic con 4 timestamps + payload correcto | KafkaPublisher + Schema | Protección (RF-03) |
| `test_circuit_breaker_opens_after_5_failures` | Toxiproxy + Kafka | 5 fallos consecutivos → CB OPEN → 503 en siguiente solicitud | CircuitBreaker + Handler | Protección (RF-05.4) |
| `test_circuit_breaker_recovers_after_30s_probe` | Toxiproxy + CB OPEN | Cada 30s intento recovery → HALF_OPEN → CLOSED | CircuitBreaker + Resilience | Protección (RF-05.6) |
| `test_rate_limit_rejects_13th_request_per_minute` | RateLimiter in-memory | 12 requests acepta, 13ava → 429 + headers X-RateLimit-* | Middleware + RateLimiter | Protección (RF-13) |
| `test_kafka_retry_exponential_backoff` | Toxiproxy + network delay | Fallo → 100ms → 200ms → 400ms reintento | KafkaPublisher + Resilience | Protección (RF-05.1) |
| `test_dead_letter_on_final_failure` | Toxiproxy + permanent failure | 3 reintentos agotados → evento en dead-letter log | KafkaPublisher + Observability | Protección (RF-05.3) |
| `test_circuit_breaker_with_toxiproxy_latency` | Toxiproxy latency > timeout | CB abre, 503 retornado, recupera cuando latency disminuye | CircuitBreaker + Toxiproxy + Timeout | Protección (RF-05) |

**Validación de Contrato (OpenAPI):**
- Cada respuesta HTTP validada contra `openapi.yaml` (usando `kin-openapi`)
- Cada ProblemDetails verifica RFC 9457 (type, title, status, detail, instance presentes)
- Un cambio incompatible en OpenAPI rompe una prueba antes del merge

---

### Object Mother & Test Data Builder

**Ubicación:** `tests/support/mothers/position_request_mother.go`

**Patrón:**

```go
// Defaults válidos y deterministas
func PositionRequestMother() *PositionRequestBuilder {
    return &PositionRequestBuilder{
        vehicleID:  "VH-001",
        latitude:   40.7128,
        longitude:  -74.0060,
        accuracy:   5.0,
        capturedAt: "2024-10-17T14:30:45.123Z",
    }
}

func (b *PositionRequestBuilder) withCapturedAt(t time.Time) *PositionRequestBuilder {
    b.capturedAt = t.Format(time.RFC3339Nano)
    return b
}

func (b *PositionRequestBuilder) build() PositionRequest {
    return PositionRequest{
        VehicleID:  b.vehicleID,
        Latitude:   b.latitude,
        Longitude:  b.longitude,
        Accuracy:   &b.accuracy,
        CapturedAt: b.capturedAt,
    }
}
```

**Uso:**
- Suite Rápida: todos los tests
- Suite Integración: seeds de datos
- k6 load tests: generadores de payload

---

### Trazabilidad: Requisito → Prueba

| Requisito | Prueba | Suite | Capas | Status |
|---|---|---|---|---|
| RF-01.1 | test_accepts_valid_position | Rápida | HTTP + UseCase | ✅ |
| RF-02.3 | test_rejects_empty_vehicle_id | Rápida | VO + Error | ✅ |
| RF-02.4 | test_rejects_out_of_range_latitude | Rápida | VO + Error | ✅ |
| RF-02.5 | test_rejects_out_of_range_longitude | Rápida | VO + Error | ✅ |
| RF-02.6 | test_rejects_future_timestamp | Rápida | VO + Error | ✅ |
| RF-02.9 | test_rejects_invalid_json_format | Rápida | HTTP parsing | ✅ |
| RF-03 | test_kafka_receives_correct_payload | Integración | KafkaPublisher | ✅ |
| RF-04 | test_http_202_sent_before_kafka_attempt | Integración | HTTP + Async | ✅ |
| RF-05.1 | test_kafka_retry_exponential_backoff | Integración | KafkaPublisher | ✅ |
| RF-05.4 | test_circuit_breaker_opens_after_5_failures | Integración | CircuitBreaker | ✅ |
| RF-05.6 | test_circuit_breaker_recovers_after_30s_probe | Integración | CircuitBreaker | ✅ |
| RF-07 | test_emits_domain_event_on_success | Rápida | Entity + Domain Event | ✅ |
| RF-09.3 | test_error_mapper_formats_rfc9457 | Rápida | Error + Mapper | ✅ |
| RF-11 | test_emits_domain_event_on_success | Rápida | Entity + Event | ✅ |
| RF-12 | test_error_mapper_formats_rfc9457 | Rápida | Error + Mapper | ✅ |
| RF-13 | test_rate_limit_rejects_13th_request_per_minute | Integración | Middleware | ✅ |

---

### Antipatrones Prohibidos (testing.md sección 22)

- ❌ NO: Pruebas separadas por cada Value Object ("test VO VehicleID", "test VO GeoPosition", etc.)
- ❌ NO: Mockear colaboradores internos (Entity, UseCase, Mapper, VO)
- ❌ NO: Verificar número de llamadas internas o interacciones
- ❌ NO: Probar métodos privados o detalles de implementación
- ❌ NO: Snapshots de respuestas JSON
- ❌ NO: Pruebas que dependen del orden de ejecución o estado compartido
- ❌ NO: Sleep sin espera de condición
- ❌ NO: Duplicar lógica de producción dentro de la prueba para calcular resultado esperado

---

### Checklist de Revisión (testing.md sección 23)

Antes de aceptar una prueba:

1. ✅ **¿Entra y sale por bordes públicos?** (UseCase.Execute() o HTTP handler → Entity/Error/Response)
2. ✅ **¿Pasaría si reescribo la implementación sin cambiar comportamiento?** (Resistencia al refactoring)
3. ✅ **¿Falla si rompo la regla que dice proteger?** (Ej: cambiar `latitude > 90` → `latitude > 89` debe fallar)
4. ✅ **¿Solo sustituye dependencias no gestionadas?** (Fakes para Clock/IDGenerator, Kafka real en integración)
5. ✅ **¿Es determinista y rápida?** (< 100ms Suite Rápida, < 10s Integración)
6. ✅ **¿Referencia el requisito que verifica?** (Comentario: `// Verifies: RF-02.6, RF-07.3`)
7. ✅ **¿Fué escrita antes del fix si es regresión?** (Si cubre un bug, debe fallar sin el fix)

---

### Resumen: Por Qué Este Enfoque Funciona

| Aspecto | Beneficio |
|---|---|
| **Black Box** | No pruebas frágiles que rompan con refactoring |
| **Comportamiento observable** | Pruebas cumplen Requisitos de Negocio |
| **Multi-capa por test** | 12 pruebas cubren todas las capas, NO 50+ pruebas separadas |
| **Object Mother** | Datos deterministas, reutilizables, fáciles de mantener |
| **Suite Rápida + Integración** | Feedback rápido en CI, validación completa sin lentitud |
| **Trazabilidad explícita** | Cada prueba referencia el requisito que valida |
| **RFC 9457 validation** | Cambios incompatibles rompen pruebas antes del merge |

---

## Summary: Beneficios de COMPLETO (Actualizado)

✅ **Separación clara de responsabilidades** (cada carpeta una función)  
✅ **DTOs + Mappers centralizados** (reutilizables, seguros, protegen dominio)  
✅ **Dominio protegido** (sin HTTP, sin Kafka, sin detalles, emite eventos)  
✅ **Domain Events desacoplados** (Application decide acciones)  
✅ **Error Handling explícito** (Domain → Application → HTTP ErrorMapper → RFC 9457)  
✅ **RFC 9457 Conformance** (Problem Details seguros, sin leaks)  
✅ **Testeable en cada capa** (aislada)  
✅ **Escalable** (agregar endpoints = reutilizar Mappers + Events)  
✅ **Mantenible** (cambios localizados)  
✅ **Profesional** (patrón verificable, event-driven ready)  

### Requisitos Cubiertos

| Requirement | Coverage | Notes |
|---|---|---|
| RF-01 | ✅ Completo | Recepción HTTP POST |
| RF-02 | ✅ Completo | Validación de contrato |
| RF-03 | ✅ Completo | Publicación asincrónica Kafka |
| RF-04 | ✅ Completo | Desacoplamiento ingesta/procesamiento |
| RF-05 | ✅ Completo | Reintentos + Circuit Breaker |
| RF-06 | ✅ Completo | Distinción aceptación vs persistencia |
| RF-07 | ✅ Completo | 4 timestamps + validación |
| RF-08 | ✅ Completo | Logging + monitoreo |
| RF-09 | ✅ Completo | OpenAPI contract |
| RF-10 | ✅ Completo | Clean Architecture COMPLETO |
| RF-11 | ✅ Nuevo | Domain Events (PositionReceivedEvent) |
| RF-12 | ✅ Nuevo | Error Handling Domain → HTTP (RFC 9457) |
| RF-13 | ✅ Nuevo | Rate Limiting in-memory (rolling window 60s, 12/min per vehicle_id) |
| RNF-01 | ✅ Completo | Clean Architecture aplicada |
| RNF-03 | ✅ Completo | Desacoplamiento por eventos |
| RNF-04 | ✅ Completo | Idempotencia + duplicados |
| RNF-06 | ✅ Completo | Resiliencia (CB, reintentos) |
