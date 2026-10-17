# Especificación: Servicio HTTP de Ingesta de Posiciones

## 1. Propósito

Implementar un endpoint HTTP que reciba posiciones individuales de vehículos, valide el contrato contra una especificación explícita, publique el evento en Kafka de forma asíncrona y responda rápidamente al cliente con una respuesta clara sobre la aceptación o rechazo de la solicitud.

La implementación debe contemplar explícitamente los fallos de Kafka y definir una estrategia clara de reintentos, alertas y garantías de entrega.

## 2. Requisitos funcionales

### RF-01. Endpoint HTTP de ingesta

El servicio HTTP debe exponer un endpoint que acepte posiciones de vehículos:

- **Método:** `POST`
- **Ruta:** `/positions` o `/api/v1/positions` (según arquitectura)
- **Content-Type:** `application/json`
- **Timeout:** 5 segundos (configurables)

### RF-02. Validación del contrato de solicitud

Cada solicitud debe validarse contra un esquema definido. Los campos requeridos son:

```json
{
  "vehicleId": "string (uuid)",
  "latitude": "number (-90 a 90)",
  "longitude": "number (-180 a 180)",
  "timestamp": "RFC 3339 (ISO 8601)",
  "accuracy": "number (metros, >= 0, optional)"
}
```

**Validaciones:**
- `vehicleId` es requerido y debe ser un UUID válido
- `latitude` debe estar en rango [-90, 90]
- `longitude` debe estar en rango [-180, 180]
- `timestamp` debe ser válido y no más de 1 hora en el futuro
- `accuracy` es opcional; si se proporciona, debe ser >= 0
- Rechazar campos desconocidos (strict mode)
- Rechazar payloads > 10 KB
- Rechazar lotes (solo una posición por solicitud en este endpoint)

**Respuesta de error de validación:**

```http
HTTP/1.1 422 Unprocessable Content
Content-Type: application/problem+json
```

```json
{
  "type": "https://api.fleet-platform.local/problems/validation-error",
  "title": "La solicitud contiene datos inválidos",
  "status": 422,
  "detail": "El campo 'latitude' debe estar entre -90 y 90",
  "instance": "/problems/occurrences/abc123"
}
```

### RF-03. Publicación asíncrona en Kafka

Una vez validada la solicitud:

1. **Transformación:** Convertir la posición en un evento de dominio con:
   - `eventId`: UUID generado (para deduplicación)
   - `vehicleId`: copiado de la solicitud
   - `latitude`, `longitude`, `accuracy`: copiados
   - `timestamp`: usar el del cliente si es válido, o el server si es futuro
   - `receivedAt`: timestamp del servidor (cuando se recibió)
   - `version`: "1.0" (para versionado del evento)

2. **Publicación:**
   - Topic Kafka: `position.ingested` (configurable)
   - Key: `vehicleId` (para particionamiento por vehículo)
   - Formato: JSON
   - Confirmación: Esperar ACK de al menos 1 replica antes de responder

3. **Reintentos de Kafka:**
   - Máximo 3 intentos
   - Backoff exponencial: 100ms, 300ms, 1s
   - Timeout por intento: 2 segundos
   - Si falla tras 3 intentos: ir a estrategia de fallos (RF-04)

### RF-04. Estrategia explícita de fallos de Kafka

**Opción A: Fast Fail (recomendada para MVP)**

Si la publicación en Kafka falla tras los reintentos:

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/problem+json
```

```json
{
  "type": "https://api.fleet-platform.local/problems/kafka-unavailable",
  "title": "El servicio de mensajería no está disponible",
  "status": 503,
  "detail": "No se pudo publicar el evento después de 3 intentos. Reintente más tarde.",
  "instance": "/problems/occurrences/xyz789"
}
```

El cliente debe reintentar. El evento no se pierde (no se aceptó), así que no hay garantía de entrega en este caso.

**Opción B: Dead Letter Queue (alternativa)**

Si se implementa DLQ (Decision Pending):

- Mantener una cola de eventos fallidos locales (en memoria o SQLite ephemeral)
- Responder `202 Accepted` inmediatamente
- Background worker reintenta la publicación con backoff más agresivo
- Si persisten los fallos después de N intentos, loguear alerta crítica y mantener el evento en DLQ

Decisión: **Elegir Opción A para el MVP**. Si el tiempo lo permite, se implementa Opción B con revisión de decisión.

### RF-05. Respuesta de éxito

Si la solicitud es válida y se publicó en Kafka:

```http
HTTP/1.1 202 Accepted
Content-Type: application/json
```

```json
{
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "accepted",
  "message": "La posición ha sido aceptada para procesamiento"
}
```

**Semántica:** `202 Accepted` indica que la solicitud fue aceptada pero el procesamiento aún está en curso (Kafka está procesando el evento).

### RF-06. Observabilidad

- **Logging:** Cada solicitud debe registrar:
  - `traceId` (generado o recibido en header `X-Trace-ID`)
  - `vehicleId`
  - Resultado: `accepted`, `rejected` (validation), `rejected` (kafka)
  - Latencia de la solicitud
- **Métricas:**
  - `positions.received.total` (counter)
  - `positions.accepted.total` (counter)
  - `positions.rejected.total` (counter, con etiqueta `reason`)
  - `positions.kafka.errors.total` (counter)
  - `positions.request.duration.seconds` (histogram)
- **Alertas:**
  - Si `positions.kafka.errors` > 5 en 1 minuto → alerta WARN en logs
  - Si `positions.kafka.errors` > 10 en 1 minuto → alerta ERROR

### RF-07. Límites y configuración

- **Rate limiting:** No implementar en endpoint individual; dejar para fase 2 (reverse proxy / API Gateway)
- **Timeout HTTP:** 5 segundos (configurable en `config/application.yml`)
- **Timeout Kafka:** 2 segundos por intento
- **Buffer Kafka:** 1000 mensajes (configurable)
- **Compresión Kafka:** snappy (configurable)

## 3. Criterios de aceptación

### CA-01. Validación de contrato

- [ ] La API rechaza solicitudes sin `vehicleId` con `400 Bad Request`
- [ ] La API rechaza `latitude` fuera de rango con `422 Unprocessable Content`
- [ ] La API rechaza `longitude` fuera de rango con `422 Unprocessable Content`
- [ ] La API rechaza timestamps inválidos con `422 Unprocessable Content`
- [ ] La API rechaza timestamps futuros (> 1 hora) con `422 Unprocessable Content`
- [ ] La API rechaza campos desconocidos con `400 Bad Request` (strict mode)
- [ ] La API rechaza payloads > 10 KB con `413 Content Too Large`
- [ ] Los mensajes de error siguen RFC 9457 Problem Details

**Verifies:** RF-02, Security/validation

### CA-02. Publicación exitosa en Kafka

- [ ] Una solicitud válida es aceptada (`202 Accepted`)
- [ ] El evento llega a Kafka en el topic `position.ingested`
- [ ] El evento contiene los campos esperados (vehicleId, latitude, longitude, timestamp, accuracy, eventId, receivedAt, version)
- [ ] El evento tiene una `key` equal a `vehicleId` para particionamiento
- [ ] La respuesta `202` incluye un `eventId` único

**Verifies:** RF-03, RF-05

### CA-03. Fallos de Kafka (estrategia Fast Fail)

- [ ] Si Kafka no responde, el servidor reintenta 3 veces con backoff
- [ ] Si tras 3 reintentos sigue sin respuesta, se retorna `503 Service Unavailable`
- [ ] El cliente recibe `503` sin haber guardado nada localmente (el evento no fue aceptado)
- [ ] Los logs registran cada reintento fallido con `traceId`

**Verifies:** RF-04, RF-06, RNF-06

### CA-04. Observabilidad

- [ ] Cada solicitud tiene un `traceId` (generado o en header `X-Trace-ID`)
- [ ] Los logs contienen: `vehicleId`, `status`, `traceId`, `latency`
- [ ] Las métricas `positions.received.total`, `positions.accepted.total`, `positions.rejected.total` se incrementan correctamente
- [ ] Las métricas `positions.kafka.errors.total` y `positions.request.duration.seconds` se registran
- [ ] Se generan alertas WARN/ERROR cuando `positions.kafka.errors` exceden umbrales

**Verifies:** RF-06, RNF-10

### CA-05. Contrato OpenAPI

- [ ] El endpoint está documentado en OpenAPI con esquema de solicitud/respuesta
- [ ] Las respuestas de error se representan con `application/problem+json`
- [ ] Los consumidores pueden validar las respuestas contra el contrato

**Verifies:** RF-02, api-standards/9

## 4. Decisiones de diseño

### D-01. Estrategia de fallos: Fast Fail (MVP)

**Decisión:** La estrategia inicial es "Fast Fail" (RF-04 Opción A):
- Si Kafka falla, responder `503` inmediatamente
- No guardar eventos en DLQ local
- Permitir que el cliente reintente

**Justificación:**
- Más simple de implementar y probar en el MVP
- Explícito: el cliente sabe que el evento fue rechazado
- DLQ agrega complejidad sin necesidad demostrada

**Implicaciones:**
- Los clientes deben implementar reintentos exponenciales
- No hay garantía de entrega en caso de fallos de Kafka (aceptable para telemetría)
- Posible pérdida de eventos si el cliente no reintenta

**Revisión:** Si los requisitos cambian (ej: garantía de entrega exactamente una vez), se reevaluará y se documentará en ADR.

### D-02. Semántica HTTP: `202 Accepted`

**Decisión:** Usar `202 Accepted` en lugar de `201 Created` o `200 OK`

**Justificación:**
- La solicitud es válida y fue aceptada
- El procesamiento (almacenamiento en TimescaleDB) ocurre asincronicamente después
- `202` comunica explícitamente al cliente que debe esperar procesamiento futuro
- RFC 9110: "202 (Accepted) indica que la solicitud fue aceptada para procesamiento, pero el procesamiento no ha sido completado"

### D-03. Particionamiento en Kafka

**Decisión:** Usar `vehicleId` como key de Kafka

**Justificación:**
- Todas las posiciones de un vehículo van a la misma partición
- Garantiza orden de eventos por vehículo (importante para state machine del vehículo)
- Facilita el procesamiento en el worker

### D-04. Versionado de eventos

**Decisión:** Incluir `version: "1.0"` en cada evento

**Justificación:**
- Permite evolucionar el esquema sin romper consumidores
- El worker puede ignorar eventos con versiones no soportadas
- Facilita migraciones futuras

## 5. Consideraciones de seguridad

### SEC-01. Validación estricta

- Validar todas las entradas en el servidor
- No confiar en tipos del cliente
- Rechazar campos desconocidos (strict mode JSON)

### SEC-02. DoS

- Limitar tamaño de payload (10 KB)
- Limitar tasa de solicitudes (implementar en fase 2 con API Gateway)
- No permitir lotes (una posición por solicitud)

### SEC-03. Información sensible

- No exponer errores internos en respuestas
- No loguear payloads completos
- Usar RFC 9457 Problem Details para errores

### SEC-04. Autenticación

No implementar en esta fase (fase 2). Asumir que la solicitud viene de una aplicación confiable.

## 6. Estructura de implementación

```
backend/go/
├── internal/
│   ├── ingestion/
│   │   ├── domain/
│   │   │   ├── errors.go          # Errores de dominio
│   │   │   ├── position.go        # Aggregate Position (NEW)
│   │   │   └── trace.go
│   │   ├── http/
│   │   │   ├── handler/
│   │   │   │   └── position.go    # Handler HTTP (NEW)
│   │   │   ├── middleware/        # error.go, logging.go, trace_id.go
│   │   │   └── router.go          # Rutas HTTP
│   │   ├── ports/
│   │   │   ├── messaging.go       # Interface para Kafka (NEW)
│   │   │   ├── health.go
│   │   │   ├── logging.go
│   │   │   └── server.go
│   │   └── config/
│   │       ├── composer.go        # Inyección de dependencias
│   │       └── ...
│   └── ...
├── tests/
│   └── support/
│       ├── mothers/
│       │   └── position_mother.go (NEW) # Object Mother para posiciones
│       ├── network/
│       │   └── handlers.go (NEW)  # Handlers de MSW (web)
│       └── fakes/
│           └── kafka_producer.go  (NEW) # Fake de Kafka para pruebas
```

## 7. Plan de implementación

### Fase 1: Dominio y puertos (rápido)

1. Definir aggregate `Position` en `internal/ingestion/domain/position.go`
2. Definir interface `MessagingPort` en `internal/ingestion/ports/messaging.go`
3. Definir casos de uso / handlers de aplicación
4. Crear Object Mother para posiciones

### Fase 2: Adaptadores HTTP y Kafka

1. Implementar handler HTTP en `internal/ingestion/http/handler/position.go`
2. Implementar producer Kafka en adaptador (nuevo)
3. Configurar middleware de error y logging
4. Registrar rutas en router

### Fase 3: Pruebas

1. Pruebas de dominio (validación de Position)
2. Pruebas de handler HTTP con fakes de Kafka
3. Pruebas de integración con Kafka en contenedor
4. Pruebas de resiliencia: fallos de Kafka con Toxiproxy

### Fase 4: Documentación y CI/CD

1. Documentar en OpenAPI
2. Configurar validación OpenAPI en tests
3. Añadir a pipelines CI/CD

## 8. Referencias

- RFC 9457: Problem Details for HTTP APIs
- RFC 9110: HTTP Semantics
- `testing.md` (sección 8 y 9): Interceptación de red y contrato OpenAPI
- `api-standards.md`: Diseño e implementación de API
- `security.md` (sección 6): Validación de entradas

## 9. Matriz de trazabilidad

| Criterio | Suite | Ruta |
|----------|-------|------|
| CA-01 | Rápida | `tests/ingestion/handler_position_test.go` |
| CA-02 | Integración | `tests/ingestion/integration/kafka_publisher_test.go` |
| CA-03 | Integración (Toxiproxy) | `tests/ingestion/resilience/kafka_failures_test.go` |
| CA-04 | Rápida + Integración | Logs y métricas en cada prueba |
| CA-05 | Integración | `tests/ingestion/contract/openapi_validation_test.go` |
