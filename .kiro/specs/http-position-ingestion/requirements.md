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
- **Ingesta de datos**: Recepción y aceptación de datos sin validación de negocio; solo validación de contrato y formato. Las decisiones de negocio (filtrado, deduplicación, valoración) ocurren downstream en el worker.
- **positions_topic**: Topic de Kafka donde se publican los eventos de posición.
- **Worker**: Consumidor de Kafka que procesa eventos de positions_topic y los persiste en TimescaleDB.
- **Circuit Breaker**: Mecanismo de resiliencia que gestiona fallos en la comunicación con Kafka.
- **Reintentos**: Estrategia para reintentar la publicación en Kafka ante fallos transitorios.
- **Fallo transitorio**: Error temporal (timeout, conexión rechazada) que potencialmente puede resolverse con un reintento.
- **Fallo permanente**: Error irrecuperable (validación fallida, payload malformado en Kafka) que no debe reintentarse.
- **Alerta**: Notificación de un fallo crítico que requiere atención operativa.
- **HTTP 202**: Código de estado HTTP que indica que la solicitud fue aceptada para procesamiento asincrónico.
- **Payload**: Contenido de la solicitud HTTP (vehicle_id, latitude, longitude, accuracy, captured_at).
- **X-Trace-ID**: Encabezado HTTP opcional (UUID v4) para rastreo de solicitudes. Si se envía, debe ser validado como UUID v4 válido.
- **X-Idempotency-Key**: Encabezado HTTP opcional (UUID v4) enviado por el cliente. Si se envía, debe ser validado como UUID v4 válido. Nota: deduplicación NO se implementa en esta spec (trabajo futuro).

## Requirements

### Requirement 1: Recibir posiciones individuales mediante HTTP POST

**User Story:** Como cliente de la API, quiero enviar una posición individual de un vehículo mediante HTTP POST, de modo que el sistema registre la ubicación del vehículo.

#### Acceptance Criteria

1. WHEN se recibe una solicitud HTTP POST válida en el endpoint `/api/v1/positions`, THE servicio DEBE aceptar la solicitud y procesarla.
2. WHEN una solicitud HTTP POST llega sin el encabezado `Content-Type: application/json`, THE servicio DEBE rechazar la solicitud con código de estado 415.
3. WHERE el cliente desea enviar una sola posición (no un lote), THE servicio DEBE permitir enviar una posición individual en el body de la solicitud como un objeto JSON único.
4. WHEN el cliente envía un lote de posiciones (array JSON) en lugar de una posición individual, THE servicio DEBE rechazar la solicitud con código de estado 422 (Contenido No Procesable).
5. WHEN un evento está encolado esperando publicación, THE servicio DEBE responder al cliente con HTTP 202 (Aceptado) antes de intentar publicar en Kafka.

### Requirement 2: Validar el contrato de entrada explícitamente

**User Story:** Como responsable de la API, quiero validar explícitamente el contrato de cada solicitud de posición, de modo que solo se acepten datos bien formados y seguros.

#### Acceptance Criteria

1. THE servicio DEBE validar que el body contiene un objeto JSON único con la estructura esperada.
2. WHEN falta un campo obligatorio (vehicle_id, latitude, longitude, captured_at), THE servicio DEBE rechazar la solicitud con código de estado 422 y devolver un Problem Details indicando el campo faltante según RFC 9457.
3. WHEN el campo `vehicle_id` está vacío o es nulo, THE servicio DEBE rechazar la solicitud con código de estado 422.
4. WHEN el campo `latitude` no es un número o está fuera del rango [-90.0, 90.0] (precisión IEEE 754 de 64 bits), THE servicio DEBE rechazar la solicitud con código de estado 422.
5. WHEN el campo `longitude` no es un número o está fuera del rango [-180.0, 180.0] (precisión IEEE 754 de 64 bits), THE servicio DEBE rechazar la solicitud con código de estado 422.
6. WHEN el campo `captured_at` no es una cadena ISO 8601 válida con zona horaria UTC (formato: YYYY-MM-DDTHH:MM:SS.sssZ o YYYY-MM-DDTHH:MM:SS+00:00), THE servicio DEBE rechazar la solicitud con código de estado 422. El servicio DEBE rechazar si captured_at es posterior a la hora actual del servidor (futuro no válido).
7. WHEN la solicitud contiene campos JSON adicionales desconocidos, THE servicio DEBE procesarla ignorando los campos adicionales (permisivo en la entrada).
8. WHEN la solicitud supera el tamaño máximo permitido (10 KB, medido en bytes UTF-8 del body HTTP), THE servicio DEBE rechazar la solicitud con código de estado 413 (Contenido Demasiado Grande).
9. WHEN el body JSON es malformado (sintaxis JSON inválida), THE servicio DEBE rechazar la solicitud con código de estado 400 (Solicitud Malformada).

### Requirement 3: Publicar posiciones en Kafka de forma asincrónica

**User Story:** Como responsable del sistema, quiero que el servicio HTTP publique posiciones en Kafka sin bloquear la respuesta al cliente, de modo que el servicio sea rápido y resiliente ante fallos en el procesamiento.

#### Acceptance Criteria

1. WHEN se recibe una solicitud de posición válida, THE servicio DEBE publicar el evento en positions_topic de Kafka de forma asincrónica (sin esperar la confirmación de Kafka antes de responder al cliente).
2. THE servicio DEBE mantener una cola interna de hasta 10,000 eventos para encolar publicaciones y procesarlas en background.
3. WHEN se publica en Kafka, THE servicio DEBE usar:
   - Clave de partición: `vehicle_id` (garantiza orden por vehículo)
   - Formato: JSON con campos: event_id, vehicle_id, latitude, longitude, accuracy, captured_at, received_at, processed_at
   - Confirmación Kafka: `acks=1` (ACK de al menos 1 replica)
4. WHEN se publica con éxito en Kafka, THE servicio DEBE registrar un mensaje de nivel INFO indicando el trace_id y el vehicle_id.
5. WHERE el tiempo es crítico, THE servicio DEBE garantizar latencia p95 ≤ 500ms entre recepción HTTP y confirmación de Kafka.

### Requirement 4: Responder rápidamente al cliente con HTTP 202 Accepted

**User Story:** Como cliente de la API, quiero recibir una respuesta HTTP rápida después de confirmar que mi solicitud fue aceptada, sin esperar a que se complete el procesamiento remoto.

#### Acceptance Criteria

1. WHEN se recibe una solicitud de posición válida, THE servicio DEBE responder con HTTP 202 (Aceptado) en menos de 100 milisegundos (p99, medido desde el primer byte del socket hasta el último byte de respuesta).
2. WHEN responde con HTTP 202, THE servicio DEBE incluir un body JSON con los datos de aceptación (y solo estos campos):
   - `status`: cadena con valor "accepted"
   - `trace_id`: identificador único de la solicitud (UUID v4)
   - `vehicle_id`: identificador del vehículo (eco de la solicitud)
   - `received_at`: marca de tiempo de recepción en formato ISO 8601 con milisegundos
3. WHEN responde con HTTP 202, THE servicio DEBE incluir el encabezado `X-Trace-ID` con el trace_id.
4. WHEN responde con error de validación (códigos 400, 413, 415, 422), THE servicio DEBE responder con JSON Problem Details según RFC 9457 en menos de 50 milisegundos (p99).
5. WHEN ocurre un error inesperado del servidor (código 500), THE servicio DEBE devolver un Problem Details genérico sin exponer stack traces, consultas SQL, rutas internas, nombres de configuración ni detalles técnicos.

### Requirement 5: Implementar reintentos y manejo de fallos de Kafka

**User Story:** Como responsable de la confiabilidad del sistema, quiero que el servicio maneje explícitamente los fallos de Kafka con reintentos limitados y alertas, de modo que los eventos no se pierdan silenciosamente.

#### Acceptance Criteria

1. WHEN Kafka no es accesible o rechaza la publicación, THE servicio DEBE reintentar la publicación hasta 3 veces con backoff exponencial:
   - 1er reintento: 100ms después del fallo inicial
   - 2do reintento: 200ms después del 1er fallo
   - 3er reintento: 400ms después del 2do fallo
2. THE servicio DEBE clasificar errores de Kafka en dos categorías:
   - Errores retryables (transitorios): timeout, conexión rechazada, broker no disponible
   - Errores no retryables: mensaje malformado, esquema inválido, topic no existe
3. WHEN falla el reintento final (3 intentos agotados), THE servicio DEBE:
   - Registrar un mensaje de nivel ERROR con trace_id, tipo de error y razón del fallo
   - Escribir el evento en un dead-letter log estructurado para recuperación posterior
   - Descartar el evento de la cola para evitar bloquear solicitudes futuras
4. WHEN la conexión con Kafka falla repetidamente (5 fallos consecutivos en ventana de 60 segundos), THE servicio DEBE aplicar el Circuit Breaker y transicionar a estado OPEN.
5. IF el Circuit Breaker está en estado OPEN y se recibe una nueva solicitud HTTP, THEN THE servicio DEBE responder al cliente con HTTP 503 (Servicio No Disponible) indicando que la dependencia no está disponible.
6. WHEN el Circuit Breaker está en estado OPEN, THE servicio DEBE:
   - Intentar recuperación periódicamente (cada 30 segundos) sin bloquear solicitudes nuevas
   - Enviar un mensaje de prueba sintetizado con timeout de 5 segundos
   - Transicionar a estado HALF_OPEN para permitir recuperación gradual
   - Transicionar a estado CLOSED cuando se confirme la recuperación de Kafka

### Requirement 6: Distinguir entre confirmación de aceptación y persistencia definitiva

**User Story:** Como diseñador del contrato, quiero que el cliente entienda claramente que HTTP 202 confirma solo aceptación en Kafka, no persistencia en TimescaleDB.

#### Acceptance Criteria

1. WHEN el cliente recibe HTTP 202, THE body de respuesta DEBE indicar explícitamente que el evento fue aceptado para procesamiento asincrónico, NO que está definitivamente persistido.
2. THE servicio DEBE documentar en el contrato OpenAPI que HTTP 202 garantiza SOLO que el evento fue publicado en Kafka. NO garantiza:
   - Validación del evento
   - Procesamiento por el worker
   - Persistencia en TimescaleDB
3. WHERE el cliente requiera confirmar persistencia definitiva, THE servicio DEBE proporcionar un endpoint separado de consulta de estado (fuera del alcance de este requisito).
4. WHEN un evento se pierde después de publicarse en Kafka (por fallo del worker o TimescaleDB), THE cliente NO recibirá notificación automática; la responsabilidad es del worker y del sistema de monitoreo.

### Requirement 7: Registrar cuatro timestamps de auditoría y rastreo

**User Story:** Como operador del sistema, quiero rastrear exactamente cuándo ocurre cada etapa del procesamiento de la posición, de modo que pueda diagnosticar cuellos de botella y garantizar integridad.

#### Acceptance Criteria

1. EACH evento DEBE incluir EXACTAMENTE CUATRO timestamps UTC en formato ISO 8601 con milisegundos (YYYY-MM-DDTHH:MM:SS.sssZ):

   a. **captured_at** (obligatorio, cliente)
      - Cuándo: Cuándo fue capturada la ubicación en el dispositivo móvil
      - Origen: Enviado por el cliente en la solicitud HTTP
      - Responsabilidad: Cliente garantiza captura en UTC
      - Validación: No puede ser posterior a la hora actual del servidor (no puede ser futuro)

   b. **received_at** (obligatorio, servidor HTTP)
      - Cuándo: Cuándo el servidor HTTP recibió completamente la solicitud (último byte del body)
      - Origen: Generado automáticamente por el servidor al recibir la solicitud
      - Precisión: Milisegundos exactos desde el reloj del servidor
      - Cálculo: timestamp_recepción = ahora_servidor_al_recibir_último_byte

   c. **processed_at** (obligatorio, servidor HTTP)
      - Cuándo: Cuándo completó validación de contrato y fue encolado para Kafka
      - Origen: Generado automáticamente después de validación exitosa
      - Marca: Inicio del procesamiento asincrónico
      - Cálculo: timestamp_procesamiento = ahora_servidor_después_validación_exitosa

   d. **stored_at** (obligatorio, worker)
      - Cuándo: Cuándo fue persistido en TimescaleDB (INSERT completado)
      - Origen: Generado por worker después de INSERT exitoso
      - Confirmación: Confirma persistencia definitiva
      - Cálculo: timestamp_almacenamiento = ahora_servidor_después_INSERT_exitoso

2. EACH evento publicado en Kafka DEBE incluir estos cuatro timestamps en el payload JSON con los tres primeros campos completados:
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

3. WHEN el evento es recibido por HTTP, THE servicio DEBE validar que `captured_at`:
   - Es un timestamp ISO 8601 válido con zona horaria UTC
   - Tiene precisión de milisegundos (YYYY-MM-DDTHH:MM:SS.sssZ o YYYY-MM-DDTHH:MM:SS+00:00)
   - No es posterior a la hora actual del servidor (no puede ser futuro)
   - Si falla validación: rechazar con código 422 y motivo específico
   - El servicio ACEPTA cualquier timestamp en el pasado (sin límite de antigüedad) para soportar clientes offline

4. WHEN el evento llega al worker desde Kafka, THE worker DEBE:
   - Recibir los tres primeros timestamps (captured_at, received_at, processed_at) inalterados
   - Agregar automáticamente stored_at = ahora_servidor cuando el INSERT sea exitoso
   - Persistir todos cuatro timestamps en TimescaleDB sin modificación
   - Usar estos timestamps para auditoría y detección de retrasos

5. WHEN se persiste en TimescaleDB, THE tabla DEBE incluir columnas separadas para cada timestamp:
   - `captured_at` (timestamp with timezone, NOT NULL)
   - `received_at` (timestamp with timezone, NOT NULL)
   - `processed_at` (timestamp with timezone, NOT NULL)
   - `stored_at` (timestamp with timezone, NOT NULL)
   - Todas con zona horaria UTC (AT TIME ZONE 'UTC')

6. THE timestamps DEBEN permitir cálculos de latencia para diagnóstico:
   - Latencia cliente-servidor: `received_at - captured_at`
   - Latencia de validación: `processed_at - received_at`
   - Latencia de ingesta: `processed_at - captured_at`
   - Latencia de persistencia: `stored_at - processed_at`

7. THE servicio DEBE rechazar timestamps que:
   - No sean ISO 8601 válido
   - Tengan zona horaria distinta de UTC
   - Tengan precisión menor a milisegundos
   - Sean posteriores a la hora actual del servidor (futuro no válido)
   - NO se rechaza por antigüedad; el servicio acepta cualquier timestamp pasado para soportar sincronización offline

### Requirement 8: Registrar y monitorear cada solicitud

**User Story:** Como operador del sistema, quiero registrar y monitorear cada solicitud para diagnosticar problemas y rastrear eventos.

#### Acceptance Criteria

1. FOR cada solicitud recibida, THE servicio DEBE registrar (en logs JSON):
   - trace_id: único (generado o extraído del encabezado `X-Trace-ID` si lo proporciona el cliente)
   - vehicle_id: identificador del vehículo
   - http_status: código de estado HTTP de respuesta
   - duration_ms: duración de la solicitud en milisegundos (desde primer byte recibido hasta último byte enviado)
   - operation_status: "accepted", "rejected_validation" o "rejected_kafka"
   - log_level: según la gravedad

2. WHEN se publica en Kafka, THE servicio DEBE registrar (nivel INFO):
   - trace_id, vehicle_id, timestamps (captured_at, received_at, processed_at)
   - Sin incluir datos sensibles (coordenadas NO se registran completas; solo accuracy)

3. WHEN ocurre un error, THE servicio DEBE registrar (nivel ERROR):
   - trace_id, error_type, error_message
   - Sin exponer: stack traces, consultas SQL, rutas internas, nombres de configuración, tokens

4. THE logs DEBEN ser estructurados en formato JSON e incluir timestamp en ISO 8601:
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

5. THE servicio DEBE aplicar límite de cardinality: máximo 1000 logs por segundo por nivel, con sampling adaptativo para prevenir explosión de logs.

### Requirement 9: Contrato OpenAPI explícito

**User Story:** Como consumidor de la API, quiero disponer de una especificación OpenAPI clara del endpoint de ingesta.

#### Acceptance Criteria

1. THE servicio DEBE incluir un archivo `openapi.yaml` (OpenAPI 3.1.0) que documente:
   - Endpoint: POST /api/v1/positions
   - Esquema de solicitud con campos, validaciones y restricciones
   - Esquemas de respuesta para códigos 202, 400, 413, 415, 422, 500, 503
   - Encabezados de solicitud (Content-Type, X-Trace-ID, X-Idempotency-Key)
2. WHEN se publica la API, THE contrato OpenAPI DEBE estar disponible en `/api/v1/openapi.yaml`.
3. THE respuestas de Problem Details DEBEN seguir RFC 9457 con campos obligatorios: type, title, status, detail, instance.

## Correctness Properties

Las siguientes propiedades DEBEN verificarse mediante testing:

1. **Rechazo de Validación**: Cualquier solicitud que viole criterios en Requisito 2 DEBE ser rechazada con el código de estado correcto (422 para semántica, 415 para media type, 413 para tamaño).

2. **Contrato de Publicación Asincrónica**: HTTP 202 se envía antes de intentar publicar en Kafka; si Kafka falla, el cliente NUNCA recibe notificación de ese fallo (por Requisito 5).

3. **Recuperación del Circuit Breaker**: Después de que Circuit Breaker se abre por fallos repetidos de Kafka, intentos de recuperación DEBEN ocurrir cada 30 segundos sin bloquear respuestas HTTP.

4. **Continuidad de Rastreo**: Cada entrada de log y respuesta de error DEBE referenciar el mismo trace_id durante todo el ciclo de vida de la solicitud.

5. **Integridad de Timestamps**: Los cuatro timestamps DEBEN estar presentes en cada evento, en orden cronológico (captured_at ≤ received_at ≤ processed_at ≤ stored_at), y permanecer inalterados desde cliente hasta persistencia final. captured_at NO puede ser futuro (rechazado con 422); puede ser cualquier timestamp pasado.

### Requirement 10: Arquitectura COMPLETO con Separación Clara de Capas

**User Story:** Como arquitecto del sistema, quiero que el servicio implemente una arquitectura de **Clean Architecture COMPLETO** con separación explícita de capas (Domain, Application, HTTP Adapters, Outbound Adapters, Ports), DTOs centralizados y Mappers, de modo que el sistema sea mantenible, testeable y escalable.

#### Acceptance Criteria

1. THE servicio DEBE organizar el código en carpetas que representen explícitamente cada capa arquitectónica:
   - `domain/`: Value Objects, Entity, Errors (lógica pura de negocio, sin HTTP, sin persistencia)
   - `application/`: Commands, UseCases (orquestación, sin HTTP, sin detalles de infraestructura)
   - `adapters/inbound/http/`: Handler HTTP, Middleware (traducción HTTP → Application)
   - `adapters/outbound/`: Implementaciones de puertos (Application → Kafka, Circuit Breaker, RateLimiter)
   - `ports/`: 6 interfaces públicas (contratos con la aplicación)
   - `http/`: DTOs y Mappers centralizados (protegen el dominio, traducen entre contextos)
   - `config/`: Composición de dependencias (conecta puertos con implementaciones)

2. THE carpeta `http/` DEBE contener:
   - `dto/request.go`: PositionRequest (entrada HTTP cruda, solo tipos simples)
   - `dto/response.go`: PositionResponse (salida HTTP 202), ProblemDetails (RFC 9457)
   - `mapper/position_mapper.go`: ToCommand() (DTO → Command) y FromEntity() (Entity → DTO)
   - `mapper/error_mapper.go`: ToProblemDetails() (Domain Error → RFC 9457 ProblemDetails)

3. THE carpeta `domain/` DEBE contener EXACTAMENTE 9 archivos:
   - `vehicle_id.go`: VehicleID (Value Object)
   - `geo_position.go`: GeoPosition (Value Object)
   - `captured_timestamp.go`: CapturedTimestamp (Value Object)
   - `received_timestamp.go`: ReceivedTimestamp (Value Object)
   - `processed_timestamp.go`: ProcessedTimestamp (Value Object)
   - `event_id.go`: EventID (Value Object)
   - `trace_id.go`: TraceID (Value Object)
   - `idempotency_key.go`: IdempotencyKey (Value Object)
   - `rate_limit_config.go`: RateLimitConfig (Value Object)
   - `vehicle_position.go`: VehiclePosition (Entity: orquesta Value Objects)
   - `errors.go`: Errores del dominio (struct + constantes)

4. THE carpeta `application/` DEBE contener:
   - `commands/receive_position.go`: ReceivePositionCommand (DTO interno de applicación)
   - `usecases/receive_position.go`: ReceivePositionUseCase (orquesta puertos, crea Entity, retorna Entity)

5. THE carpeta `adapters/inbound/http/middleware/` DEBE contener 5 middlewares:
   - `trace_id.go`: TraceIDMiddleware (valida/genera X-Trace-ID)
   - `idempotency.go`: IdempotencyKeyMiddleware (valida X-Idempotency-Key)
   - `content_type.go`: ContentTypeMiddleware (valida Content-Type: application/json)
   - `rate_limit.go`: RateLimitMiddleware (verifica límite por vehicle_id)
   - `error_handler.go`: ErrorHandlerMiddleware (mapea errores a Problem Details)

6. THE carpeta `ports/` DEBE contener exactamente 6 interfaces:
   - `kafka_publisher.go`: KafkaPublisher (puerto de salida)
   - `circuit_breaker_manager.go`: CircuitBreakerManager (puerto de salida)
   - `rate_limiter.go`: RateLimiter (puerto de salida)
   - `logger.go`: Logger (puerto de salida)
   - `clock.go`: Clock (puerto de salida, inyectable para testing)
   - `id_generator.go`: IDGenerator (puerto de salida)

7. THE carpeta `adapters/outbound/` DEBE contener:
   - `messaging/kafka_publisher.go`: Implementa KafkaPublisher (Kafka, colas, reintentos)
   - `resilience/circuit_breaker.go`: Implementa CircuitBreakerManager (estados, transiciones)
   - `resilience/rate_limiter.go`: Implementa RateLimiter (rolling window en memoria)
   - `time/clock.go`: Implementa Clock (time.Now() envuelto)

8. EACH archivo del dominio DEBE contener:
   - Constructor `New{Type}()` con validaciones
   - Métodos públicos expresivos del dominio (sin "Get" ni "Set")
   - Métodos privados para cálculos internos
   - Sin referencias a HTTP, Kafka, DTOs, ni otras capas externas

9. THE UseCase DEBE implementar el flujo completo sin bloquear:
   - Entrada: ReceivePositionCommand (DTO de aplicación)
   - Crear Value Objects a partir del command (validación dominio)
   - Construir VehiclePosition Entity
   - Verificar Circuit Breaker
   - Publicar asincrónicamente a Kafka
   - Retornar Entity (sin esperar Kafka)

10. THE Mappers DEBE ser bidireccionales y reutilizables:
    - `PositionMapper.ToCommand(request DTO) → Command`: convierte DTO HTTP en Command interno
    - `PositionMapper.FromEntity(entity) → DTO`: convierte Entity de dominio en DTO HTTP
    - `ErrorMapper.ToProblemDetails(error) → DTO`: convierte error de dominio en RFC 9457

11. THE data flow DEBE ser claramente lineal:
    ```
    Cliente HTTP
       ↓
    PositionRequest DTO (deserializado)
       ↓
    PositionMapper.ToCommand() → ReceivePositionCommand
       ↓
    ReceivePositionUseCase.Execute(command)
       ↓
    Crea: 8 Value Objects + 1 Entity (VehiclePosition)
       ↓
    Valida, verifica Circuit Breaker, publica a Kafka (async)
       ↓
    Retorna Entity (del dominio)
       ↓
    PositionMapper.FromEntity(entity) → PositionResponse DTO
       ↓
    Cliente recibe JSON (sin exponer dominio)
    ```

12. THE carpeta `config/` DEBE contener `composer.go` que:
    - Instancia todas las dependencias (Clock, IDGenerator, Logger, etc.)
    - Conecta puertos (interfaces) con implementaciones (adapters)
    - Construye UseCase con dependencias inyectadas
    - Construye Middleware pipeline
    - Retorna HTTP handler completo listo para usar

### Requirement 11: Generación de Domain Events

**User Story:** Como diseñador del sistema, quiero que la Entity VehiclePosition emita Domain Events, de modo que la Application pueda consumirlos y decidir acciones (como publicar a Kafka) sin que el Domain conozca detalles de infraestructura.

#### Acceptance Criteria

1. WHEN se crea una VehiclePosition Entity exitosamente, THE Entity DEBE emitir un PositionReceivedEvent automáticamente.

2. EACH PositionReceivedEvent DEBE contener exactamente estos campos:
   - vehicleID: VehicleID del vehículo
   - eventID: EventID único (UUID v4)
   - capturedAt: CapturedTimestamp (del cliente)
   - receivedAt: ReceivedTimestamp (del servidor HTTP)
   - processedAt: ProcessedTimestamp (post-validación)
   - traceID: TraceID (para auditoría)
   - occurredAt: time.Time (cuándo se emitió el event)

3. EACH PositionReceivedEvent DEBE implementar métodos expresivos del dominio:
   - AggregateID() → string (retorna vehicleID como identificador del agregado)
   - EventType() → string (retorna "PositionReceived" constante)
   - OccurredAt() → time.Time (retorna cuándo ocurrió)

4. THE PositionReceivedEvent DEBE estar ubicado en `domain/events.go` y SER un Value Object (inmutable, sin dependencias externas).

5. THE Domain DEBE exponer el event a través de un método del Entity (ej: Entity.Events()) para que Application lo consume. THE event NO es publicado automáticamente por el Domain.

6. THE PositionReceivedEvent DEBE ser completamente desacoplado de Kafka, HTTP, base de datos. El event NO referencia ningún detalle de infraestructura.

7. WHERE la Application recibe el PositionReceivedEvent, THE Application DEBE decidir qué hacer (ej: publicar a Kafka). El Domain no impone esa decisión.

### Requirement 12: Manejo de Errores Domain → HTTP

**User Story:** Como consumidor de la API, quiero que los errores del Domain se traduzcan a respuestas HTTP claras y seguras según RFC 9457.

#### Acceptance Criteria

1. THE Domain DEBE emitir errores específicos (no genéricos). Cada error corresponde a una violación de invariante o regla de negocio:
   - FutureTimestampError: captured_at es posterior a ahora
   - InvalidVehicleIDError: vehicle_id está vacío o excede 255 caracteres
   - InvalidGeoPositionError: latitud/longitud fuera de rango válido
   - InvalidEventIDError: event_id no es UUID v4 válido
   - InvalidTraceIDError: X-Trace-ID no es UUID v4 válido (si se proporciona)

2. THE Application DEBE capturar cualquier error del Domain SIN transformarlo. La Application registra el error en logs (con trace_id) y lo propaga hacia HTTP Layer.

3. THE HTTP Layer (ErrorMapper.ToProblemDetails()) DEBE traducir cada Domain Error a RFC 9457 ProblemDetails con:
   - type: URI identifying problem type (ej: "urn:fleet:error:future-timestamp")
   - title: Brief, stable summary (ej: "Future Timestamp")
   - status: HTTP status code (422 para errores de validación, 429 para rate limit, 503 para CB)
   - detail: Human-readable explanation (ej: "Captured timestamp cannot be in the future")
   - instance: URI of this occurrence (ej: "/problems/validation/{trace_id}")

4. EACH Domain Error DEBE mapear a exactamente UNO de estos HTTP status codes:
   - FutureTimestampError → 422 Unprocessable Content
   - InvalidVehicleIDError → 422 Unprocessable Content
   - InvalidGeoPositionError → 422 Unprocessable Content
   - InvalidEventIDError → 422 Unprocessable Content
   - InvalidTraceIDError → 400 Bad Request
   - RateLimitExceededError (Application) → 429 Too Many Requests
   - CircuitBreakerOpenError (Application) → 503 Service Unavailable
   - Unexpected/Unknown errors → 500 Internal Server Error

5. THE ProblemDetails response DEBE NUNCA exponer:
   - Stack traces
   - SQL queries
   - Internal paths or filenames
   - Secret keys or tokens
   - Implementation details (ej: "sarama producer error")

6. THE ErrorMapper DEBE ser testeable aisladamente. Cada Domain Error tiene un test que verifica su mapeo correcto a RFC 9457 ProblemDetails.

7. WHEN un error inesperado ocurre (no mapeado), THE ErrorMapper DEBE retornar:
   ```json
   {
     "type": "urn:fleet:error:internal-error",
     "title": "Internal Server Error",
     "status": 500,
     "detail": "An unexpected error occurred",
     "instance": "/problems/error/{trace_id}"
   }
   ```

### Requirement 13: Rate Limiting por vehículo (in-memory, sin Redis)

**User Story:** Como responsable de la disponibilidad del sistema, quiero proteger el servicio de abuso limitando la tasa de solicitudes por vehículo, de modo que clientes honestos no queden afectados por clientes que envíen solicitudes excesivas.

#### Acceptance Criteria

1. THE servicio DEBE implementar rate limiting **in-memory** (sin Redis) con rolling window de 60 segundos por vehicle_id.

2. THE límite por defecto DEBE ser 12 solicitudes por minuto por vehículo, calculado como: `(60 segundos / frequency_seconds) * 1.2`, donde frequency_seconds es el intervalo esperado entre solicitudes.

3. WHEN un cliente excede el límite en la ventana de 60 segundos, THE servicio DEBE rechazar la solicitud con HTTP 429 (Too Many Requests) y devolver RFC 9457 ProblemDetails.

4. WHEN la solicitud es rechazada por rate limit, THE servicio DEBE incluir en la respuesta HTTP estos encabezados:
   - `Retry-After`: segundos a esperar antes de reintentar
   - `X-RateLimit-Limit`: límite máximo de solicitudes en la ventana
   - `X-RateLimit-Remaining`: solicitudes restantes en la ventana actual
   - `X-RateLimit-Reset`: timestamp (Unix) de reinicio de la ventana

5. THE rate limiting se evalúa PER VEHICLE_ID (no global), permitiendo que múltiples vehículos envíen solicitudes simultáneamente sin interferencia mutua.

6. THE límite es por VEHÍCULO IDENTIFICADO EN request (vehicle_id), no por dirección IP del cliente.

7. THE implementación DEBE ser memory-safe (no race conditions) incluso con alta concurrencia (miles de vehículos en paralelo).

8. THE rolling window DEBE ser eficiente: evita scans o iteraciones completas de datos históricos. Se recomienda usar una estructura que soporte O(1) agregar entrada y O(1) contar en ventana.

9. THE valor de rate limit DEBE ser configurable (variable de entorno o archivo de configuración) pero el valor por defecto es 12/min.

10. NOTE: Rate limiting se implementa SIN Redis. Esto implica que:
    - Válido solo en UNA instancia del servicio (si hay múltiples réplicas, cada una tendrá su propio contador)
    - Contadores se pierden si la instancia se reinicia
    - Suficiente para MVP de una sola instancia; escalabilidad multi-instancia será trabajo futuro con Redis

---

## Scope NO cubierto

Las siguientes funcionalidades NO son parte de esta spec y serán responsabilidad de especificaciones futuras:

1. **Deduplicación de eventos**: El encabezado X-Idempotency-Key es aceptado y validado como UUID v4, pero NO se implementa mecanismo de deduplicación. Esto incluye almacenamiento en Redis, PostgreSQL o cualquier otro mecanismo de caché.

2. **Rate limiting multi-instancia (Redis)**: El rate limiting de esta spec es in-memory y válido solo en una sola instancia. Para escalabilidad a múltiples réplicas con límites compartidos y no perder contadores ante reinicio, se usará Redis en especificación futura.

3. **Autenticación y autorización**: El servicio NO valida permisos ni autenticidad del cliente. Estos controles se implementarán en un componente middleware separado.

4. **Versionado de esquema Kafka**: El payload de Kafka no incluye metadatos de versión del esquema. La gestión de compatibilidad será responsabilidad del worker y las herramientas de monitoreo.

