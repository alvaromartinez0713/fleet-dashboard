# Migraciones de Base de Datos

## Propósito

Este directorio contiene las migraciones SQL versionadas que definen y evolucionan el esquema de TimescaleDB para el sistema de monitoreo de flotas.

Las migraciones se ejecutan de forma ordenada, idempotente y automática mediante el servicio `migrate` durante la inicialización de la infraestructura en docker-compose.

---

## Convención de Nombres

Cada migración sigue la convención de nombres establecida por `golang-migrate`:

```
NNNNNN_description_of_migration.up.sql
NNNNNN_description_of_migration.down.sql
```

### Estructura del nombre

| Componente | Descripción | Ejemplo |
|---|---|---|
| `NNNNNN` | Número secuencial de 6 dígitos, rellenado con ceros | `000001`, `000002`, `000003` |
| `_` | Separador literal | — |
| `description_of_migration` | Descripción breve en inglés de la intención de la migración | `create_telemetry_schema` |
| `.up.sql` o `.down.sql` | Sufijo que indica dirección: aplicar o revertir | — |

### Ejemplos

- `000001_create_telemetry_schema.up.sql` — Aplicar: crear la extensión TimescaleDB y el schema `telemetry`
- `000001_create_telemetry_schema.down.sql` — Revertir: eliminar el schema `telemetry`
- `000002_create_positions_table.up.sql` — Aplicar: crear tabla `telemetry.positions` como hypertable
- `000002_create_positions_table.down.sql` — Revertir: eliminar tabla `telemetry.positions`

### Reglas de nombrado

1. **Numeración secuencial**: cada migración incrementa el número en 1
2. **Sin espacios**: usar guiones bajos (`_`) para separar palabras
3. **Descripción clara**: el nombre debe reflejar la intención (crear schema, crear tabla, otorgar permisos)
4. **Inglés en el nombre**: seguir convención de código en inglés
5. **Minúsculas**: todo en minúsculas para compatibilidad

---

## Orden Secuencial

Las migraciones se aplican en **orden ascendente por número secuencial**:

```
1. 000001_create_telemetry_schema
   ├─ Crea: extension TimescaleDB
   ├─ Crea: schema telemetry
   └─ Efecto: estructura base lista

2. 000002_create_positions_table
   ├─ Crea: tabla telemetry.positions
   ├─ Crea: hypertable (TimescaleDB)
   ├─ Crea: índices para queries
   └─ Efecto: almacenamiento de telemetría disponible

3. 000003_grant_permissions
   ├─ Concede: permisos a fleet_user
   ├─ Aplica: mínimo privilegio (SELECT, INSERT)
   └─ Efecto: worker/ingestion pueden acceder
```

### Garantías del orden

- **Dependencias respetadas**: una migración posterior no se ejecuta si la anterior falló
- **Idempotencia por número**: dentro de cada número, las migraciones `.up` no pueden volver a ejecutarse
- **Determinismo**: el mismo conjunto de migraciones aplicadas siempre genera el mismo estado final de la base de datos
- **Recuperabilidad**: si una migración falla, el proceso se detiene; la base de datos permanece en estado consistente

---

## Relación .up.sql / .down.sql

Cada migración consta de un par de archivos:

### .up.sql — Aplicar la migración

**Responsabilidad**: ejecutar los cambios que la migración introduce.

**Ejemplos**:
- Crear schema, tabla, índice, función, rol
- Otorgar permisos, agregar columnas, modificar tipos
- Cualquier operación que avance el estado de la base de datos

**Características**:
- Debe ser **idempotente**: se puede ejecutar múltiples veces sin provocar errores
- Usar `IF NOT EXISTS` cuando sea apropiado para garantizar idempotencia
- Debe completarse exitosamente o la migración falla

### .down.sql — Revertir la migración

**Responsabilidad**: deshacer exactamente lo que `.up.sql` hizo.

**Ejemplos**:
- `CREATE TABLE` → `DROP TABLE`
- `GRANT PERMISSIONS` → `REVOKE PERMISSIONS`
- `CREATE SCHEMA` → `DROP SCHEMA`

**Características**:
- Debe ser la **inversa exacta** de `.up.sql`
- Usar `IF EXISTS` para garantizar idempotencia en rollback
- El orden de reversión es inverso: primero se revierte `000003`, luego `000002`, luego `000001`

### Ejemplo de par

```sql
-- 000002_create_positions_table.up.sql
CREATE TABLE IF NOT EXISTS telemetry.positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id VARCHAR(50) NOT NULL,
    latitude NUMERIC(9, 6) NOT NULL,
    longitude NUMERIC(9, 6) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

-- 000002_create_positions_table.down.sql
DROP TABLE IF EXISTS telemetry.positions CASCADE;
```

### Relación de dependencia

| Migración | Estado después de .up | Estado después de .down |
|---|---|---|
| 000001 | Extension TimescaleDB + schema `telemetry` creados | Schema eliminado |
| 000002 | Tabla `positions` como hypertable e índices listos | Tabla eliminada |
| 000003 | Permisos otorgados a `fleet_user` | Permisos revocados |

---

## Tabla schema_migrations

### Registro automático

`golang-migrate` mantiene automáticamente una tabla llamada `schema_migrations` en la base de datos. Esta tabla registra el **historial de todas las migraciones aplicadas**.

### Estructura de schema_migrations

```sql
CREATE TABLE schema_migrations (
    version bigint PRIMARY KEY,
    dirty boolean
);
```

| Campo | Tipo | Propósito |
|---|---|---|
| `version` | `bigint` | Número secuencial de la migración (ejemplo: 1, 2, 3) |
| `dirty` | `boolean` | Indica si la migración se interrumpió sin completar (= `true`) |

### Comportamiento al aplicar migraciones

1. **Primera ejecución**: 
   - Se ejecuta `000001_create_telemetry_schema.up.sql`
   - Se registra en `schema_migrations`: `INSERT INTO schema_migrations (version, dirty) VALUES (1, false)`
   - Se ejecuta `000002_create_positions_table.up.sql`
   - Se registra: `INSERT INTO schema_migrations (version, dirty) VALUES (2, false)`
   - Se ejecuta `000003_grant_permissions.up.sql`
   - Se registra: `INSERT INTO schema_migrations (version, dirty) VALUES (3, false)`

2. **Re-ejecución (idempotencia)**:
   - golang-migrate consulta `schema_migrations`
   - Verifica que las versiones `1`, `2`, `3` ya están registradas
   - Las omite (no ejecuta `up.sql` nuevamente)
   - Busca nuevas migraciones (versión `4`, `5`, etc.) que no están registradas
   - Solo ejecuta las nuevas

3. **Fallo durante migración**:
   - Si `000002_create_positions_table.up.sql` falla a mitad de ejecución
   - Se establece `dirty = true` en `schema_migrations` para versión 2
   - El proceso se detiene
   - La migración posterior (`000003`) no se ejecuta
   - La base de datos queda en estado inconsistente; requiere intervención manual o rollback

### Consultando el registro

Para verificar qué migraciones se han ejecutado:

```sql
SELECT version, dirty FROM schema_migrations ORDER BY version;
```

**Salida esperada** (después de aplicar todas):
```
 version | dirty
---------+-------
       1 | false
       2 | false
       3 | false
```

### Idempotencia garantizada

Porque `schema_migrations` registra todas las ejecuciones exitosas, el servicio `migrate` puede ser invocado múltiples veces:

- **1ª ejecución**: aplica 3 migraciones, registra en `schema_migrations`
- **2ª ejecución**: consulta `schema_migrations`, ve que las 3 ya están aplicadas, termina sin hacer cambios (exit 0)
- **3ª ejecución**: idem a la 2ª

**Garantía**: después de cualquier ejecución, el estado de la base de datos es idéntico y correcto.

### Rollback

Aunque el servicio `migrate` nunca ejecuta rollbacks automáticos, los archivos `.down.sql` permiten revertir migraciones manualmente si es necesario:

```bash
# No implementado en el MVP, pero disponible para operaciones manuales
# El comando `migrate down` usaría los archivos .down.sql en orden inverso
migrate -path ./migrations -database "postgres://..." down 1  # Revierte 000003
```

---

## Ciclo de Vida en docker-compose

### Inicio del sistema

```
1. timescaledb inicia
   └─ Health check: pg_isready ✓ (healthy)

2. migrate inicia (depende de timescaledb healthy)
   ├─ Lee variables de entorno: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
   ├─ Conecta a timescaledb
   ├─ Consulta schema_migrations (tabla vacía en primer inicio)
   ├─ Ejecuta migraciones 000001, 000002, 000003 secuencialmente
   ├─ Registra cada ejecución en schema_migrations
   ├─ Termina con exit 0
   └─ docker-compose reconoce: service_completed_successfully

3. worker, ingestion inician (dependen de migrate completed)
   ├─ worker asume: schema telemetry existe, tabla positions existe, índices existen, permisos otorgados
   └─ ingestion asume: igual
```

### Re-inicios o escalabilidad

Si docker-compose se reinicia:

```
1. timescaledb se levanta
   └─ Datos persisten en volumen pgdata

2. migrate se ejecuta nuevamente
   ├─ schema_migrations ya tiene registros de 000001, 000002, 000003
   ├─ Omite las migraciones ya aplicadas
   ├─ Busca nuevas (no hay)
   ├─ Termina con exit 0
   └─ sin cambios en la BD

3. worker, ingestion se inician
   └─ BD está lista
```

---

## Diseño de Migraciones Existentes

### 000001_create_telemetry_schema

**Propósito**: establecer la base de TimescaleDB.

**.up.sql**:
- `CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE` — Activa extensión TimescaleDB
- `CREATE SCHEMA IF NOT EXISTS telemetry` — Crea namespace para objetos de telemetría

**.down.sql**:
- `DROP SCHEMA IF EXISTS telemetry CASCADE` — Limpia el schema y sus objetos

**Idempotencia**: `IF NOT EXISTS` permite re-ejecución sin errores.

---

### 000002_create_positions_table

**Propósito**: crear tabla optimizada para series de tiempo.

**.up.sql**:
- Crea tabla `telemetry.positions` con columnas:
  - `id` (UUID, clave primaria, generada aleatoriamente)
  - `vehicle_id` (VARCHAR 50, obligatorio)
  - `latitude`, `longitude`, `altitude`, `speed`, `heading`, `accuracy` (NUMERIC)
  - `timestamp` (TIMESTAMPTZ, obligatorio, dimensión hypertable)
- `SELECT create_hypertable(...)` — Convierte tabla en hypertable para optimización de series de tiempo
- Índices en `(vehicle_id, timestamp DESC)` e `(timestamp DESC)` para queries comunes

**.down.sql**:
- `DROP TABLE IF EXISTS telemetry.positions CASCADE` — Elimina tabla y sus índices

**Optimizaciones**:
- Hypertable: CompactTime-series data en chunks automáticos
- Índices: acelera consultas por vehículo y por fecha
- Tipos: NUMERIC para precisión de coordenadas, TIMESTAMPTZ para timezone-awareness

---

### 000003_grant_permissions

**Propósito**: otorgar acceso seguro a `fleet_user`.

**.up.sql**:
- `GRANT USAGE ON SCHEMA telemetry TO fleet_user` — Permite acceder al schema
- `GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA telemetry TO fleet_user` — Permite leer y escribir
- `GRANT SELECT, INSERT ON telemetry.positions TO fleet_user` — Permisos explícitos en tabla

**.down.sql**:
- `REVOKE USAGE ...`, `REVOKE SELECT, INSERT ...` — Revoca permisos

**Seguridad**:
- Mínimo privilegio: solo SELECT e INSERT (sin DELETE, UPDATE, ALTER)
- Sin acceso a `public` ni otros roles
- Permite que worker/ingestion funcionen sin permisos administrativos

---

## Nota Operativa: Ajuste del Esquema de `positions_table` (Migración 000002)

### Contexto

La migración `000002` fue editada para alinearse con el contrato del Telemetry_Worker, introduciendo las siguientes modificaciones:

- **Nueva columna**: `event_id UUID NOT NULL` (clave de deduplicación por `event_id`)
- **Nueva única constraint**: `(captured_at, event_id)` sobre `positions_event_id_key` (exigencia de TimescaleDB para unique indexes sobre hypertables)
- **Columnas de contrato**: `received_at`, `processed_at`, `stored_at` (timestamps del flujo de procesamiento)
- **Eliminadas columnas legadas**: `altitude`, `speed`, `heading`, `timestamp`
- **Dimensión temporal actualizada**: cambio de `timestamp` a `captured_at` como dimensión del hypertable

**Razón**: la aplicación no está en producción y no hay entornos con el esquema anterior ya aplicado. Las bases de datos de prueba siempre se crean desde cero aplicando todas las migraciones, por lo que el ajuste directo de `000002` no requiere una nueva migración hacia adelante.

### Impacto en bases de datos locales existentes

**Caveat importante**: si un desarrollador ya tiene una base de datos local con la versión anterior de `000002` aplicada, `golang-migrate` **no** reaplicará la migración editada (porque `schema_migrations` ya registra la versión como completada).

**Solución**: recrear la base de datos local:

```bash
# Opción 1: drop de la base de datos completa
docker-compose down -v  # -v elimina volúmenes
docker-compose up migrate  # Re-aplica todas las migraciones desde cero

# Opción 2: drop del schema telemetry únicamente (si quiere preservar otros datos)
docker-compose exec timescaledb psql -U fleet_user -d fleet_telemetry \
  -c "DROP SCHEMA IF EXISTS telemetry CASCADE;"
docker-compose up migrate  # Re-aplica 000001, 000002, 000003
```

**CI y entornos efímeros**: no se ven afectados porque siempre parten de bases de datos vacías.

---

## Adición de Nuevas Migraciones

Para agregar una nueva migración en el futuro:

1. **Determinar el siguiente número**: si la última es `000003`, la nueva es `000004`
2. **Crear el par de archivos**:
   ```bash
   touch 000004_your_migration_name.up.sql
   touch 000004_your_migration_name.down.sql
   ```
3. **Escribir `.up.sql`**: SQL que aplica el cambio (idempotente con `IF NOT EXISTS`, `IF NOT` etc.)
4. **Escribir `.down.sql`**: SQL que revierte exactamente lo que `.up.sql` hace
5. **Probar localmente**:
   ```bash
   docker-compose up migrate --force-recreate
   docker-compose logs migrate
   ```
6. **Verificar `schema_migrations`**:
   ```bash
   docker-compose exec timescaledb psql -U fleet_user -d fleet_telemetry -c "SELECT * FROM schema_migrations;"
   ```
7. **Verificar estado**: consultas SQL directas para confirmar que la estructura es correcta

---

## Troubleshooting

### Migración falla: "schema already exists"

**Causa**: `.up.sql` intenta crear un schema sin `IF NOT EXISTS`.

**Solución**: agregar `IF NOT EXISTS` a la declaración `CREATE`.

**Ejemplo**:
```sql
-- ❌ Incorrecto
CREATE SCHEMA telemetry;

-- ✅ Correcto
CREATE SCHEMA IF NOT EXISTS telemetry;
```

### schema_migrations marcada como dirty

**Causa**: una migración se interrumpió antes de completar (timeout, connection error, etc.).

**Efecto**: estado inconsistente; `migrate` no continuará automáticamente.

**Solución manual**:
1. Verificar el estado de la migración que falló
2. Completar manualmente si es posible, o rollback
3. Actualizar `schema_migrations` para marcar como `dirty = false`

```sql
UPDATE schema_migrations SET dirty = false WHERE version = 2;
```

### ¿Cómo verificar que todas las migraciones se han aplicado?

```bash
# Conectarse a TimescaleDB
docker-compose exec timescaledb psql -U fleet_user -d fleet_telemetry

# Dentro del psql:
SELECT * FROM schema_migrations;  -- Ver registro de migraciones
SELECT * FROM information_schema.tables WHERE table_schema = 'telemetry';  -- Ver tablas creadas
SELECT * FROM pg_indexes WHERE schemaname = 'telemetry';  -- Ver índices
```

---

## Referencias

- [golang-migrate Documentación](https://github.com/golang-migrate/migrate)
- [PostgreSQL CREATE SCHEMA](https://www.postgresql.org/docs/current/sql-createschema.html)
- [TimescaleDB Hypertables](https://docs.timescale.com/latest/develop/hypertables/)
- [PostgreSQL GRANT](https://www.postgresql.org/docs/current/sql-grant.html)

