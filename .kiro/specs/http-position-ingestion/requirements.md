# Requirements Document

## Introduction

El servicio HTTP de ingesta recibe posiciones individuales de vehículos desde clientes móviles o externos, las valida contra un contrato explícito y las publica de forma asincrónica en Kafka. El servicio responde rápidamente al cliente después de confirmar la aceptación de la solicitud para procesamiento, sin esperar a que la persistencia definitiva en TimescaleDB sea completada.

Este requisito es fundamental para RF-04 (Ingesta asincrónica de telemetría) y RNF-03 (Desacoplamiento del procesamiento) de product.md.

## Glossary

- **Posición**: Coordenadas de ubicación (latitud, longitud) de un vehículo en un momento específico, acompañadas de metadatos identificadores.
- **Vehículo**: Entidad identificada unívocamente por un identificador del dominio (ID_Vehículo). Es responsabilidad del cliente proporcionar este identificador.
- **Cliente**: Sistema que envía solicitudes HTTP. Puede ser la aplicación móvil, un simulador de carga o un servicio externo.
- **Validación de contrato**: Verificación de que la solicitud HTTP cumple el esquema esperado (tipos, formatos, campos obligatorios, límites).
- **Publicación asincrónica**: La solicitud HTTP se acepta y se publica en Kafka sin esperar a que el worker procese el evento.
- **Confirmación de aceptación**: Respuesta HTTP que indica que la solicitud fue recibida correctamente y publicada en el tópico de Kafka, sin garantizar persistencia definitiva.
- **Tópico_Posiciones**: Canal de mensajería donde se publican los eventos de posición en Kafka.
- **Worker**: Consumidor de Kafka que procesa eventos de Tópico_Posiciones y los persiste en TimescaleDB.
- **Circuit Breaker**: Mecanismo de resiliencia que gestiona fallos en la comunicación con Kafka.
- **Reintentos**: Estrategia para reintentar la publicación en Kafka ante fallos transitorios.
- **Fallo transitorio**: Error temporal (timeout, conexión rechazada) que potencialmente puede resolverse con un reintento.
- **Fallo permanente**: Error irrecuperable (validación fallida, payload malformado en Kafka) que no debe reintentarse.
- **Alerta**: Notificación de un fallo crítico que requiere atención operativa.
- **HTTP 202 Aceptado**: Código de estado HTTP que indica que la solicitud fue aceptada para procesamiento asincrónico.
- **Clave-Idempotencia**: Encabezado HTTP que permite al cliente indicar que una solicitud debe ser tratada idempotentemente.
- **ID_Rastreo**: Identificador único (UUID) asignado a cada solicitud para rastreabilidad en logs.

## Requirements

### Requirement 1: Recibir posiciones individuales mediante HTTP POST

**Historia de usuario:** Como cliente de la API, quiero enviar una posición individual de un vehículo mediante HTTP POST, de modo que el sistema registre la ubicación del vehículo.

#### Acceptance Criteria

1. CUANDO se recibe una solicitud HTTP POST válida en el endpoint `/api/v1/posiciones`, EL Servicio_Ingesta DEBE aceptar la solicitud y procesarla.
2. CUANDO una solicitud HTTP POST llega sin el encabezado `Content-Type: application/json`, EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 415.
3. DONDE el cliente desea enviar una sola posición (no un lote), EL Servicio_Ingesta DEBE permitir enviar una posición individual en el cuerpo de la solicitud como un objeto JSON único.
4. CUANDO el cliente envía un lote de posiciones (array JSON) en lugar de una posición individual, EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422 (Contenido No Procesable).
5. CUANDO un evento está encolado esperando publicación, EL Servicio_Ingesta DEBE responder al cliente con HTTP 202 (Aceptado) antes de intentar publicar en Kafka.

### Requirement 2: Validar el contrato de entrada explícitamente

**Historia de usuario:** Como responsable de la API, quiero validar explícitamente el contrato de cada solicitud de posición, de modo que solo se acepten datos bien formados y seguros.

#### Acceptance Criteria

1. EL Servicio_Ingesta DEBE validar que el cuerpo contiene un objeto JSON único con la estructura esperada.
2. CUANDO falta un campo obligatorio (id_vehiculo, latitud, longitud, capturado_en), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422 y devolver un Detalle_Problema indicando el campo faltante según RFC 9457.
3. CUANDO el campo `id_vehiculo` está vacío o es nulo, EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422.
4. CUANDO el campo `latitud` no es un número o está fuera del rango [-90.0, 90.0] (precisión IEEE 754 de 64 bits), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422.
5. CUANDO el campo `longitud` no es un número o está fuera del rango [-180.0, 180.0] (precisión IEEE 754 de 64 bits), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422.
6. CUANDO el campo `capturado_en` no es una cadena ISO 8601 válida con zona horaria UTC (formato: YYYY-MM-DDTHH:MM:SS.sssZ o YYYY-MM-DDTHH:MM:SS+00:00), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 422.
7. CUANDO la solicitud contiene campos JSON adicionales desconocidos, EL Servicio_Ingesta DEBE procesarla ignorando los campos adicionales (permisivo en la entrada).
8. CUANDO la solicitud supera el tamaño máximo permitido (10 KB, medido en bytes UTF-8 del cuerpo HTTP), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 413 (Contenido Demasiado Grande).
9. CUANDO el cuerpo JSON es malformado (sintaxis JSON inválida), EL Servicio_Ingesta DEBE rechazar la solicitud con código de estado 400 (Solicitud Malformada).

### Requirement 3: Publicar posiciones en Kafka de forma asincrónica

**Historia de usuario:** Como responsable del sistema, quiero que el servicio HTTP publique posiciones en Kafka sin bloquear la respuesta al cliente, de modo que el servicio sea rápido y resiliente ante fallos en el procesamiento.

#### Acceptance Criteria

1. CUANDO se recibe una solicitud de posición válida, EL Servicio_Ingesta DEBE publicar el evento en Tópico_Posiciones de Kafka de forma asincrónica (sin esperar la confirmación de Kafka antes de responder al cliente).
2. EL Servicio_Ingesta DEBE mantener una cola interna de hasta 10,000 eventos para encolar publicaciones y procesarlas en background.
3. CUANDO se publica en Kafka, EL Servicio_Ingesta DEBE usar:
   - Clave de partición: `id_vehiculo` (garantiza orden por vehículo)
   - Formato: JSON con campos: id_evento, id_vehiculo, latitud, longitud, precisión, capturado_en, recibido_en, procesado_en
   - Confirmación Kafka: `acks=1` (ACK de 1 replica como mínimo)
4. CUANDO se publica con éxito en Kafka, EL Servicio_Ingesta DEBE registrar un mensaje de nivel INFO indicando el id_rastreo y el id_vehiculo.
5. DONDE el tiempo es crítico, EL Servicio_Ingesta DEBE garantizar latencia p95 ≤ 500ms entre recepción HTTP y confirmación de Kafka.

### Requirement 4: Responder rápidamente al cliente con HTTP 202 Aceptado

**Historia de usuario:** Como cliente de la API, quiero recibir una respuesta HTTP rápida después de confirmar que mi solicitud fue aceptada, sin esperar a que se complete el procesamiento remoto.

#### Acceptance Criteria

1. CUANDO se recibe una solicitud de posición válida, EL Servicio_Ingesta DEBE responder con HTTP 202 (Aceptado) en menos de 100 milisegundos (p99, medido desde el primer byte del socket hasta el último byte de respuesta).
2. CUANDO responde con HTTP 202, EL Servicio_Ingesta DEBE incluir un cuerpo JSON con los datos de aceptación (y solo estos campos):
   - `estado`: cadena con valor "aceptado"
   - `id_rastreo`: identificador único de la solicitud (UUID v4)
   - `id_vehiculo`: identificador del vehículo (eco de la solicitud)
   - `recibido_en`: marca de tiempo de recepción en formato ISO 8601 con milisegundos
3. CUANDO responde con HTTP 202, EL Servicio_Ingesta DEBE incluir el encabezado `X-ID-Rastreo` con el id_rastreo.
4. CUANDO responde con error de validación (códigos 400, 413, 415, 422), EL Servicio_Ingesta DEBE responder con JSON Detalle_Problema según RFC 9457 en menos de 50 milisegundos (p99).
5. CUANDO ocurre un error inesperado del servidor (código 500), EL Servicio_Ingesta DEBE devolver un Detalle_Problema genérico sin exponer trazas de stack, consultas SQL, rutas internas, nombres de configuración ni detalles técnicos.

### Requirement 5: Implementar reintentos y manejo de fallos de Kafka

**Historia de usuario:** Como responsable de la confiabilidad del sistema, quiero que el servicio maneje explícitamente los fallos de Kafka con reintentos limitados y alertas, de modo que los eventos no se pierdan silenciosamente.

#### Acceptance Criteria

1. CUANDO Kafka no es accesible o rechaza la publicación, EL Servicio_Ingesta DEBE reintentar la publicación hasta 3 veces con backoff exponencial:
   - 1er reintento: 100ms después del fallo inicial
   - 2do reintento: 200ms después del 1er fallo
   - 3er reintento: 400ms después del 2do fallo
2. EL Servicio_Ingesta DEBE clasificar errores de Kafka en dos categorías:
   - Errores retryables (transitorios): timeout, conexión rechazada, broker no disponible
   - Errores no retryables: mensaje malformado, esquema inválido, topic no existe
3. CUANDO falla el reintento final (3 intentos agotados), EL Servicio_Ingesta DEBE:
   - Registrar un mensaje de nivel ERROR con id_rastreo, tipo de error y razón del fallo
   - Escribir el evento en un dead-letter log estructurado para recuperación posterior
   - Descartar el evento de la cola para evitar bloquear solicitudes futuras
4. CUANDO la conexión con Kafka falla repetidamente (5 fallos consecutivos en ventana de 60 segundos), EL Servicio_Ingesta DEBE aplicar el Circuit Breaker y transicionar a estado ABIERTO.
5. SI el Circuit Breaker está en estado ABIERTO y se recibe una nueva solicitud HTTP, ENTONCES EL Servicio_Ingesta DEBE responder al cliente con HTTP 503 (Servicio No Disponible) indicando que la dependencia no está disponible.
6. CUANDO el Circuit Breaker está en estado ABIERTO, EL Servicio_Ingesta DEBE:
   - Intentar recuperación periódicamente (cada 30 segundos) sin bloquear solicitudes nuevas
   - Enviar un mensaje de prueba sintetizado con timeout de 5 segundos
   - Transicionar a estado SEMI_ABIERTO para permitir recuperación gradual
   - Transicionar a estado CERRADO cuando se confirme la recuperación de Kafka

### Requirement 6: Distinguir entre confirmación de aceptación y persistencia definitiva

**Historia de usuario:** Como diseñador del contrato, quiero que el cliente entienda claramente que HTTP 202 confirma solo aceptación en Kafka, no persistencia en TimescaleDB.

#### Acceptance Criteria

1. CUANDO el cliente recibe HTTP 202, EL cuerpo de respuesta DEBE indicar explícitamente que el evento fue aceptado para procesamiento asincrónico, NO que está definitivamente persistido.
2. EL Servicio_Ingesta DEBE documentar en el contrato OpenAPI que HTTP 202 garantiza SOLO que el evento fue publicado en Kafka. NO garantiza:
   - Validación del evento
   - Procesamiento por el worker
   - Persistencia en TimescaleDB
3. DONDE el cliente requiera confirmar persistencia definitiva, EL Servicio_Ingesta DEBE proporcionar un endpoint separado de consulta de estado (fuera del alcance de este requisito).
4. CUANDO un evento se pierde después de publicarse en Kafka (por fallo del worker o TimescaleDB), EL cliente NO recibirá notificación automática; la responsabilidad es del worker y del sistema de monitoreo.

### Requirement 7: Registrar cuatro timestamps de auditoría y rastreo

**Historia de usuario:** Como operador del sistema, quiero rastrear exactamente cuándo ocurre cada etapa del procesamiento de la posición, de modo que pueda diagnosticar cuellos de botella y garantizar integridad.

#### Acceptance Criteria

1. CADA evento DEBE incluir EXACTAMENTE CUATRO timestamps UTC en formato ISO 8601 con milisegundos (YYYY-MM-DDTHH:MM:SS.sssZ):

   a. **capturado_en** (obligatorio, cliente)
      - CUÁNDO: Cuándo fue capturada la ubicación en el dispositivo móvil
      - ORIGEN: Enviado por el cliente en la solicitud HTTP
      - RESPONSABILIDAD: Cliente garantiza captura en UTC
      - VALIDACIÓN: No puede ser más de 1 hora en el futuro ni más de 24 horas en el pasado

   b. **recibido_en** (obligatorio, servidor HTTP)
      - CUÁNDO: Cuándo el servidor HTTP recibió completamente la solicitud (último byte del cuerpo)
      - ORIGEN: Generado automáticamente por el servidor al recibir la solicitud
      - PRECISIÓN: Milisegundos exactos desde el reloj del servidor
      - CÁLCULO: timestamp_recepción = ahora_servidor_al_recibir_último_byte

   c. **procesado_en** (obligatorio, servidor HTTP)
      - CUÁNDO: Cuándo completó validación de contrato y fue encolado para Kafka
      - ORIGEN: Generado automáticamente después de validación exitosa
      - MARCA: Inicio del procesamiento asincrónico
      - CÁLCULO: timestamp_procesamiento = ahora_servidor_después_validación_exitosa

   d. **almacenado_en** (obligatorio, worker)
      - CUÁNDO: Cuándo fue persistido en TimescaleDB (INSERT completado)
      - ORIGEN: Generado por worker después de INSERT exitoso
      - CONFIRMACIÓN: Confirma persistencia definitiva
      - CÁLCULO: timestamp_almacenamiento = ahora_servidor_después_INSERT_exitoso

2. CADA evento publicado en Kafka DEBE incluir estos cuatro timestamps en el payload JSON con los tres primeros campos completados:
   ```json
   {
     "id_evento": "uuid",
     "id_vehiculo": "id",
     "latitud": 40.7128,
     "longitud": -74.0060,
     "precisión": 5.0,
     "capturado_en": "2024-10-17T14:30:45.123Z",
     "recibido_en": "2024-10-17T14:30:45.456Z",
     "procesado_en": "2024-10-17T14:30:45.467Z",
     "almacenado_en": null
   }
   ```

3. CUANDO el evento es recibido por HTTP, EL Servicio_Ingesta DEBE validar que `capturado_en`:
   - Es un timestamp ISO 8601 válido con zona horaria UTC
   - No está más de 1 hora en el futuro (reloj del cliente desincronizado)
   - No está más de 24 horas en el pasado (eventos obsoletos)
   - Si falla validación: rechazar con código 422 y motivo específico

4. CUANDO el evento llega al worker desde Kafka, EL worker DEBE:
   - Recibir los tres primeros timestamps (capturado_en, recibido_en, procesado_en) inalterados
   - Agregar automáticamente almacenado_en = ahora_servidor cuando el INSERT sea exitoso
   - Persistir todos cuatro timestamps en TimescaleDB sin modificación
   - Usar estos timestamps para auditoría y detección de retrasos

5. CUANDO se persiste en TimescaleDB, LA tabla DEBE incluir columnas separadas para cada timestamp:
   - `captured_at` (timestamp with timezone, NOT NULL)
   - `received_at` (timestamp with timezone, NOT NULL)
   - `processed_at` (timestamp with timezone, NOT NULL)
   - `stored_at` (timestamp with timezone, NOT NULL)
   - Todas con zona horaria UTC (AT TIME ZONE 'UTC')

6. LOS timestamps DEBEN permitir cálculos de latencia para diagnóstico:
   - Latencia cliente-servidor: `recibido_en - capturado_en`
   - Latencia de validación: `procesado_en - recibido_en`
   - Latencia de ingesta: `procesado_en - capturado_en`
   - Latencia de persistencia: `almacenado_en - procesado_en`

7. EL Servicio_Ingesta DEBE rechazar timestamps que:
   - No sean ISO 8601 válido
   - Tengan zona horaria distinta de UTC
   - Tengan precisión menor a milisegundos
   - Causen que capturado_en sea posterior a recibido_en o procesado_en

### Requirement 8: Registrar y monitorear cada solicitud

**Historia de usuario:** Como operador del sistema, quiero registrar y monitorear cada solicitud para diagnosticar problemas y rastrear eventos.

#### Acceptance Criteria

1. PARA cada solicitud recibida, EL Servicio_Ingesta DEBE registrar (en logs JSON):
   - id_rastreo: único (generado o extraído del encabezado `X-ID-Rastreo` si lo proporciona el cliente)
   - id_vehiculo: identificador del vehículo
   - código_http: código de estado HTTP de respuesta
   - duracion_ms: duración de la solicitud en milisegundos (desde primer byte recibido hasta último byte enviado)
   - estado_operacion: "aceptado", "rechazado_validacion" o "rechazado_kafka"
   - nivel_log: según la gravedad

2. CUANDO se publica en Kafka, EL Servicio_Ingesta DEBE registrar (nivel INFO):
   - id_rastreo, id_vehiculo, timestamps (capturado_en, recibido_en, procesado_en)
   - Sin incluir datos sensibles (coordenadas NO se registran completas; solo precisión)

3. CUANDO ocurre un error, EL Servicio_Ingesta DEBE registrar (nivel ERROR):
   - id_rastreo, tipo_error, mensaje_error
   - Sin exponer: trazas de stack, consultas SQL, rutas internas, nombres de configuración, tokens

4. LOS logs DEBEN ser estructurados en formato JSON e incluir timestamp en ISO 8601:
   ```json
   {
     "timestamp": "2024-10-17T14:30:45.123Z",
     "nivel": "INFO",
     "id_rastreo": "550e8400-e29b-41d4-a716-446655440000",
     "id_vehiculo": "VH-001",
     "evento": "solicitud_recibida",
     "codigo_http": 202,
     "duracion_ms": 45,
     "estado_operacion": "aceptado"
   }
   ```

5. EL Servicio_Ingesta DEBE aplicar límite de cardinality: máximo 1000 logs por segundo por nivel, con sampling adaptativo para prevenir explosión de logs.

### Requirement 9: Soportar idempotencia mediante Clave-Idempotencia

**Historia de usuario:** Como desarrollador del cliente, quiero que el cliente maneje reintentos de solicitud de forma segura.

#### Acceptance Criteria

1. CUANDO el cliente reintenta una solicitud (ej. timeout), EL cliente DEBE incluir un encabezado `Clave-Idempotencia` con un identificador único UUID v4 de la operación.
2. CUANDO el servidor recibe una solicitud con la misma `Clave-Idempotencia`, EL Servicio_Ingesta DEBE:
   - Responder con la misma respuesta HTTP 202 (sin duplicar la publicación en Kafka)
   - Usar el mismo id_rastreo de la solicitud anterior
3. EL Servicio_Ingesta DEBE mantener un cache de Clave-Idempotencia con duración limitada (24 horas).
4. CUANDO el cache expira, EL Servicio_Ingesta DEBE permitir reintentos sin deduplicación (responsabilidad del worker detectar duplicados por id_evento).
5. SI se recibe una solicitud con mismo Clave-Idempotencia pero distinto id_vehiculo, ENTONCES EL Servicio_Ingesta DEBE rechazarla con código 422 (Contenido No Procesable, "error de idempotencia: id_vehiculo inconsistente").

### Requirement 10: Contrato OpenAPI explícito

**Historia de usuario:** Como consumidor de la API, quiero disponer de una especificación OpenAPI clara del endpoint de ingesta.

#### Acceptance Criteria

1. EL Servicio_Ingesta DEBE incluir un archivo `openapi.yaml` (OpenAPI 3.1.0) que documente:
   - Endpoint: POST /api/v1/posiciones
   - Esquema de solicitud con campos, validaciones y restricciones
   - Esquemas de respuesta para códigos 202, 400, 413, 415, 422, 500, 503
   - Encabezados de solicitud (Content-Type, X-ID-Rastreo, Clave-Idempotencia)
2. CUANDO se publica la API, EL contrato OpenAPI DEBE estar disponible en `/api/v1/openapi.yaml`.
3. LAS respuestas de Detalle_Problema DEBEN seguir RFC 9457 con campos obligatorios: tipo, título, estado, detalle, instancia.

## Correctness Properties

Las siguientes propiedades DEBEN verificarse mediante testing:

1. **Rechazo de Validación**: Cualquier solicitud que viole criterios en Requisito 2 DEBE ser rechazada con el código de estado correcto (422 para semántica, 415 para media type, 413 para tamaño).

2. **Contrato de Publicación Asincrónica**: HTTP 202 se envía antes de intentar publicar en Kafka; si Kafka falla, el cliente NUNCA recibe notificación de ese fallo (por Requisito 5).

3. **Idempotencia**: Dos solicitudes idénticas con la misma Clave-Idempotencia dentro de 24 horas DEBEN resultar en la misma respuesta y una sola publicación en Kafka.

4. **Recuperación del Circuit Breaker**: Después de que Circuit Breaker se abre por fallos repetidos de Kafka, intentos de recuperación DEBEN ocurrir cada 30 segundos sin bloquear respuestas HTTP.

5. **Continuidad de Rastreo**: Cada entrada de log y respuesta de error DEBE referenciar el mismo id_rastreo durante todo el ciclo de vida de la solicitud.

6. **Integridad de Timestamps**: Los cuatro timestamps DEBEN estar presentes en cada evento, en orden cronológico estricto (capturado_en ≤ recibido_en ≤ procesado_en ≤ almacenado_en), y permanecer inalterados desde cliente hasta persistencia final.
