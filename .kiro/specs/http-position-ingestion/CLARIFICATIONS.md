# Clarifications & Implementation Notes

## Testing Strategy Alignment with testing.md

### Problema Identificado

Inicialmente, se crearon archivos `*_test.go` en `internal/ingestion/domain/` para cada Value Object:
- `event_id_test.go`
- `vehicle_id_test.go`
- `geo_position_test.go`
- `timestamps_test.go`
- `trace_id_test.go`
- `idempotency_key_test.go`
- `rate_limit_config_test.go`
- `vehicle_position_test.go`
- Y otros en adaptadores

### Causa Raíz

Las Ondas 0-6 (domain, ports, adapters, middleware) no indicaban explícitamente que **no debían incluir archivos `_test.go`**. Por lo tanto, fue lógico crear tests unitarios por archivo al siguiente las convenciones estándar de Go (`*_test.go` junto al código).

### Solución

**Actualización de tasks.md (hecho el 2024-XX-XX):**
- Se agregó un encabezado `⚠️ ESTRATEGIA DE TESTING: Crítico` al inicio del archivo
- Cada Onda 0-6 ahora incluye una nota explícita: `⚠️ Nota: Estas tareas crean solo archivos `.go` (sin `_test.go`)...`
- Cada tarea de dominio, puertos y adaptadores incluye: `**Nota:** No crear [archivo]_test.go. [Explicación]`

**Eliminación de archivos unitarios:**
- Se eliminaron todos los `*_test.go` de `internal/ingestion/domain/`

### Regla Correcta según testing.md

**Sección 3 (Qué se prueba):**
```
| Código trivial: DTOs, getters, mapeos directos, configuración | No probar de forma dedicada |
| Reglas de negocio y algoritmos | Probar a fondo |
```

**Sección 10 (Backend):**
```
Pruebas del núcleo (la mayoría, en milisegundos):
- Entran por los casos de uso de `internal/application/usecases/`
- Usan fakes en memoria para los `ports/` de salida
- Cubren reglas de dominio, validación de eventos, idempotencia, ...
```

### Estructura Correcta de Tests

**Suite Rápida (Onda 7-11):**
- Ubicación: `tests/suite_rapida/*.go` (SOLO aquí van archivos `_test.go`)
- Entrada: Por casos de uso (e.g., `ReceivePositionUseCase.Execute()`)
- Fakes inyectados: `FakeKafkaPublisher`, `FakeClock`, `FakeIDGenerator`, `FakeRateLimiter`
- Validación indirecta: Los VOs se validan al ejercitarlos a través del caso de uso

**Suite Integración (Onda 12-13):**
- Ubicación: `tests/suite_integracion/*.go` (SOLO aquí van archivos `_test.go`)
- Entrada: Por casos de uso con infraestructura real (Kafka vía Testcontainers, Toxiproxy)
- Validación: Comportamiento observable contra sistemas reales

### Test Data (No son tests)

**Ubicación: `tests/support/`**
- `tests/support/mothers/position_mother.go` → Object Mother (NO es `_test.go`)
- `tests/support/fakes/` → Fakes (NO son `_test.go`)
- `tests/support/http_server.go` → TestHTTPServer harness (NO es `_test.go`)

Estos son **herramientas de prueba**, no tests. No tienen sufijo `_test.go`.

### Black-Box Verification

Cada prueba verifica lo **observable**:
- ✅ Respuesta HTTP (status code, headers, body JSON)
- ✅ Estado final (registros en Kafka, contadores en fakes)
- ❌ Internos (si `EventID` llamó a `IDGenerator`, estructura de `VehiclePosition`, etc.)

### Beneficios de Esta Estructura

1. **Resistencia al refactoring:** Si reescribo `GeoPosition` internamente, el test sigue pasando si el comportamiento observable es igual.
2. **Cobertura inteligente:** 17 pruebas Suite Rápida cubren 14 Correctness Properties sin necesidad de 100+ tests unitarios.
3. **Mantenibilidad:** Una prueba por comportamiento, no per-clase.
4. **Velocidad:** Suite Rápida < 1s (sin contenedores).

### Próximos Pasos

**Ondas 7-11:** Crear `tests/suite_rapida/` con tests que entren por casos de uso
**Ondas 12-13:** Crear `tests/suite_integracion/` con tests contra Kafka real

Si en futuras ejecuciones se detecta que se están creando `*_test.go` en domain/, adapters/, o middleware/, el prompt de Kiro debe recordar: **"testing.md: Solo tests en suite_rapida/ y suite_integracion/"**

---

## Changelog

- **2024-XX-XX:** Se identificó que tasks.md no dejaba claro la estrategia de testing. Se actualiza tasks.md con notas explícitas en cada Onda 0-6. Se eliminan 9 archivos `*_test.go` de domain/.
