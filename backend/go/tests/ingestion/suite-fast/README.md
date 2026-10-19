# Suite Fast - Pruebas Rápidas (<1 segundo)

Pruebas negras que entran por casos de uso con **fakes inyectados**. Sin Docker, sin estado compartido, sin mocks de colaboradores internos.

## 📋 Overview

- **Cantidad:** 17 tests
- **Tiempo total:** <1 segundo
- **Entrada:** Casos de uso + DTO mappers
- **Salida:** Entity, Error, o Response
- **Aislamiento:** Fakes: Clock, IDGenerator, KafkaPublisher, RateLimiter, CircuitBreaker

## 🎯 Tests por Categoría

### Validación (4 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| missing_vehicle_id_returns_422 | P1 | vehicleId requerido |
| latitude_out_of_range_returns_422 | P1 | -90 ≤ latitude ≤ 90 |
| longitude_out_of_range_returns_422 | P1 | -180 ≤ longitude ≤ 180 |
| future_captured_at_returns_422 | P5 | capturedAt ≤ now |

### Middleware (2 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| invalid_content_type_returns_415 | P7 | Content-Type: application/json |
| idempotency_key_validation | P1 | X-Idempotency-Key UUID format |

### Rate Limiting (2 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| rate_limit_exceeded_returns_429 | P6 | 12/60s por vehicle |
| rate_limit_window_sliding | P6 | Ventana se desliza |

### Circuit Breaker (3 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| circuit_breaker_opens_on_repeated_failures | P3 | Abre tras 5 fallos |
| circuit_breaker_half_opens_after_timeout | P3 | OPEN → HALF_OPEN (30s) |
| circuit_breaker_closes_after_successes | P3 | HALF_OPEN → CLOSED (2 éxitos) |

### Timestamps (1 test)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| latency_timestamps_correct | P4 | capturedAt ≤ receivedAt ≤ processedAt |

### TraceID (2 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| response_includes_trace_id_header | P9 | X-Trace-ID presente en response |
| invalid_trace_id_header_returns_400 | P9 | X-Trace-ID UUID v4 |

### Posición Válida (1 test)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| valid_position_returns_202 | P2 | 202 Accepted, evento publicado |

### Malformación (2 tests)
| Test | Propiedad | Verifica |
|------|-----------|----------|
| malformed_json_returns_400 | P1 | JSON inválido → 400 |
| request_exceeds_max_size_returns_413 | P1 | > 1MB → 413 |

## 🏃 Ejecución

```bash
cd backend/go

# Todos los tests
make test-ingestion-fast

# Un archivo específico
go test -v ./tests/ingestion/suite-fast/validation_test.go

# Un test específico
go test -v -run TestValidPositionReturns202 ./tests/ingestion/suite-fast/

# Con cobertura
go test -v -cover ./tests/ingestion/suite-fast/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🏗 Estructura de Archivos

```
suite-fast/
├── valid_position_test.go        (1 test: válido → 202)
├── validation_test.go            (4 tests: campos requeridos, ranges)
├── middleware_test.go            (2 tests: content-type, idempotency)
├── rate_limit_test.go            (2 tests: límite 12/60s, ventana deslizante)
├── circuit_breaker_test.go       (3 tests: opens, half-open, closes)
├── timestamp_test.go             (1 test: latencias correctas)
├── trace_id_test.go              (2 tests: presente, válido)
└── validation_test.go            (2 tests: malformed JSON, oversized)
```

## 🔍 Cómo se Estructura un Test

Todos siguen **Arrange / Act / Assert**:

```go
func TestValidPositionReturns202(t *testing.T) {
    // ========== ARRANGE ==========
    // 1. Crear TestHTTPServer con fakes
    server, _ := support.NewTestHTTPServer()
    defer server.Stop()
    
    // 2. Construir datos con Object Mother
    request := mothers.PositionMother.Valid().Build()
    
    // ========== ACT ==========
    // 3. Enviar HTTP POST
    resp, _ := http.Post(server.BaseURL()+"/api/v1/positions", ...)
    
    // ========== ASSERT ==========
    // 4. Verificar observable: status, response body
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)
    assert.Equal(t, 1, server.GetKafkaPublisher().GetPublishedEventCount())
}
```

## 🎯 Correctness Properties Validadas

| Propiedad | Tests | Descripción |
|-----------|-------|-------------|
| P1 (Validation) | 4, 2 | Inputs validados, ranges, tipos |
| P2 (Async+202) | 1 | 202 Accepted, evento publicado |
| P3 (CB-Recovery) | 3 | Circuit Breaker state machine |
| P4 (4-Timestamps) | 1 | Timestamps en orden correcto |
| P5 (Future-Validation) | 1 | capturedAt ≤ now |
| P6 (Rate-Limit) | 2 | 12/60s, ventana deslizante |
| P7 (RFC9457) | 2 | ProblemDetails estructura |
| P9 (Trace-ID) | 2 | Presente, válido UUID v4 |

## ❌ Antipatrones Prohibidos (testing.md §22)

- ❌ NO mockear colaboradores internos (VO, UseCase)
- ❌ NO verificar número de llamadas internas
- ❌ NO probar métodos privados
- ❌ NO usar snapshots
- ❌ NO compartir estado entre tests
- ❌ NO usar sleep (esperar por condición)

## ✅ Checklist Antes de Push

```bash
# 1. Suite Fast debe PASAR
make test-ingestion-fast

# 2. Verificar Composer
./verify_composer.sh

# 3. Sin binarios generados
ls -la *.test 2>/dev/null || echo "✓ Sin .test binarios"

# 4. Sin test coverage degradation
go test -cover ./tests/ingestion/suite-fast/...

# 5. Linter
go fmt ./...
go vet ./...
```

## 📚 Referencias

- `.kiro/steering/testing.md` — Estándar completo (24 secciones)
- `.kiro/specs/http-position-ingestion/design.md` — API design
- `backend/go/tests/support/fakes/README.md` — Fakes disponibles
- `backend/go/tests/support/mothers/` — Object Mothers

---

**Status:** ✅ 17/17 tests implementados

**Última actualización:** Ver git log

**Próximo:** Suite Integration (Kafka real + Toxiproxy)
