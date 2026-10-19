# Diseño: Infraestructura Docker Compose (Kafka + TimescaleDB)

## 1. Resumen Ejecutivo

Este documento presenta el diseño técnico de la infraestructura local Docker Compose para el Portal de Monitoreo de Flotas, extendiendo la configuración existente con tres servicios nuevos:

1. **Zookeeper** (confluentinc/cp-zookeeper:7.6.0): Coordinador de metadatos para Kafka
2. **Kafka** (confluentinc/cp-kafka:7.6.0): Bus de eventos desacoplado para telemetría
3. **TimescaleDB** (timescale/timescaledb:2.14.0-pg15): Persistencia de series de tiempo

**Objetivo**: Permitir a desarrolladores ejecutar `docker compose up` en la raíz del proyecto y obtener un entorno completamente funcional, aislado, reproducible y sin dependencias externas.

---

## 2. Arquitectura de Alto Nivel

### 2.1 Diagrama de Servicios y Relaciones

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Docker Compose Stack                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   fleet-network (bridge)                    │   │
│  ├─────────────────────────────────────────────────────────────┤   │
│  │                                                              │   │
│  │  ┌──────────────────┐                                       │   │
│  │  │   Zookeeper      │                                       │   │
│  │  │   :2181          │                                       │   │
│  │  │                  │                                       │   │
│  │  │  zookeeper-data  │                                       │   │
│  │  └──────────────────┘                                       │   │
│  │         △                                                   │   │
│  │         │                                                   │   │
│  │         │ (depends_on: condition: service_healthy)         │   │
│  │         │                                                   │   │
│  │  ┌──────────────────┐         ┌──────────────────┐         │   │
│  │  │   Kafka Broker   │         │   TimescaleDB    │         │   │
│  │  │   :9092          │         │   :5432          │         │   │
│  │  │                  │         │   5433→5432 (*)  │         │   │
│  │  │  kafka-data      │         │                  │         │   │
│  │  │  (logs)          │         │ fleet_db         │         │   │
│  │  └──────────────────┘         │ └─telemetry.*    │         │   │
│  │         △                     │   └─positions    │         │   │
│  │         │                     │ fleet_user       │         │   │
│  │         │                     │                  │         │   │
│  │         │                     │ timescaledb-data │         │   │
│  │         │ (consumer)          └──────────────────┘         │   │
│  │         │                              △                   │   │
│  │         │                              │ (external access) │   │
│  │         │                              │                   │   │
│  │  ┌──────────────────┐    ┌──────────────────┐             │   │
│  │  │  Ingestion API   │    │  Worker Service  │             │   │
│  │  │  (publisher)     │    │  (consumer)      │             │   │
│  │  │  :8080           │    │                  │             │   │
│  │  │                  │    │                  │             │   │
│  │  │ Existing         │    │ Existing         │             │   │
│  │  └──────────────────┘    └──────────────────┘             │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  (*) Port mapping configured via ${TIMESCALEDB_PORT:-5432}          │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘

Host Machine
  ↓ (exposed ports)
  - localhost:8080 → Ingestion API (HTTP)
  - localhost:${TIMESCALEDB_PORT} → TimescaleDB (PostgreSQL)
```

### 2.2 Flujo de Comunicación

```
Mobile Device / Test Client
  │
  ├─→ POST /positions (Ingestion API, localhost:8080)
  │
  └─→ Ingestion Service (internal)
      │
      ├─→ [Kafka Topic: telemetry.positions]
      │   - Key: vehicle_id (partition key)
      │   - Value: PositionReceivedEvent
      │
      ├─→ Worker Service (consumer)
      │   │
      │   ├─→ Validation & Processing
      │   │
      │   └─→ INSERT into telemetry.positions (TimescaleDB)
      │
      └─→ Dashboard / External Tools
          │
          └─→ SELECT from telemetry.positions (localhost:${TIMESCALEDB_PORT})
```

### 2.3 Principios de Diseño

1. **Aislamiento**: Todos los servicios en red bridge privada sin exposición innecesaria
2. **Reproducibilidad**: Misma configuración para todos los desarrolladores
3. **Persistencia**: Datos sobreviven reinicios de contenedor (pero no docker compose down)
4. **Ordenamiento**: Zookeeper → Kafka → TimescaleDB (via depends_on + health checks)
5. **Recuperación**: Restart policies `unless-stopped` para recuperación automática
6. **Configurabilidad**: Variables .env para personalización sin editar YAML

---

## 3. Especificaciones de Servicios

### 3.1 Zookeeper Service

**Propósito**: Coordinador de metadatos distribuido que Kafka utiliza para gestionar brokers, topics, particiones y sincronizar estado de cluster.

#### Imagen y Versión

```yaml
image: confluentinc/cp-zookeeper:7.6.0
```

**Justificación**: Confluent proporciona versiones certificadas y bien mantenidas de Apache Kafka y Zookeeper. La versión 7.6.0 es estable, ampliamente adoptada y compatible con cp-kafka:7.6.0.

#### Configuración

```yaml
zookeeper:
  image: confluentinc/cp-zookeeper:7.6.0
  container_name: fleet-zookeeper
  environment:
    # Identificador único del nodo Zookeeper en el cluster (1 node local)
    ZOO_MY_ID: 1
    
    # Puerto de servidor Zookeeper (coordinación interna)
    ZOO_PORT: 2181
    
    # Port para líderes de Zookeeper (no necesario en single node)
    ZOO_SERVERS: "server.1=zookeeper:2888:3888"
    
    # Logging
    LOG4J_ROOT_LOGLEVEL: WARN
  
  ports:
    # Puerto 2181 NO se expone al host (interno solo)
    # Kafka y aplicaciones locales acceden via DNS fleet-network
  
  volumes:
    # Persistencia de snapshots y logs de transacción
    - zookeeper-data:/var/lib/zookeeper/data
    - zookeeper-logs:/var/lib/zookeeper/log
  
  networks:
    - fleet-network
  
  restart: unless-stopped
  
  healthcheck:
    # Conectar a puerto 2181 y verificar que Zookeeper responde
    test: ["CMD", "echo", "ruok", "|", "nc", "-q", "1", "localhost", "2181"]
    interval: 10s
    timeout: 3s
    retries: 3
    start_period: 5s
```

#### Volúmenes

- **zookeeper-data**: Snapshots y datos de estado (requerido para persistencia)
- **zookeeper-logs**: Logs de transacción (WAL para recuperación)

#### Health Check

Ejecuta `echo ruok | nc localhost 2181` que envía comando "ruok" (Are You OK) al puerto 2181. Zookeeper responde con "imok" si está listo. Este mecanismo es estándar para Zookeeper.

---

### 3.2 Kafka Service

**Propósito**: Bus de eventos que desacopla productores (Ingestion API) de consumidores (Worker). Proporciona durabilidad, ordenamiento por partición y reintentos.

#### Imagen y Versión

```yaml
image: confluentinc/cp-kafka:7.6.0
```

**Justificación**: Versión compatible con cp-zookeeper:7.6.0, incluye herramientas para crear topics y administrar particiones.

#### Configuración

```yaml
kafka:
  image: confluentinc/cp-kafka:7.6.0
  container_name: fleet-kafka
  
  depends_on:
    zookeeper:
      condition: service_healthy
  
  environment:
    # ID único del broker en el cluster (1 broker local)
    KAFKA_BROKER_ID: 1
    
    # Coordinador Zookeeper
    KAFKA_ZOOKEEPER_CONNECT: "zookeeper:2181"
    
    # Listener para conexiones internas y externas
    KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://kafka:9092"
    KAFKA_LISTENERS: "PLAINTEXT://0.0.0.0:9092"
    KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
    
    # Protocolo de seguridad (PLAINTEXT para desarrollo local)
    KAFKA_SECURITY_INTER_BROKER_PROTOCOL: PLAINTEXT
    
    # Replicación: 1 broker = 1 factor (no es redundante)
    KAFKA_DEFAULT_REPLICATION_FACTOR: 1
    KAFKA_MIN_INSYNC_REPLICAS: 1
    
    # Particiones por defecto para topics auto-creados
    KAFKA_NUM_PARTITIONS: 3
    
    # Log cleanup y retention
    KAFKA_LOG_RETENTION_HOURS: 24
    KAFKA_LOG_SEGMENT_BYTES: 1073741824
    
    # Memória heap
    KAFKA_HEAP_OPTS: "-Xmx512M -Xms512M"
    
    # Logging
    KAFKA_LOG4J_ROOT_LOGLEVEL: WARN
  
  ports:
    # Puerto 9092 NO se expone al host (interno solo)
    # Ingestion y Worker acceden via DNS fleet-network
  
  volumes:
    # Logs de Kafka (particiones, índices, snapshots)
    - kafka-data:/var/lib/kafka/data
  
  networks:
    - fleet-network
  
  restart: unless-stopped
  
  healthcheck:
    # Verificar que el broker está listo usando kafka-broker-api-versions
    test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server=localhost:9092"]
    interval: 10s
    timeout: 3s
    retries: 3
    start_period: 10s
```

#### Topic: telemetry.positions

El topic se crea automáticamente con el siguiente script de inicialización (ejecutado en entrypoint de Kafka):

**Archivo**: `backend/go/scripts/create-kafka-topic.sh` (se proporciona en tasks)

```bash
#!/bin/bash
# Esperar a que Kafka esté completamente listo
sleep 15

# Crear topic con 3 particiones, 1 replication factor
kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic telemetry.positions \
  --partitions 3 \
  --replication-factor 1 \
  --config cleanup.policy=delete \
  --config retention.ms=86400000 \
  --if-not-exists 2>/dev/null

echo "✓ Topic telemetry.positions created or already exists"
```

**Decisión de diseño**: Usar script separado en lugar de init container para mantener docker-compose.yml limpio. El script se ejecuta después de que Kafka esté healthy mediante healthcheck.

#### Volúmenes

- **kafka-data**: Logs de broker, índices de particiones, snapshots

#### Health Check

Ejecuta `kafka-broker-api-versions --bootstrap-server=localhost:9092` que verifica que el broker responde a llamadas de API y está operativo.

---

### 3.3 TimescaleDB Service

**Propósito**: Base de datos PostgreSQL con extensión TimescaleDB para almacenar series de tiempo de telemetría de vehículos optimizadas para alta frecuencia y consultas analíticas.

#### Imagen y Versión

```yaml
image: timescale/timescaledb:2.14.0-pg15
```

**Justificación**: Versión estable con PostgreSQL 15 (soporte de características modernas como GENERATED COLUMNS). Timescale 2.14 incluye optimizaciones para escritura de alta frecuencia e índices adaptivos.

#### Configuración

```yaml
timescaledb:
  image: timescale/timescaledb:2.14.0-pg15
  container_name: fleet-timescaledb
  
  environment:
    # Credenciales
    POSTGRES_USER: ${POSTGRES_USER:-fleet_user}
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-fleet_password}
    POSTGRES_DB: ${POSTGRES_DB:-fleet_db}
    
    # Configuración PostgreSQL para desarrollo local
    # (no optimizado para producción)
    POSTGRES_INITDB_ARGS: "-c shared_preload_libraries=timescaledb"
    
    # Logging
    POSTGRES_LOG_STATEMENT: "mod"  # Loguea ALTER/CREATE/DROP pero no SELECT
  
  ports:
    # Mapeo configurable para acceso desde host
    - "${TIMESCALEDB_PORT:-5432}:5432"
  
  volumes:
    # Datos de base de datos
    - timescaledb-data:/var/lib/postgresql/data
    
    # Script de inicialización (SQL a ejecutar en primer start)
    - ./backend/go/scripts/init-timescaledb.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
  
  networks:
    - fleet-network
  
  restart: unless-stopped
  
  healthcheck:
    # Conectar a base de datos y ejecutar SELECT 1
    test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-fleet_user} -d ${POSTGRES_DB:-fleet_db}"]
    interval: 10s
    timeout: 3s
    retries: 3
    start_period: 15s
```

#### Diseño de Schema

##### Decisión Arquitectónica: Schema `telemetry`

El schema se organiza en un namespace dedicado `telemetry` por las siguientes razones:

1. **Organización por Bounded Context**: Separa la telemetría de otros dominios futuros (alerts, vehicle_state, maintenance)
2. **Escalabilidad**: Facilita agregar tablas relacionadas (vehicle_metadata, alert_events, geofences)
3. **Permisos Granulares**: Permite controlar acceso por schema
4. **Convención Estándar**: Alinea con prácticas de microservicios y DDD

##### Decisión Arquitectónica: Tabla Única `telemetry.positions`

Se utiliza una **tabla única** en lugar de dividir en `positions` + `metadata`:

**Ventajas**:
- **Sin duplicación**: `captured_at`, `vehicle_id`, `event_id` se almacenan una sola vez
- **Simplicidad en Worker**: 1 INSERT en lugar de 2 transaccionales
- **Sin JOINs para audit**: Toda la información de trazabilidad está en la misma fila
- **Compresión eficiente**: TimescaleDB comprime timestamps y UUIDs sin penalización notable
- **Índices optimizados**: Búsquedas por event_id, trace_id sin navegar múltiples tablas

**Desventajas descartadas**:
- "Redundancia de timestamps": Comprimidos eficientemente, costo marginal ~5%
- "Tabla ancha": 250 bytes/row es aceptable para series de tiempo modernas

##### Decisión Arquitectónica: `captured_at` como Partitioning Column

Se usa `captured_at` (timestamp del dispositivo) en lugar de `received_at` o un `time` sintético:

**Justificación**:
1. **Correctitud Semántica**: Refleja el momento real del evento, no cuándo llegó al servidor
2. **Offline Support**: Soporta captura sin conexión correctamente
3. **Queries Historicos**: "¿Dónde estuvo VH-001 entre 10-11 AM?" usa `captured_at`
4. **Compatibilidad con Domain Event**: `PositionReceivedEvent.capturedAt` mapea directamente

**Riesgo mitigado**: Clock skew de dispositivos se maneja con validación en el boundary (máximo ±15 min).

##### Campos de Timestamps: Latency Tracking

La tabla incluye **4 timestamps** para rastrear latencia end-to-end:

| Timestamp | Origen | Propósito |
|---|---|---|
| `captured_at` | Dispositivo móvil | Momento real del evento |
| `received_at` | Ingestion API | Latencia de red (captured → received) |
| `processed_at` | Worker | Latencia de procesamiento (received → processed) |
| `stored_at` | TimescaleDB trigger | Latencia de persistencia (processed → stored) |

**Ejemplo de análisis**:
```sql
SELECT 
  vehicle_id,
  AVG(received_at - captured_at) AS avg_network_latency,
  AVG(processed_at - received_at) AS avg_processing_latency,
  AVG(stored_at - processed_at) AS avg_storage_latency
FROM telemetry.positions
WHERE captured_at >= NOW() - INTERVAL '1 hour'
GROUP BY vehicle_id;
```

##### Campos de Trazabilidad

| Campo | Tipo | Propósito | Origen |
|---|---|---|---|
| `event_id` | UUID | Identificador único del evento (idempotencia) | Domain Event |
| `trace_id` | UUID | Rastreo de request end-to-end | HTTP Header (X-Trace-ID) |
| `idempotency_key` | UUID | Cliente puede enviar key para retry seguro | HTTP Header (X-Idempotency-Key) |

**Unicidad**: `event_id` tiene constraint UNIQUE para detectar duplicados.

#### Inicialización de Base de Datos

**Archivo**: `backend/go/scripts/init-timescaledb.sql`

```sql
-- ============================================================================
-- TimescaleDB Initialization for Fleet Telemetry
-- ============================================================================
-- This script runs once when the container starts for the first time.
-- Subsequent starts use existing data from timescaledb-data volume.
--
-- Schema Design:
-- - Single table `telemetry.positions` with all fields (no split into metadata)
-- - Partitioned by `captured_at` (device timestamp, not server timestamp)
-- - Includes 4 timestamps for latency tracking (captured, received, processed, stored)
-- - Includes traceability fields (event_id, trace_id, idempotency_key)
-- ============================================================================

-- Create TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- ============================================================================
-- Schema: telemetry (bounded context organization)
-- ============================================================================
CREATE SCHEMA IF NOT EXISTS telemetry;

-- ============================================================================
-- Table: telemetry.positions
-- ============================================================================
-- Single table design with all position data and traceability metadata.
-- Partitioned by `captured_at` (device timestamp).
--
CREATE TABLE IF NOT EXISTS telemetry.positions (
  -- Timestamp (partitioning column, device time)
  captured_at TIMESTAMPTZ NOT NULL,
  
  -- Vehicle identification
  vehicle_id VARCHAR(50) NOT NULL,
  
  -- Geographic coordinates
  latitude NUMERIC(10, 8) NOT NULL,
  longitude NUMERIC(11, 8) NOT NULL,
  accuracy NUMERIC(10, 2),
  
  -- Additional timestamps (latency tracking)
  received_at TIMESTAMPTZ NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL,
  stored_at TIMESTAMPTZ DEFAULT NOW(),
  
  -- Traceability
  event_id UUID NOT NULL UNIQUE,
  trace_id UUID NOT NULL,
  idempotency_key UUID,
  
  -- Primary key: (captured_at, vehicle_id)
  -- Enables efficient lookups and supports TimescaleDB partitioning
  PRIMARY KEY (captured_at, vehicle_id)
);

-- ============================================================================
-- Convert to Hypertable (partitioned by captured_at)
-- ============================================================================
-- Partitions by 1-hour intervals of captured_at.
-- Each partition holds data for 1 hour of device-captured events.
--
SELECT create_hypertable(
  'telemetry.positions',
  by_range('captured_at', INTERVAL '1 hour'),
  if_not_exists => TRUE
);

-- ============================================================================
-- Indexes for Query Performance
-- ============================================================================

-- Index for vehicle history queries: "WHERE vehicle_id = ? ORDER BY captured_at DESC"
CREATE INDEX IF NOT EXISTS idx_vehicle_captured 
  ON telemetry.positions (vehicle_id, captured_at DESC);

-- Index for event lookup by event_id (deduplication, debugging)
CREATE INDEX IF NOT EXISTS idx_event_id 
  ON telemetry.positions (event_id);

-- Index for tracing queries: "WHERE trace_id = ?" (correlate HTTP request → DB row)
CREATE INDEX IF NOT EXISTS idx_trace_id 
  ON telemetry.positions (trace_id);

-- ============================================================================
-- Compression Policy
-- ============================================================================
-- Compress data older than 7 days to save disk space.
-- TimescaleDB compresses timestamps, UUIDs, and coordinates efficiently.
-- Compressed data remains queryable (automatically decompressed on read).
--
SELECT add_compression_policy(
  'telemetry.positions',
  INTERVAL '7 days',
  if_not_exists => TRUE
);

-- ============================================================================
-- Retention Policy
-- ============================================================================
-- Delete data older than 30 days automatically.
-- Configurable for production (e.g., 90 days, 1 year).
--
SELECT add_retention_policy(
  'telemetry.positions',
  INTERVAL '30 days',
  if_not_exists => TRUE
);

-- ============================================================================
-- Grants: Usuario fleet_user
-- ============================================================================
GRANT USAGE ON SCHEMA telemetry TO fleet_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA telemetry TO fleet_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA telemetry TO fleet_user;

-- Grant future tables (if schema evolves)
ALTER DEFAULT PRIVILEGES IN SCHEMA telemetry 
  GRANT ALL PRIVILEGES ON TABLES TO fleet_user;
```

#### Estimaciones de Almacenamiento

##### Tamaño por Fila

```
Campo                 | Tipo           | Bytes
----------------------|----------------|-------
captured_at           | TIMESTAMPTZ    | 8
vehicle_id            | VARCHAR(50)    | ~15 (avg)
latitude              | NUMERIC(10,8)  | ~12
longitude             | NUMERIC(11,8)  | ~12
accuracy              | NUMERIC(10,2)  | ~8
received_at           | TIMESTAMPTZ    | 8
processed_at          | TIMESTAMPTZ    | 8
stored_at             | TIMESTAMPTZ    | 8
event_id              | UUID           | 16
trace_id              | UUID           | 16
idempotency_key       | UUID           | 16 (nullable)
Overhead (header)     |                | ~24
Indexes               |                | ~50 (amortizado)
----------------------|----------------|-------
TOTAL POR FILA        |                | ~250 bytes
```

##### Volumen Estimado

**Escenario MVP**: 100 vehículos, 12 posiciones/minuto

```
Posiciones/minuto:  1,200 (100 × 12)
Posiciones/día:     1,728,000
Posiciones/mes:     51,840,000

Almacenamiento sin compresión:
  51.84M rows × 250 bytes = ~12.96 GB/mes

Almacenamiento con compresión (ratio 3-5x):
  12.96 GB ÷ 4 (avg) = ~3.2 GB/mes

Indices adicionales: ~1 GB/mes

TOTAL ESTIMADO: ~4-5 GB/mes para MVP
```

**Escenario producción**: 1,000 vehículos

```
Posiciones/minuto:  12,000
Posiciones/día:     17,280,000
Posiciones/mes:     518,400,000

Almacenamiento con compresión:
  518.4M rows × 250 bytes ÷ 4 = ~32 GB/mes

Con retention de 30 días: ~32 GB constante
Con retention de 90 días: ~96 GB constante
```

**Conclusión**: El schema es escalable para MVP y producción temprana. Para flotas >5,000 vehículos, considerar continuous aggregates de TimescaleDB.

#### Port Mapping

```yaml
ports:
  - "${TIMESCALEDB_PORT:-5432}:5432"
```

- **Interno**: Contenedor escucha en 5432
- **Host**: Exposición configurable via `.env`
- **Default**: 5432 si `TIMESCALEDB_PORT` no está definido
- **Ejemplo personalizado**: `TIMESCALEDB_PORT=5433` permite ejecutar instancia secundaria en 5433

#### Health Check

Ejecuta `pg_isready -U fleet_user -d fleet_db` que verifica:
1. PostgreSQL está respondiendo en puerto 5432
2. Usuario fleet_user puede conectarse
3. Base de datos fleet_db es accesible

#### Volúmenes

- **timescaledb-data**: Archivos de base de datos PostgreSQL (tablas, índices, logs de transacción)
- **init script**: SQL que se ejecuta UNA VEZ en primer start (creación de schema, hypertable, políticas)

---

## 4. Diseño de Redes

### 4.1 Fleet Network (Bridge Driver)

```yaml
networks:
  fleet-network:
    driver: bridge
    name: fleet-network
```

**Características**:

1. **Driver bridge**: Red privada interna con DNS embebido
2. **Nombre personalizado**: `fleet-network` para claridad
3. **Alcance**: Solo los contenedores conectados a esta red se ven entre sí
4. **Aislamiento**: No interfiere con otras instancias Docker ni con host

### 4.2 Resolución DNS Interna

Dentro de `fleet-network`, los servicios se resuelven automáticamente por nombre:

```
zookeeper:2181     → Zookeeper para Kafka (usado por Kafka, no por aplicaciones)
kafka:9092         → Kafka broker (usado por Ingestion API y Worker)
timescaledb:5432   → PostgreSQL (usado por Worker, opcional desde host)
ingestion:8080     → API de ingesta (usado por Worker para lógica, si aplica)
worker             → Worker service (usado por Ingestion para notificaciones, si aplica)
```

### 4.3 Acceso desde Host

Solo se exponen puertos selectivos:

| Servicio | Puerto Host | Puerto Contenedor | Propósito |
|---|---|---|---|
| Ingestion API | 8080 | 8080 | HTTP requests desde clientes móviles/postman |
| TimescaleDB | ${TIMESCALEDB_PORT:-5432} | 5432 | PostgreSQL clients (psql, pgAdmin, etc.) |
| Zookeeper | ❌ No expuesto | 2181 | Solo interno para Kafka |
| Kafka | ❌ No expuesto | 9092 | Solo interno para aplicaciones |
| Worker | ❌ No expuesto | N/A | No escucha HTTP, procesa Kafka |

---

## 5. Gestión de Volúmenes

### 5.1 Definición de Volúmenes

```yaml
volumes:
  zookeeper-data:
    name: fleet-zookeeper-data
    driver: local
  
  zookeeper-logs:
    name: fleet-zookeeper-logs
    driver: local
  
  kafka-data:
    name: fleet-kafka-data
    driver: local
  
  timescaledb-data:
    name: fleet-timescaledb-data
    driver: local
```

### 5.2 Estrategia de Persistencia

| Operación | Volúmenes | Datos |
|---|---|---|
| `docker compose up` (primera vez) | Creados vacíos | Se inicializan desde scripts |
| `docker compose up` (subsecuente) | Reutilizados | Se recuperan del volumen |
| `docker compose stop` | Persisten | ✓ Datos intactos |
| `docker compose start` | Reutilizados | ✓ Datos intactos |
| `docker compose down` | Persisten | ✓ Datos intactos (seguridad) |
| `docker compose down -v` | Se eliminan | ❌ Todos los datos se pierden |

**Justificación**: Preservar por defecto evita pérdida accidental de datos. Requiere `-v` explícito para limpiar.

### 5.3 Ubicación de Volúmenes en Host

```
/var/lib/docker/volumes/fleet-zookeeper-data/_data/
/var/lib/docker/volumes/fleet-zookeeper-logs/_data/
/var/lib/docker/volumes/fleet-kafka-data/_data/
/var/lib/docker/volumes/fleet-timescaledb-data/_data/
```

(Exacta ubicación depende de Docker Engine, usualmente `/var/lib/docker/volumes/`)

---

## 6. Configuración de Entorno

### 6.1 Variables .env

```bash
# ============================================================================
# Docker Compose Environment Configuration
# ============================================================================
# Copy this file to .env and customize values for your environment.
# Variables listed below are injected into docker-compose.yml via ${VAR_NAME:-default}.

# --------------------------------------------------------------------------
# HTTP API (Ingestion Service)
# --------------------------------------------------------------------------
# Puerto en host para acceder a API HTTP
HTTP_PORT=8080

# Timeouts para lectura/escritura de HTTP
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=30s

# --------------------------------------------------------------------------
# Logging (compartido entre servicios)
# --------------------------------------------------------------------------
# Nivel de log: debug, info, warn, error
LOG_LEVEL=info

# Formato de log: json o text
LOG_FORMAT=json

# --------------------------------------------------------------------------
# Kafka / Worker Configuration
# --------------------------------------------------------------------------
# Timeout de shutdown para worker (consumidor)
WORKER_SHUTDOWN_TIMEOUT=30s

# --------------------------------------------------------------------------
# TimescaleDB / PostgreSQL
# --------------------------------------------------------------------------
# Usuario administrador (debe ser `fleet_user` para aplicaciones)
POSTGRES_USER=fleet_user

# Contraseña de usuario (CAMBIAR EN PRODUCCIÓN)
POSTGRES_PASSWORD=fleet_password

# Nombre de base de datos
POSTGRES_DB=fleet_db

# Puerto del host para acceder a TimescaleDB
# Por defecto 5432, pero puede ser 5433, 5434, etc. si hay conflicto
TIMESCALEDB_PORT=5432
```

### 6.2 .env.example

Archivo de plantilla que se incluye en el repositorio (NO contiene credenciales reales):

```bash
# Copy this file to .env and customize
# ============================================================================

# HTTP API
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Worker
WORKER_SHUTDOWN_TIMEOUT=30s

# TimescaleDB
POSTGRES_USER=fleet_user
POSTGRES_PASSWORD=changeme
POSTGRES_DB=fleet_db
TIMESCALEDB_PORT=5432
```

### 6.3 Precedencia de Variables

1. Variables ya definidas en shell: `export TIMESCALEDB_PORT=5433`
2. Archivo `.env` en directorio raíz del proyecto
3. Valores por defecto en `docker-compose.yml`: `${VAR_NAME:-default}`

**Ejemplo**:

```bash
# Sin .env y sin export:
TIMESCALEDB_PORT=5432  # (default)

# Con .env TIMESCALEDB_PORT=5433:
TIMESCALEDB_PORT=5433

# Con export y .env:
export TIMESCALEDB_PORT=5434  # (shell override wins)
TIMESCALEDB_PORT=5434
```

---

## 7. Estructura de docker-compose.yml

### 7.1 Organización General

```yaml
version: '3.9'

# ============================================================================
# SERVICES
# ============================================================================
services:
  # Coordinador para Kafka
  zookeeper:
    # ... (config detallada en sección 3.1)
  
  # Bus de eventos desacoplado
  kafka:
    # ... (config detallada en sección 3.2)
  
  # Persistencia de series de tiempo
  timescaledb:
    # ... (config detallada en sección 3.3)
  
  # Servicio de ingesta (EXISTENTE)
  ingestion:
    # ... (sin cambios, pero se conecta a kafka)
  
  # Servicio worker (EXISTENTE)
  worker:
    # ... (sin cambios, consume de kafka, escribe en timescaledb)

# ============================================================================
# NETWORKS
# ============================================================================
networks:
  fleet-network:
    driver: bridge
    name: fleet-network

# ============================================================================
# VOLUMES
# ============================================================================
volumes:
  zookeeper-data:
    name: fleet-zookeeper-data
  zookeeper-logs:
    name: fleet-zookeeper-logs
  kafka-data:
    name: fleet-kafka-data
  timescaledb-data:
    name: fleet-timescaledb-data
```

### 7.2 Comentarios y Documentación

Cada servicio incluye:

```yaml
  # ========================================================================
  # SERVICE NAME: Description of purpose
  # ========================================================================
  # Purpose: What this service does and why it's needed
  # Image: Docker image and version
  # Network: Which networks it connects to
  # Ports: External access configuration
  # Volumes: What data persists
  # Dependencies: What other services it requires
  #
  service-name:
    image: ...
    # Detailed comments explaining each environment variable
    environment:
      VAR_NAME: explanation  # What this controls
    # etc.
```

---

## 8. Diseño del Makefile

**Ubicación**: `/home/developer/Projects/fleet-dashboard/Makefile` (raíz del proyecto)

### 8.1 Targets Principales

```makefile
# ============================================================================
# Fleet Platform - Local Development Orchestration
# ============================================================================
# Makefile provides convenient commands for managing Docker Compose stack
# from project root. All targets are PHONY (not file-based).
#
# Usage: make [target]
#        make help          (show all targets)
#        make up            (start stack in background)
#        make down          (stop stack)
#        make logs          (stream logs)
# ============================================================================

.PHONY: help up down logs logs-timescaledb logs-kafka status clean clean-hard restart build

# Default target when 'make' is called without args
.DEFAULT_GOAL := help

# ============================================================================
# HELP
# ============================================================================
help:
	@echo "Fleet Platform - Docker Compose Commands"
	@echo "=========================================="
	@echo ""
	@echo "Startup & Management:"
	@echo "  make up              - Start infrastructure in background"
	@echo "  make down            - Stop all services"
	@echo "  make restart         - Restart all services (no recreation)"
	@echo "  make build           - Rebuild Docker images"
	@echo ""
	@echo "Inspection:"
	@echo "  make status          - Show status of all services"
	@echo "  make logs            - Stream logs from all services"
	@echo "  make logs-kafka      - Stream logs from Kafka only"
	@echo "  make logs-timescaledb - Stream logs from TimescaleDB only"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean           - Stop and remove containers (preserve volumes)"
	@echo "  make clean-hard      - Remove everything including volumes and prune unused"
	@echo ""

# ============================================================================
# STARTUP & MANAGEMENT
# ============================================================================

up:
	@echo "Starting Fleet Platform infrastructure..."
	docker compose up -d
	@echo "✓ Stack started. Services will be healthy in 20-30 seconds."
	@echo "✓ Ingestion API: http://localhost:8080"
	@echo "✓ TimescaleDB: localhost:$$(grep TIMESCALEDB_PORT .env 2>/dev/null || echo 5432)"

down:
	@echo "Stopping Fleet Platform infrastructure..."
	docker compose down
	@echo "✓ Services stopped. Volumes preserved for next start."

restart:
	@echo "Restarting all services..."
	docker compose restart
	@echo "✓ Services restarted."

build:
	@echo "Building Docker images..."
	docker compose build
	@echo "✓ Build complete."

# ============================================================================
# INSPECTION
# ============================================================================

status:
	@echo "Service Status:"
	@echo "==============="
	docker compose ps

logs:
	@echo "Streaming logs from all services (Ctrl+C to exit)..."
	docker compose logs -f

logs-kafka:
	@echo "Streaming logs from Kafka (Ctrl+C to exit)..."
	docker compose logs -f kafka

logs-timescaledb:
	@echo "Streaming logs from TimescaleDB (Ctrl+C to exit)..."
	docker compose logs -f timescaledb

# ============================================================================
# CLEANUP
# ============================================================================

clean:
	@echo "Stopping and removing containers (volumes preserved)..."
	docker compose down
	@echo "✓ Cleanup complete. Use 'make clean-hard' to remove volumes."

clean-hard:
	@echo "WARNING: This will remove all containers, volumes, and prune unused resources."
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Removing everything..."; \
		docker compose down -v; \
		docker system prune -f; \
		echo "✓ Hard cleanup complete. All data removed."; \
	else \
		echo "Aborted."; \
	fi
```

### 8.2 .PHONY Declaration

```makefile
.PHONY: help up down logs logs-timescaledb logs-kafka status clean clean-hard restart build
```

Esto previene conflictos si existen archivos con estos nombres (ej: `up`, `down`, `build`).

---

## 9. Estrategia de Inicialización

### 9.1 Secuencia de Startup

```
docker compose up -d
  │
  ├─→ [1] Create networks
  │        └─→ fleet-network (bridge)
  │
  ├─→ [2] Create volumes (if not exist)
  │        ├─→ zookeeper-data
  │        ├─→ zookeeper-logs
  │        ├─→ kafka-data
  │        └─→ timescaledb-data
  │
  ├─→ [3] Start Zookeeper container
  │        ├─→ Pull image (if not cached)
  │        ├─→ Mount volumes
  │        ├─→ Start process: /etc/confluent/docker/run
  │        ├─→ Wait for health check (ruok → imok)
  │        └─→ Status: healthy ✓
  │
  ├─→ [4] Start Kafka container
  │        ├─→ Pull image (if not cached)
  │        ├─→ Mount volumes
  │        ├─→ Wait for Zookeeper healthy (depends_on: condition)
  │        ├─→ Connect to zookeeper:2181 for metadata
  │        ├─→ Start broker process: /etc/confluent/docker/run
  │        ├─→ Wait for health check (broker-api-versions)
  │        ├─→ Run init script: create-kafka-topic.sh
  │        │   └─→ Create topic: telemetry.positions (3 partitions, RF=1)
  │        └─→ Status: healthy ✓
  │
  ├─→ [5] Start TimescaleDB container
  │        ├─→ Pull image (if not cached)
  │        ├─→ Mount volumes
  │        ├─→ Initialize: run init-timescaledb.sql (first time only)
  │        │   ├─→ Create extension timescaledb
  │        │   ├─→ Create hypertable: positions
  │        │   ├─→ Add compression policy
  │        │   ├─→ Add retention policy
  │        │   └─→ Grant privileges to fleet_user
  │        ├─→ Start postgres process
  │        ├─→ Wait for health check (pg_isready)
  │        └─→ Status: healthy ✓
  │
  ├─→ [6] Start Ingestion API container
  │        ├─→ Pull image (if not cached)
  │        ├─→ Start HTTP server on :8080
  │        ├─→ Connect to kafka:9092 (for publishing)
  │        └─→ Status: healthy (health check /health)
  │
  └─→ [7] Start Worker container
           ├─→ Pull image (if not cached)
           ├─→ Connect to kafka:9092 (for consuming)
           ├─→ Connect to timescaledb:5432 (for persisting)
           └─→ Ready to process events
```

**Timeline Estimado**:
- Primero start (con descargas): ~5-10 minutos
- Subsecuente start (imágenes en caché): ~20-30 segundos

### 9.2 Garantías de Ordenamiento

Usando `depends_on` con condiciones `condition: service_healthy`:

```yaml
kafka:
  depends_on:
    zookeeper:
      condition: service_healthy
  # Kafka NO INICIA hasta que Zookeeper esté healthy

worker:
  depends_on:
    kafka:
      condition: service_healthy
    timescaledb:
      condition: service_healthy
    ingestion:
      condition: service_healthy
  # Worker NO INICIA hasta que todos sean healthy
```

**Beneficio**: Evita errores por conexiones a servicios que aún no están listos.

---

## 10. Health Checks en Detalle

### 10.1 Zookeeper Health Check

```yaml
healthcheck:
  test: ["CMD", "echo", "ruok", "|", "nc", "-q", "1", "localhost", "2181"]
  interval: 10s       # Ejecutar cada 10 segundos
  timeout: 3s         # Fallar si tarda más de 3s
  retries: 3          # Fallar después de 3 fallos
  start_period: 5s    # Gracia de 5s antes de comenzar checks
```

**Lógica**:
1. Envía comando "ruok" al puerto 2181
2. Si Zookeeper responde "imok", es healthy
3. Falla si no responde en 3s
4. Después de 3 fallos consecutivos, estado unhealthy

### 10.2 Kafka Health Check

```yaml
healthcheck:
  test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server=localhost:9092"]
  interval: 10s
  timeout: 3s
  retries: 3
  start_period: 10s   # Más tiempo que Zookeeper para inicialización
```

**Lógica**:
1. Llama API de Kafka que devuelve versiones de protocolo
2. Si responde, broker está operativo
3. Más tolerante en start_period porque tarda más en iniciar

### 10.3 TimescaleDB Health Check

```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-fleet_user} -d ${POSTGRES_DB:-fleet_db}"]
  interval: 10s
  timeout: 3s
  retries: 3
  start_period: 15s   # Máximo tiempo para inicialización SQL
```

**Lógica**:
1. Ejecuta `pg_isready` para verificar conectividad PostgreSQL
2. Comprueba que usuario fleet_user puede conectarse
3. Comprueba que base de datos fleet_db es accesible
4. Mayor start_period para permitir inicialización de schema

---

## 11. Gestión de Limpieza y Datos

### 11.1 Comandos de Limpieza

```bash
# Detener servicios (preservar datos)
make down
# Equivalente: docker compose down

# Detener y eliminar volúmenes (DESTRUCTIVO)
make clean-hard
# Equivalente: docker compose down -v && docker system prune -f
```

### 11.2 Impacto en Datos

| Comando | Contenedores | Volúmenes | Red | Resultado |
|---|---|---|---|---|
| `make up` | Crean | Crean (vacío) | Crea | Fresh stack |
| `make up` (subsecuente) | Reutilizan | Reutilizan ✓ | Reutiliza | Datos recuperados |
| `make down` | Remover | **Preservan** ✓ | Remover | Limpio, datos seguros |
| `make clean-hard` | Remover | **Remover** ❌ | Remover | Slate completamente limpio |

### 11.3 Volumen Persistencia

**Cómo Docker Compose gestiona volúmenes**:

1. **`docker compose down`** (sin `-v`):
   - Detiene contenedores
   - Remueve contenedores
   - Remueve red
   - **PRESERVA volúmenes** (default safe behavior)

2. **`docker compose down -v`**:
   - Detiene contenedores
   - Remueve contenedores
   - Remueve red
   - **REMUEVE volúmenes** (irreversible)

3. **`docker system prune -f`**:
   - Limpia imágenes no usadas
   - Limpia contenedores detenidos
   - Limpia redes no usadas
   - NO toca volúmenes (seguridad)

---

## 12. Estrategia de Mapeo de Puertos

### 12.1 Puerto Mapping

| Servicio | Interno | Host | Expuesto | Razón |
|---|---|---|---|---|
| **Zookeeper** | 2181 | ❌ | No | Solo para Kafka (DNS internal) |
| **Kafka** | 9092 | ❌ | No | Solo para Ingestion/Worker (DNS internal) |
| **TimescaleDB** | 5432 | ✓ | ${TIMESCALEDB_PORT:-5432} | Acceso desde host (psql, pgAdmin) |
| **Ingestion** | 8080 | ✓ | 8080 | Acceso desde móvil/postman |
| **Worker** | N/A | ❌ | No | Sin interfaz HTTP (event-driven) |

### 12.2 Resolución de Conflictos de Puertos

Si puerto ya está ocupado, Docker Compose falla con:

```
ERROR: for timescaledb  Cannot start service timescaledb: 
Bind for 0.0.0.0:5432 failed: port is already allocated
```

**Solución**: Cambiar `TIMESCALEDB_PORT` en `.env`:

```bash
# En .env
TIMESCALEDB_PORT=5433

# Luego:
make up
# TimescaleDB estará en localhost:5433
```

---

## 13. Configuración de Logging

### 13.1 Fuentes de Logs

Cada servicio genera logs que se capturan en Docker:

```
Zookeeper    → /var/lib/zookeeper/log/* (dentro del contenedor)
Kafka        → stdout de broker (capturado por Docker)
TimescaleDB  → /var/log/postgresql/* (dentro del contenedor)
Ingestion    → LOG_FORMAT (json o text según .env)
Worker       → LOG_FORMAT (json o text según .env)
```

### 13.2 Acceso a Logs

**Vía Docker Compose**:

```bash
# Ver todos los logs
make logs

# Ver solo TimescaleDB
make logs-timescaledb

# Ver últimas 50 líneas
docker compose logs --tail 50

# Ver logs históricos (sin -f)
docker compose logs
```

**Dentro del contenedor**:

```bash
# Conectar interactivamente
docker exec -it fleet-timescaledb psql -U fleet_user -d fleet_db

# Ver logs PostgreSQL
docker exec fleet-timescaledb tail -f /var/log/postgresql/postgresql.log
```

### 13.3 Configuración de Niveles

**Variables .env**:

```bash
LOG_LEVEL=info        # debug, info, warn, error
LOG_FORMAT=json       # json o text

# Kafka (interno)
KAFKA_LOG4J_ROOT_LOGLEVEL=WARN

# PostgreSQL (internal)
POSTGRES_LOG_STATEMENT=mod   # ddl, mod, all, none
```

---

## 14. Consideraciones de Seguridad

### 14.1 Credenciales TimescaleDB

**Ubicación**: `.env` (NO en repositorio)

```bash
# .env (local only, no commitear)
POSTGRES_USER=fleet_user
POSTGRES_PASSWORD=fleet_password

# .env.example (plantilla pública)
POSTGRES_USER=fleet_user
POSTGRES_PASSWORD=changeme
```

**.gitignore**:

```
.env
.env.local
.env.*.local
```

### 14.2 Ejecución de Procesos

Todos los servicios ejecutan como usuarios no-root:

```
Zookeeper    → usuario: appuser (en imagen confluentinc)
Kafka        → usuario: appuser (en imagen confluentinc)
TimescaleDB  → usuario: postgres (en imagen timescale)
```

Verificación:

```bash
docker exec fleet-zookeeper whoami
# appuser

docker exec fleet-timescaledb whoami
# postgres
```

### 14.3 Aislamiento de Red

Todos los servicios conectan a `fleet-network` (bridge privada):

- No accesible desde otras redes Docker
- No accesible desde host a menos que puerto esté explícitamente expuesto
- Tráfico interno es local a Docker (no transita por red externa)

---

## 15. Puntos de Integración

### 15.1 Conexiones de Servicios

**Ingestion API** → Kafka:

```go
// Backend connect string en código
kafkaProducers := []string{"kafka:9092"}  // Via DNS fleet-network
```

**Worker** → Kafka:

```go
// Consumer connect string
kafkaBootstrap := []string{"kafka:9092"}  // Via DNS fleet-network
```

**Worker** → TimescaleDB:

```go
// PostgreSQL connection string
pgConnectionString := "postgres://fleet_user:fleet_password@timescaledb:5432/fleet_db?search_path=telemetry"
// Or via environment: DATABASE_URL
```

### 15.2 No Breaking Changes

El docker-compose.yml existente se extiende (no se modifica):

- Servicios `ingestion` y `worker` continúan igual
- Nueva red `fleet-network` (todos conectan automáticamente si no tienen `networks`)
- Nuevos servicios (zookeeper, kafka, timescaledb) se agregan sin afectar existentes

### 15.3 Flujo de Mensajes

```
1. POST /positions (cliente → Ingestion)
2. ValidatePosition usecase
3. Publish PositionReceivedEvent → Kafka topic: telemetry.positions
4. Worker consume del topic
5. ProcessPosition usecase
6. INSERT into telemetry.positions (TimescaleDB)
7. Dashboard query SELECT FROM telemetry.positions (optional, via localhost:${TIMESCALEDB_PORT})
```

---

## 16. Decisiones Técnicas Clave

### 16.1 ¿Por qué confluentinc/cp-kafka y cp-zookeeper?

**Ventajas**:
- Versiones certificadas y mantenidas por Confluent
- Compatibilidad garantizada entre versiones
- Incluyen herramientas administrativas (`kafka-topics`, `kafka-broker-api-versions`)
- Ampliamente adoptadas en industria
- Excelente documentación

**Alternativas consideradas**:
- `apache/kafka`: Más minimal, pero requiere Zookeeper separado; confluentinc es mejor integrado
- Imágenes personalizadas: Mayor control pero más mantenimiento

### 16.2 ¿Por qué timescale/timescaledb:2.14.0-pg15?

**Ventajas**:
- PostgreSQL 15: Soporte moderno, GENERATED COLUMNS, JSON mejorado
- TimescaleDB 2.14: Optimizaciones de escritura, índices adaptivos, compresión automática
- Producción-ready para MVP
- Imagen oficial mantenida por Timescale

**Alternativas consideradas**:
- `postgres:15`: Sin TimescaleDB; requeriría instalación manual
- `timescale/timescaledb:latest`: Menos predecible, pueden haber breaking changes

### 16.3 ¿Topic creation via script en lugar de init container?

**Decisión**: Script separado (`create-kafka-topic.sh`)

**Ventajas**:
- Docker Compose YAML más limpio
- Script reutilizable en producción
- Idempotente (crea si no existe, else se ignora)
- Fácil de debugguear

**Alternativa**: Init container adicional

```yaml
# No hacer esto (complejidad):
init-kafka-topics:
  image: confluentinc/cp-kafka:7.6.0
  depends_on:
    kafka:
      condition: service_healthy
  command: |
    kafka-topics --create
    --bootstrap-server kafka:9092
    --topic telemetry.positions
    --partitions 3
    --replication-factor 1
    --if-not-exists
```

Rechazado porque: requiere servicio extra, más lógica en compose, más difícil de mantener.

### 16.4 ¿Health check con retraso de startup_period?

Cada servicio tiene `start_period` diferente:

- **Zookeeper**: 5s (fast startup)
- **Kafka**: 10s (tarda más, conecta a Zookeeper)
- **TimescaleDB**: 15s (ejecuta SQL init, tarda más)

**Justificación**: Evita falsos negativos durante inicialización. Sin `start_period`, health check podría fallar antes de que el servicio esté completamente listo.

### 16.5 ¿Makefile en raíz vs makefile por servicio?

**Decisión**: Makefile único en raíz

**Justificación**:
- Desarrolladores ejecutan `make up` desde cualquier directorio
- Orquestación centralizada y clara
- Fácil de descubrir (`make help`)
- Convención estándar

**Alternativa**: Makefiles por servicio (backend/Makefile, frontend/Makefile)

Rechazada porque: requiere ejecutar `cd backend && make up`, menos convenient.

---

## 17. Testing Strategy (Correctness Properties)

**NOTA**: Esta feature es de infraestructura (IaC - docker-compose configuration). No es aplicable property-based testing.

**Estrategia de pruebas**:

1. **Smoke Tests** (verificación manual básica):
   ```bash
   make up
   # Esperar 30 segundos
   docker compose ps | grep -E "Up|healthy"
   # Todos deben estar running/healthy
   ```

2. **Integration Tests** (verificar comunicación):
   ```bash
   # Kafka conecta a Zookeeper
   docker exec fleet-kafka kafka-broker-api-versions --bootstrap-server localhost:9092
   
   # TimescaleDB es accesible
   psql -h localhost -p 5432 -U fleet_user -d fleet_db -c "SELECT 1"
   
   # Topic existe
   docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092 | grep telemetry.positions
   ```

3. **Configuration Validation** (verificar variables):
   ```bash
   docker compose config | grep -E "timescale.*5432|kafka.*9092"
   ```

4. **Manual Tests** (demo de flujo):
   ```bash
   # Publicar evento a Ingestion
   curl -X POST http://localhost:8080/positions \
     -H "Content-Type: application/json" \
     -d '{"vehicleId":"VH-001","latitude":40.0,"longitude":-74.0}'
   
   # Verificar en TimescaleDB
   psql -h localhost -p 5432 -U fleet_user -d fleet_db \
     -c "SELECT * FROM positions ORDER BY time DESC LIMIT 1"
   ```

---

## 18. Archivos de Configuración

### 18.1 Árbol de Archivos

```
fleet-dashboard/
├── docker-compose.yml           (MODIFICAR: agregar servicios)
├── .env.example                 (NUEVO: plantilla de variables)
├── Makefile                     (NUEVO: comandos de orquestación)
├── backend/go/
│   └── scripts/
│       ├── create-kafka-topic.sh    (NUEVO: inicialización de topic)
│       └── init-timescaledb.sql     (NUEVO: inicialización de DB)
└── .gitignore                   (ACTUALIZAR: agregar .env)
```

### 18.2 .gitignore

Agregar/actualizar:

```
# Environment
.env
.env.local
.env.*.local

# Docker
.docker-compose.override.yml
```

---

## 19. Matriz de Requisitos → Diseño

| Requisito | Sección de Diseño | Implementación |
|---|---|---|
| RF-01 Zookeeper | 3.1 | Imagen cp-zookeeper, health check |
| RF-02 Kafka + Topic | 3.2 | Imagen cp-kafka, script create-topic |
| RF-03 Health checks | 10 | Healthcheck per servicio con timeouts |
| RF-04 Fleet-network | 4 | Network bridge con DNS |
| RF-05 TimescaleDB | 3.3 | Imagen timescaledb, init SQL |
| RF-06 Port mapping | 12 | Port 5432 → ${TIMESCALEDB_PORT} |
| RF-07 Volúmenes | 5 | 4 volúmenes definidos, estrategia |
| RF-08 .env | 6 | Variables con defaults |
| RF-09 .env.example | 6.2 | Plantilla pública |
| RF-10 docker-compose comments | 7 | Documentación inline |
| RF-11 Startup order | 9, 10 | depends_on con conditions |
| RF-12 Restart policies | 3.1-3.3 | unless-stopped en todos |
| RF-13 Logs | 13 | Acceso via make logs-* |
| RF-14 Limpieza | 11 | make clean, make clean-hard |
| RF-15 Multiplataforma | 7, 12 | API version 3.9 |
| RF-16 Seguridad | 14 | Credenciales en .env, no-root |
| RF-17 Makefile | 8 | Makefile con targets |

---

## 20. Restricciones y Limitaciones

### 20.1 Entorno de Desarrollo Local Solamente

```yaml
# NO adecuado para producción:
KAFKA_DEFAULT_REPLICATION_FACTOR: 1  # Single broker, no redundancia
KAFKA_MIN_INSYNC_REPLICAS: 1         # No confirmación distribuida
POSTGRES_LOG_STATEMENT: mod          # Logging excesivo
KAFKA_HEAP_OPTS: -Xmx512M            # RAM limitada
```

Para producción, estos valores deben ajustarse (múltiples brokers, replicación, etc.).

### 20.2 Single Node Cluster

Kafka con 1 broker es vulnerable:

- Si broker falla, no hay replicación a otro broker
- Datos se pierden si volumen `kafka-data` se corrompe

**Mitigación para desarrollo**: Volumes persistentes + restart policies.

### 20.3 Recursos de Host

Stack requiere aproximadamente:

- **CPU**: 2-4 cores
- **RAM**: 3-4 GB (Zookeeper ~256MB, Kafka ~512MB, TimescaleDB ~1GB, app ~512MB)
- **Disco**: 5-10 GB (volúmenes vacíos inicialmente)

---

## 21. Próximos Pasos (Tasks)

Este documento proporciona base para las siguientes tareas:

1. **Task 1**: Crear/actualizar `docker-compose.yml` con servicios nuevos
2. **Task 2**: Crear `.env.example` en raíz
3. **Task 3**: Crear `Makefile` en raíz
4. **Task 4**: Crear scripts: `create-kafka-topic.sh`, `init-timescaledb.sql`
5. **Task 5**: Actualizar `.gitignore` para `.env`
6. **Task 6**: Verificar integración con servicios existentes (ingestion, worker)
7. **Task 7**: Escribir tests de integración (Go, verificar conectividad)
8. **Task 8**: Documentar en README: cómo usar docker-compose

---

## 22. Referencias y Estándares Aplicados

- **Docker Compose Spec**: Version 3.9 (https://docs.docker.com/compose/compose-file/compose-file-v3/)
- **Confluent Docker Images**: Documentation (https://docs.confluent.io/kafka-docker/current/)
- **TimescaleDB Docs**: Docker setup (https://docs.timescale.com/self-hosted/latest/install/docker/)
- **Health Checks**: Docker Healthcheck spec
- **Clean Architecture**: Aplicado a infraestructura (separación de concerns)
- **Estándares del proyecto**: `tech.md`, `testing.md`, `lenguaje.md`

