# Requirements Document

## Introduction

El servicio HTTP de ingesta recibe posiciones individuales de vehículos desde clientes móviles o externos, las valida contra un contrato explícito y las publica de forma asincrónica en Kafka. El servicio responde rápidamente al cliente después de confirmar la aceptación de la solicitud para procesamiento, sin esperar a que la persistencia definitiva en TimescaleDB sea completada.

Este requisito es fundamental para RF-04 (Ingesta asincrónica de telemetría) y RNF-03 (Desacoplamiento del procesamiento) de product.md.

## Glossary

- **Posición**: Coordenadas de ubicación (latitud, longitud) de un vehículo en un momento específico, acompañadas de metadatos identificadores.
- **Vehículo**: Entidad identificada unívocamente por un identificador del dominio (vehicle_id). Es responsabilidad del cliente proporcionar este identificador.
- **Cliente**: Sistema que envía solicitudes HTTP. Puede ser la aplicación móvil, un simulador de carga o un servicio externo.
- **Validación de contrato**: Verificación de que la solicitud HTTP cumple el esquema esperado (tipos, formatos, campos obligatorios, límites).
- **Publicación asincrónica**: La solicitud HTTP se acepta y se publica en Kafka sin esperar a que el worker procese el evento.
- **Confirmación de aceptación**: Respuesta HTTP que indica que la solicitud fue recibida correctamente y publicada en el topic de Kafka, sin garantizar persistencia definitiva.
- **positions_topic**: Topic de Kafka donde se publican los eventos de posición.
- **Worker**: Consumidor de Kafka que procesa eventos de positions_topic y los persiste en TimescaleDB.
- **Circuit Breaker**: Mecanismo de resiliencia que gestiona fallos en la comunicación con Kafka.
- **Reintentos**: Estrategia para reintentar la publicación en Kafka ante fallos transitorios.
- **Fallo transitorio**: Error temporal (timeout, conexión rechazada) que potencialmente puede resolverse con un reintento.
- **Fallo permanente**: Error irrecuperable (validación fallida, payload malformado en Kafka) que no debe reintentarse.
- **Alerta**: Notificación de un fallo crítico que requiere atención operativa.
- **HTTP 202**: Código de estado HTTP que indica que la solicitud fue aceptada para procesamiento asincrónico.

## Requirements

### Requirement 1: Recibir posiciones individuales mediante HTTP POST

**Historia de usuario:** Como cliente de la API, quiero enviar una posición individual de un vehículo mediante HTTP POST, de modo que el sistema registre la ubicación del vehículo.

#### Acceptance Criteria

1. WHEN a valid HTTP POST request is received at endpoint `/api/v1/positions`, THE service SHALL accept the request and process it.
2. WHEN an HTTP POST request arrives without the `Content-Type: application/json` header, THE service SHALL reject the request with status code 415.
3. WHERE the client wants to send a single position (not a batch), THE service SHALL allow sending a single position in the request body as a unique JSON object.
4. WHEN the client sends a batch of positions (JSON array) instead of a single position, THE service SHALL reject the request with status code 422 (Unprocessable Content).
5. WHEN an event is queued waiting for publication, THE service SHALL respond to the client with HTTP 202 (Accepted) before attempting to publish to Kafka.

### Requirement 2: Validar el contrato de entrada explícitamente

**Historia de usuario:** Como responsable de la API, quiero validar explícitamente el contrato de cada solicitud de posición, de modo que solo se acepten datos bien formados y seguros.

#### Acceptance Criteria

1. THE service SHALL validate that the body contains a unique JSON object with the expected structure.
2. WHEN a required field is missing (vehicle_id, latitude, longitude, captured_at), THE service SHALL reject the request with status code 422 and return a Problem Details indicating the missing field according to RFC 9457.
3. WHEN the `vehicle_id` field is empty or null, THE service SHALL reject the request with status code 422.
4. WHEN the `latitude` field is not a number or is outside the range [-90.0, 90.0] (IEEE 754 64-bit precision), THE service SHALL reject the request with status code 422.
5. WHEN the `longitude` field is not a number or is outside the range [-180.0, 180.0] (IEEE 754 64-bit precision), THE service SHALL reject the request with status code 422.
6. WHEN the `captured_at` field is not a valid ISO 8601 string with UTC timezone (format: YYYY-MM-DDTHH:MM:SS.sssZ or YYYY-MM-DDTHH:MM:SS+00:00), THE service SHALL reject the request with status code 422.
7. WHEN the request contains unknown additional JSON fields, THE service SHALL process it while ignoring the additional fields (permissive on input).
8. WHEN the request exceeds the maximum allowed size (10 KB, measured in UTF-8 bytes of the HTTP body), THE service SHALL reject the request with status code 413 (Content Too Large).
9. WHEN the JSON body is malformed (invalid JSON syntax), THE service SHALL reject the request with status code 400 (Bad Request).

### Requirement 3: Publicar posiciones en Kafka de forma asincrónica

**Historia de usuario:** Como responsable del sistema, quiero que el servicio HTTP publique posiciones en Kafka sin bloquear la respuesta al cliente, de modo que el servicio sea rápido y resiliente ante fallos en el procesamiento.

#### Acceptance Criteria

1. WHEN a valid position request is received, THE service SHALL publish the event to positions_topic in Kafka asynchronously (without waiting for Kafka confirmation before responding to the client).
2. THE service SHALL maintain an internal queue of up to 10,000 events to queue publications and process them in the background.
3. WHEN publishing to Kafka, THE service SHALL use:
   - Partition key: `vehicle_id` (ensures order per vehicle)
   - Format: JSON with fields: event_id, vehicle_id, latitude, longitude, accuracy, captured_at, received_at, processed_at
   - Kafka confirmation: `acks=1` (ACK from at least 1 replica)
4. WHEN successfully published to Kafka, THE service SHALL log an INFO level message indicating the trace_id and vehicle_id.
5. WHERE time is critical, THE service SHALL guarantee p95 latency ≤ 500ms between HTTP reception and Kafka confirmation.

### Requirement 4: Responder rápidamente al cliente con HTTP 202 Accepted

**Historia de usuario:** Como cliente de la API, quiero recibir una respuesta HTTP rápida después de confirmar que mi solicitud fue aceptada, sin esperar a que se complete el procesamiento remoto.

#### Acceptance Criteria

1. WHEN a valid position request is received, THE service SHALL respond with HTTP 202 (Accepted) in less than 100 milliseconds (p99, measured from the first byte of the socket to the last byte of response).
2. WHEN responding with HTTP 202, THE service SHALL include a JSON body with acceptance data (and only these fields):
   - `status`: string with value "accepted"
   - `trace_id`: unique request identifier (UUID v4)
   - `vehicle_id`: vehicle identifier (echo of the request)
   - `received_at`: reception timestamp in ISO 8601 format with milliseconds
3. WHEN responding with HTTP 202, THE service SHALL include the `X-Trace-ID` header with the trace_id.
4. WHEN responding with validation error (status codes 400, 413, 415, 422), THE service SHALL respond with JSON Problem Details according to RFC 9457 in less than 50 milliseconds (p99).
5. WHEN an unexpected server error occurs (status code 500), THE service SHALL return a generic Problem Details without exposing stack traces, SQL queries, internal paths, configuration names, or technical details.

### Requirement 5: Implementar reintentos y manejo de fallos de Kafka

**Historia de usuario:** Como responsable de la confiabilidad del sistema, quiero que el servicio maneje explícitamente los fallos de Kafka con reintentos limitados y alertas, de modo que los eventos no se pierdan silenciosamente.

#### Acceptance Criteria

1. WHEN Kafka is not accessible or rejects the publish, THE service SHALL retry the publication up to 3 times with exponential backoff:
   - 1st retry: 100ms after initial failure
   - 2nd retry: 200ms after 1st failure
   - 3rd retry: 400ms after 2nd failure
2. THE service SHALL classify Kafka errors into two categories:
   - Retryable errors (transient): timeout, connection refused, broker unavailable
   - Non-retryable errors: malformed message, invalid schema, topic does not exist
3. WHEN the final retry fails (3 attempts exhausted), THE service SHALL:
   - Log an ERROR level message with trace_id, error type, and failure reason
   - Write the event to a structured dead-letter log for later recovery
   - Discard the event from the queue to avoid blocking future requests
4. WHEN the connection with Kafka fails repeatedly (5 consecutive failures in 60-second window), THE service SHALL apply the Circuit Breaker and transition to OPEN state.
5. IF the Circuit Breaker is in OPEN state and a new HTTP request is received, THEN THE service SHALL respond to the client with HTTP 503 (Service Unavailable) indicating that the dependency is unavailable.
6. WHEN the Circuit Breaker is in OPEN state, THE service SHALL:
   - Attempt recovery periodically (every 30 seconds) without blocking new requests
   - Send a synthesized probe message with 5-second timeout
   - Transition to HALF_OPEN state to allow gradual recovery
   - Transition to CLOSED state when Kafka recovery is confirmed

### Requirement 6: Distinguir entre confirmación de aceptación y persistencia definitiva

**Historia de usuario:** Como diseñador del contrato, quiero que el cliente entienda claramente que HTTP 202 confirma solo aceptación en Kafka, no persistencia en TimescaleDB.

#### Acceptance Criteria

1. WHEN the client receives HTTP 202, THE response body SHALL explicitly indicate that the event was accepted for asynchronous processing, NOT that it is definitively persisted.
2. THE service SHALL document in the OpenAPI contract that HTTP 202 guarantees ONLY that the event was published to Kafka. It does NOT guarantee:
   - Event validation
   - Processing by the worker
   - Persistence to TimescaleDB
3. WHERE the client requires confirmation of final persistence, THE service SHALL provide a separate status query endpoint (outside the scope of this requirement).
4. WHEN an event is lost after being published to Kafka (due to worker or TimescaleDB failure), THE client SHALL NOT receive automatic notification; responsibility lies with the worker and monitoring system.

### Requirement 7: Registrar cuatro timestamps de auditoría y rastreo

**Historia de usuario:** Como operador del sistema, quiero rastrear exactamente cuándo ocurre cada etapa del procesamiento de la posición, de modo que pueda diagnosticar cuellos de botella y garantizar integridad.

#### Acceptance Criteria

1. EACH event SHALL include EXACTLY FOUR UTC timestamps in ISO 8601 format with milliseconds (YYYY-MM-DDTHH:MM:SS.sssZ):

   a. **captured_at** (mandatory, client)
      - WHEN: When the location was captured on the mobile device
      - SOURCE: Sent by the client in the HTTP request
      - RESPONSIBILITY: Client guarantees capture in UTC
      - VALIDATION: Cannot be more than 1 hour in the future or more than 24 hours in the past

   b. **received_at** (mandatory, HTTP server)
      - WHEN: When the HTTP server completely received the request (last byte of body)
      - SOURCE: Automatically generated by the server when receiving the request
      - PRECISION: Exact milliseconds from the server clock
      - CALCULATION: timestamp_reception = now_server_at_receiving_last_byte

   c. **processed_at** (mandatory, HTTP server)
      - WHEN: When contract validation completed and was queued for Kafka
      - SOURCE: Automatically generated after successful validation
      - MARK: Start of asynchronous processing
      - CALCULATION: timestamp_processing = now_server_after_successful_validation

   d. **stored_at** (mandatory, worker)
      - WHEN: When persisted to TimescaleDB (INSERT completed)
      - SOURCE: Generated by worker after successful INSERT
      - CONFIRMATION: Confirms final persistence
      - CALCULATION: timestamp_storage = now_server_after_successful_INSERT

2. EACH event published to Kafka SHALL include these four timestamps in the JSON payload with the first three fields completed:
   ```json
   {
     "event_id": "uuid",
     "vehicle_id": "id",
     "latitude": 40.7128,
     "longitude": -74.0060,
     "accuracy": 5.0,
     "captured_at": "2024-10-17T14:30:45.123Z",
     "received_at": "2024-10-17T14:30:45.456Z",
     "processed_at": "2024-10-17T14:30:45.467Z",
     "stored_at": null
   }
   ```

3. WHEN the event is received via HTTP, THE service SHALL validate that `captured_at`:
   - Is a valid ISO 8601 timestamp with UTC timezone
   - Is not more than 1 hour in the future (client clock skew)
   - Is not more than 24 hours in the past (stale events)
   - If validation fails: reject with status code 422 and specific reason

4. WHEN the event arrives at the worker from Kafka, THE worker SHALL:
   - Receive the first three timestamps (captured_at, received_at, processed_at) unchanged
   - Automatically add stored_at = now_server when INSERT is successful
   - Persist all four timestamps to TimescaleDB without modification
   - Use these timestamps for audit and latency detection

5. WHEN persisting to TimescaleDB, THE table SHALL include separate columns for each timestamp:
   - `captured_at` (timestamp with timezone, NOT NULL)
   - `received_at` (timestamp with timezone, NOT NULL)
   - `processed_at` (timestamp with timezone, NOT NULL)
   - `stored_at` (timestamp with timezone, NOT NULL)
   - All with UTC timezone (AT TIME ZONE 'UTC')

6. THE timestamps SHALL enable latency calculations for diagnosis:
   - Client-to-server latency: `received_at - captured_at`
   - Validation latency: `processed_at - received_at`
   - Ingestion latency: `processed_at - captured_at`
   - Persistence latency: `stored_at - processed_at`

7. THE service SHALL reject timestamps that:
   - Are not valid ISO 8601
   - Have timezone other than UTC
   - Have precision less than milliseconds
   - Cause captured_at to be after received_at or processed_at

### Requirement 8: Registrar y monitorear cada solicitud

**Historia de usuario:** Como operador del sistema, quiero registrar y monitorear cada solicitud para diagnosticar problemas y rastrear eventos.

#### Acceptance Criteria

1. FOR each request received, THE service SHALL log (in JSON):
   - trace_id: unique (generated or extracted from `X-Trace-ID` header if provided by the client)
   - vehicle_id: vehicle identifier
   - http_status: HTTP response status code
   - duration_ms: request duration in milliseconds (from first byte received to last byte sent)
   - operation_status: "accepted", "rejected_validation", or "rejected_kafka"
   - log_level: according to severity

2. WHEN published to Kafka, THE service SHALL log (INFO level):
   - trace_id, vehicle_id, timestamps (captured_at, received_at, processed_at)
   - Without sensitive data (coordinates NOT logged completely; only accuracy)

3. WHEN an error occurs, THE service SHALL log (ERROR level):
   - trace_id, error_type, error_message
   - Without exposing: stack traces, SQL queries, internal paths, configuration names, tokens

4. THE logs SHALL be structured in JSON format and include timestamp in ISO 8601:
   ```json
   {
     "timestamp": "2024-10-17T14:30:45.123Z",
     "level": "INFO",
     "trace_id": "550e8400-e29b-41d4-a716-446655440000",
     "vehicle_id": "VH-001",
     "event": "request_received",
     "http_status": 202,
     "duration_ms": 45,
     "operation_status": "accepted"
   }
   ```

5. THE service SHALL apply cardinality limit: maximum 1000 logs per second per level, with adaptive sampling to prevent log explosion.

### Requirement 9: Idempotencia mediante X-Idempotency-Key obligatorio

**Historia de usuario:** Como desarrollador del cliente, quiero que el cliente maneje reintentos de solicitud de forma segura usando un identificador idempotente.

#### Acceptance Criteria

1. THE `X-Idempotency-Key` header is MANDATORY in all HTTP POST requests to `/api/v1/positions`.
2. WHEN the `X-Idempotency-Key` header is absent, THE service SHALL reject the request with status code 400 (Bad Request).
3. WHEN the `X-Idempotency-Key` header is not a valid UUID v4, THE service SHALL reject the request with status code 400 (Bad Request).
4. THE service SHALL store in database each `X-Idempotency-Key` processed for 24 hours for durability.
5. WHEN a request is received with the same `X-Idempotency-Key` within the 24-hour window, THE service SHALL:
   - If the payload is identical: respond with the same HTTP 202 response (without duplicating the publication to Kafka) and use the same trace_id
   - If the payload is different: reject with status code 409 (Conflict)
6. WHEN the client reuses an `X-Idempotency-Key` after 24 hours (TTL expired) or generates a new one, THE service SHALL treat it as a new request.
7. IT IS the client's responsibility to generate a unique UUID v4 for each new request; do not reuse keys for different requests.

### Requirement 10: Contrato OpenAPI explícito

**Historia de usuario:** Como consumidor de la API, quiero disponer de una especificación OpenAPI clara del endpoint de ingesta.

#### Acceptance Criteria

1. THE service SHALL include an `openapi.yaml` file (OpenAPI 3.1.0) that documents:
   - Endpoint: POST /api/v1/positions
   - Request schema with fields, validations, and constraints
   - Response schemas for status codes 202, 400, 409, 413, 415, 422, 500, 503
   - Request headers (Content-Type, X-Trace-ID, X-Idempotency-Key)
2. WHEN the API is published, THE OpenAPI contract SHALL be available at `/api/v1/openapi.yaml`.
3. THE Problem Details responses SHALL follow RFC 9457 with mandatory fields: type, title, status, detail, instance.

## Correctness Properties

Las siguientes propiedades DEBEN verificarse mediante testing:

1. **Rechazo de Validación**: Cualquier solicitud que viole criterios en Requisito 2 DEBE ser rechazada con el código de estado correcto (422 para semántica, 415 para media type, 413 para tamaño).

2. **Contrato de Publicación Asincrónica**: HTTP 202 se envía antes de intentar publicar en Kafka; si Kafka falla, el cliente NUNCA recibe notificación de ese fallo (por Requisito 5).

3. **Idempotencia**: Dos solicitudes idénticas con la misma X-Idempotency-Key dentro de 24 horas DEBEN resultar en la misma respuesta y una sola publicación en Kafka.

4. **Recuperación del Circuit Breaker**: Después de que Circuit Breaker se abre por fallos repetidos de Kafka, intentos de recuperación DEBEN ocurrir cada 30 segundos sin bloquear respuestas HTTP.

5. **Continuidad de Rastreo**: Cada entrada de log y respuesta de error DEBE referenciar el mismo trace_id durante todo el ciclo de vida de la solicitud.

6. **Integridad de Timestamps**: Los cuatro timestamps DEBEN estar presentes en cada evento, en orden cronológico estricto (captured_at ≤ received_at ≤ processed_at ≤ stored_at), y permanecer inalterados desde cliente hasta persistencia final.
