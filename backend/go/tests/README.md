# Organización de Pruebas - Backend Go

Este directorio contiene todas las pruebas automatizadas para los servicios backend en Go, organizadas por módulo y tipo de suite. Cada módulo sigue una estructura consistente con dos suites principales: **Suite Fast** y **Suite Integration**.

## Filosofía

Las pruebas siguen los principios definidos en `testing.md`:
- **Pocas pruebas, valiosas y difíciles de romper** que verifican el comportamiento importante del sistema como caja negra
- **Unidades de comportamiento**, no pruebas clase por clase o capa por capa
- **Salidas observables**: respuestas, estado final o efectos en sistemas externos
- **Resistencia al refactoring** no se negocia (sin falsos positivos)

## Estructura de Directorios

```
tests/
├── README.md                  # Este archivo (visión general)
├── ingestion/                 # Módulo HTTP Position Ingestion
│   ├── README.md              # Detalles específicos del módulo
│   ├── suite-fast/            # Pruebas rápidas (fakes, <1s)
│   └── suite-integration/     # Pruebas de integración (deps reales, <120s)
├── worker/                    # Módulo Telemetry Worker
│   ├── README.md              # Detalles específicos del módulo
│   ├── suite-fast/            # Pruebas rápidas
│   └── suite-integration/     # Pruebas de integración
└── support/                   # Infraestructura compartida de pruebas
    ├── mothers/               # Object Mothers (builders de datos de prueba)
    ├── fakes/                 # Implementaciones fake (en memoria)
    └── harness/               # Servidores de prueba y utilidades
```

Cada directorio de módulo contiene:
- **README.md**: Documentación específica de las pruebas del módulo
- **suite-fast/**: Pruebas rápidas que entran por casos de uso con dependencias fake
- **suite-integration/**: Pruebas de integración con dependencias externas reales (Kafka, TimescaleDB, Toxiproxy)

## Suites de Prueba

### Suite Fast
**Propósito:** Verificar lógica de dominio y casos de uso sin dependencias externas.

| Característica | Valor |
|----------------|-------|
| **Punto de entrada** | Casos de uso de aplicación (puertos públicos) |
| **Dependencias** | Todas fake (Clock, IDGenerator, Kafka, DB, etc.) |
| **Velocidad** | <1 segundo total |
| **Aislamiento** | Efímero, sin estado compartido, sin Docker |
| **Alcance** | Cobertura de comportamiento de reglas de dominio y validación |
| **Cuándo ejecutar** | En cada commit (pre-commit hook) |

**Ejemplo de prueba:**
- Valida restricciones de entrada (rango de latitud, formato de timestamp)
- Verifica códigos de estado HTTP y estructura de respuesta
- Comprueba comportamiento de middleware (rate limiting, circuit breaker)
- Usa Object Mothers para datos de prueba

### Suite Integration
**Propósito:** Verificar contratos con sistemas externos reales y patrones de resiliencia.

| Característica | Valor |
|----------------|-------|
| **Punto de entrada** | Endpoints HTTP + brokers de mensajes reales |
| **Dependencias** | Kafka real (Testcontainers), TimescaleDB real, Toxiproxy para fallos |
| **Velocidad** | <120 segundos |
| **Aislamiento** | Contenedores Docker efímeros, estado limpio por prueba |
| **Alcance** | Flujos end-to-end, idempotencia, inyección de fallos, validación de schema |
| **Cuándo ejecutar** | Pipeline CI, antes de merge |

**Ejemplo de prueba:**
- Publica a Kafka real y verifica schema/particionamiento
- Inyecta fallos de red con Toxiproxy (latencia, caída de conexión)
- Verifica transiciones de estado del Circuit Breaker (CLOSED → OPEN → HALF_OPEN → CLOSED)
- Prueba manejo de duplicados y rate limiting con ventana deslizante

### Suite Property-Based (Opcional)
**Propósito:** Verificar invariantes a través de secuencias arbitrarias de entrada.

Usa `rapid` (framework de Go para property-based testing) para verificar propiedades como:
- Monotonía de latencia (processingLatency ≥ receptionLatency)
- Correctitud de ventana deslizante del rate limiter (cualquier ventana de 60s acepta ≤ cuota)
- Idempotencia bajo reintentos

Parte de Suite Fast (ejecución rápida, seeds deterministas).

## Ejecución de Pruebas

### Comandos Rápidos (desde raíz del backend)

```bash
# Ejecutar todas las pruebas rápidas (siempre antes de commit)
make test-fast

# Ejecutar todas las pruebas de integración (requiere Docker)
make test-integration

# Ejecutar todas las pruebas (fast + integration)
make test

# Ejecutar pruebas de un módulo específico
make test-ingestion-fast
make test-ingestion-integration
make test-worker-fast
make test-worker-integration

# Limpiar binarios generados
make clean-tests
```

### Comandos Directos con go test

```bash
# Ejecutar pruebas rápidas de un módulo
go test -v ./tests/ingestion/suite-fast/...

# Ejecutar pruebas de integración (requiere build tag)
go test -tags=integration -v ./tests/ingestion/suite-integration/...

# Ejecutar una prueba específica
go test -v -run TestValidPositionReturns202 ./tests/ingestion/suite-fast/

# Ejecutar con cobertura
go test -v -cover ./tests/ingestion/suite-fast/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Build Tags

- **Suite Fast**: No requiere build tags
- **Suite Integration**: Requiere `-tags=integration`

Esta separación evita la ejecución accidental de pruebas lentas durante ciclos rápidos de desarrollo.

## Infraestructura de Pruebas (support/)

### Object Mothers (`support/mothers/`)
Proveen datos de prueba reutilizables con valores por defecto sensatos y un patrón builder.

**Propósito:**
- Fuente única de verdad para escenarios de prueba
- Valores por defecto válidos listos para usar
- Sobrescribir solo los campos relevantes por prueba

**Uso:**
```go
// Escenarios predefinidos
position := mothers.PositionMother{}.Valid().Build()
position := mothers.PositionMother{}.OutOfRangeLatitude().Build()

// Sobrescrituras personalizadas
position := mothers.PositionMother{}.Valid()
    .WithVehicleID("VH-001")
    .WithLatitude(40.7128)
    .Build()
```

### Fakes (`support/fakes/`)
Implementaciones en memoria de puertos (interfaces) para pruebas rápidas y deterministas.

**Fakes disponibles:**
- `FakeClock`: Control de tiempo sin esperar
- `FakeKafkaPublisher`: Verificar eventos publicados sin Kafka real
- `FakeCircuitBreaker`: Simular transiciones de estado
- `FakeRateLimiter`: Controlar cuota sin ventanas de tiempo reales
- `FakeIDGenerator`: UUIDs deterministas

**Uso:**
```go
clock := fakes.NewFakeClock()
clock.Advance(30 * time.Second)  // Viaje en el tiempo

kafka := fakes.NewFakeKafkaPublisher()
count := kafka.PublishedCount()  // Verificar interacciones
```

### Harness (`support/harness/`)
Servidores de prueba y utilidades que componen implementaciones reales con fakes.

**TestHTTPServer:**
- Compone router Gin + middleware + casos de uso
- Inyecta todos los fakes mediante inyección de dependencias
- Proporciona accesores para verificar estado interno

**TestLogger:**
- Logging minimalista para pruebas
- Suprime ruido en salida de pruebas

## Convenciones de Nombres

| Elemento | Convención | Ejemplo |
|---------|------------|---------|
| **Directorios** | kebab-case | `suite-fast`, `suite-integration` |
| **Archivos Go** | snake_case | `validation_test.go`, `fake_clock.go` |
| **Paquetes** | lowercase (sin guiones) | `package suite_fast` |
| **Funciones test** | TestCamelCase | `TestValidPositionReturns202` |

## Patrón de Estructura de Pruebas

Todas las pruebas siguen **Arrange / Act / Assert**:

```go
func TestExampleBehavior(t *testing.T) {
    // ========== ARRANGE ==========
    // Setup: crear servidor, preparar entradas, inyectar dependencias
    server, _ := support.NewTestHTTPServer()
    defer server.Stop()
    server.Start()
    
    request := mothers.PositionMother{}.Valid().Build()
    reqBody, _ := json.Marshal(request)
    
    // ========== ACT ==========
    // Ejecutar: realizar la acción bajo prueba
    httpReq, _ := http.NewRequest(http.MethodPost, 
        server.BaseURL()+"/api/v1/positions", 
        bytes.NewReader(reqBody))
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, _ := http.DefaultClient.Do(httpReq)
    defer resp.Body.Close()
    
    // ========== ASSERT ==========
    // Verificar: comprobar salidas observables
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)
    assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
    // ... más aserciones sobre body, headers, eventos publicados
}
```

**Una sola acción (Act) por prueba**: Cada prueba ejecuta una acción y verifica sus efectos observables.

## Checklist Pre-Commit

Antes de hacer commit:

```bash
# 1. Todas las pruebas rápidas deben pasar
make test-fast

# 2. Sin binarios de prueba generados en raíz del repo
ls -la *.test 2>/dev/null || echo "✓ Limpio"

# 3. Pruebas de integración pasan (si Docker está disponible)
make test-integration

# 4. Sin pruebas flaky (usar fakes deterministas, no tiempo real)
```

## Anti-Patrones (Prohibidos)

Siguiendo `testing.md`:

- ❌ **Pruebas unitarias por archivo/clase** (ej. `vehicle_id_test.go`) — probar comportamientos, no clases
- ❌ **Mockear colaboradores internos** (servicios de dominio, otros casos de uso) — solo mockear dependencias externas
- ❌ **Probar métodos privados** — entrar por puertos públicos
- ❌ **Mockear clientes HTTP** (axios, fetch) — usar interceptación de red (MSW para frontend)
- ❌ **Duplicar lógica de producción** en aserciones — verificar salidas observables
- ❌ **Pruebas acopladas al orden de ejecución** — cada prueba debe ser independiente
- ❌ **Reintentos automáticos** para ocultar inestabilidad — arreglar la causa raíz
- ❌ **Snapshots para validación** — asertar comportamientos específicos

## Métricas

Métricas actuales de pruebas por módulo:

| Módulo | Suite Fast | Suite Integration | Property-Based |
|--------|------------|-------------------|----------------|
| **ingestion** | 17 pruebas (<1s) | 8 pruebas (<120s) | 2 pruebas |
| **worker** | TBD | TBD | TBD |

## Documentación Específica por Módulo

Para escenarios de prueba detallados, criterios de aceptación y patrones específicos del módulo:

- [ingestion/README.md](./ingestion/README.md) - Pruebas de HTTP Position Ingestion
- [worker/README.md](./worker/README.md) - Pruebas de Telemetry Worker

## Referencias

- [testing.md](../../.kiro/steering/testing.md) - Estándares de pruebas (fuente única de verdad)
- [api-standards.md](../../.kiro/steering/api-standards.md) - Estándares de contratos API (RFC 9457)
- [tech.md](../../.kiro/steering/tech.md) - Stack tecnológico
- [structure.md](../../.kiro/steering/structure.md) - Límites de Clean Architecture
