# Requirements Document

## Introduction

The HTTP ingestion service receives individual vehicle positions from mobile or external clients, validates them against an explicit contract, and publishes them asynchronously to Kafka. The service responds quickly to the client after confirming acceptance of the request for processing, without waiting for final persistence in TimescaleDB.

This requirement is fundamental to RF-04 (Ingesta asincrónica de telemetría) and RNF-03 (Desacoplamiento del procesamiento) from product.md.

## Glossary

- **Posición**: Coordenadas de ubicación (latitud, longitud) de un vehículo en un momento específico, acompañadas de metadatos identificadores.
- **Vehículo**: Entidad identificada unívocamente por un identificador del dominio (Vehicle_ID). Es responsabilidad del cliente proporcionar este identificador.
- **Cliente**: Sistema que envía solicitudes HTTP. Puede ser la aplicación móvil, un simulador de carga o un servicio externo.
- **Validación de contrato**: Verificación de que la solicitud HTTP cumple el esquema esperado (tipos, formatos, campos obligatorios, límites).
- **Publicación asincrónica**: La solicitud HTTP se acepta y se publica en Kafka sin esperar a que el worker procese el evento.
- **Confirmación de aceptación**: Respuesta HTTP que indica que la solicitud fue recibida correctamente y publicada en el tópico de Kafka, sin garantizar persistencia definitiva.
- **Position_Topic**: Canal de mensajería donde se publican los eventos de posición en Kafka.
- **Worker**: Consumidor de Kafka que procesa eventos de Position_Topic y los persiste en TimescaleDB.
- **Circuit Breaker**: Mecanismo de resiliencia que gestiona fallos en la comunicación con Kafka.
- **Reintentos**: Estrategia para reintentar la publicación en Kafka ante fallos transitorios.
- **Fallo transitorio**: Error temporal (timeout, conexión rechazada) que potencialmente puede resolverse con un reintento.
- **Fallo permanente**: Error irrecuperable (validación fallida, payload malformado en Kafka) que no debe reintentarse.
- **Alerta**: Notificación de un fallo crítico que requiere atención operativa.
- **HTTP 202 Accepted**: Código de estado HTTP que indica que la solicitud fue aceptada para procesamiento asincrónico.
- **Idempotency-Key**: Header HTTP que permite al cliente indicar que una solicitud debe ser tratada idempotentemente.
- **trace_id**: Identificador único (UUID) asignado a cada solicitud para rastreabilidad en logs.

## Requirements

### Requirement 1: Receive individual positions via HTTP POST

**User Story:** As an API client, I want to send an individual vehicle position via HTTP POST, so that the system records the vehicle's location.

#### Acceptance Criteria

1. WHEN a valid HTTP POST request is received at endpoint `/api/v1/positions`, THE Ingestion_Service SHALL accept the request and process it.
2. WHEN an HTTP POST request arrives without the `Content-Type: application/json` header, THE Ingestion_Service SHALL reject the request with status code 415.
3. WHERE the client wants to send a single position (not a batch), THE Ingestion_Service SHALL allow sending an individual position in the request body.
4. WHEN the client sends a batch of positions instead of an individual position, THE Ingestion_Service SHALL reject the request or process only the first position according to the contract.

### Requirement 2: Validate input contract explicitly

**User Story:** As an API owner, I want to validate explicitly the contract of each position request, so that only well-formed and secure data is accepted.

#### Acceptance Criteria

1. THE Ingestion_Service SHALL validate that the body contains a JSON object with the expected structure.
2. WHEN a required field is missing (vehicle_id, latitude, longitude, timestamp), THE Ingestion_Service SHALL reject the request with status code 422 and return a Problem Details indicating the missing field.
3. WHEN the `vehicle_id` field is empty or null, THE Ingestion_Service SHALL reject the request with status code 422.
4. WHEN the `latitude` field is not a number or is outside the range [-90, 90], THE Ingestion_Service SHALL reject the request with status code 422.
5. WHEN the `longitude` field is not a number or is outside the range [-180, 180], THE Ingestion_Service SHALL reject the request with status code 422.
6. WHEN the `timestamp` field is not a valid ISO 8601 string, THE Ingestion_Service SHALL reject the request with status code 422.
7. WHEN the request contains unknown additional fields, THE Ingestion_Service SHALL process it while ignoring the additional fields (be permissive on input).
8. WHEN the request exceeds the maximum allowed size (10 KB), THE Ingestion_Service SHALL reject the request with status code 413.

### Requirement 3: Publish positions to Kafka asynchronously

**User Story:** As a system owner, I want the HTTP service to publish positions to Kafka without blocking the response to the client, so that the service is fast and resilient to failures in processing.

#### Acceptance Criteria

1. WHEN a valid position request is received, THE Ingestion_Service SHALL publish the event to Position_Topic in Kafka asynchronously (without waiting for Kafka confirmation).
2. WHEN an event is published to Kafka, THE Ingestion_Service SHALL generate a unique trace_id for the event and log it.
3. THE Ingestion_Service SHALL maintain an internal queue or use Go channels to enqueue publishing events and process them in the background.
4. WHEN an event is queued waiting for publication, THE Ingestion_Service SHALL respond to the client with HTTP 202 before attempting to publish.
5. WHEN successfully published to Kafka, THE Ingestion_Service SHALL log an INFO level message indicating the trace_id and vehicle_id.
6. WHERE timing is critical, THE Ingestion_Service SHALL document the expected latency between HTTP reception and Kafka publication confirmation.

### Requirement 4: Respond quickly to client with HTTP 202 Accepted

**User Story:** As an API client, I want to receive a quick HTTP response after my request is confirmed as accepted, without waiting for remote processing to complete.

#### Acceptance Criteria

1. WHEN a valid position request is received, THE Ingestion_Service SHALL respond with HTTP 202 Accepted in less than 100 milliseconds.
2. WHEN responding with HTTP 202, THE Ingestion_Service SHALL include a JSON body with acceptance data, including:
   - `status`: string with value "accepted"
   - `trace_id`: unique identifier of the request (UUID)
   - `vehicle_id`: vehicle identifier (echo of the request)
   - `timestamp_received`: reception timestamp in ISO 8601 format
3. WHEN responding with HTTP 202, THE Ingestion_Service SHALL include the `X-Trace-ID` header with the trace_id.
4. WHEN responding with a validation failure (status 400-422), THE Ingestion_Service SHALL respond with a JSON Problem Details according to RFC 9457.
5. WHEN an unexpected server error occurs (status 500), THE Ingestion_Service SHALL return a generic Problem Details without exposing technical details.

### Requirement 5: Implement retries and Kafka failure handling

**User Story:** As a system reliability owner, I want the service to explicitly handle Kafka failures with limited retries and alerts, so that events are not lost silently.

#### Acceptance Criteria

1. WHEN Kafka is not accessible or rejects the publish, THE Ingestion_Service SHALL retry the publication up to 3 times with exponential backoff (100ms, 200ms, 400ms).
2. WHEN the final retry fails, THE Ingestion_Service SHALL:
   - Log an ERROR level message with the trace_id and failure reason.
   - Generate an alert (send to alerting system or structured log with ALERT level).
   - Discard the event from the queue to avoid blocking future requests.
3. WHEN publication is successful on any retry, THE Ingestion_Service SHALL consider the operation completed.
4. WHEN the connection with Kafka fails repeatedly, THE Ingestion_Service SHALL apply Circuit Breaker to avoid continuous attempts.
5. IF the Circuit Breaker is open and a new request is received, THEN THE Ingestion_Service SHALL respond to the client with HTTP 503 Service Unavailable indicating the dependency is not available.
6. WHEN the Circuit Breaker is open, THE Ingestion_Service SHALL attempt to recover periodically (every 30 seconds) without blocking new requests.

### Requirement 6: Distinguish between acceptance confirmation and final persistence

**User Story:** As a contract designer, I want the client to clearly understand that HTTP 202 confirms only Kafka acceptance, not persistence in TimescaleDB.

#### Acceptance Criteria

1. WHEN the client receives HTTP 202, THE response body SHALL explicitly indicate that the event was accepted for processing, not that it is definitively persisted.
2. THE Ingestion_Service SHALL document in the OpenAPI contract that HTTP 202 does not guarantee final persistence.
3. WHERE the client requires confirmation of final persistence, THE Ingestion_Service SHALL provide a separate status query endpoint (outside the scope of this requirement).
4. WHEN the event is lost after being published to Kafka (due to worker or TimescaleDB failure), THE client SHALL NOT receive automatic notification; responsibility lies with the worker and monitoring system.

### Requirement 7: Observability and logging

**User Story:** As a system operator, I want to log and monitor each request to diagnose problems and trace events.

#### Acceptance Criteria

1. FOR each request received, THE Ingestion_Service SHALL log:
   - trace_id unique (generated or extracted from `X-Trace-ID` header if provided by the client).
   - vehicle_id.
   - HTTP response code.
   - Request duration in milliseconds.
   - Operation status: accepted, rejected_validation, rejected_kafka.
2. WHEN published to Kafka, THE Ingestion_Service SHALL log the trace_id and event published (without sensitive data).
3. WHEN an error occurs, THE Ingestion_Service SHALL log:
   - trace_id.
   - Error type.
   - Error message (without exposing internal technical details, stack traces, SQL queries, or internal paths).
4. THE logs SHALL be structured in JSON format and include timestamp in ISO 8601 format.

### Requirement 8: Idempotency and deduplication on client

**User Story:** As a client developer, I want to safely handle request retries.

#### Acceptance Criteria

1. WHEN the client retries a request (e.g., timeout), THE client SHOULD include an `Idempotency-Key` header with a unique operation identifier.
2. WHEN the server receives a request with the same `Idempotency-Key`, THE Ingestion_Service SHALL respond with the same response (HTTP 202) without duplicating the publication to Kafka.
3. THE Ingestion_Service SHALL maintain a cache of Idempotency-Keys with limited duration (24 hours).
4. WHEN the cache expires, THE Ingestion_Service SHALL allow retries without deduplication (responsibility of the worker to detect duplicates by event_id).

### Requirement 9: Explicit OpenAPI contract

**User Story:** As an API consumer, I want a clear OpenAPI specification of the ingestion endpoint.

#### Acceptance Criteria

1. THE Ingestion_Service SHALL include an `openapi.yaml` file (or `openapi.json`) that documents:
   - Endpoint: POST /api/v1/positions
   - Request schema with fields and validations.
   - Response schemas for 202, 400, 413, 415, 422, 500, 503.
   - Header fields (Content-Type, X-Trace-ID, Idempotency-Key).
2. WHEN the API is published, THE OpenAPI contract SHALL be available at `/api/v1/openapi.json` or similar.
3. THE Problem Details responses SHALL follow RFC 9457 with type, title, status, detail, and instance.

## Correctness Properties

The following properties must be verified through testing:

1. **Validation Rejection**: Any request that violates validation criteria in Requirement 2 must be rejected with the appropriate status code (422 for semantic, 415 for media type, 413 for size).

2. **Async Publish Contract**: HTTP 202 is sent before Kafka publication attempts; if Kafka fails, the client never receives notification of that failure (per Requirement 5).

3. **Idempotency**: Two identical requests with the same Idempotency-Key within 24 hours must result in the same response and a single publication to Kafka.

4. **Circuit Breaker Recovery**: After Circuit Breaker opens due to repeated Kafka failures, recovery attempts must occur every 30 seconds without blocking HTTP responses.

5. **Trace Continuity**: Every log entry and error response must reference the same trace_id throughout the request lifecycle.
