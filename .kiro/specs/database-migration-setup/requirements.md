# Requisitos: Sistema de Migraciones de Base de Datos

## 1. Propósito

Definir los requisitos funcionales y no funcionales para un servicio Go independiente que ejecuta migraciones de base de datos en TimescaleDB.

El servicio migrate:
- Se levanta como contenedor independiente en docker-compose (como ingestion o worker)
- Depende SOLO de TimescaleDB en estado healthy
- Ejecuta las migraciones SQL de forma idempotente
- Prepara el esquema de la base de datos para que worker e ingestion funcionen
- Termina exitosamente (exit 0) después de completar su tarea
- NO es un servicio daemon (no queda corriendo)

Reemplaza el enfoque anterior de script estático (backend/go/scripts/init-timescaledb.sql) con un sistema versionado basado en golang-migrate.

## 2. Requisitos Funcionales

### RF-01: Iniciar cuando TimescaleDB está healthy

**Descripción**: El servicio migrate debe iniciar solo cuando TimescaleDB esté en estado healthy, garantizado por docker-compose mediante health check.

**Criterios de aceptación**:
- El servicio 'migrate' está definido como servicio en docker-compose.yml
- Tiene configuración: depends_on: timescaledb (condition: service_healthy)
- No inicia si TimescaleDB no está healthy
- Registra en logs: "Starting database migrations"
- La salida de logs es accesible vía docker-compose logs migrate

### RF-02: Conectar a TimescaleDB usando variables de entorno

**Descripción**: El servicio lee credenciales de conexión desde variables de entorno estándar de PostgreSQL.

**Criterios de aceptación**:
- Lee variables de entorno: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
- Construye connection string válida: postgres://user:password@host:port/dbname
- Valida que todas las variables requeridas estén presentes antes de conectar
- Si alguna variable falta, retorna error descriptivo y exit code 1
- No imprime ni registra contraseñas en ningún log
- El error indica claramente cuál variable falta

### RF-03: Ejecutar todas las migraciones en orden secuencial

**Descripción**: El servicio ejecuta las 3 migraciones en orden correcto usando golang-migrate.

**Criterios de aceptación**:
- Ejecuta las migraciones en orden: 000001 → 000002 → 000003
- Cada migración se ejecuta completamente antes de pasar a la siguiente
- Si una migración falla, el proceso se detiene inmediatamente
- Al fallar, registra el error completo y retorna exit code 1
- Las migraciones ejecutadas se registran en tabla schema_migrations de la BD
- Registra en logs: "Running migration: 000001_create_telemetry_schema"
- Registra en logs: "Migration 000001 completed successfully"

### RF-04: Crear schema telemetry, tabla positions como hypertable e índices

**Descripción**: Las tres migraciones crean la estructura de base de datos necesaria para almacenar telemetría de vehículos.

**Criterios de aceptación**:
- Migración 000001: crea extension timescaledb (si no existe)
- Migración 000001: crea schema telemetry (si no existe)
- Migración 000002: crea tabla telemetry.positions con columnas: id, vehicle_id, latitude, longitude, altitude, speed, heading, accuracy, timestamp
- Migración 000002: convierte telemetry.positions en hypertable con dimensión time en columna timestamp
- Migración 000002: crea índice en columna vehicle_id
- Migración 000002: crea índice en columna timestamp
- Migración 000003: ejecuta GRANT USAGE ON SCHEMA telemetry TO fleet_user
- Migración 000003: ejecuta GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry TO fleet_user
- Los tipos de datos son correctos (UUID, NUMERIC, TIMESTAMPTZ)
- Las constraints NOT NULL están presentes según diseño
- Todas las migraciones son idempotentes

### RF-05: Otorgar permisos a fleet_user

**Descripción**: La migración 000003 configura permisos para que worker e ingestion puedan acceder a la BD.

**Criterios de aceptación**:
- Ejecuta GRANT USAGE ON SCHEMA telemetry TO fleet_user
- Ejecuta GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry TO fleet_user
- Ejecuta GRANT SELECT, INSERT ON telemetry.positions TO fleet_user
- Los permisos se aplican solo a fleet_user, no a role public
- Los permisos persisten después de que migrate termina

### RF-06: Terminar exitosamente después de ejecutar migraciones

**Descripción**: El servicio debe finalizar con el código de salida correcto después de completar las migraciones.

**Criterios de aceptación**:
- Después de ejecutar todas las migraciones exitosamente, el proceso termina con exit code 0
- No queda corriendo como daemon
- docker-compose reconoce que el servicio ha completado (service_completed_successfully)
- Los servicios worker e ingestion pueden iniciar después de que migrate termina
- Si hay error en cualquier migración, termina con exit code 1
- Registra en logs: "All migrations completed successfully" si exit 0

### RF-07: Idempotencia: re-ejecutar es seguro

**Descripción**: Si el servicio se ejecuta múltiples veces, debe detectar migraciones ya aplicadas.

**Criterios de aceptación**:
- golang-migrate mantiene tabla schema_migrations con registro de migraciones ejecutadas
- Al re-ejecutar, golang-migrate verifica qué migraciones ya están aplicadas
- Las migraciones ya aplicadas se omiten (no se ejecutan nuevamente)
- No hay errores tipo "schema already exists" o "table already exists"
- La BD permanece en estado consistente después de re-ejecución
- El exit code sigue siendo 0 en re-ejecuciones


## 3. Requisitos No Funcionales

### RNF-01: Servicio completamente aislado

**Descripción**: El servicio migrate es un contenedor independiente con responsabilidad única.

**Criterios de aceptación**:
- El servicio migrate está definido como un servicio separado en docker-compose.yml
- Tiene su propio Dockerfile stage llamado migrate
- No comparte volúmenes, networks ni puertos con otros servicios
- La arquitectura permite ejecutar migrate sin ejecutar worker, ingestion o kafka
- Logs de migrate son separados e identificables

### RNF-02: Única dependencia TimescaleDB

**Descripción**: Solo depende de TimescaleDB healthy.

**Criterios de aceptación**:
- depends_on: timescaledb (condition: service_healthy)
- Sin kafka, zookeeper, worker, ingestion u otros servicios
- Funciona incluso si kafka no está disponible

### RNF-03: Código limpio sin complejidad

**Descripción**: Go golang-migrate. Sin embed, sin compilación especial.

**Criterios de aceptación**:
- cmd/migrate/main.go: lee variables, conecta, ejecuta
- internal/infrastructure/migrations/migrator.go: lógica golang-migrate
- Sin //go:embed, migraciones del filesystem
- Código directo, fácil de entender

### RNF-04: Migraciones en backend/go/migrations

**Descripción**: SQL en directorio estándar, versionado en Git.

**Criterios de aceptación**:
- backend/go/migrations/ existe
- 6 archivos SQL versionados
- README.md documentación

### RNF-05: Migraciones accesibles en contenedor

**Descripción**: SQL disponible en runtime.

**Criterios de aceptación**:
- Dockerfile copia: COPY backend/go/migrations /app/migrations
- Programa lee de /app/migrations/

### RNF-06: Dockerfile stage migrate

**Descripción**: Stage específico para compilar migrate.

**Criterios de aceptación**:
- FROM golang:1.21 AS migrate
- Compila cmd/migrate/main.go
- Binario en /app/migrate

### RNF-07: Dependencias go.mod

**Descripción**: Dependencias especificadas.

**Criterios de aceptación**:
- golang-migrate/v4
- lib/pq
- go.sum sincronizado

### RNF-08: docker-compose orden correcto

**Descripción**: timescaledb → migrate → worker/ingestion.

**Criterios de aceptación**:
- Orden de inicio correcto
- Worker/ingestion esperan migrate (completed)
- Si migrate falla, no inician

### RNF-09: Observabilidad logging

**Descripción**: Eventos importantes, sin datos sensibles.

**Criterios de aceptación**:
- Registra: "Starting migrations"
- "Running migration: 000001"
- "Migration completed"
- Sin contraseñas

### RNF-10: Worker/ingestion sin migraciones

**Descripción**: Solo migrate ejecuta migraciones.

**Criterios de aceptación**:
- Sin importes de paquete migrations
- Sin RunMigrations() en bootstrap
- Asumen BD lista

