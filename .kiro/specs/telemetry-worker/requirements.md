# Documento de Requisitos: Telemetry Worker

## 1. Introducción

El Telemetry Worker es un servicio Go independiente que consume eventos de posición desde Kafka, valida datos requeridos de cada evento, deduplica por `event_id` y persiste las posiciones en TimescaleDB con garantías de idempotencia y sin pérdida silenciosa.

Cierra el flujo de ingesta asincrónica definido en `product.md` (RF-04, RF-05, RF-10) implementando desacoplamiento (RNF-03), integridad e idempotencia (RNF-04), persistencia especializada (RNF-05), resiliencia (RNF-06) y Clean Architecture (RNF-01).

## 2. Glosario

- **Telemetry_Worker**: Servicio Go independiente que consume eventos de Kafka, los valida y los persiste en TimescaleDB
- **positions_topic**: Topic de Kafka donde el servicio de ingesta publica eventos PositionReceived
- **PositionReceived_Event**: Evento JSON con campos requeridos (event_id, vehicle_id, latitude, longitude, captured_at) y opcionales (received_at, processed_at, accuracy)
- **event_id**: Identificador único (UUID v4) del evento, clave de deduplicación
- **vehicle_id**: Identificador del vehículo (alfanumérico 1-50)
- **positions_table**: Hypertable `telemetry.positions` de TimescaleDB donde se persisten las posiciones procesadas
- **Event_Time_Dimension**: Columna temporal del hypertable, en este caso `captured_at`
- **stored_at**: Timestamp UTC (milisegundos, ISO 8601) generado por el worker al confirmar el INSERT
- **Offset_Commit**: Confirmación del offset de Kafka que indica consumo exitoso
- **Idempotencia**: Propiedad de que procesar el mismo evento N veces (N ≥ 1) deja exactamente el mismo estado final en la base de datos
- **Deduplicación**: Mecanismo de garantizar que solo existe una fila por `event_id` único
- **Error_Recuperable**: Error transitorio (timeout DB, conexión rechazada, dependencia no disponible) susceptible de reintento
- **Error_No_Recuperable**: Error permanente (evento inválido, payload malformado) que no debe reintentarse
- **Dead_Letter_Sink**: Destino para registrar eventos descartados (MVP: log JSON estructurado)
- **Circuit_Breaker**: Mecanismo de resiliencia que protege TimescaleDB mediante estados (CLOSED, OPEN, HALF_OPEN)

## 3. Requisitos Funcionales

### Requisito 1: Consumir eventos de posición desde Kafka

**Descripción**: El Telemetry Worker debe consumir eventos de `positions_topic` de forma continua como parte de un Consumer_Group, preservando el orden por vehículo dentro de cada partición.

**Criterios de Aceptación**:

1. WHEN el Telemetry Worker inicia con conexión a Kafka disponible, THEN se suscribe a `positions_topic` y comienza a consumir mensajes de las particiones asignadas.
2. WHEN hay mensajes en una partición, THEN los procesa en orden de offset ascendente de esa partición.
3. WHEN procesa mensajes de una misma partición, THEN el procesamiento es secuencial (uno a uno), preservando orden por `vehicle_id`.
4. WHEN el payload JSON no es válido, THEN lo clasifica como Error_No_Recuperable, lo registra en el Dead_Letter_Sink y confirma el offset (commit).
5. WHEN Kafka revoca particiones durante rebalanceo, THEN completa el procesamiento de mensajes en curso dentro de `WORKER_SHUTDOWN_TIMEOUT` antes de liberarlas.
6. IF el tiempo de shutdown se agota, THEN libera las particiones sin confirmar offsets, permitiendo reproceso idempotente.
7. WHEN Kafka no está disponible, THEN registra WARN, clasifica como Error_Recuperable y reintenta con Backoff_Exponencial.

**Trazabilidad**: RF-04, RNF-03

---

### Requisito 2: Validar los datos requeridos de cada evento

**Descripción**: El Telemetry Worker debe validar que cada evento consumido contiene y cumple con los campos requeridos antes de intentar persistir.

**Campos Requeridos** (rechazo si faltan o inválidos):
- `event_id`: UUID v4 válido
- `vehicle_id`: Alfanumérico 1-50 caracteres
- `latitude`: float64 en rango [-90.0, 90.0]
- `longitude`: float64 en rango [-180.0, 180.0]
- `captured_at`: Timestamp ISO 8601 en UTC

**Campos Opcionales** (NULL en base de datos si ausentes):
- `received_at`: Timestamp ISO 8601 en UTC (si presente, >= `captured_at`)
- `processed_at`: Timestamp ISO 8601 en UTC
- `accuracy`: float64

**Criterios de Aceptación**:

1. WHEN valida `event_id`, THEN verifica que es UUID v4 válido; si no, Evento_Invalido.
2. WHEN valida `vehicle_id`, THEN verifica alfanumérico, trim de espacios, longitud 1-50; si no, Evento_Invalido.
3. WHEN valida `latitude`, THEN verifica rango [-90, 90], no es NaN ni Infinity; si no, Evento_Invalido.
4. WHEN valida `longitude`, THEN verifica rango [-180, 180], no es NaN ni Infinity; si no, Evento_Invalido.
5. WHEN valida `captured_at`, THEN verifica ISO 8601 UTC válido y no en futuro; si no, Evento_Invalido.
6. WHEN ambos `captured_at` y `received_at` están presentes y son válidos, IF `captured_at > received_at`, THEN Evento_Invalido.
7. WHEN `received_at` está presente, THEN verifica ISO 8601 UTC válido; si no, Evento_Invalido.
8. WHEN `received_at` está ausente, THEN persiste como NULL sin rechazar el evento.
9. WHEN `processed_at` está presente, THEN verifica ISO 8601 UTC válido; si no, Evento_Invalido.
10. WHEN el evento contiene campos desconocidos, THEN los ignora sin rechazar el evento.

**Si Evento_Invalido**:
- Clasifica como Error_No_Recuperable
- Registra ERROR log con `trace_id`, `event_id`, `vehicle_id` y motivo
- Escribe en Dead_Letter_Sink
- Confirma offset (commit)

**Trazabilidad**: RF-05, RNF-04

---

### Requisito 3: Persistir las posiciones en TimescaleDB

**Descripción**: El Telemetry Worker debe persistir cada posición válida en `positions_table` usando un INSERT parametrizado con deduplicación automática.

**Criterios de Aceptación**:

1. WHEN persiste una posición, THEN ejecuta un INSERT único parametrizado (sin concatenación de valores).
2. WHEN construye el INSERT, THEN incluye todos los campos: `event_id`, `vehicle_id`, `latitude`, `longitude`, `captured_at`, `received_at`, `processed_at`, `stored_at` y `accuracy` (si presente).
3. WHEN asigna `stored_at`, THEN usa el reloj actual del Clock en UTC con precisión de milisegundos.
4. WHEN un campo opcional (`received_at`, `processed_at`, `accuracy`) está ausente, THEN persiste como NULL.
5. WHEN un timestamp presente en el evento, THEN lo persiste con el valor exacto recibido (sin modificar).
6. WHEN el INSERT agrega una fila, THEN es Inserción Exitosa → commit offset.
7. WHEN el INSERT no agrega fila por conflicto `ON CONFLICT (captured_at, event_id)`, THEN es Duplicado → commit offset (garantía ya satisfecha).
8. WHEN un Duplicado existe, THEN no sobrescribe su `stored_at` preexistente.
9. IF al ejecutar el INSERT, la columna `event_id` o el índice unique no existen, THEN Error_No_Recuperable → dead-letter → commit.

**Trazabilidad**: RF-05, RNF-04, RNF-05

---

### Requisito 4: Procesamiento idempotente y deduplicación por event_id

**Descripción**: El Telemetry Worker debe garantizar que reprocesar el mismo evento produce el mismo estado final sin duplicados.

**Criterios de Aceptación**:

1. WHEN procesa el mismo evento N veces (N ≥ 2), THEN la base de datos contiene exactamente una fila para ese `event_id`.
2. WHEN la fila se crea la primera vez, THEN `stored_at` refleja el momento de la primera inserción.
3. WHEN se intenta insertar el mismo evento de nuevo, THEN `ON CONFLICT (captured_at, event_id) DO NOTHING` evita duplicado.
4. WHEN procesa un lote de hasta 500 eventos con duplicados, THEN el número de filas = número de `event_id` únicos y válidos.
5. WHEN cualquier `event_id` ya existe, THEN se cuenta como Duplicado y avanza el offset sin sobrescribir la fila.
6. WHEN los eventos con el mismo `event_id` llegan en cualquier orden, THEN el estado final es idéntico.

**Implementación**: Unique index `(captured_at, event_id)` + `INSERT ... ON CONFLICT (captured_at, event_id) DO NOTHING`.

**Trazabilidad**: RNF-04

---

### Requisito 5: Gestión de reintentos y clasificación de errores

**Descripción**: El Telemetry Worker debe reintentar Errores_Recuperables con límite y backoff determinista, y no reintentar Errores_No_Recuperables.

**Criterios de Aceptación**:

1. WHEN encuen tra un Error_Recuperable (timeout, conexión rechazada, etc.), THEN reintenta hasta `WORKER_RETRY_MAX` veces (defecto 3, rango 0-10).
2. WHEN reinten ta, THEN aplica Backoff_Exponencial: 100ms, 200ms, 400ms (sin jitter, determinista).
3. WHEN agota reintentos, THEN registra ERROR con número de intentos y escribe en Dead_Letter_Sink.
4. WHEN encuen tra un Error_No_Recuperable, THEN cero reintentos, derivación inmediata a Dead_Letter_Sink.
5. IF falla la escritura en Dead_Letter_Sink, THEN no confirma offset, registra ERROR y preserva el evento (sin pérdida silenciosa).

**Trazabilidad**: RNF-06

---

### Requisito 6: Confirmación de offset sin pérdida silenciosa

**Descripción**: El offset de Kafka solo se confirma cuando se garantiza que el evento fue persistido, deduplicado o registrado en el Dead_Letter_Sink.

**Reglas de Offset Commit**:

| Resultado | ¿Commit? | Razón |
|-----------|----------|-------|
| INSERT exitoso (SaveResultInserted) | SÍ | Persistencia confirmada |
| Duplicado (SaveResultDuplicate, ON CONFLICT resuelto) | SÍ | Garantía ya satisfecha |
| Error no recuperable + dead-letter escribió OK | SÍ | Registrado para diagnóstico |
| Error recuperable agotado + dead-letter escribió OK | SÍ | Registrado |
| Error recuperable agotado + dead-letter falló | NO | Sin pérdida silenciosa |
| Circuit Breaker OPEN | NO | Pausa consumo |

**Criterios de Aceptación**:

1. WHEN una inserción es exitosa, THEN confirma el offset.
2. WHEN un evento es duplicado, THEN confirma el offset (garantía ya satisfecha).
3. WHEN un evento se escribe en dead-letter con éxito, THEN confirma el offset.
4. WHILE un evento no se haya persistido, deduplicado ni registrado en dead-letter, THEN no confirma offset.
5. WHEN la escritura en dead-letter falla, THEN no confirma offset, registra ERROR y el mensaje se reprocesa.

**Auto-commit OFF**: Confirmación siempre explícita.

**Trazabilidad**: RNF-04, RNF-06

---

### Requisito 7: Circuit Breaker sobre persistencia en TimescaleDB

**Descripción**: El Telemetry Worker protege la dependencia TimescaleDB con un Circuit Breaker que se abre ante fallos repetidos y pausa el consumo de Kafka.

**Estados del Circuit Breaker**:

- **CLOSED**: Operación normal. Al acumular `WORKER_CB_FAILURE_THRESHOLD` (defecto 5) fallos consecutivos Error_Recuperable, transiciona a OPEN.
- **OPEN**: Pausa fetch de Kafka, no confirma offsets, espera `WORKER_CB_RECOVERY_INTERVAL` (defecto 30s) antes de transicionar a HALF_OPEN.
- **HALF_OPEN**: Permite un único intento de persistencia de prueba. Éxito → CLOSED (reanuda consumo); Fallo → OPEN.

**Criterios de Aceptación**:

1. WHEN Error_Recuperable en persistencia acumula ≥ `WORKER_CB_FAILURE_THRESHOLD`, THEN transiciona a OPEN.
2. WHILE OPEN, THEN pausa fetch de nuevos mensajes.
3. WHILE OPEN, THEN no confirma offsets de mensajes no persistidos.
4. WHEN transcurren `WORKER_CB_RECOVERY_INTERVAL`, THEN transiciona a HALF_OPEN.
5. WHEN intento de persistencia en HALF_OPEN es exitoso, THEN transiciona a CLOSED y reanuda consumo.
6. WHEN intento de persistencia en HALF_OPEN falla, THEN vuelve a OPEN.
7. WHEN cada operación de persistencia, THEN aplica timeout `WORKER_DB_OPERATION_TIMEOUT` (defecto 5s).
8. IF una operación excede el timeout, THEN cancela, clasifica como Error_Recuperable y cuenta para el umbral.

**Trazabilidad**: RF-10, RNF-06

---

### Requisito 8: Observabilidad de fallos y resultados

**Descripción**: El Telemetry Worker debe emitir logs JSON estructurados para diagnóstico sin exponer información sensible.

**Criterios de Aceptación**:

1. WHEN persiste con éxito, THEN registra INFO con `trace_id`, `event_id`, `vehicle_id`.
2. WHEN deriva a dead-letter, THEN registra ERROR con `trace_id`, `event_id`, clasificación y categoría de motivo (estable, p.ej. "invalid_event").
3. WHEN no puede confirmar offset, THEN registra WARN con `trace_id`, `event_id`.
4. WHEN emite cualquier log, THEN formato JSON con `timestamp` ISO 8601 UTC milisegundos.
5. WHEN emite cualquier log, THEN **NO incluye**: latitude, longitude, SQL, stack traces, secretos, DSN.
6. WHEN consume un evento con `trace_id` UUID v4 válido, THEN propaga ese `trace_id` en todos los logs.
7. IF evento no tiene `trace_id` o es inválido, THEN genera uno nuevo (UUID v4) para correlacionar logs.

**Trazabilidad**: RNF-06, security.md

---

### Requisito 9: Configuración parametrizable

**Descripción**: El Telemetry Worker debe leer su configuración desde variables de entorno con prefijo `WORKER_*`.

**Claves de Configuración**:

| Clave | Defecto | Requerido | Descripción |
|-------|---------|-----------|-------------|
| `WORKER_KAFKA_BROKERS` | - | Sí | Host1:Puerto,Host2:Puerto (CSV) |
| `WORKER_KAFKA_TOPIC` | positions | No | Nombre del topic |
| `WORKER_KAFKA_CONSUMER_GROUP` | telemetry-worker | No | Consumer group |
| `WORKER_RETRY_MAX` | 3 | No | Max reintentos (0-10) |
| `WORKER_RETRY_BASE_BACKOFF` | 100ms | No | Backoff inicial |
| `WORKER_CB_FAILURE_THRESHOLD` | 5 | No | Umbral apertura breaker |
| `WORKER_CB_RECOVERY_INTERVAL` | 30s | No | Intervalo recuperación |
| `WORKER_DB_DSN` | - | Sí | PostgreSQL connection string |
| `WORKER_DB_OPERATION_TIMEOUT` | 5s | No | Timeout operación DB |
| `WORKER_SHUTDOWN_TIMEOUT` | 30s | No | Timeout apagado graceful |

**Criterios de Aceptación**:

1. WHEN falta `WORKER_KAFKA_BROKERS` o `WORKER_DB_DSN`, THEN falla validación sin exponer valor.
2. WHEN `WORKER_RETRY_MAX` está fuera de rango, THEN falla validación.
3. WHEN `WORKER_CB_FAILURE_THRESHOLD` es < 1, THEN falla validación.
4. WHEN se lee la configuración, THEN **NO expone el DSN** en logs ni mensajes de error.

**Trazabilidad**: RF-09

---

### Requisito 10: Schema de positions_table alineado con el contrato del worker

**Descripción**: El schema de `telemetry.positions` debe editarse (no crearse nuevo) para incluir `event_id`, unique constraint de deduplicación, y los campos de timestamps del contrato.

**Decisión de Migración**: Editar `000002_create_positions_table.up.sql` en lugar de crear `000004` porque:
- Sin producción ni datos a preservar
- Bases efímeras de pruebas arrancan de cero
- Schema final más limpio

**Schema de positions_table**:

```sql
CREATE TABLE telemetry.positions (
    id               UUID NOT NULL DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL,
    vehicle_id       VARCHAR(50) NOT NULL,
    latitude         NUMERIC(9, 6) NOT NULL,
    longitude        NUMERIC(9, 6) NOT NULL,
    accuracy         NUMERIC(5, 2),
    captured_at      TIMESTAMPTZ NOT NULL,
    received_at      TIMESTAMPTZ,
    processed_at     TIMESTAMPTZ,
    stored_at        TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX positions_event_id_key
    ON telemetry.positions (captured_at, event_id);

SELECT create_hypertable('telemetry.positions', 'captured_at', if_not_exists => TRUE);

CREATE INDEX positions_vehicle_captured_at_idx
    ON telemetry.positions (vehicle_id, captured_at DESC);
```

**Criterios de Aceptación**:

1. WHEN ejecuta la migración, THEN crea `telemetry.positions` con columnas event_id, vehicle_id, lat, lon, accuracy (nullable), captured_at, received_at (nullable), processed_at (nullable), stored_at.
2. WHEN ejecuta, THEN crea unique index `(captured_at, event_id)` (columna temporal primero, requerimiento TimescaleDB).
3. WHEN ejecuta, THEN convierte a hypertable con `captured_at` como Event_Time_Dimension.
4. WHEN ejecuta down, THEN DROP TABLE IF EXISTS telemetry.positions CASCADE.
5. WHEN un desarrollador ya tiene base local con `000002` anterior aplicada, THEN debe recrear la base (drop + re-aplicar migraciones).

**Trazabilidad**: RNF-05

---

## 4. Propiedades Verificables

Las siguientes propiedades deben validarse mediante testing (testing.md):

1. **Idempotencia**: Para cualquier evento válido, procesarlo N veces (N ≥ 1) deja exactamente una fila con ese `event_id`.
2. **Deduplicación de Lotes**: Un lote de eventos con duplicados produce exactamente una fila por `event_id` único.
3. **Confluence**: Cualquier permutación del orden de llegada de eventos con el mismo `event_id` deja el mismo estado final.
4. **Sin Pérdida Silenciosa**: Un mensaje nunca obtiene offset commit salvo que haya sido persistido, deduplicado o registrado en dead-letter.
5. **Clasificación de Errores**: Error_No_Recuperable → cero reintentos; Error_Recuperable → ≤ maxRetries+1 intentos con backoff determinista antes de dead-letter.
6. **Preservación de Timestamps**: Timestamps presentes se persisten sin modificar; ausentes → NULL; `stored_at ≥ processed_at` cuando `processed_at` presente.

---

## 5. Criterios de Aceptación Globales

- ✅ Consumir eventos Kafka secuencialmente por partición
- ✅ Validar 10 criterios de campos requeridos
- ✅ Persistir con INSERT parametrizado + ON CONFLICT
- ✅ Reintentos con backoff determinista 100/200/400ms
- ✅ Dead-letter JSON sin secretos
- ✅ Circuit Breaker 3 estados
- ✅ Offset commit solo cuando garantía satisfecha
- ✅ Logs JSON con trace_id
- ✅ Schema con unique constraint `(captured_at, event_id)`
- ✅ 9 claves de configuración WORKER_*
- ✅ Compilación exitosa: `go build ./cmd/worker`

