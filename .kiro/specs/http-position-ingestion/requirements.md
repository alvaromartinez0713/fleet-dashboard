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

1. WHEN se recibe una solicitud HTTP POST válida en el endpoint `/api/v1/positions`, THE servicio DEBE aceptar la solicitud y procesarla.
2. WHEN una solicitud HTTP POST llega sin el encabezado `Content-Type: application/json`, THE servicio DEBE rechazar la solicitud con código de estado 415.
3. WHERE el cliente desea enviar una sola posición (no un lote), THE servicio DEBE permitir enviar una posición individual en el body de la solicitud como un objeto JSON único.
4. WHEN el cliente envía un lote de posiciones (array JSON) en lugar de una posición individual, THE servicio DEBE rechazar la solicitud con código de estado 422 (Contenido No Procesable).
5. WHEN un evento está encolado esperando publicación, THE servicio DEBE responder al cliente con HTTP 202 (Aceptado) antes de intentar publicar en Kafka.

### Requirement 2: Validar el contrato de entrada explícitamente

**Historia de usuario:** Como responsable de la API, quiero validar explícitamente el contrato de cada solicitud de posición, de modo que solo se acepten datos bien formados y seguros.

#### Acceptance Criteria

1. THE servicio DEBE validar que el body contiene un objeto JSON único con la estructura esperada.
2. WHEN falta un campo obligatorio (vehicle_id, latitude, longitude, captured_at), THE servicio DEBE rechazar la solicitud con código de estado 422 y devolver un Problem Details indicando el campo faltante según RFC 9457.
3. WHEN el campo `vehicle_id` está vacío o es nulo, THE servicio DEBE rechazar la solicitud con código de estado 422.
4. WHEN el campo `latitude` no es un número o está fuera del rango [-90.0, 90.0] (precisión IEEE 754 de 64 bits), THE servicio DEBE rechazar la solicitud con código de estado 422.
5. WHEN el campo `longitude` no es un número o está fuera del rango [-180.0, 180.0] (precisión IEEE 754 de 64 bits), THE servicio DEBE rechazar la solicitud con código de estado 422.
6. WHEN el campo `captured_at` no es una cadena ISO 8601 válida con zona horaria UTC (formato: YYYY-MM-DDTHH:MM:SS.sssZ o YYYY-MM-DDTHH:MM:SS+00:00), THE servicio DEBE rechazar la solicitud con código de estado 422.
7. WHEN la solicitud contiene campos JSON adicionales desconocidos, THE servicio DEBE procesarla ignorando los campos adicionales (permisivo en la entrada).
8. WHEN la solicitud supera el tamaño máximo permitido (10 KB, medido en bytes UTF-8 del body HTTP), THE servicio DEBE rechazar la solicitud con código de estado 413 (Contenido Demasiado Grande).
9. WHEN el body JSON es malformado (sintaxis JSON inválida), THE servicio DEBE rechazar la solicitud con código de estado 400 (Solicitud Malformada).

### Requirement 3: Publicar posiciones en Kafka de forma asincrónica

**Historia de usuario:** Como responsable del sistema, quiero que el servicio HTTP publique posiciones en Kafka sin bloquear la respuesta al cliente, de modo que el servicio sea rápido y resiliente ante fallos en el procesamiento.

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

**Historia de usuario:** Como cliente de la API, quiero recibir una respuesta HTTP rápida después de confirmar que mi solicitud fue aceptada, sin esperar a que se complete el procesamiento remoto.

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

**Historia de usuario:** Como responsable de la confiabilidad del sistema, quiero que el servicio maneje explícitamente los fallos de Kafka con reintentos limitados y alertas, de modo que los eventos no se pierdan silenciosamente.

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

**Historia de usuario:** Como diseñador del contrato, quiero que el cliente entienda claramente que HTTP 202 confirma solo aceptación en Kafka, no persistencia en TimescaleDB.

#### Acceptance Criteria

1. WHEN el cliente recibe HTTP 202, THE body de respuesta DEBE indicar explícitamente que el evento fue aceptado para procesamiento asincrónico, NO que está definitivamente persistido.
2. THE servicio DEBE documentar en el contrato OpenAPI que HTTP 202 garantiza SOLO que el evento fue publicado en Kafka. NO garantiza:
   - Validación del evento
   - Procesamiento por el worker
   - Persistencia en TimescaleDB
3. WHERE el cliente requiera confirmar persistencia definitiva, THE servicio DEBE proporcionar un endpoint separado de consulta de estado (fuera del alcance de este requisito).
4. WHEN un evento se pierde después de publicarse en Kafka (por fallo del worker o TimescaleDB), THE cliente NO recibirá notificación automática; la responsabilidad es del worker y del sistema de monitoreo.

### Requirement 7: Registrar cuatro timestamps de auditoría y rastreo

**Historia de usuario:** Como operador del sistema, quiero rastrear exactamente cuándo ocurre cada etapa del procesamiento de la posición, de modo que pueda diagnosticar cuellos de botella y garantizar integridad.

#### Acceptance Criteria

1. EACH evento DEBE incluir EXACTAMENTE CUATRO timestamps UTC en formato ISO 8601 con milisegundos (YYYY-MM-DDTHH:MM:SS.sssZ):

   a. **captured_at** (obligatorio, cliente)
      - Cuándo: Cuándo fue capturada la ubicación en el dispositivo móvil
      - Origen: Enviado por el cliente en la solicitud HTTP
      - Responsabilidad: Cliente garantiza captura en UTC
      - Validación: No puede ser más de 1 hora en el futuro ni más de 24 horas en el pasado

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
   - No está más de 1 hora en el futuro (reloj del cliente desincronizado)
   - No está más de 24 horas en el pasado (eventos obsoletos)
   - Si falla validación: rechazar con código 422 y motivo específico

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
   - Causen que captured_at sea posterior a received_at o processed_at

### Requirement 8: Registrar y monitorear cada solicitud

**Historia de usuario:** Como operador del sistema, quiero registrar y monitorear cada solicitud para diagnosticar problemas y rastrear eventos.

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

### Requirement 9: Idempotencia mediante X-Idempotency-Key obligatorio

**Historia de usuario:** Como desarrollador del cliente, quiero que el cliente maneje reintentos de solicitud de forma segura usando un identificador idempotente.

#### Acceptance Criteria

1. THE encabezado `X-Idempotency-Key` es OBLIGATORIO en toda solicitud HTTP POST a `/api/v1/positions`.
2. WHEN el encabezado `X-Idempotency-Key` está ausente, THE servicio DEBE rechazar la solicitud con código de estado 400 (Solicitud Malformada).
3. WHEN el encabezado `X-Idempotency-Key` no es un UUID v4 válido, THE servicio DEBE rechazar la solicitud con código de estado 400 (Solicitud Malformada).
4. THE servicio DEBE almacenar en base de datos cada `X-Idempotency-Key` procesado durante 24 horas para durabilidad.
5. WHEN se recibe una solicitud con la misma `X-Idempotency-Key` dentro de la ventana de 24 horas, THE servicio DEBE:
   - Si el payload es idéntico: responder con la misma respuesta HTTP 202 (sin duplicar la publicación en Kafka) y usar el mismo trace_id
   - Si el payload es diferente: rechazar con código de estado 409 (Conflicto)
6. WHEN el cliente reutiliza un `X-Idempotency-Key` después de 24 horas (TTL expirado) o genera uno nuevo, THE servicio DEBE tratarlo como una nueva solicitud.
7. IT IS responsabilidad del cliente generar un UUID v4 único para cada solicitud nueva; no reutilizar claves para solicitudes distintas.

### Requirement 10: Contrato OpenAPI explícito

**Historia de usuario:** Como consumidor de la API, quiero disponer de una especificación OpenAPI clara del endpoint de ingesta.

#### Acceptance Criteria

1. THE servicio DEBE incluir un archivo `openapi.yaml` (OpenAPI 3.1.0) que documente:
   - Endpoint: POST /api/v1/positions
   - Esquema de solicitud con campos, validaciones y restricciones
   - Esquemas de respuesta para códigos 202, 400, 409, 413, 415, 422, 500, 503
   - Encabezados de solicitud (Content-Type, X-Trace-ID, X-Idempotency-Key)
2. WHEN se publica la API, THE contrato OpenAPI DEBE estar disponible en `/api/v1/openapi.yaml`.
3. THE respuestas de Problem Details DEBEN seguir RFC 9457 con campos obligatorios: type, title, status, detail, instance.

## Correctness Properties

Las siguientes propiedades DEBEN verificarse mediante testing:

1. **Rechazo de Validación**: Cualquier solicitud que viole criterios en Requisito 2 DEBE ser rechazada con el código de estado correcto (422 para semántica, 415 para media type, 413 para tamaño).

2. **Contrato de Publicación Asincrónica**: HTTP 202 se envía antes de intentar publicar en Kafka; si Kafka falla, el cliente NUNCA recibe notificación de ese fallo (por Requisito 5).

3. **Idempotencia**: Dos solicitudes idénticas con la misma X-Idempotency-Key dentro de 24 horas DEBEN resultar en la misma respuesta y una sola publicación en Kafka.

4. **Recuperación del Circuit Breaker**: Después de que Circuit Breaker se abre por fallos repetidos de Kafka, intentos de recuperación DEBEN ocurrir cada 30 segundos sin bloquear respuestas HTTP.

5. **Continuidad de Rastreo**: Cada entrada de log y respuesta de error DEBE referenciar el mismo trace_id durante todo el ciclo de vida de la solicitud.

6. **Integridad de Timestamps**: Los cuatro timestamps DEBEN estar presentes en cada evento, en orden cronológico estricto (captured_at ≤ received_at ≤ processed_at ≤ stored_at), y permanecer inalterados desde cliente hasta persistencia final.
