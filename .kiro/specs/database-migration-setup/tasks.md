# Implementation Plan: Sistema de Migraciones de Base de Datos

## Overview

Plan de implementación para el servicio Go `migrate`: un contenedor independiente que ejecuta migraciones SQL versionadas sobre TimescaleDB usando `golang-migrate`, de forma idempotente, y termina con exit 0.

El plan avanza de forma incremental: primero las dependencias y los archivos SQL, luego la lógica de migración y el punto de entrada, después el empaquetado (Dockerfile) y la orquestación (docker-compose), y finalmente la limpieza del enfoque anterior. Cada paso se apoya en el anterior y termina con el cableado completo del servicio en docker-compose.

Las pruebas siguen `testing.md`: entran por bordes públicos y usan TimescaleDB efímero (Testcontainers) para los adaptadores de persistencia y migración (dependencia gestionada). No hay sección de "Correctness Properties" en el diseño, por lo que no se usan property-based tests; se emplean pruebas de integración y unitarias. Las sub-tareas de prueba están marcadas con `*` y son opcionales.

## Tasks

- [x] 1. Preparar dependencias y estructura de migraciones SQL
  - [x] 1.1 Agregar dependencias de migración a go.mod y go.sum
    - Agregar `github.com/golang-migrate/migrate/v4` y `github.com/lib/pq` a `backend/go/go.mod`
    - Ejecutar la sincronización del gestor de módulos para actualizar `backend/go/go.sum`
    - Verificar que el driver `postgres` y el source `file` de golang-migrate queden disponibles vía imports anónimos
    - _Requirements: RNF-03, RNF-07_

  - [x] 1.2 Crear migración 000001 (schema telemetry)
    - Crear `backend/go/migrations/000001_create_telemetry_schema.up.sql` con `CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE` y `CREATE SCHEMA IF NOT EXISTS telemetry`
    - Crear `backend/go/migrations/000001_create_telemetry_schema.down.sql` con `DROP SCHEMA IF EXISTS telemetry CASCADE`
    - Usar `IF NOT EXISTS` para garantizar idempotencia
    - _Requirements: RF-04, RF-07, RNF-04_

  - [x] 1.3 Crear migración 000002 (tabla positions como hypertable e índices)
    - Crear `backend/go/migrations/000002_create_positions_table.up.sql`: tabla `telemetry.positions` con columnas `id` (UUID PK), `vehicle_id` (VARCHAR NOT NULL), `latitude`/`longitude` (NUMERIC NOT NULL), `altitude`, `speed`, `heading`, `accuracy` (NUMERIC), `timestamp` (TIMESTAMPTZ NOT NULL)
    - Convertir a hypertable con `create_hypertable('telemetry.positions', 'timestamp', if_not_exists => TRUE)`
    - Crear índice en `(vehicle_id, timestamp DESC)` e índice en `(timestamp DESC)`
    - Crear `backend/go/migrations/000002_create_positions_table.down.sql` con `DROP TABLE IF EXISTS telemetry.positions CASCADE`
    - Asegurar tipos correctos (UUID, NUMERIC, TIMESTAMPTZ) y constraints NOT NULL
    - _Requirements: RF-04, RF-07, RNF-04_

  - [x] 1.4 Crear migración 000003 (permisos a fleet_user)
    - Crear `backend/go/migrations/000003_grant_permissions.up.sql` con `GRANT USAGE ON SCHEMA telemetry`, `GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry` y `GRANT SELECT, INSERT ON telemetry.positions` a `fleet_user`
    - Crear `backend/go/migrations/000003_grant_permissions.down.sql` con los `REVOKE` correspondientes
    - Aplicar solo a `fleet_user` (principio de mínimo privilegio, sin DELETE/UPDATE/ALTER, sin `public`)
    - _Requirements: RF-05, RF-04, RNF-04_

  - [x] 1.5 Crear README.md de migraciones
    - Crear `backend/go/migrations/README.md` documentando la convención de nombres, el orden secuencial, la relación `.up.sql`/`.down.sql` y cómo se registran en `schema_migrations`
    - Documentación en español, identificadores y nombres de archivo en inglés
    - _Requirements: RNF-04_

- [x] 2. Implementar la lógica de migración (migrator)
  - [x] 2.1 Implementar internal/infrastructure/migrations/migrator.go
    - Crear `backend/go/internal/infrastructure/migrations/migrator.go` con una función pública `RunMigrations` que reciba la connection string y la ruta del directorio de migraciones
    - Construir la instancia de golang-migrate con source `file://` sobre el directorio de migraciones y el driver PostgreSQL
    - Ejecutar `Up()`, tratar `ErrNoChange` como éxito (idempotencia) y retornar error en cualquier otro fallo
    - Registrar inicio y resultado de la operación sin exponer la contraseña (comments en inglés)
    - _Requirements: RF-03, RF-07, RNF-03, RNF-09_

  - [ ]* 2.2 Escribir prueba de integración del migrator con TimescaleDB efímero
    - Levantar TimescaleDB en contenedor efímero (Testcontainers) según `testing.md` 5.1
    - Ejecutar `RunMigrations` sobre la BD vacía y verificar estado final observable: existe schema `telemetry`, existe `telemetry.positions` como hypertable, existen los índices y los grants a `fleet_user`
    - Verificar idempotencia: una segunda ejecución deja el mismo estado y no retorna error
    - Build tag `integration` (`// Verifies: RF-03, RF-04, RF-05, RF-07`)
    - _Requirements: RF-03, RF-04, RF-05, RF-07_

- [x] 3. Implementar el punto de entrada del servicio migrate
  - [x] 3.1 Implementar lectura y validación de variables de entorno
    - En `backend/go/cmd/migrate/main.go`, leer `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
    - Validar que todas estén presentes; si falta alguna, registrar un error descriptivo que indique la variable faltante y salir con exit code 1
    - Construir la connection string `postgres://user:password@host:port/dbname` sin registrar la contraseña en logs
    - _Requirements: RF-02, RNF-09_

  - [x] 3.2 Cablear main con el migrator y el manejo de exit codes
    - Registrar "Starting database migrations", invocar `migrations.RunMigrations` apuntando al directorio `/app/migrations`
    - En caso de error de conexión o de migración, registrar el error completo y salir con exit code 1
    - En caso de éxito, registrar "All migrations completed successfully" y salir con exit code 0 (no queda como daemon)
    - Logs en inglés, sin contraseñas ni datos sensibles
    - _Requirements: RF-01, RF-03, RF-06, RNF-03, RNF-09_

  - [ ]* 3.3 Escribir prueba de integración end-to-end del binario migrate
    - Contra TimescaleDB efímero, ejecutar el flujo de `main` (o una función extraída invocable) con variables de entorno válidas y verificar exit 0 y el estado final de la BD
    - Verificar caso de variable faltante: exit 1 y mensaje que nombra la variable, sin exponer la contraseña
    - Build tag `integration` (`// Verifies: RF-01, RF-02, RF-06`)
    - _Requirements: RF-01, RF-02, RF-06_

- [x] 4. Checkpoint - Verificar migraciones y servicio
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Empaquetar el servicio en el Dockerfile
  - [x] 5.1 Agregar stages de build y runtime para migrate
    - En `backend/go/Dockerfile`, agregar un stage `migrate-builder` que compile `./cmd/migrate` reutilizando el stage `builder` existente
    - Agregar un stage runtime `migrate` (base ligera coherente con los stages actuales) que copie el binario y copie `backend/go/migrations` a `/app/migrations`
    - Definir `ENTRYPOINT` al binario migrate; no exponer puertos ni health check (no es daemon)
    - _Requirements: RNF-05, RNF-06_

- [x] 6. Orquestar el servicio en docker-compose
  - [x] 6.1 Agregar el servicio migrate a docker-compose.yml
    - Definir el servicio `migrate` con `build` target `migrate`
    - `depends_on: timescaledb` con `condition: service_healthy` y sin dependencia de kafka/zookeeper/worker/ingestion
    - Variables de entorno: `DB_HOST=timescaledb`, `DB_PORT=5432`, `DB_USER=fleet_user`, `DB_NAME=fleet_telemetry` y `DB_PASSWORD` desde `.env`
    - No compartir volúmenes/puertos propios de otros servicios; logs identificables vía `docker-compose logs migrate`
    - _Requirements: RF-01, RF-02, RNF-01, RNF-02_

  - [x] 6.2 Actualizar dependencias de worker e ingestion
    - Agregar a `worker`: `depends_on` `migrate` con `condition: service_completed_successfully` (manteniendo timescaledb healthy y kafka healthy)
    - Agregar a `ingestion`: `depends_on` `migrate` con `condition: service_completed_successfully` (manteniendo kafka healthy)
    - Garantizar el orden timescaledb → migrate → worker/ingestion
    - _Requirements: RF-06, RNF-08_

  - [x] 6.3 Verificar el health check de timescaledb en docker-compose
    - Confirmar/definir el health check `pg_isready` para timescaledb con intervalo, timeout y start_period de forma que migrate pueda depender de `service_healthy`
    - _Requirements: RF-01, RNF-02_

- [x] 7. Limpiar el enfoque anterior de inicialización
  - [x] 7.1 Eliminar el script estático de inicialización
    - Eliminar `backend/go/scripts/init-timescaledb.sql` y cualquier montaje/referencia a `init-timescaledb.sql` en docker-compose
    - _Requirements: RNF-10_

  - [x] 7.2 Asegurar que worker e ingestion no ejecutan migraciones
    - Revisar los bootstrap de `cmd/worker` y `cmd/ingestion` para confirmar que no importan el paquete `migrations` ni invocan `RunMigrations`; eliminar cualquier lógica de ese tipo si existiera
    - _Requirements: RNF-10_

  - [ ]* 7.3 Escribir prueba que verifica ausencia de lógica de migración en worker/ingestion
    - Prueba estática/de compilación que falle si `cmd/worker` o `cmd/ingestion` referencian el paquete `migrations` o `RunMigrations`
    - `// Verifies: RNF-10`
    - _Requirements: RNF-10_

- [x] 8. Checkpoint final - Verificar orquestación completa
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Las sub-tareas marcadas con `*` son pruebas opcionales y pueden omitirse para un MVP más rápido; no deben implementarse automáticamente.
- Las pruebas de integración usan TimescaleDB efímero (Testcontainers) y se separan con el build tag `integration` según `testing.md` (secciones 5.1 y 5.2).
- TimescaleDB es una dependencia gestionada: se verifica el estado final, no las llamadas. No se mockea la base de datos, para detectar SQL incorrecto, tipos, hypertables y grants reales.
- Documentación en español; nombres de archivos, identificadores y comments en inglés; logs en inglés sin datos sensibles.
- Cada tarea referencia los requisitos (RF/RNF) que verifica para trazabilidad (sección 18 de `testing.md`).

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3", "1.4", "1.5"] },
    { "id": 1, "tasks": ["2.1"] },
    { "id": 2, "tasks": ["2.2", "3.1"] },
    { "id": 3, "tasks": ["3.2"] },
    { "id": 4, "tasks": ["3.3", "5.1"] },
    { "id": 5, "tasks": ["6.1"] },
    { "id": 6, "tasks": ["6.2", "6.3", "7.1", "7.2"] },
    { "id": 7, "tasks": ["7.3"] }
  ]
}
```
