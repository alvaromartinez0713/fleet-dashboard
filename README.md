# Fleet Platform - Portal Corporativo de Monitoreo de Flotas

Sistema de monitoreo de flotas en tiempo real que captura, procesa y visualiza telemetría de vehículos con capacidades offline-first y análisis operativo mediante IA.

## Descripción General

El Portal de Monitoreo de Flotas es un MVP funcional que integra:

- **Captura móvil** de coordenadas con persistencia local y sincronización por lotes
- **Procesamiento asíncrono** mediante Kafka para desacoplar ingesta y persistencia
- **Almacenamiento optimizado** en TimescaleDB para series temporales de alta frecuencia
- **Dashboard web reactivo** con visualización de mapa y alertas en tiempo real
- **Agente operativo de IA** para consultas en lenguaje natural

## Arquitectura

### Descripción General

La Plataforma de Flotas consta de servicios de infraestructura y servicios de aplicación que trabajan en conjunto para capturar, procesar y visualizar telemetría de flotas en tiempo real. La arquitectura implementa Clean Architecture con una clara separación entre dominio, aplicación, adaptadores e infraestructura, permitiendo que cada componente sea independiente, verificable y mantenible.

**Principios arquitectónicos:**
- **Desacoplamiento**: Kafka separa la ingesta del procesamiento
- **Idempotencia**: El worker procesa eventos de forma idempotente, tolerando duplicados
- **Resiliencia**: Circuit Breaker y timeouts en comunicaciones remotas
- **Observabilidad**: Trazas distribuidas (trace_id) y logging estructurado

### Servicios de Infraestructura

#### **Zookeeper** (`confluentinc/cp-zookeeper:7.6.0`)

- **Propósito**: Coordinador de metadatos para la gestión de clusters Kafka
- **Responsabilidades**: 
  - Coordinación de brokers Kafka
  - Registro de brokers y asignación de particiones
  - Gestión de state del cluster
- **Puerto**: 2181 (solo interno, no expuesto al host)
- **Volúmenes**:
  - `zookeeper-data`: Almacenamiento de snapshots (estado del cluster)
  - `zookeeper-logs`: Transaction logs para durabilidad
- **Requisitos de salud**: Responde a comando `ruok` con `imok`

#### **Kafka** (`confluentinc/cp-kafka:7.6.0`)

- **Propósito**: Bus de eventos para desacoplamiento entre ingesta y procesamiento de telemetría
- **Responsabilidades**:
  - Recibir eventos de múltiples fuentes (Ingestion API)
  - Mantener orden de eventos por vehículo (particionamiento)
  - Distribuir eventos a consumidores (Worker)
- **Topic**: `telemetry.positions`
  - **Particiones**: 3 (permite paralelismo en procesamiento)
  - **Particionamiento**: Por `vehicle_id` para garantizar orden de eventos por vehículo
  - **Replication Factor**: 1 (MVP - sin replicación)
  - **Retención**: 24 horas (configurable vía `KAFKA_LOG_RETENTION_HOURS`)
- **Puerto**: 9092 (solo interno, no expuesto al host)
- **Volúmenes**:
  - `kafka-data`: Almacenamiento de particiones, índices y logs del broker
- **Configuración**:
  - Compresión: No configurada (optimizable para producción)
  - Segment size: 1GB
  - Heap: 512MB (MVP)

#### **TimescaleDB** (`timescale/timescaledb:2.14.0-pg15`)

- **Propósito**: Almacenamiento persistente optimizado para datos de series temporales de alta frecuencia
- **Responsabilidades**:
  - Persistencia de eventos de posición de vehículos
  - Consultas para dashboard y agente de IA
  - Deduplicación mediante `event_id` (UNIQUE)
- **Schema**: `telemetry`
  - **Hypertable**: `telemetry.positions`
  - **Particionamiento**: Por `captured_at` (timestamp del evento)
  - **Chunk size**: 1 hora (optimizado para consultas de corta duración)
- **Puerto**: ${TIMESCALEDB_PORT} (default: 5433, expuesto al host)
- **Volúmenes**:
  - `timescaledb-data`: Datos de PostgreSQL, índices y archivos de control
- **Características TimescaleDB**:
  - Hypertables: Particionamiento automático por timestamp
  - Compresión: Disponible para datos antiguos (reducción de almacenamiento)
  - Continuous Aggregates: Precomputed views para dashboards (futuro)
  - Automatic data retention: Políticas de limpieza (futuro)

### Servicios de Aplicación

#### **Ingestion API**

- **Propósito**: Endpoint HTTP que recibe telemetría de dispositivos móviles
- **Responsabilidades**:
  - Validar solicitudes (formato, rango de coordenadas, campos obligatorios)
  - Transformar datos HTTP a eventos internos
  - Publicar eventos en Kafka
  - Devolver respuesta al cliente
- **Endpoint**: `POST /positions`
  - **Contrato**: RFC 9457 Problem Details para errores
  - **Validaciones**: 
    - `vehicle_id`: Requerido, varchar(50)
    - `latitude`: Requerido, rango [-90, 90]
    - `longitude`: Requerido, rango [-180, 180]
    - `accuracy`: Opcional, numérico en metros
- **Respuesta**: 
  - `202 Accepted`: Evento aceptado para procesamiento asíncrono
  - `400 Bad Request`: Datos inválidos (incluye detalles de validación)
  - `422 Unprocessable Content`: Datos semánticamente inválidos
  - `429 Too Many Requests`: Rate limit excedido
- **Puerto**: 8080 (expuesto al host)
- **Dependencias**: Kafka (publicación de eventos)
- **Timeouts configurables**:
  - `HTTP_READ_TIMEOUT`: 15s (default)
  - `HTTP_WRITE_TIMEOUT`: 15s (default)
  - `HTTP_SHUTDOWN_TIMEOUT`: 30s (graceful shutdown)

#### **Worker**

- **Propósito**: Consumidor Kafka que procesa eventos de telemetría
- **Responsabilidades**:
  - Consumir eventos del topic `telemetry.positions`
  - Validar estructura y completitud de datos
  - Detectar y descartar duplicados (mediante `event_id`)
  - Persistir eventos en TimescaleDB
  - Confirmar mensaje en Kafka solo después de persistencia
  - Manejar fallos recuperables (reintentos) y no recuperables
- **Garantías de procesamiento**:
  - **Idempotencia**: Procesar el mismo evento N veces produce el mismo estado que procesarlo una vez
  - **Deduplicación**: Duplicados se detectan mediante `event_id` (UNIQUE constraint)
  - **Orden**: Eventos de un vehículo se procesan en orden (particionamiento por `vehicle_id`)
  - **No hay pérdidas**: Offset de Kafka se confirma solo después de persistencia en DB
- **Dependencias**: 
  - Kafka (consumo de eventos)
  - TimescaleDB (persistencia)
- **Configuración**:
  - `KAFKA_GROUP_ID`: `fleet-worker-group` (permite múltiples workers en paralelo)
  - `WORKER_SHUTDOWN_TIMEOUT`: 30s (graceful shutdown)
- **Resiliencia**:
  - Reintentos con backoff exponencial para errores transitorios
  - Circuit Breaker para fallos de TimescaleDB
  - Timeout en operaciones de BD

### Flujo de Datos

```
┌─────────────────┐
│  Mobile App     │
└────────┬────────┘
         │
         │ HTTP POST /positions
         │ {vehicle_id, latitude, longitude, ...}
         ↓
┌────────────────────────────────────────┐
│  Ingestion API (localhost:8080)        │
│                                        │
│  1. Validate request                   │
│  2. Create PositionReceivedEvent       │
│  3. Publish to Kafka                   │
└────────┬───────────────────────────────┘
         │
         │ Publish event (idempotency_key, event_id)
         ↓
┌────────────────────────────────────────┐
│  Kafka Topic: telemetry.positions      │
│                                        │
│  Partitions: 3 (by vehicle_id)         │
│  Retention: 24 hours                   │
│  Replication: 1                        │
└────────┬───────────────────────────────┘
         │
         │ Subscribe to topic
         ↓
┌────────────────────────────────────────┐
│  Worker Service                        │
│                                        │
│  1. Consume from partition             │
│  2. Deserialize event                  │
│  3. Check for duplicate (event_id)     │
│  4. Validate geo coordinates           │
│  5. Persist to TimescaleDB             │
│  6. Commit offset in Kafka             │
└────────┬───────────────────────────────┘
         │
         │ INSERT or UPDATE (on duplicate, skip)
         ↓
┌──────────────────────────────────────────────┐
│  TimescaleDB (localhost:5433)                │
│                                             │
│  Schema: telemetry                          │
│  Table: positions (hypertable)              │
│  Partitioning: by captured_at (1h chunks)   │
│                                             │
│  Columns:                                   │
│  - captured_at (TIMESTAMPTZ, partition key) │
│  - vehicle_id (VARCHAR, PK part)            │
│  - latitude, longitude, accuracy            │
│  - received_at, processed_at, stored_at     │
│  - event_id (UNIQUE, deduplication)         │
│  - trace_id (distributed tracing)           │
└──────────────────────────────────────────────┘
         ↑
         │ SELECT for queries
         │
┌────────┴──────────────┐
│  Dashboard / Queries  │
│  Agente de IA         │
└───────────────────────┘
```

### Topología de Red

Todos los servicios se ejecutan en contenedores Docker conectados a través de un bridge network llamado `fleet-network`:

```
┌──────────────────────────────────────────────────────────┐
│  Host (localhost)                                        │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  fleet-network (bridge)                            │ │
│  │                                                    │ │
│  │  Servicios internos (sin exposición al host):     │ │
│  │  • zookeeper:2181                                 │ │
│  │  • kafka:9092                                     │ │
│  │                                                    │ │
│  │  Servicios expuestos al host:                     │ │
│  │  • ingestion:8080 → localhost:8080                │ │
│  │  • timescaledb:5432 → localhost:5433              │ │
│  │    (puerto configurable vía TIMESCALEDB_PORT)     │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  Comunicación:                                          │
│  - Servicios internos: Por nombre (zookeeper:2181)     │
│  - Desde host: Por localhost y puerto expuesto        │
└──────────────────────────────────────────────────────────┘
```

**Comunicación entre servicios:**
- **Zookeeper ↔ Kafka**: Interno (`zookeeper:2181`)
- **Ingestion API ↔ Kafka**: Interno (`kafka:9092`)
- **Worker ↔ Kafka**: Interno (`kafka:9092`)
- **Worker ↔ TimescaleDB**: Interno (`timescaledb:5432`)
- **Host ↔ Ingestion API**: Puerto 8080 (configurable)
- **Host ↔ TimescaleDB**: Puerto 5433 (configurable vía `TIMESCALEDB_PORT`)

### Consideraciones de Escalabilidad

**MVP (configuración actual):**
- 1 broker Kafka (single-node)
- 1 instancia TimescaleDB (sin replicación)
- 1 instancia Ingestion API (sin load balancer)
- 1 Worker (puede paralelizarse)

**Escalabilidad horizontal:**

**Para Kafka:**
- Aumentar número de brokers (replica factor)
- Aumentar particiones de topic (permite más workers paralelos)
- Particionamiento por `vehicle_id` permite escalar workers manteniendo orden por vehículo

**Para TimescaleDB:**
- Replicación con streaming (PostgreSQL native)
- Read replicas para consultas de dashboard
- Compresión de datos antiguos (hypertable feature)
- Políticas de retención automática

**Para Ingestion API:**
- Load balancer (NGINX, AWS ALB)
- Múltiples réplicas detrás del load balancer
- Rate limiting distribuido (Redis)

**Optimizaciones de costo:**
- Compresión TimescaleDB: Hasta 90% de reducción en datos viejos
- Tiered storage: Datos fríos en S3 (TimescaleDB con extension)
- Log retention policies: 24h por defecto, ajustable
- Kafka log compaction: Mantener solo último estado por clave (futuro)

## Quick Start

Get the Fleet Platform running locally in 5 minutes.

### Prerequisites

- **Docker Engine**: version 24.0 or later
- **Docker Compose**: plugin version 2.0 or later (included with Docker Desktop)
- **Make**: optional, simplifies command execution (available on macOS/Linux; can use `docker compose` directly on Windows)
- **curl**: for verifying service health (included on most systems)

**For development:**
- Go 1.23+ (backend development)
- Node.js 20+ (frontend development, when implemented)

### Getting Started (5 Steps)

**Step 1: Clone the repository**

```bash
git clone <repository-url>
cd fleet-dashboard
```

**Step 2: Copy environment template and customize (optional)**

```bash
cp .env.example .env
```

By default:
- TimescaleDB listens on `localhost:5433` (to avoid conflicts with local PostgreSQL on 5432)
- Ingestion API listens on `localhost:8080`
- Kafka is internal only (accessible via `kafka:9092` from services)

Edit `.env` only if you need different ports or credentials.

**Step 3: Start all services**

```bash
# Using Make (recommended)
make up

# Or directly with Docker Compose
docker compose up -d
```

Services are started in the background. The compose file defines health checks for all services.

**Step 4: Wait for services to be healthy (20-30 seconds)**

```bash
# Check service health
make status

# Or verify each service individually
docker compose ps

# Wait for all services to show as "healthy" or "Up"
```

**Step 5: Verify with a health check**

```bash
# Verify Ingestion API is ready
curl http://localhost:8080/health

# Expected response:
# {"status": "ok"}
```

If all checks pass, you're ready to work with the platform!

### Service Access

Once services are running, you can access them locally:

| Service | URL | Purpose |
|---------|-----|---------|
| **Ingestion API** | `http://localhost:8080` | POST telemetry data here |
| **API Health** | `http://localhost:8080/health` | Check API status |
| **TimescaleDB** | `localhost:${TIMESCALEDB_PORT}` | Database (default: 5433) |
| **Kafka** | `localhost:9092` | Message broker (dev only, internal) |

**Connect to TimescaleDB:**

```bash
# From your host (requires psql)
psql -h localhost -p 5433 -U fleet_user -d fleet_db
# Password: (check POSTGRES_PASSWORD in .env, default: fleet_password)

# Or use Docker to run psql
docker compose exec timescaledb psql -U fleet_user -d fleet_db
```

### Next Steps

**Explore common tasks:**

- 📖 See **[Makefile Commands](#makefile-commands)** section for available operations
- 🏗️ Read **[Architecture](#architecture)** section for system overview
- 🔧 Check **[Troubleshooting](#troubleshooting)** for common issues
- 📝 Review `.kiro/steering/` for project standards

**Verify the flow:**

1. Send a test position to the Ingestion API
2. Check that it appears in TimescaleDB
3. Review logs to understand the data flow

**Stop services when done:**

```bash
# Stop containers (data preserved)
make down

# Or remove everything including volumes (caution: deletes data!)
make clean-hard
```

## Makefile Commands

### Overview

The Makefile at the project root provides convenient commands for managing the Docker Compose stack. All targets simplify common operations without requiring verbose docker compose syntax.

**Windows users**: If Make is not available, you can invoke docker compose commands directly:
```bash
docker compose up -d      # instead of: make up
docker compose down       # instead of: make down
docker compose ps         # instead of: make status
```

Install [GNU Make](https://gnuwin32.sourceforge.net/packages/make.htm) or [Git Bash](https://gitforwindows.org/) to use Make on Windows.

### Available Commands

| Command | Description | Usage Example |
|---------|-------------|---|
| `make help` | Display all available commands | `make help` |
| `make up` | Start infrastructure in background | `make up` |
| `make down` | Stop all services | `make down` |
| `make restart` | Restart all services without recreating containers | `make restart` |
| `make build` | Rebuild Docker images | `make build` |
| `make status` | Show status of all services | `make status` |
| `make logs` | Stream logs from all services (Ctrl+C to exit) | `make logs` |
| `make logs-kafka` | Stream logs from Kafka only | `make logs-kafka` |
| `make logs-timescaledb` | Stream logs from TimescaleDB only | `make logs-timescaledb` |
| `make clean` | Stop and remove containers (preserves volumes) | `make clean` |
| `make clean-hard` | Full cleanup including volumes and pruned images | `make clean-hard` |

### Common Workflows

**1. Start the stack for development**

Get services running and verify they're healthy:

```bash
make up
make status
curl http://localhost:8080/health
```

Services take 20-30 seconds to become healthy. The output shows service endpoints.

**2. Debug a service**

Stream logs from a specific service to diagnose issues:

```bash
make status                # See which service has issues
make logs-timescaledb      # Debug database-specific problems
make logs-kafka            # Debug message broker issues
make logs                  # See all logs at once
```

Press Ctrl+C to stop streaming.

**3. Clean restart after code changes**

Restart services without losing data:

```bash
make down
make up
make status
```

Or restart all services in place:

```bash
make restart
```

**4. Full cleanup and reset**

Remove everything including volumes when you need a clean slate:

```bash
make clean-hard
make up
```

**⚠️ Warning**: `make clean-hard` will delete all data in volumes. Use only when you want to reset completely.

## Configuración de Variables de Entorno

Archivo `.env` (copia de `.env.example`):

### TimescaleDB

- `POSTGRES_USER`: Usuario de PostgreSQL (default: `fleet_user`)
- `POSTGRES_PASSWORD`: Contraseña (default: `fleet_password`)
- `POSTGRES_DB`: Nombre de base de datos (default: `fleet_db`)
- `TIMESCALEDB_PORT`: Puerto de acceso desde host (default: `5433`)

### Kafka

- `KAFKA_BROKERS`: Dirección del broker (default: `kafka:9092`)
- `KAFKA_TOPIC`: Topic de telemetría (default: `telemetry.positions`)
- `KAFKA_GROUP_ID`: Grupo de consumidor del worker (default: `fleet-worker-group`)

### Ingestion API

- `HTTP_PORT`: Puerto de la API (default: `8080`)
- `HTTP_READ_TIMEOUT`: Timeout de lectura (default: `15s`)
- `HTTP_WRITE_TIMEOUT`: Timeout de escritura (default: `15s`)
- `HTTP_SHUTDOWN_TIMEOUT`: Timeout de apagado graceful (default: `30s`)

### Worker

- `WORKER_SHUTDOWN_TIMEOUT`: Timeout de apagado graceful (default: `30s`)
- `DB_HOST`: Host de TimescaleDB (default: `timescaledb`)
- `DB_PORT`: Puerto interno (default: `5432`)

### Logging

- `LOG_LEVEL`: Nivel de logs: `debug`, `info`, `warn`, `error` (default: `info`)
- `LOG_FORMAT`: Formato: `json`, `text` (default: `json`)

## Troubleshooting

### Conflicto de puerto en 5432

Si ya tienes PostgreSQL corriendo localmente en puerto 5432:

```bash
# Editar .env y cambiar puerto
TIMESCALEDB_PORT=5433

# Reiniciar stack
make down
make up
```

### Servicios no healthy

```bash
# Revisar logs para diagnóstico
make logs

# Ver estado detallado
make status

# Si persiste, aumentar recursos de Docker:
# Docker Desktop → Settings → Resources → Memory (mínimo 4GB recomendado)
```

### Reinicio desde cero

```bash
# ADVERTENCIA: Esto eliminará todos los datos
make clean-hard

# Reiniciar limpio
make up
```

### Error "topic ya existe"

El script `create-kafka-topic.sh` es idempotente. Si ves "topic already exists", puedes ignorarlo con seguridad. El script continuará sin errores.

### No se puede conectar a TimescaleDB

```bash
# Verificar que el servicio está healthy
make status

# Si está "Up" pero no "healthy", revisar logs
make logs-timescaledb

# Conexión desde host
psql -h localhost -p 5433 -U fleet_user -d fleet_db
# Contraseña: valor de POSTGRES_PASSWORD en .env (default: fleet_password)
```

## Troubleshooting

### Port Conflict on 5432 (TimescaleDB)

**Problem:**
`docker compose up` fails with an error about port 5432 already in use:
```
bind: address already in use
Error response from daemon: driver failed programming external connectivity on endpoint fleet-timescaledb
```

**Diagnosis:**
Another PostgreSQL instance (local PostgreSQL server or Docker container) is already running on port 5432 or 5433.

**Solution:**

**Option 1: Stop local PostgreSQL service**
```bash
# macOS (with Homebrew)
brew services stop postgresql

# Linux (systemd)
sudo systemctl stop postgresql

# Windows (Services)
# Open Services → Find PostgreSQL → Right-click → Stop
```

**Option 2: Use a different port**

Edit `.env` and specify a different port for TimescaleDB:
```bash
echo "TIMESCALEDB_PORT=5434" >> .env
```

Then restart:
```bash
make down
make up
make status
```

Verify the connection works:
```bash
psql -h localhost -p 5434 -U fleet_user -d fleet_db
```

---

### Services Not Healthy

**Problem:**
`make status` shows services as "Up" but not marked as "healthy", or services crash shortly after startup.

**Diagnosis:**
Services may need more time to initialize, or Docker doesn't have sufficient resources allocated.

**Solution:**

**Step 1: Wait for initialization (20-30 seconds)**
Allow services time to perform initialization tasks (schema creation, topic creation, health checks):
```bash
make status
# Repeat every 5-10 seconds until all services show "healthy"
```

**Step 2: Check logs for errors**
```bash
# View all service logs
make logs

# Or check specific services
make logs-kafka
make logs-timescaledb

# Look for error messages like "ERROR", "Exception", "Failed"
```

**Step 3: Increase Docker resources**
If logs don't show errors and services are just slow, increase Docker's allocation:

- **Docker Desktop (macOS/Windows):**
  - Click Docker icon (top menu bar) → Preferences/Settings
  - Go to Resources tab
  - Increase Memory to at least 4GB (8GB recommended)
  - Increase CPUs to at least 4
  - Click Apply and Restart

- **Verify allocation:**
  ```bash
  docker stats
  # Check that services aren't hitting memory/CPU limits
  ```

**Step 4: If still failing, restart from scratch**
```bash
make clean-hard
make up
make status
```

---

### Topic Already Exists Error

**Problem:**
During startup, you see a warning or error:
```
Topic telemetry.positions already exists
```

**Diagnosis:**
The Kafka topic was created in a previous run. The script `create-kafka-topic.sh` is idempotent (safe to run multiple times).

**Solution:**
This is expected behavior and safe to ignore. The script uses `--if-not-exists` to prevent errors:
```bash
kafka-topics --create --if-not-exists --topic telemetry.positions ...
```

No action needed. The platform will function normally.

---

### "Docker not found" Error

**Problem:**
Running `make up` or any docker command fails with:
```
docker: command not found
```

**Diagnosis:**
Docker Engine is not installed on your system, or it's not in your system PATH.

**Solution:**

**Step 1: Install Docker Desktop**
- Visit: https://www.docker.com/products/docker-desktop/
- Download and install for your operating system (macOS, Windows, or Linux)
- Follow the installation wizard and allow privileged access when prompted

**Step 2: Verify installation**
```bash
# Check Docker Engine
docker --version
# Expected: Docker version 24.0 or later

# Check Docker Compose
docker compose version
# Expected: Docker Compose version 2.0 or later
```

**Step 3: Restart your terminal**
Close and reopen your terminal (or IDE terminal) to reload PATH environment.

**Step 4: Test connectivity**
```bash
docker run hello-world
# Should download and run a test container
```

---

### Cannot Connect to TimescaleDB from Host

**Problem:**
Trying to connect to TimescaleDB fails with:
```
psql: error: could not connect to server: Connection refused
Is the server running on host "localhost" (127.0.0.1) and accepting
TCP/IP connections on port 5433?
```

**Diagnosis:**
- TimescaleDB container is not running
- Port is not correctly mapped from container to host
- Firewall is blocking the connection

**Solution:**

**Step 1: Verify the container is running**
```bash
make status
# or
docker compose ps

# Look for "fleet-timescaledb" row with status "Up" and "(healthy)"
```

If not running:
```bash
make up
make status  # Wait for healthy status
```

**Step 2: Verify port is accessible**
```bash
# macOS/Linux
netstat -an | grep 5433

# Windows (PowerShell)
netstat -an | findstr 5433

# Expected output should show LISTEN on port 5433
```

**Step 3: Check Docker port mapping**
```bash
docker port fleet-timescaledb
# Expected output:
# 5432/tcp -> 127.0.0.1:5433
```

**Step 4: Try connecting directly through Docker**
```bash
# This bypasses network mapping issues
docker compose exec timescaledb psql -U fleet_user -d fleet_db

# If this works but host connection fails, it's a network/firewall issue
# Try connecting to different port or adjust firewall rules
```

---

### Disk Space Issues

**Problem:**
Services fail to start or operations fail with:
```
No space left on device
docker: error response from daemon: mkdir /var/lib/docker/overlay2/...: no space left on device
```

**Diagnosis:**
Docker volumes, images, and containers are consuming significant disk space.

**Solution:**

**Step 1: Clean up old Docker resources**
```bash
# Stop and remove containers, networks (safe)
make clean-hard

# Or use Docker directly
docker system prune -a --volumes
```

**Step 2: Check Docker disk usage**
```bash
docker system df

# Shows:
# IMAGES      (number of images and space used)
# CONTAINERS  (running and stopped containers)
# VOLUMES     (Docker volumes)
# BUILD CACHE (build cache data)
```

**Step 3: Prune specific resources**
```bash
# Remove unused images (not in use by any container)
docker image prune -a

# Remove unused volumes
docker volume prune

# Remove build cache
docker builder prune
```

**Step 4: Check available disk space**
```bash
# macOS/Linux
df -h

# Windows (PowerShell)
Get-Volume

# Aim for at least 10-15GB free space for comfortable development
```

**Step 5: If still low, free up system disk**
- Delete old project files or archives
- Clear browser cache and downloads
- Check Applications folder for unused software
- Consider external storage for Docker data (Docker Desktop → Resources → Advanced → Docker Data Folder)

---

### Unable to Create Schema After Multiple Restarts

**Problem:**
After restarting services multiple times, you see:
```
ERROR: schema "telemetry" already exists
```

**Diagnosis:**
TimescaleDB volume is persisting schema from previous runs. Schema creation script (`.sql` file) is being run again but the schema already exists.

**Solution:**
This is expected and safe. The initialization script uses `CREATE SCHEMA IF NOT EXISTS`:

```sql
CREATE SCHEMA IF NOT EXISTS telemetry;
```

This is idempotent and will not raise an error. The platform will function normally.

**If you want a clean database:**
```bash
# Remove volumes and start fresh
make clean-hard
make up
```

**Note:** This will delete all data stored in TimescaleDB.

---

### Still Need Help?

If the troubleshooting steps above don't resolve your issue:

**Gather diagnostic information:**

```bash
# Review all service logs
docker compose logs

# Check overall service health
docker compose ps

# Inspect a specific service
docker inspect fleet-kafka
docker inspect fleet-timescaledb

# Check Docker system status
docker system info
docker system df
```

**Review project documentation:**

- `.kiro/steering/` — Project standards and architectural decisions
- `.kiro/specs/docker-compose-infrastructure/` — Infrastructure specification
- `Makefile` — Available commands and their definitions

**Check service-specific documentation:**

- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [TimescaleDB Documentation](https://docs.timescale.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

## Estructura del Proyecto

```
fleet-dashboard/
├── .kiro/                      # Specs de Kiro
│   ├── steering/              # Reglas generales del proyecto
│   └── specs/                 # Especificaciones de funcionalidades
├── backend/go/                # Servicios backend en Go
│   ├── cmd/                   # Puntos de entrada
│   │   ├── ingestion/        # API de ingesta
│   │   └── worker/           # Worker de procesamiento
│   ├── internal/             # Código interno (Clean Architecture)
│   ├── scripts/              # Scripts de inicialización
│   │   ├── init-timescaledb.sql
│   │   └── create-kafka-topic.sh
│   └── tests/                # Pruebas
├── frontend/                  # (Futuro) Dashboard web en React
├── mobile/                    # (Futuro) App móvil en React Native
├── infrastructure/           # (Futuro) Infraestructura como código (Terraform)
├── docker-compose.yml        # Configuración de servicios
├── .env.example              # Plantilla de variables de entorno
├── .env                      # Variables locales (no commitear)
├── Makefile                  # Comandos de orquestación
└── README.md                 # Este archivo
```

## Stack Tecnológico

- **Backend**: Go 1.23 con Gin
- **Mensajería**: Apache Kafka (Confluent Platform 7.6.0)
- **Persistencia**: TimescaleDB 2.14 (PostgreSQL 15 + extensión)
- **Frontend**: React + TypeScript (futuro)
- **Móvil**: React Native (futuro)
- **Infraestructura**: Docker Compose (local), Terraform (producción, futuro)
- **CI/CD**: GitHub Actions (futuro)

## Base de Datos: Schema de TimescaleDB

### Tabla: `telemetry.positions`

Hypertable particionada por `captured_at` (chunks de 1 hora):

| Columna | Tipo | Descripción |
|---------|------|-------------|
| `captured_at` | `TIMESTAMPTZ` | Timestamp del dispositivo (partitioning key) |
| `vehicle_id` | `VARCHAR(50)` | Identificador del vehículo (parte de PK) |
| `latitude` | `NUMERIC(10,8)` | Latitud (-90 a 90) |
| `longitude` | `NUMERIC(11,8)` | Longitud (-180 a 180) |
| `accuracy` | `NUMERIC(10,2)` | Precisión GPS en metros |
| `received_at` | `TIMESTAMPTZ` | Timestamp de recepción en API |
| `processed_at` | `TIMESTAMPTZ` | Timestamp de procesamiento en worker |
| `stored_at` | `TIMESTAMPTZ` | Timestamp de almacenamiento en DB |
| `event_id` | `UUID` | Identificador único del evento (UNIQUE) |
| `trace_id` | `UUID` | ID de rastreo distribuido |
| `idempotency_key` | `UUID` | Clave de idempotencia del cliente |

**Primary Key**: `(captured_at, vehicle_id)`

**Índices**:
- `idx_positions_vehicle_id` on `(vehicle_id, captured_at DESC)`
- `idx_positions_event_id` on `(event_id)` — para deduplicación
- `idx_positions_trace_id` on `(trace_id)` — para rastreo distribuido

### Conectar a TimescaleDB

```bash
# Desde el host
psql -h localhost -p 5433 -U fleet_user -d fleet_db

# Listar tablas
\dt telemetry.*

# Ver estructura de la tabla
\d telemetry.positions

# Consulta de ejemplo
SELECT vehicle_id, captured_at, latitude, longitude
FROM telemetry.positions
WHERE captured_at >= NOW() - INTERVAL '1 hour'
ORDER BY captured_at DESC
LIMIT 10;
```

## Desarrollo

### Construir Servicios Backend

```bash
cd backend/go

# Construir API de ingesta
make build-ingestion

# Construir worker
make build-worker

# Ejecutar tests
make test

# Ejecutar tests de integración (requiere Docker)
make test-integration
```

### Agregar Dependencias

```bash
cd backend/go
go get github.com/example/package
go mod tidy
```

## Testing

El proyecto sigue los estándares definidos en `.kiro/steering/testing.md`:

- **Suite rápida**: Núcleo del dominio con fakes (sin infraestructura externa)
- **Suite de integración**: Pruebas con contenedores efímeros (Testcontainers)
- **End-to-end**: Recorridos completos con Docker Compose

```bash
# Tests rápidos (dominio + fakes)
cd backend/go
go test ./...

# Tests de integración (requiere Docker)
go test -tags=integration ./...

# Property-based testing (idempotencia)
go test -run TestIdempotency ./...
```

## Seguridad

- **Credenciales**: Nunca commitear archivos `.env` con valores reales
- **Puertos**: En desarrollo, TimescaleDB se expone en puerto configurable (default 5433)
- **Autenticación**: TimescaleDB usa SCRAM-SHA-256 por defecto
- **Permisos**: Servicios corren con usuarios no-root en contenedores

Ver estándares completos en `.kiro/steering/security.md`.

## Convenciones de Código

- **Idioma**: Código y comentarios en inglés
- **Documentación**: Español (specs, README, documentación técnica)
- **Mensajes de error de usuario**: Español
- **Logs de producción**: Inglés (compatibilidad con herramientas de monitoreo)

Ver guía completa en `.kiro/steering/lenguaje.md`.

## Contribución

Este proyecto sigue Clean Architecture y Domain-Driven Design:

- **Backend**: Separación estricta entre dominio, aplicación, adaptadores e infraestructura
- **Frontend**: Separación entre presentación, lógica de aplicación e infraestructura
- **Móvil**: Mismos principios arquitectónicos

Ver estándares en:
- `.kiro/steering/structure.md` — Arquitectura y organización
- `.kiro/steering/tech.md` — Stack tecnológico y restricciones técnicas
- `.kiro/steering/api-standards.md` — Diseño de API (RFC 9457 Problem Details)

## Licencia

[Definir según necesidades del proyecto]

## Contacto

[Definir según necesidades del proyecto]
