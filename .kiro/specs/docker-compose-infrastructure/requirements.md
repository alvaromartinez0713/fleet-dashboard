# Requisitos: Infraestructura Docker Compose (Kafka + TimescaleDB)

## Introducción

Este documento especifica los requisitos para agregar servicios de infraestructura a Docker Compose: Kafka (con Zookeeper) para mensajería y TimescaleDB para persistencia de series de tiempo.

**Scope**: Exclusivamente infraestructura de mensajería y base de datos. NO incluye servicios de aplicación (ingestion, worker) ni su lógica. Se extiende el archivo `docker-compose.yml` existente que ya contiene servicios de aplicación.

**Objetivo**: Permitir a desarrolladores ejecutar localmente una capa de infraestructura reproducible, aislada e independiente de producción, en un único comando: `docker compose up`.

## Glosario

- **Kafka**: Bus de eventos que desacopla productores (ingestion) de consumidores (worker).
- **Topic**: Concepto de Kafka donde se publican eventos. Ejemplo: `telemetry.positions`.
- **Partición**: Subdivisión de un Topic para paralelismo y ordenamiento. Se particiona por `vehicle_id`.
- **Zookeeper**: Coordinador de metadatos que Kafka utiliza para gestionar el cluster (en modo legacy o transición).
- **TimescaleDB**: Extensión de PostgreSQL optimizada para series de tiempo de alta frecuencia.
- **Hypertable**: Tabla de TimescaleDB que distribuye automáticamente datos en segmentos de tiempo.
- **Volume**: Almacenamiento persistente en disco que survive reinicios de contenedor.
- **Network**: Red Docker interna que permite comunicación entre servicios por nombres de servicio (DNS).
- **Health Check**: Verificación periódica de que un servicio está listo y operativo.
- **Environment Variable**: Configuración inyectada en contenedores desde `.env` o `docker-compose.yml`.
- **Service**: Instancia de contenedor definida en `docker-compose.yml`.

## Requisitos

### RF-01: Servicio Zookeeper para coordinación

**Historia de usuario:** Como desarrollador, quiero que Zookeeper esté disponible como coordinador para Kafka, para que el broker gestione correctamente sus metadatos.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE servicio `zookeeper` SHALL iniciarse con imagen `confluentinc/cp-zookeeper:7.x` (versión específica pendiente de decisión)
2. WHILE Zookeeper ejecuta, THE servicio SHALL escuchar en puerto interno `2181` para coordinación con Kafka
3. WHEN Zookeeper inicia completamente, THE contenedor SHALL tener estado `healthy` según su health check
4. WHILE Zookeeper ejecuta, THE datos de coordinación SHALL persistirse en volumen `zookeeper-data`
5. WHEN Docker Compose se detiene, THE contenedor Zookeeper SHALL cerrarse gracefully

### RF-02: Servicio Kafka con Topic telemetry.positions

**Historia de usuario:** Como desarrollador, quiero que Kafka esté operativo localmente con el topic `telemetry.positions` pre-creado, para que la ingestion puede publicar eventos sin configuración manual.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE servicio `kafka` SHALL iniciarse con imagen `confluentinc/cp-kafka:7.x` (versión específica pendiente)
2. WHILE Kafka ejecuta, THE broker SHALL estar conectado a `zookeeper:2181` para coordinación
3. WHEN Kafka inicia completamente, THE Topic `telemetry.positions` SHALL existir con 3 particiones y 1 replication factor (para desarrollo local)
4. WHILE Kafka ejecuta, THE broker SHALL escuchar en puerto interno `9092` para clientes
5. WHERE el Topic `telemetry.positions` existe, THE particionamiento SHALL usar `vehicle_id` como clave para garantizar orden por vehículo
6. WHEN Kafka inicia por primera vez, THE configuración SHALL crear el Topic automáticamente (via init script o comando)
7. WHILE Kafka ejecuta, THE datos de Kafka (logs) SHALL persistirse en volumen `kafka-data`

### RF-03: Health checks para infraestructura

**Historia de usuario:** Como desarrollador, quiero que Docker Compose valide que los servicios de infraestructura estén listos, para evitar fallos por startup asincrónico.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE servicio `timescaledb` SHALL ejecutar health check cada 10 segundos usando `pg_isready`
2. WHEN el health check de TimescaleDB es exitoso, THE container alcanza estado `healthy`
3. WHEN se ejecuta `docker compose up`, THE servicio `kafka` SHALL ejecutar health check para validar que el broker está listo
4. WHILE los servicios son `healthy`, THE docker compose podrá considerar el stack como operativo

### RF-04: Red interna fleet-network

**Historia de usuario:** Como desarrollador, quiero que los servicios de infraestructura se comuniquen internamente sin exponerse innecesariamente, para mantener aislamiento y seguridad.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE Network `fleet-network` SHALL crearse si no existe (driver: bridge)
2. WHILE los servicios ejecutan, THE resolución de DNS interno SHALL permitir `zookeeper:2181`, `kafka:9092`, `timescaledb:5432`
3. WHEN un contenedor en la red se reinicia, THE nombre de servicio SHALL resolver a la nueva IP automáticamente
4. WHEN se ejecuta `docker compose down`, THE Network `fleet-network` SHALL removerse automáticamente

### RF-05: TimescaleDB como persistencia de telemetría

**Historia de usuario:** Como desarrollador, quiero ejecutar TimescaleDB localmente para almacenar telemetría de vehículos, sin depender de una base de datos externa.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE servicio `timescaledb` SHALL iniciarse con imagen `timescale/timescaledb:latest-pg15` (versión pendiente de decisión)
2. WHILE TimescaleDB ejecuta, THE servicio SHALL escuchar en puerto interno `5432` (PostgreSQL estándar)
3. WHEN TimescaleDB inicia, THE base de datos `fleet_db` SHALL crearse automáticamente
4. WHEN TimescaleDB inicia, THE usuario `fleet_user` SHALL crearse con contraseña desde variable `${POSTGRES_PASSWORD:-fleet_password}`
5. WHEN TimescaleDB inicia completamente, THE health check SHALL retornar exitoso dentro de 3 segundos
6. WHILE TimescaleDB ejecuta, THE datos SHALL persistirse en volumen `timescaledb-data`
7. WHEN Docker Compose se detiene, THE datos en el volumen SHALL preservarse para próximos inicios

### RF-06: Port mapping para TimescaleDB

**Historia de usuario:** Como desarrollador, quiero acceder a TimescaleDB desde herramientas externas (psql, pgAdmin) en mi máquina host.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE puerto host para TimescaleDB SHALL mapearse desde variable `${TIMESCALEDB_PORT:-5432}`
2. WHEN el puerto host está disponible, THE mapping SHALL ser `localhost:${TIMESCALEDB_PORT}:5432` (container)
3. WHERE el puerto ya está en uso, THE `docker compose up` SHALL fallar con error indicando el conflicto
4. WHEN el desarrollador configura `TIMESCALEDB_PORT=5433` en `.env`, THE puerto host SHALL ser `5433` en lugar del default

### RF-07: Volúmenes para persistencia

**Historia de usuario:** Como desarrollador, quiero que los datos persistan cuando reinicio Docker Compose, para no perder telemetría o state de Kafka durante desarrollo iterativo.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE volúmenes `kafka-data` y `timescaledb-data` SHALL crearse si no existen
2. WHILE los servicios ejecutan, THE escriben datos a estos volúmenes
3. WHEN se ejecuta `docker compose stop`, THE volúmenes permanecerán intactos
4. WHEN se ejecuta `docker compose up` nuevamente, THE volúmenes serán reutilizados y datos previos serán accesibles
5. WHEN se ejecuta `docker compose down` (sin `-v`), THE volúmenes NO deberán eliminarse
6. WHERE el desarrollador ejecuta `docker compose down -v`, THE volúmenes podrán eliminarse explícitamente

### RF-08: Variables de entorno en .env

**Historia de usuario:** Como desarrollador, quiero configurar los servicios mediante variables de entorno sin editar `docker-compose.yml`.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE archivo `.env` (si existe) SHALL cargarse automáticamente
2. WHEN se inicia docker-compose, THE variable `POSTGRES_USER` SHALL usarse en TimescaleDB (default: `fleet_user`)
3. WHEN se inicia docker-compose, THE variable `POSTGRES_PASSWORD` SHALL usarse en TimescaleDB (default: `fleet_password`)
4. WHEN se inicia docker-compose, THE variable `POSTGRES_DB` SHALL usarse como nombre de DB (default: `fleet_db`)
5. WHEN se inicia docker-compose, THE variable `TIMESCALEDB_PORT` SHALL usarse como puerto host (default: `5432`)
6. WHEN se inicia docker-compose, THE variable `KAFKA_HEAP_OPTS` SHALL configurar memoria de Kafka (default: `"`)
7. WHERE una variable no está definida, THE valor por defecto SHALL aplicarse

### RF-09: Archivo .env.example

**Historia de usuario:** Como desarrollador nuevo que clona el repositorio, quiero ver qué variables de entorno puedo personalizar.

#### Criterios de Aceptación

1. WHEN el repositorio se clona, THE archivo `.env.example` SHALL existir en raíz del proyecto
2. WHEN el desarrollador revisa `.env.example`, THE archivo SHALL listar todas las variables soportadas con comentarios
3. WHEN el desarrollador revisa `.env.example`, THE valores dummy (NO reales) deberán estar presentes para variables sensibles (contraseñas)
4. WHEN el desarrollador copia `.env.example` a `.env` y personaliza, THE Docker Compose usará los valores de `.env`

### RF-10: Documentación en docker-compose.yml

**Historia de usuario:** Como desarrollador, quiero que `docker-compose.yml` esté bien documentado para entender la estructura sin necesidad de investigar.

#### Criterios de Aceptación

1. WHEN el desarrollador abre `docker-compose.yml`, THE archivo SHALL tener sección de comentarios explicativa al inicio
2. WHEN el desarrollador revisa cada servicio, THE archivo SHALL documentar: qué es, qué imagen usa, qué puertos expone, qué volúmenes usa
3. WHEN el desarrollador revisa secciones de networks y volumes, THE archivo SHALL explicar su propósito
4. WHEN el desarrollador revisa dependencias entre servicios, THE archivo SHALL documentar el orden de startup

### RF-11: Dependencias de startup entre servicios

**Historia de usuario:** Como desarrollador, quiero que los servicios inicien en el orden correcto (Zookeeper → Kafka → TimescaleDB).

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up`, THE Zookeeper SHALL iniciar primero
2. WHEN Zookeeper está `healthy`, THE Kafka SHALL iniciar
3. WHEN Kafka está `healthy`, THE TimescaleDB SHALL estar en paralelo o después (sin dependencia explícita)
4. WHEN todos los servicios son `healthy`, THE stack está completamente operativo

### RF-12: Restart policies automáticas

**Historia de usuario:** Como desarrollador, quiero que los servicios se reinicien automáticamente si fallan inesperadamente.

#### Criterios de Aceptación

1. WHEN un contenedor Kafka falla, THE restart policy SHALL reintentar (policy: `unless-stopped`)
2. WHEN un contenedor TimescaleDB falla, THE restart policy SHALL reintentar (policy: `unless-stopped`)
3. WHEN un contenedor Zookeeper falla, THE restart policy SHALL reintentar (policy: `unless-stopped`)
4. WHERE el desarrollador ejecuta `docker compose stop`, THE servicios se detendrán sin reintentos
5. WHERE el desarrollador ejecuta `docker compose start`, THE servicios detenidos manualmente SHALL reiniciarse

### RF-13: Logs accesibles vía docker compose logs

**Historia de usuario:** Como desarrollador, quiero ver logs centralizados de los servicios para debugging sin entrar en cada contenedor.

#### Criterios de Aceptación

1. WHEN el desarrollador ejecuta `docker compose logs`, THE logs de zookeeper, kafka, timescaledb SHALL ser visibles
2. WHEN el desarrollador ejecuta `docker compose logs timescaledb`, THE solo logs de TimescaleDB SHALL mostrarse
3. WHEN el desarrollador ejecuta `docker compose logs -f`, THE logs SHALL mostrarse en tiempo real

### RF-14: Limpieza determinística

**Historia de usuario:** Como desarrollador, quiero poder limpiar completamente el stack con comandos simples.

#### Criterios de Aceptación

1. WHEN el desarrollador ejecuta `docker compose down`, THE contenedores deberán detenerse y removerse
2. WHEN el desarrollador ejecuta `docker compose down -v`, THE contenedores, volúmenes y redes se removerán
3. WHEN el desarrollador ejecuta `docker compose down`, THE volúmenes NO se eliminarán (preservar datos por defecto)

### RF-15: Compatibilidad multiplataforma

**Historia de usuario:** Como desarrollador en Windows, Mac o Linux, quiero que Docker Compose funcione sin cambios en cualquier plataforma.

#### Criterios de Aceptación

1. WHEN Docker Compose ejecuta en Docker Desktop (Windows/Mac), THE stack SHALL funcionar identicamente
2. WHEN Docker Compose ejecuta en Docker Engine (Linux), THE stack SHALL funcionar identicamente
3. WHEN Docker Compose ejecuta, THE API version SHALL ser 3.9

### RF-16: Seguridad mínima

**Historia de usuario:** Como desarrollador, quiero que TimescaleDB esté configurado con autenticación básica.

#### Criterios de Aceptación

1. WHEN TimescaleDB inicia, THE usuario `fleet_user` SHALL tener contraseña (NO acceso sin autenticación)
2. WHEN los contenedores ejecutan, THE procesos NO deberán ejecutarse como `root`
3. WHERE credenciales están en `.env`, THE archivo `.env` SHALL estar en `.gitignore`

### RF-17: Makefile para comandos de orquestación local

**Historia de usuario:** Como desarrollador, quiero un Makefile en la raíz del proyecto que proporcione comandos convenientes para desplegar, administrar y limpiar la infraestructura local, evitando memorizar comandos complejos de docker-compose.

#### Criterios de Aceptación

1. WHEN el desarrollador está en la raíz del proyecto, THE archivo `Makefile` SHALL existir
2. WHEN el desarrollador ejecuta `make help`, THE comando SHALL mostrar lista de targets disponibles con breve descripción
3. WHEN el desarrollador ejecuta `make up`, THE comando SHALL equivaler a `docker compose up -d` (inicia infraestructura en background)
4. WHEN el desarrollador ejecuta `make down`, THE comando SHALL equivaler a `docker compose down` (detiene infraestructura)
5. WHEN el desarrollador ejecuta `make logs`, THE comando SHALL mostrar logs en tiempo real de todos los servicios (`docker compose logs -f`)
6. WHEN el desarrollador ejecuta `make logs-timescaledb`, THE comando SHALL mostrar solo logs de TimescaleDB en tiempo real
7. WHEN el desarrollador ejecuta `make logs-kafka`, THE comando SHALL mostrar solo logs de Kafka en tiempo real
8. WHEN el desarrollador ejecuta `make status`, THE comando SHALL mostrar estado de todos los servicios (`docker compose ps`)
9. WHEN el desarrollador ejecuta `make clean`, THE comando SHALL ejecutar `docker compose down -v` para limpiar completamente (contenedores, volúmenes, redes)
10. WHEN el desarrollador ejecuta `make clean-hard`, THE comando SHALL ejecutar `docker system prune -f` además de `docker compose down -v` para limpiar recursos no utilizados
11. WHEN el desarrollador ejecuta `make restart`, THE comando SHALL ejecutar `docker compose restart` (reinicia todos los servicios sin recrearlos)
12. WHEN el desarrollador ejecuta `make build`, THE comando SHALL ejecutar `docker compose build` (reconstruye imágenes de servicios que lo requieran)
13. WHERE el Makefile tiene targets, THE `.PHONY` SHALL estar declarado explícitamente para prevenir conflictos con archivos
14. WHEN el desarrollador revisa el Makefile, THE archivo SHALL incluir comentarios explicando cada target
15. WHEN el desarrollador ejecuta un target que requiere estar en directorio raíz, THE Makefile debe validar o el target debe documentarse que requiere estar en raíz

## Requisitos No Funcionales

### RNF-01: Aislamiento del entorno local

Los servicios deberán estar completamente aislados de producción y del host sin interferencias.

#### Criterios de Aceptación

1. WHILE Docker Compose ejecuta, THE base de datos local NO deberá interferir con otras instancias PostgreSQL en el host
2. WHILE Docker Compose ejecuta, THE puertos NO deberán estar en conflicto con otros servicios (o usar variables configurables)
3. WHEN Docker Compose se detiene, THE host NO deberá quedar modificado (excepto volúmenes)

### RNF-02: Reproducibilidad

Dos desarrolladores ejecutando el mismo comando deberán obtener el mismo resultado.

#### Criterios de Aceptación

1. WHEN dos desarrolladores clonan el repositorio, THE `docker-compose.yml` será idéntico
2. WHEN ejecutan `docker compose up`, THE servicios iniciarán en el mismo orden
3. WHEN ejecutan tests, THE comportamiento será determinístico

### RNF-03: Rendimiento de startup

El stack deberá iniciar rápidamente sin bloquear el flujo de desarrollo.

#### Criterios de Aceptación

1. WHEN se ejecuta `docker compose up` (con imágenes en caché), THE stack operativo dentro de 30 segundos
2. WHEN se ejecuta `docker compose up` (primera ejecución), THE stack operativo dentro de 5 minutos

### RNF-04: Tolerancia de recursos

El stack deberá funcionar en máquinas de desarrollo típicas.

#### Criterios de Aceptación

1. WHEN Docker Compose ejecuta en 4 GB RAM, THE sin errores de memoria insuficiente
2. WHEN Docker Compose ejecuta en 2 CPU cores, THE sin throttling notable

## Matriz de Trazabilidad: Requisitos → Pruebas

| ID | Descripción | Suite | Ruta | Tipo |
|---|---|---|---|---|
| RF-01 | Servicio Zookeeper | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-02 | Kafka + Topic telemetry.positions | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-03 | Health checks | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-04 | Network fleet-network | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-05 | TimescaleDB setup | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-06 | Port mapping para TimescaleDB | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-07 | Volúmenes persistencia | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-08 | Variables .env | Unit | `tests/unit/config_test.go` | Go |
| RF-09 | .env.example | Manual | README review | Doc |
| RF-10 | Documentación docker-compose.yml | Manual | YAML review | Doc |
| RF-11 | Dependencias startup | Docker | `tests/docker/startup_order_test.go` | Go |
| RF-12 | Restart policies | Docker | `tests/docker/startup_order_test.go` | Go |
| RF-13 | Logs accesibles | Manual | CLI verification | CLI |
| RF-14 | Limpieza determinística | Docker | `tests/docker/cleanup_test.go` | Go |
| RF-15 | Compatibilidad multiplataforma | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-16 | Seguridad mínima | Docker | `tests/docker/infrastructure_test.go` | Go |
| RF-17 | Makefile para orquestación local | Manual | Makefile review | CLI |
| RNF-01 | Aislamiento local | Docker | `tests/docker/cleanup_test.go` | Go |
| RNF-02 | Reproducibilidad | Docker | `tests/docker/infrastructure_test.go` | Go |
| RNF-03 | Rendimiento startup | Performance | `tests/docker/startup_order_test.go` | Go |
| RNF-04 | Tolerancia recursos | Docker | `tests/docker/infrastructure_test.go` | Go |

## Decisiones Pendientes

1. **Versión de Kafka**: `confluentinc/cp-kafka:7.x` (decidir punto-x)
2. **Versión de Zookeeper**: `confluentinc/cp-zookeeper:7.x` (decidir punto-x)
3. **Versión de TimescaleDB**: `timescale/timescaledb:latest-pg15` vs versión fija (decidir)
4. **Kafdrop opcional**: Incluir en docker-compose (comentado) o documentar instalación manual
5. **Init script Kafka**: Script Go, shell script, o usar `confluentinc/cp-kafka` init
6. **Init script TimescaleDB**: Usar Dockerfile personalizado o script SQL en volumen
7. **Logging centralizado**: No en MVP; evaluar logging agregado en futuro
