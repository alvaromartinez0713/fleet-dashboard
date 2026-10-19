# Infraestructura de Pruebas - Support

Este directorio contiene toda la infraestructura compartida para las pruebas del backend Go: **Object Mothers** (datos de prueba), **Fakes** (implementaciones en memoria), y **Harness** (servidores de prueba).

Ver [../README.md](../README.md) para la organización general de pruebas.

## Estructura de Directorios

```
support/
├── README.md              # Este archivo (visión general de infraestructura)
├── mothers/               # Object Mothers (builders de datos de prueba)
│   └── position_mother.go
├── fakes/                 # Implementaciones fake de puertos (en memoria)
│   ├── fake_clock.go
│   ├── fake_id_generator.go
│   ├── fake_kafka_publisher.go
│   ├── fake_circuit_breaker.go
│   └── fake_rate_limiter.go
└── harness/               # Servidores de prueba y composición
    ├── http_server.go
    └── test_logger.go
```

## Filosofía

Según `testing.md` (secciones 5-7):

1. **Object Mothers + Test Data Builder**: Datos de prueba con valores por defecto válidos y sobrescrituras explícitas
2. **Fakes para dependencias gestionadas**: Implementaciones en memoria, deterministas, con estado rastreable
3. **Harness para composición**: Servidores de prueba que inyectan dependencias y exponen accesores

**Principio clave:** Los tests entran por los **bordes públicos** (casos de uso, HTTP) y verifican **salidas observables** (respuestas, estado final, eventos publicados).

---

## 1. Object Mothers (`mothers/`)

### Propósito

Proveen datos de prueba reutilizables siguiendo el patrón **Object Mother + Test Data Builder**:

- **Object Mother**: Punto único que ofrece escenarios con nombre del dominio (`Valid()`, `OutOfRangeLatitude()`, `FutureTimestamp()`)
- **Test Data Builder**: Cada Mother devuelve un builder con valores por defecto válidos; cada prueba sobrescribe solo los datos relevantes

### Implementación

```go
// mothers/position_mother.go
type PositionRequestBuilder struct {
    vehicleID  string
    latitude   float64
    longitude  float64
    capturedAt time.Time
}

// Escenarios predefinidos
func (m PositionMother) Valid() *PositionRequestBuilder {
    return &PositionRequestBuilder{
        vehicleID:  "VH-001",
        latitude:   40.7128,
        longitude:  -74.0060,
        capturedAt: time.Now().Add(-1 * time.Minute), // 1 min atrás (pasado válido)
    }
}

func (m PositionMother) OutOfRangeLatitude() *PositionRequestBuilder {
    return m.Valid().WithLatitude(120.0) // > 90
}

func (m PositionMother) FutureTimestamp() *PositionRequestBuilder {
    return m.Valid().WithCapturedAt(time.Now().Add(10 * time.Minute)) // futuro
}

// Builder methods
func (b *PositionRequestBuilder) WithVehicleID(id string) *PositionRequestBuilder {
    b.vehicleID = id
    return b
}

func (b *PositionRequestBuilder) WithLatitude(lat float64) *PositionRequestBuilder {
    b.latitude = lat
    return b
}

func (b *PositionRequestBuilder) Build() *PositionRequest {
    return &PositionRequest{
        VehicleID:  b.vehicleID,
        Latitude:   b.latitude,
        Longitude:  b.longitude,
        CapturedAt: b.capturedAt.Format(time.RFC3339),
    }
}
```

### Uso en Pruebas

```go
// Caso 1: Usar escenario predefinido directamente
position := mothers.PositionMother{}.Valid().Build()

// Caso 2: Escenario predefinido con sobrescrituras
position := mothers.PositionMother{}.Valid()
    .WithVehicleID("VH-CUSTOM")
    .WithLatitude(45.5017)
    .Build()

// Caso 3: Escenario de error predefinido
position := mothers.PositionMother{}.OutOfRangeLatitude().Build()
```

### Reglas de Object Mothers (testing.md sección 7)

- ✅ Valores por defecto **válidos y deterministas** (sin aleatoriedad sin seed, sin hora real)
- ✅ Cada llamada crea una **instancia nueva** (sin compartir objetos mutables)
- ✅ El builder construye a través del **constructor o factory del dominio** (respeta invariantes)
- ✅ Solo escenarios **reutilizados en varias pruebas**; variantes puntuales se expresan con el builder en la prueba

---

## 2. Fakes (`fakes/`)

### Propósito

Implementaciones en memoria de **puertos** (interfaces) que permiten pruebas rápidas, deterministas y sin dependencias externas.

Según `testing.md` (sección 5):
- **Dependencias gestionadas** (solo las usa nuestra aplicación): usar el mismo motor en instancia efímera (ej. Kafka con Testcontainers en Suite Integration)
- **Dependencias no gestionadas** (observadas por terceros): sustituir con fakes en el borde (ej. Clock, IDGenerator, KafkaPublisher)

### Fakes Disponibles

#### 2.1 FakeClock

Implementa `ports.Clock` con tiempo controlable.

**Características:**
- Thread-safe con `sync.RWMutex`
- `Advance(duration)`: Avanza el tiempo sin esperar
- `SetTime(t)`: Establece un tiempo específico
- Inicializa en `time.Now()` por defecto

**Uso:**
```go
clock := fakes.NewFakeClock()

// Avanzar tiempo sin esperar
clock.Advance(30 * time.Second)

// Establecer tiempo específico
clock.SetTime(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))

// Obtener tiempo actual (fake)
now := clock.Now()
```

**Cuándo usar:** Siempre que se necesite control determinista del tiempo (timestamps, ventanas de rate limiting, Circuit Breaker timeouts).

---

#### 2.2 FakeIDGenerator

Implementa `ports.IDGenerator` con UUIDs determinísticos.

**Características:**
- Thread-safe con `sync.Mutex`
- Genera UUIDs basados en contador (predecibles)
- Soporta UUIDs predefinidos (cicla a través de ellos)

**Uso:**
```go
// Generación determinista basada en contador
gen := fakes.NewFakeIDGenerator()
id1 := gen.Generate() // Predecible
id2 := gen.Generate() // Siguiente predecible

// UUIDs predefinidos
uuids := []uuid.UUID{
    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
    uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
}
gen := fakes.NewFakeIDGeneratorWithUUIDs(uuids)
id := gen.Generate() // Retorna el primero, luego cicla
```

**Cuándo usar:** Para event_id, trace_id, request_id deterministas en pruebas.

---

#### 2.3 FakeKafkaPublisher

Implementa `ports.KafkaPublisher` con rastreo de eventos publicados.

**Características:**
- Thread-safe con `sync.RWMutex`
- Registra todos los eventos publicados
- Soporta simulación de errores (`SetFailNextPublish()`, `SetSimulateError()`)
- Soporta delay de publicación (`SetPublishDelay()`)
- `WaitForEvent()` para esperar eventos en pruebas asíncronas

**Uso:**
```go
pub := fakes.NewFakeKafkaPublisher()

// Publicar (sin Kafka real)
pub.PublishAsync(ctx, event, callback)

// Verificar eventos publicados
events := pub.GetPublishedEvents()
count := pub.GetPublishedEventCount()
assert.Equal(t, 1, count)

// Simular error en próxima publicación
pub.SetFailNextPublish(true)

// Simular error continuo
pub.SetSimulateError(true)

// Esperar evento (para pruebas async)
pub.WaitForEvent(1 * time.Second)

// Reset para siguiente prueba
pub.Reset()
```

**Cuándo usar:** Suite Fast (sin Kafka real). Suite Integration usa Kafka real con Testcontainers.

---

#### 2.4 FakeCircuitBreaker

Implementa `ports.CircuitBreaker` con transiciones de estado controlables.

**Características:**
- Thread-safe con `sync.RWMutex`
- Soporta estados CLOSED, OPEN, HALF_OPEN
- Simula fallos (`SetSimulateFailure()`)
- Fuerza transiciones de estado (`ForceOpen()`, `ForceClosed()`)

**Uso:**
```go
cb := fakes.NewFakeCircuitBreaker()

// Uso normal
err := cb.Execute(ctx, func() error {
    // Operación protegida
    return nil
})

// Forzar estado OPEN (simula 5 fallos consecutivos)
cb.ForceOpen()

// Simular fallo en próxima ejecución
cb.SetSimulateFailure(true)

// Verificar estado
state := cb.GetState() // "CLOSED", "OPEN", "HALF_OPEN"
```

**Cuándo usar:** Suite Fast para verificar lógica de manejo de Circuit Breaker (503 cuando OPEN). Suite Integration prueba transiciones reales con Toxiproxy.

---

#### 2.5 FakeRateLimiter

Implementa `ports.RateLimiter` con límites configurables.

**Características:**
- Thread-safe con `sync.RWMutex`
- Soporta whitelist y blacklist de vehículos
- Modos extremos: `AlwaysAllow`, `AlwaysDeny`
- Rastreo de llamadas por vehículo

**Uso:**
```go
limiter := fakes.NewFakeRateLimiter(12) // Límite: 12 requests

// Verificar límite
allowed := limiter.Allow(ctx, "VH-001")

// Permitir vehículo específico (bypass límite)
limiter.AllowVehicle("VH-VIP")

// Bloquear vehículo específico
limiter.BlockVehicle("VH-BAD")

// Modo siempre permitir (bypass límite para todos)
limiter.SetAlwaysAllow(true)

// Reset para siguiente prueba
limiter.Reset()
limiter.ResetForVehicle("VH-001")

// Verificar cuántas llamadas hubo
count := limiter.GetCallCount("VH-001")
```

**Cuándo usar:** Suite Fast para verificar lógica de rate limiting (429 al exceder límite). Suite Integration prueba ventana deslizante real con FakeClock inyectado en RateLimiterImpl real.

---

### Diferencia: Fakes vs Mocks

Según `testing.md` (sección 5):

| Fake | Mock |
|------|------|
| Implementación simplificada con estado rastreable | Verifica llamadas y secuencias de interacción |
| ✅ Usado en este proyecto | ❌ Evitado en este proyecto |
| Ejemplo: `kafka.GetPublishedEventCount() == 1` | Ejemplo: `mock.ExpectCall("Publish").Times(1)` |

**Por qué preferimos fakes:**
- **Resistencia al refactoring**: Un fake sigue funcionando si cambio la implementación interna
- **Observable**: Verificamos salidas (eventos publicados), no cómo se llegó al resultado
- **Determinista**: Mismo estado inicial → mismo resultado

---

## 3. Harness (`harness/`)

### Propósito

Servidores de prueba que componen implementaciones reales con fakes inyectados, exponiendo accesores para verificar estado interno.

### 3.1 TestHTTPServer

Compone:
- Router Gin con rutas reales (`POST /api/v1/positions`)
- Middleware stack (TraceID, Idempotency, ContentType, RateLimit, ErrorHandler)
- Casos de uso con fakes inyectados
- Servidor HTTP efímero en puerto aleatorio

**Uso:**
```go
// Arrange
server, err := harness.NewTestHTTPServer()
require.NoError(t, err)
defer server.Stop()

server.Start()

// Acceder a fakes inyectados para configurar comportamiento
clock := server.GetClock()
clock.Advance(30 * time.Second)

kafka := server.GetKafkaPublisher()
// ... configurar si es necesario

// Act
resp, err := http.Post(server.BaseURL()+"/api/v1/positions", "application/json", body)

// Assert
assert.Equal(t, http.StatusAccepted, resp.StatusCode)
assert.Equal(t, 1, kafka.GetPublishedEventCount()) // Verificar efecto observable
```

**Composición interna:**
```
TestHTTPServer
├── FakeClock (inyectado en usecase + middleware)
├── FakeIDGenerator (inyectado en usecase)
├── FakeKafkaPublisher (inyectado en usecase)
├── FakeCircuitBreaker (inyectado en middleware)
├── FakeRateLimiter (inyectado en middleware)
├── Gin Router + Middleware Stack
└── HTTP Server (puerto aleatorio)
```

---

### 3.2 TestLogger

Logger minimalista para pruebas que suprime ruido en salida.

**Uso:**
```go
logger := harness.NewTestLogger()
logger.Info("Processing request", "trace_id", traceID)
logger.Error("Failed to publish", "error", err)
```

**Características:**
- Output controlado (solo errores críticos por defecto)
- Soporta nivel de log configurable para debugging
- No interfiere con salida de `go test -v`

---

## Patrón de Uso Completo

### Suite Fast: Caso de Uso Directo

```go
func TestReceivePositionUseCaseAsync(t *testing.T) {
    // ========== ARRANGE ==========
    clock := fakes.NewFakeClock()
    idGen := fakes.NewFakeIDGenerator()
    kafkaPub := fakes.NewFakeKafkaPublisher()
    
    useCase := application.NewReceivePositionUseCase(
        clock,
        idGen,
        kafkaPub,
    )
    
    cmd := mothers.PositionMother{}.Valid().Build()
    
    // ========== ACT ==========
    result, err := useCase.Execute(context.Background(), cmd)
    
    // ========== ASSERT ==========
    require.NoError(t, err)
    assert.Equal(t, "accepted", result.Status)
    
    // Verificar efecto observable: evento publicado
    kafkaPub.WaitForEvent(500 * time.Millisecond)
    assert.Equal(t, 1, kafkaPub.GetPublishedEventCount())
}
```

### Suite Fast: HTTP Completo con Harness

```go
func TestValidPositionReturns202(t *testing.T) {
    // ========== ARRANGE ==========
    server, _ := harness.NewTestHTTPServer()
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
    
    // Verificar evento publicado
    kafka := server.GetKafkaPublisher()
    kafka.WaitForEvent(500 * time.Millisecond)
    assert.Equal(t, 1, kafka.GetPublishedEventCount())
}
```

---

## Garantías de los Fakes

Usar esta infraestructura garantiza:

1. **Aislamiento**: Pruebas no dependen de Kafka real, reloj del sistema, ni red
2. **Determinismo**: Mismas entradas → mismas salidas, sin race conditions
3. **Velocidad**: Operaciones en memoria, <1s total para Suite Fast
4. **Observabilidad**: Estado interno accesible para aserciones (`GetPublishedEventCount()`, `GetCallCount()`)
5. **Reproducibilidad**: Mismo seed → mismos resultados (FakeClock, FakeIDGenerator)

---

## Ciclo de Vida en Pruebas

Los fakes se crean **frescos** en cada prueba (sin compartir estado):

```go
func TestExampleBehavior(t *testing.T) {
    // Cada prueba crea sus propios fakes
    clock := fakes.NewFakeClock()
    idGen := fakes.NewFakeIDGenerator()
    kafkaPub := fakes.NewFakeKafkaPublisher()
    
    // ... usar en la prueba
    
    // No es necesario cleanup explícito (garbage collector se encarga)
    // Pero si se usó TestHTTPServer:
    defer server.Stop() // Libera puerto
}
```

**Regla importante:** Nunca compartir fakes entre pruebas (viola independencia).

---

## Antipatrones Prohibidos

Según `testing.md`:

- ❌ **Mockear colaboradores internos** (otros casos de uso, servicios de dominio) — solo mockear dependencias externas
- ❌ **Mockear clientes HTTP** en lugar de interceptar a nivel de red (MSW para frontend)
- ❌ **Verificar llamadas internas** cuando no son el comportamiento observable
- ❌ **Compartir fakes entre pruebas** (viola independencia)
- ❌ **Usar tiempo real** (`time.Now()`) en lugar de FakeClock

---

## Referencias

- [testing.md](../../../.kiro/steering/testing.md) - Secciones 5-7 (Dobles de prueba, Datos de prueba, Aislar acoplamiento)
- [../README.md](../README.md) - Organización general de pruebas
- [../ingestion/README.md](../ingestion/README.md) - Uso en módulo ingestion
