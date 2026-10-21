# Diseño: Sistema de Migraciones de Base de Datos

## 1. Visión General

Un servicio Go independiente, sin estado, que se ejecuta como contenedor en docker-compose. Su responsabilidad única: ejecutar migraciones SQL en TimescaleDB cuando la base de datos está healthy.

**Características clave**:
- Se levanta solo después de que TimescaleDB esté healthy (health check: `pg_isready`)
- Ejecuta 3 migraciones secuenciales de forma idempotente
- Termina exitosamente (exit 0) tras completar
- No es un daemon: se detiene automáticamente
- Worker e ingestion dependen de que migrate haya completado antes de iniciar
- Reemplaza el script estático anterior (`backend/go/scripts/init-timescaledb.sql`) con un sistema versionado

---

## 2. Flujo de Inicialización

```
1. docker-compose levanta timescaledb
   └─ Health check: pg_isready (espera hasta que sea true)

2. docker-compose inicia migrate (depende: timescaledb healthy)
   └─ Acción: ejecutar migraciones SQL

3. migrate ejecuta:
   ├─ Leer variables de entorno: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
   ├─ Validar que todas estén presentes
   ├─ Construir connection string: postgres://user:password@host:port/dbname
   ├─ Conectar a TimescaleDB
   ├─ Ejecutar migración 000001: crear schema telemetry
   ├─ Ejecutar migración 000002: crear tabla positions como hypertable
   ├─ Ejecutar migración 000003: otorgar permisos a fleet_user
   └─ Registrar en tabla schema_migrations (mantenida por golang-migrate)

4. migrate termina con exit code 0
   └─ docker-compose reconoce: service_completed_successfully

5. docker-compose inicia worker e ingestion
   ├─ worker depende: timescaledb (healthy) + migrate (completed) + kafka (healthy)
   └─ ingestion depende: kafka (healthy) + migrate (completed)

6. Servicios asumen BD lista, sin necesidad de ejecutar migraciones
```

---

## 3. Arquitectura de docker-compose

### Servicio timescaledb

- **Imagen**: `timescaledb:latest-pg15`
- **Puerto**: 5432
- **Health check**: `pg_isready -U postgres`
- **Intervalo**: 5s, timeout 3s, start_period 10s
- **Variables de entorno**: POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
- **Volumen**: `pgdata:/var/lib/postgresql/data` (persistencia)
- **Red**: `default` (shared)

### Servicio migrate (NUEVO)

- **Dockerfile target**: `migrate`
- **Comando**: `/app/migrate` (sin argumentos)
- **Dependencias**:
  - `timescaledb` (condition: `service_healthy`)
  - NO depende de: kafka, zookeeper, worker, ingestion
- **Variables de entorno**: 
  - `DB_HOST=timescaledb`
  - `DB_PORT=5432`
  - `DB_USER=fleet_user`
  - `DB_PASSWORD` (desde .env)
  - `DB_NAME=fleet_telemetry`
- **Volúmenes**: 
  - `/app/migrations/` (copy desde imagen)
- **Logs**: accesibles con `docker-compose logs migrate`
- **Comportamiento**: 
  - Inicia cuando timescaledb esté healthy
  - Ejecuta migraciones
  - Termina automáticamente (exit 0)
  - docker-compose reconoce: `service_completed_successfully`

### Servicios worker e ingestion (MODIFICADOS)

- **worker** depende: 
  - `timescaledb` (condition: `service_healthy`)
  - `migrate` (condition: `service_completed_successfully`) ← NUEVO
  - `kafka` (condition: `service_healthy`)
- **ingestion** depende:
  - `migrate` (condition: `service_completed_successfully`) ← NUEVO
  - `kafka` (condition: `service_healthy`)

---

## 4. Estructura de Código

```
backend/go/
├── migrations/                                    (NUEVO DIRECTORIO)
│   ├── README.md
│   ├── 000001_create_telemetry_schema.up.sql
│   ├── 000001_create_telemetry_schema.down.sql
│   ├── 000002_create_positions_table.up.sql
│   ├── 000002_create_positions_table.down.sql
│   ├── 000003_grant_permissions.up.sql
│   └── 000003_grant_permissions.down.sql
│
├── cmd/
│   ├── api/
│   │   └── main.go                              (existente, sin cambios)
│   ├── worker/
│   │   └── main.go                              (existente, sin cambios)
│   └── migrate/                                  (NUEVO)
│       └── main.go                              (NEW: servicio migraciones)
│
├── internal/
│   ├── domain/                                   (existente)
│   ├── application/                              (existente)
│   ├── adapters/                                 (existente)
│   └── infrastructure/
│       ├── config/                               (existente)
│       └── migrations/                           (NUEVO)
│           └── migrator.go                       (NEW: lógica golang-migrate)
│
├── Dockerfile                                    (ACTUALIZADO: agregar stage migrate)
├── go.mod                                        (ACTUALIZADO: agregar golang-migrate, lib/pq)
├── go.sum                                        (ACTUALIZADO)
└── ... (otros archivos existentes)
```

---

## 5. Contenido Detallado de Migraciones SQL

### 000001_create_telemetry_schema

**Propósito**: Crear la extensión TimescaleDB y el schema `telemetry`.

**.up.sql**:
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE SCHEMA IF NOT EXISTS telemetry;
```

**.down.sql**:
```sql
DROP SCHEMA IF EXISTS telemetry CASCADE;
```

**Notas**:
- Idempotente: `IF NOT EXISTS` permite re-ejecución
- `CASCADE` asegura que no haya referencias huérfanas

---

### 000002_create_positions_table

**Propósito**: Crear tabla `positions` como hypertable TimescaleDB e índices para query performance.

**.up.sql**:
```sql
CREATE TABLE IF NOT EXISTS telemetry.positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id VARCHAR(50) NOT NULL,
    latitude NUMERIC(9, 6) NOT NULL,
    longitude NUMERIC(9, 6) NOT NULL,
    altitude NUMERIC(8, 2),
    speed NUMERIC(7, 2),
    heading NUMERIC(5, 2),
    accuracy NUMERIC(5, 2),
    timestamp TIMESTAMPTZ NOT NULL
);

SELECT create_hypertable('telemetry.positions', 'timestamp', if_not_exists => TRUE);

CREATE INDEX ON telemetry.positions (vehicle_id, timestamp DESC);
CREATE INDEX ON telemetry.positions (timestamp DESC);
```

**.down.sql**:
```sql
DROP TABLE IF EXISTS telemetry.positions CASCADE;
```

**Notas**:
- Hypertable: optimizada para series de tiempo (TimescaleDB)
- Índices en `vehicle_id` y `timestamp` para queries comunes
- `PRIMARY KEY` en `id`: cada evento es identificable
- `timestamp` es obligatorio (dimensión hypertable)
- Tipos: UUID para ID, VARCHAR para placa vehículo, NUMERIC para coordenadas, TIMESTAMPTZ para time-zone aware

---

### 000003_grant_permissions

**Propósito**: Otorgar permisos a `fleet_user` (usuario con el que worker e ingestion conectan).

**.up.sql**:
```sql
GRANT USAGE ON SCHEMA telemetry TO fleet_user;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry TO fleet_user;
GRANT SELECT, INSERT ON telemetry.positions TO fleet_user;
```

**.down.sql**:
```sql
REVOKE USAGE ON SCHEMA telemetry FROM fleet_user;
REVOKE SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry FROM fleet_user;
REVOKE SELECT, INSERT ON telemetry.positions FROM fleet_user;
```

**Notas**:
- Solo `SELECT` e `INSERT`: sin DELETE, UPDATE, ALTER
- Principio de mínimo privilegio
- Aplica a `fleet_user`, no a `public` ni otros roles

---

## 6. Componentes Go

### cmd/migrate/main.go

**Responsabilidades**:
1. Leer variables de entorno (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
2. Validar que todas las variables requeridas estén presentes
3. Construir connection string PostgreSQL
4. Conectar a la base de datos
5. Llamar a `migrations.RunMigrations(db)`
6. Registrar eventos: inicio, cada migración, finalización
7. Exit 0 si éxito, Exit 1 si error

**Pseudocódigo**:
```
1. Read environment variables
2. Validate all required variables present
   if missing -> log error, exit 1
3. Build connection string
4. Connect to DB
   if error -> log error, exit 1
5. Log "Starting database migrations"
6. Call migrator.RunMigrations()
   if error -> log error, exit 1
7. Log "All migrations completed successfully"
8. Exit 0
```

**Observabilidad**:
- Logs a stdout/stderr
- Registra timestamps, nombres de migración, resultados
- NO imprime contraseñas

---

### internal/infrastructure/migrations/migrator.go

**Responsabilidades**:
1. Usar `golang-migrate/v4` para orquestar migraciones
2. Cargar migraciones del filesystem `/app/migrations/`
3. Conectar driver PostgreSQL
4. Ejecutar `Up()` para aplicar migraciones pendientes
5. Mantener tabla `schema_migrations` (automática en golang-migrate)
6. Retornar error si alguna migración falla

**Características**:
- **Idempotencia**: golang-migrate mantiene registro de migraciones ejecutadas
- **Re-ejecución segura**: migraciones ya aplicadas se omiten
- **Versioning**: cada archivo `.up.sql` y `.down.sql` es una versión
- **Rollback**: las migraciones `.down.sql` permiten rollback (no usado en migrate, pero disponible)

**Pseudocódigo**:
```
1. Create golang-migrate instance
   source: file:///app/migrations/
   database: postgres://...
2. Call m.Up()
3. If error: return error
4. Return nil
```

---

## 7. Dockerfile

### Stage builder (opcional, puede ser `migrate`)

```dockerfile
FROM golang:1.21 AS builder

WORKDIR /app

# Copiar go.mod y go.sum
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código
COPY . .

# Compilar servicio migrate
RUN go build -o migrate ./cmd/migrate
```

### Stage migrate

```dockerfile
FROM debian:bookworm-slim AS migrate

# Instalar dependencias runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copiar binario compilado
COPY --from=builder /app/migrate /app/migrate

# Copiar migraciones SQL
COPY backend/go/migrations /app/migrations

# Establecer punto de entrada
ENTRYPOINT ["/app/migrate"]
```

**Notas**:
- Imagen base pequeña: `debian:bookworm-slim`
- Multi-stage: separa compilación de runtime
- Migraciones copiadas: accesibles en `/app/migrations/` en runtime
- No incluye código fuente ni binarios innecesarios en imagen final

---

## 8. go.mod y go.sum

**Dependencias nuevas a agregar**:

```go
require (
    github.com/golang-migrate/migrate/v4 v4.17.0
    github.com/lib/pq v1.10.9
)
```

**Notas**:
- `golang-migrate/v4`: framework para ejecutar migraciones
- `lib/pq`: driver PostgreSQL para Go

---

## 9. Características Clave de Diseño

### Idempotencia

golang-migrate mantiene automáticamente una tabla `schema_migrations` que registra cada migración ejecutada. Al re-ejecutar:
- golang-migrate consulta `schema_migrations`
- Identifica migraciones ya aplicadas
- Las omite (no se ejecutan nuevamente)
- Ejecuta solo las nuevas
- No hay errores tipo "schema already exists"
- BD permanece consistente

### Independencia

- `migrate` NO depende de Kafka, Zookeeper, worker, ingestion
- Funciona con solo `timescaledb` healthy
- Permite testear migraciones en aislamiento
- Permite ejecutar migraciones sin necesidad de otros servicios

### Versioning y Trazabilidad

- Todas las migraciones están en Git bajo `backend/go/migrations/`
- Numeración secuencial: 000001, 000002, 000003
- Cada migración tiene `.up.sql` (aplicar) y `.down.sql` (revertir)
- Posibilita rollback futuro si es necesario

### Simplicidad

- Sin `//go:embed`: migraciones se cargan del filesystem en runtime
- Sin compilación especial ni cambios al build
- golang-migrate es estándar en la comunidad Go
- Código directo, fácil de entender y mantener

### Observabilidad

- Logs registran:
  - Inicio del servicio
  - Nombre y número de cada migración ejecutada
  - Resultado (éxito o error)
  - Finalización
- Accesibles con `docker-compose logs migrate`
- Sin contraseñas ni datos sensibles

---

## 10. Transiciones de Estado

```
timescaledb: starting → healthy ✓
                         ↓
migrate: waiting for timescaledb → running migrations → completed ✓
                         ↓
worker: waiting for migrate + timescaledb + kafka → running
ingestion: waiting for migrate + kafka → running
```

**Garantías**:
- worker e ingestion NO inician hasta que migrate haya completado
- Si migrate falla (exit 1), worker e ingestion permanecen en `waiting`
- BD está lista para su uso por worker e ingestion
