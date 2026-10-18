# Arquitectura - Backend Go

## Modelo: Bounded Contexts (Domain-Driven Design)

El proyecto usa **Bounded Contexts** como estructura principal. Cada contexto es un módulo independiente con su propio dominio, aplicación, adaptadores y puertos.

```
internal/
├── ingestion/                  # Bounded Context 1: HTTP Position Ingestion
│   ├── domain/                 # Domain específico (PLANO - sin subdirectorios)
│   │   ├── vehicle_position.go       # Entity
│   │   ├── position_received_event.go # Event
│   │   ├── vehicle_id.go             # Value Object
│   │   ├── geo_position.go
│   │   ├── event_id.go
│   │   ├── trace_id.go
│   │   ├── idempotency_key.go
│   │   ├── rate_limit_config.go
│   │   ├── timestamps.go
│   │   ├── trace.go                  # Utilities
│   │   └── errors.go                 # Domain Errors
│   │
│   ├── application/            # Casos de uso
│   │   ├── usecases/           # ReceivePositionUseCase
│   │   ├── commands/           # Comandos (ReceivePositionCommand)
│   │   ├── ports/              # Interfaces que el dominio necesita
│   │   └── dto/                # Data Transfer Objects
│   │
│   ├── adapters/               # Implementaciones concretas
│   │   ├── inbound/            # Adaptadores de entrada
│   │   │   └── http/           # Rutas, handlers, middleware
│   │   └── outbound/           # Adaptadores de salida
│   │       ├── messaging/      # Kafka publisher
│   │       ├── resilience/     # Rate limiter, Circuit breaker
│   │       └── infrastructure/ # Clock, IDGenerator
│   │
│   ├── ports/                  # Interfaces (contratos públicos)
│   │   ├── http_config.go
│   │   ├── logger.go
│   │   ├── messaging.go        # KafkaPublisher interface
│   │   ├── resilience.go       # RateLimiter, CircuitBreaker interfaces
│   │   └── clock.go
│   │
│   ├── config/                 # Composición de dependencias
│   │   ├── composer.go         # DependencyComposer (Gin + adaptadores)
│   │   ├── logger.go           # LoggerImpl
│   │   ├── loader.go           # LoadConfig
│   │   └── validator.go        # ConfigValidator
│   │
│   └── http/                   # HTTP handlers y routing
│       ├── router.go           # Rutas POST /api/v1/positions
│       ├── handler/            # PositionHandler
│       ├── mapper/             # PositionMapper (DTO ↔ Domain)
│       └── middleware/         # TraceID, IdempotencyKey, ContentType, RateLimit, ErrorHandler
│
├── worker/                     # Bounded Context 2: Position Processing Worker
│   ├── domain/                 # Domain propio (PLANO)
│   ├── application/            # Casos de uso
│   ├── adapters/               # Implementaciones
│   ├── ports/                  # Interfaces
│   └── config/                 # Composición
│
├── cmd/                        # Entry points
│   ├── api/
│   │   └── main.go             # Levanta: ingestion context
│   └── worker/
│       └── main.go             # Levanta: worker context
│
└── tests/
    ├── ingestion/
    │   ├── suite-fast/         # Black-box tests via use cases
    │   └── suite-integration/  # Kafka real + Toxiproxy
    └── worker/
        ├── suite-fast/
        └── suite-integration/
```

## Reglas de Dependencia

### Dentro de un Bounded Context

```
cmd/api/main.go
    ↓
internal/ingestion/config/composer.go (DI)
    ↓
internal/ingestion/application/usecases/receive_position.go
    ↓
internal/ingestion/domain/ (entities, events, value_objects, errors)
    ↑
internal/ingestion/adapters/ (Kafka, Clock, RateLimiter)
    ↑
internal/ingestion/ports/ (interfaces)
```

**Dirección:** Solo hacia el dominio. El dominio no conoce adaptadores ni aplicación.

### Entre Bounded Contexts

- ❌ NO importar `ingestion/domain` desde `worker/`
- ❌ NO importar `worker/domain` desde `ingestion/`
- ✅ Comunicación via **eventos publicados en Kafka**
- ✅ Si es necesario compartir tipos: crear `shared/domain/` (excepcional)

## Convenciones en domain/ (PLANA)

Dentro de cada `domain/` plano, los nombres de archivo son autodescriptivos:

| Patrón | Significado | Ejemplo |
|--------|-------------|---------|
| `*_event.go` | Domain event | `position_received_event.go` |
| `*.go` (sin sufijo) | Entity o Value Object | `vehicle_position.go`, `vehicle_id.go` |
| `errors.go` | Domain errors | `errors.go` |
| `*.go` (auxiliares) | Utilities | `trace.go` |

## Ejemplos

### Usar tipos del contexto ingestion

```go
// Dentro de ingestion/ (application, adapters, http, etc.)
import "github.com/fleet-platform/backend/internal/ingestion/domain"

vehicleID := domain.NewVehicleID("VH-001")
position := domain.NewVehiclePosition(...)
event := domain.NewPositionReceivedEvent(...)
```

### Expandir a otro Bounded Context

Cuando necesites un nuevo contexto (ej: `analytics/`):

1. Crear `internal/analytics/domain/` con sus propias entidades (PLANO)
2. Crear `internal/analytics/application/` con sus casos de uso
3. Crear `internal/analytics/adapters/` para Kafka consumer
4. Publicar eventos desde `ingestion/` a Kafka
5. `analytics/` consume eventos e itera su propio dominio

```go
// analytics/adapters/inbound/kafka_consumer.go
func (kc *KafkaConsumer) OnPositionReceivedEvent(event []byte) {
    // NO importar ingestion/domain
    // Deserializar JSON y CREAR entidades propias de analytics
    analyticsEvent := analytics.NewPositionAnalyzed(...)
}
```

## Por Qué Bounded Contexts + Domain Plano

- ✅ **Escalabilidad:** Cada contexto crece independientemente con domain/ plano
- ✅ **Cohesión:** Código relacionado junto; cambios locales
- ✅ **Go-idiomatic:** Evita subdirectorios innecesarios en domain/
- ✅ **Equipos:** Múltiples equipos trabajan en contextos diferentes
- ✅ **Testing:** Suite Fast/Integration aisladas por contexto
- ✅ **Despliegue:** Cada contexto puede desplegarse independientemente

## Futuros Contextos Esperados

```
internal/
├── ingestion/          # ✅ Implementado
├── worker/             # ✅ Preparado
├── analytics/          # 📋 Futuro
├── billing/            # 📋 Futuro
└── notifications/      # 📋 Futuro
```

Cada uno con su propio `domain/` (plano), `application/`, `adapters/`, `ports/`.
