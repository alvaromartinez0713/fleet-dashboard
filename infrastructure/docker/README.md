# Docker Compose Development Infrastructure

Este directorio contiene la configuración de Docker Compose para ejecutar localmente el stack completo del Portal de Monitoreo de Flotas.

## Servicios

### Zookeeper
- **Rol**: Coordinador de Kafka, gestiona metadata de brokers, topics y particiones
- **Puerto interno**: 2181
- **Health Check**: 4-letter word protocol `ruok`
- **Volúmenes**: zookeeper-data, zookeeper-logs (persistencia)

### Kafka
- **Rol**: Bus de eventos desacoplado para ingesta y procesamiento de telemetría
- **Puerto interno**: 9092 (PLAINTEXT)
- **Topic**: `telemetry.positions` (3 particiones, replication factor 1)
- **Health Check**: `kafka-broker-api-versions`
- **Volumen**: kafka-data (logs de broker, índices)

### TimescaleDB
- **Rol**: Base de datos PostgreSQL con extensión TimescaleDB para series temporales
- **Puerto interno**: 5432
- **Puerto host**: Configurable via `TIMESCALEDB_PORT` (.env, default: 5432)
- **Credentials**: `POSTGRES_USER`, `POSTGRES_PASSWORD` desde .env
- **Base de datos**: `POSTGRES_DB` desde .env (default: fleet_db)
- **Inicialización**: Script SQL `init-timescaledb.sql` se ejecuta en primer start
- **Health Check**: `pg_isready`
- **Volumen**: timescaledb-data (persistencia de datos)

## Quick Start

### Prerrequisitos
- Docker >= 20.10
- Docker Compose >= 2.0

### Pasos
1. **Desde la raíz del proyecto:**
   ```bash
   cd infrastructure/docker
   ```

2. **Copiar archivo de entorno:**
   ```bash
   cp .env.example .env
   ```

3. **Personalizar variables (opcional):**
   ```bash
   # Editar .env si necesitas puertos diferentes
   # TIMESCALEDB_PORT=5433  # Si 5432 está en uso
   ```

4. **Iniciar servicios:**
   ```bash
   docker compose up -d
   ```

5. **Esperar 30 segundos para que todos estén healthy:**
   ```bash
   docker compose ps
   # Todos deben mostrar "Up (healthy)" o "Up"
   ```

6. **Verificar:**
   ```bash
   # TimescaleDB
   psql -h localhost -p 5432 -U fleet_user -d fleet_db -c "\dt telemetry.*"

   # Kafka
   docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092
   ```

## Environment Variables

Archivo: `.env.example`

| Variable | Default | Descripción |
|----------|---------|-------------|
| `ZOOKEEPER_CLIENT_PORT` | 2181 | Puerto interno de Zookeeper (no expuesto al host) |
| `KAFKA_BROKERS` | kafka:9092 | Broker de Kafka desde dentro de la red |
| `KAFKA_TOPIC` | telemetry.positions | Topic principal para eventos de telemetría |
| `POSTGRES_USER` | fleet_user | Usuario de PostgreSQL |
| `POSTGRES_PASSWORD` | changeme | Contraseña (cambiar en .env) |
| `POSTGRES_DB` | fleet_db | Base de datos principal |
| `TIMESCALEDB_PORT` | 5432 | Puerto host para acceso a PostgreSQL |
| `DB_HOST` | localhost | Host para conexiones desde host |

## Troubleshooting

### Conflicto de puerto en 5432
Si tienes PostgreSQL u otro servicio usando 5432:

1. Editar `.env`:
   ```bash
   TIMESCALEDB_PORT=5433
   ```

2. Reiniciar:
   ```bash
   docker compose down
   docker compose up -d
   ```

3. Conectar con nuevo puerto:
   ```bash
   psql -h localhost -p 5433 -U fleet_user -d fleet_db
   ```

### Servicios no llegan a "healthy"
1. Revisar logs:
   ```bash
   docker compose logs -f timescaledb
   docker compose logs -f kafka
   ```

2. Verificar recursos:
   - Docker Desktop: Settings → Resources
   - Aumentar CPU, RAM si es necesario

3. Esperar más tiempo (primer start puede tomar 1-2 minutos)

### Error: "Topic already exists"
El script `create-kafka-topic.sh` es idempotente. Si el topic existe, se ignora. Es seguro reintentar.

### Reinicio desde cero
Para limpiar toda la infraestructura:

```bash
# Detener y eliminar volúmenes (PÉRDIDA DE DATOS)
docker compose down -v

# O desde raíz del proyecto:
make clean-hard
```

## Scripts

### `init-timescaledb.sql`
Crea el esquema y tablas en TimescaleDB:
- Schema: `telemetry`
- Table: `telemetry.positions` (hypertable particionada por `captured_at`)
- Índices: vehicle_id, event_id, trace_id
- Políticas: Compresión a 7 días, retención a 30 días

Se ejecuta automáticamente en el primer start del contenedor.

### `create-kafka-topic.sh`
Crea el topic `telemetry.positions` con:
- 3 particiones
- Replication factor: 1
- Retención: 24 horas
- Cleanup policy: delete

Se ejecuta automáticamente al iniciar kafka (via docker-entrypoint).

## Salir/Parar servicios

```bash
# Parar sin eliminar volúmenes (datos se preservan)
docker compose down

# O desde raíz:
make down

# Parar con eliminación de volúmenes (PÉRDIDA DE DATOS)
docker compose down -v
```

## Arquitectura de red

Todos los servicios están conectados a la red bridge `fleet-network`:
- Resolución DNS interna automática
- `zookeeper` → accesible como `zookeeper:2181`
- `kafka` → accesible como `kafka:9092`
- `timescaledb` → accesible como `timescaledb:5432`

Desde el host (máquina desarrollador):
- TimescaleDB: `localhost:5432` (configurable)
- Kafka: No expuesto (debe accederse via docker exec)
- Zookeeper: No expuesto (coordinación interna)

