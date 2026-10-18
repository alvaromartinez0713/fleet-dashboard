# Test Structure - Backend Go

Organización jerárquica de pruebas por módulo y suite.

## Estructura de Directorios

```
tests/
├── ingestion/                 # Módulo HTTP Position Ingestion
│   ├── suite-fast/            # Suite Fast (black-box via use cases, fakes)
│   │   ├── valid_position_test.go
│   │   ├── validation_test.go
│   │   ├── middleware_test.go
│   │   ├── rate_limit_test.go
│   │   ├── circuit_breaker_test.go
│   │   └── timestamp_test.go
│   └── suite-integration/     # Suite Integration (Testcontainers, Kafka real)
│       ├── kafka_test.go
│       ├── resilience_test.go
│       └── ...
├── worker/                    # Módulo Worker (próximamente)
│   ├── suite-fast/
│   └── suite-integration/
└── support/                   # Shared test infrastructure
    ├── mothers/               # Object Mothers (test data)
    │   ├── position_mother.go
    │   └── ...
    ├── fakes/                 # Fake implementations (in-memory)
    │   ├── fake_clock.go
    │   ├── fake_kafka_publisher.go
    │   ├── fake_circuit_breaker.go
    │   └── ...
    └── harness/               # Test server & utilities
        ├── http_server.go
        ├── test_logger.go
        └── ...
```

## Convenciones de Nombres

- **Directorios**: kebab-case (suite-fast, suite-integration)
- **Archivos .go**: snake_case (position_test.go, fake_clock.go)
- **Paquetes Go**: lowercase, sin guiones (package suite_fast)
- **Funciones test**: TestCamelCase (TestValidPositionReturns202)

## Ejecución de Pruebas

### Desde la raíz del backend (recomendado)

```bash
# Suite Fast (ingestion module) - SIEMPRE ejecutar antes de commit
make test-ingestion-fast

# Suite Integration (requiere Docker)
make test-ingestion-integration

# Todas las pruebas
make test

# Limpiar binarios generados
make clean-tests
```

### Directamente con go test

```bash
# Suite Fast específica
cd tests/ingestion && go test -v ./suite-fast/...

# Un archivo específico
cd tests/ingestion && go test -v ./suite-fast/validation_test.go

# Una prueba específica
cd tests/ingestion && go test -v -run TestValidPositionReturns202 ./suite-fast/

# Con cobertura
cd tests/ingestion && go test -v -cover ./suite-fast/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Estrategia de Testing

### Suite Fast
- **Entrada:** Por casos de uso (puertos públicos de aplicación)
- **Dependencias:** Todas con fakes (Clock, IDGenerator, KafkaPublisher, RateLimiter, CircuitBreaker)
- **Velocidad:** <1s total
- **Aislamiento:** Ephemeral, sin estado compartido, sin Docker
- **Cantidad:** 17 pruebas
- **Cobertura:** Comportamiento observable (status HTTP, response body, headers)

### Suite Integration
- **Entrada:** HTTP real + Kafka real (Testcontainers)
- **Dependencias:** Kafka real (Testcontainers), Toxiproxy para inyección de fallos
- **Velocidad:** <120s
- **Aislamiento:** Contenedores efímeros, limpios entre pruebas
- **Cantidad:** 8 pruebas
- **Cobertura:** End-to-end, idempotencia, resiliencia, duplicados

## Estructura de un Test

Todos los tests siguen Arrange / Act / Assert:

```go
func TestValidPositionReturns202(t *testing.T) {
    // ========== ARRANGE ==========
    server, _ := support.NewTestHTTPServer()
    defer server.Stop()
    server.Start()
    
    request := mothers.PositionMother{}.Valid().Build()
    reqBody, _ := json.Marshal(request)
    
    // ========== ACT ==========
    httpReq, _ := http.NewRequest(http.MethodPost, 
        server.BaseURL()+"/api/v1/positions", 
        bytes.NewReader(reqBody))
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, _ := http.DefaultClient.Do(httpReq)
    defer resp.Body.Close()
    
    // ========== ASSERT ==========
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)
    // ... más verificaciones
}
```

## Support: Mothers, Fakes, Harness

### Object Mothers (Datos de Prueba)
```go
// Escenarios predefinidos con builder pattern
position := mothers.PositionMother{}.Valid().Build()
position := mothers.PositionMother{}.FutureTimestamp().Build()
position := mothers.PositionMother{}.OutOfRangeLatitude().Build()
position := mothers.PositionMother{}.Valid()
    .WithVehicleID("VH-001")
    .WithLatitude(40.7128)
    .Build()
```

### Fakes (Inyectable Implementations)
```go
// TestHTTPServer inyecta todos los fakes
server, _ := support.NewTestHTTPServer()

clock := server.GetClock()
clock.Advance(30 * time.Second)  // Avanza tiempo sin esperar

kafka := server.GetKafkaPublisher()
count := kafka.PublishedCount()   // Verifica eventos publicados

limiter := server.GetRateLimiter()
limiter.Reset()                   // Reset para siguiente test
```

### Harness
- **TestHTTPServer:** Compone Gin + fakes + middleware + rutas
- **TestLogger:** Logger minimalista para no contaminar output

## Checklist Antes de Commit

```bash
# 1. Suite Fast debe PASAR
make test-ingestion-fast

# 2. Sin archivos generados en raíz
ls -la *.test 2>/dev/null || echo "✓ Sin .test binarios"

# 3. Sin mezcla de idiomas en nombres
find tests -type d | grep -E "[a-z]+-[a-z]" || echo "✓ Nombres consistentes"

# 4. Suite Integration pasa (si tienes Docker)
make test-ingestion-integration
```

## Antipatrones Prohibidos

Según `testing.md`:

- ❌ Crear tests unitarios por archivo (e.g., `vehicle_id_test.go`)
- ❌ Mockear clientes HTTP internos (use MSW o interceptación)
- ❌ Testear métodos privados o detalles de implementación
- ❌ Usar snapshots para UI
- ❌ Mockar colaboradores internos (servicios de dominio, otros casos de uso)
- ❌ Duplicar lógica de producción dentro del test
- ❌ Tests acopladas a orden de ejecución o estado compartido
- ❌ Reintentos automáticos para ocultar inestabilidad (flaky tests)

## Métricas de Pruebas

| Métrica | Suite Fast | Suite Integration |
|---------|------------|------------------|
| Cantidad | 17 pruebas | 8 pruebas |
| Tiempo | <1s | <120s |
| Aislamiento | Ephemeral fakes | Testcontainers |
| Entrada | Casos de uso | HTTP + Kafka |
| Dependencias | Todas fake | Reales |
| Cobertura | Comportamiento | End-to-end |

## Referencias

- `testing.md` - Estándar único de pruebas del proyecto
- `requirements.md` - Requisitos funcionales de HTTP Position Ingestion
- `design.md` - Diseño de la API y composición de dependencias
- `api-standards.md` - Estándares de API (RFC 9457, etc.)
