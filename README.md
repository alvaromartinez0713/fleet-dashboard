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

### Servicios de Infraestructura

- **Zookeeper** (`confluentinc/cp-zookeeper:7.6.0`): Coordinador de metadatos para Kafka
- **Kafka** (`confluentinc/cp-kafka:7.6.0`): Bus de eventos para telemetría desacoplada
  - Topic: `telemetry.positions` (3 particiones, RF=1)
  - Retención: 24 horas
- **TimescaleDB** (`timescale/timescaledb:2.14.0-pg15`): Persistencia de series temporales
  - Schema: `telemetry`
  - Hypertable: `telemetry.positions` (particionamiento por `captured_at` cada 1 hora)

### Servicios de Aplicación

- **Ingestion API** (Puerto 8080): Receptor HTTP que publica eventos en Kafka
- **Worker**: Consumidor de Kafka que procesa y persiste en TimescaleDB

### Flujo de Datos

```
Mobile App → Ingestion API → Kafka (telemetry.positions) → Worker → TimescaleDB → Dashboard
```

## Requisitos Previos

- Docker Engine 24.0+ con Docker Compose plugin
- Make (opcional, para comandos simplificados)
- Go 1.23+ (para desarrollo backend)
- Node.js 20+ (para desarrollo frontend, cuando se implemente)

## Inicio Rápido

### 1. Configuración Inicial

```bash
# Clonar el repositorio
git clone <repository-url>
cd fleet-dashboard

# Copiar plantilla de variables de entorno
cp .env.example .env

# (Opcional) Ajustar variables en .env
# Por defecto, TimescaleDB se expone en puerto 5433 para evitar conflictos
```

### 2. Iniciar Infraestructura

```bash
# Iniciar todos los servicios
make up

# Esperar 20-30 segundos para que todos los servicios estén healthy
make status
```

### 3. Verificar Servicios

```bash
# Ver logs de todos los servicios
make logs

# Ver logs de un servicio específico
make logs-kafka
make logs-timescaledb

# Verificar salud de Ingestion API
curl http://localhost:8080/health
```

### 4. Detener Servicios

```bash
# Detener y eliminar contenedores (preserva volúmenes)
make down

# Limpieza completa incluyendo volúmenes (¡elimina todos los datos!)
make clean-hard
```

## Makefile Commands

| Comando | Descripción | Uso |
|---------|-------------|-----|
| `make help` | Mostrar lista de comandos disponibles | `make` o `make help` |
| `make up` | Iniciar infraestructura en background | `make up` |
| `make down` | Detener todos los servicios | `make down` |
| `make restart` | Reiniciar servicios sin recrear contenedores | `make restart` |
| `make build` | Reconstruir imágenes Docker | `make build` |
| `make status` | Mostrar estado de todos los servicios | `make status` |
| `make logs` | Ver logs de todos los servicios (Ctrl+C para salir) | `make logs` |
| `make logs-kafka` | Ver logs solo de Kafka | `make logs-kafka` |
| `make logs-timescaledb` | Ver logs solo de TimescaleDB | `make logs-timescaledb` |
| `make clean` | Detener y eliminar contenedores (preserva volúmenes) | `make clean` |
| `make clean-hard` | Eliminar todo incluyendo volúmenes | `make clean-hard` |

### Ejemplos de Flujos de Trabajo

**Desarrollo normal:**
```bash
make up         # Iniciar stack
make logs       # Monitorear logs mientras trabajas
make restart    # Reiniciar después de cambios de código
make down       # Detener al finalizar (datos preservados)
```

**Reinicio desde cero:**
```bash
make clean-hard # Eliminar todo incluyendo volúmenes
make up         # Reiniciar limpio
```

**Inspeccionar servicio específico:**
```bash
make status            # Ver estado general
make logs-timescaledb  # Ver solo logs de base de datos
```

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
