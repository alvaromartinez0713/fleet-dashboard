# Plan de Implementación: Infraestructura Docker Compose (Kafka + TimescaleDB)

## Resumen

Este plan extiende el docker-compose.yml existente con servicios de infraestructura (Zookeeper, Kafka, TimescaleDB), agrega scripts de inicialización, un Makefile a nivel raíz para orquestación y tests de integración para verificar el stack completo.

La implementación se construye de forma incremental: configuración base → scripts → validación → documentación.

## Tareas

- [x] 1. Extender docker-compose.yml con servicios de infraestructura
  - [x] 1.1 Agregar definición del servicio Zookeeper
    - Agregar bloque de servicio para zookeeper con imagen confluentinc/cp-zookeeper:7.6.0
    - Configurar variables de entorno (ZOO_MY_ID, ZOO_PORT, ZOO_SERVERS)
    - Agregar health check usando el comando "ruok"
    - Montar volúmenes: zookeeper-data, zookeeper-logs
    - Establecer política de reinicio: unless-stopped
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-01, RF-03, RF-07, RF-10, RF-12_

  - [x] 1.2 Agregar definición del servicio Kafka
    - Agregar bloque de servicio para kafka con imagen confluentinc/cp-kafka:7.6.0
    - Configurar variables de entorno (KAFKA_BROKER_ID, KAFKA_ZOOKEEPER_CONNECT, KAFKA_ADVERTISED_LISTENERS, replication, partitions, retention)
    - Agregar health check usando el comando kafka-broker-api-versions
    - Montar volumen: kafka-data
    - Establecer política de reinicio: unless-stopped
    - Agregar depends_on: zookeeper con condition: service_healthy
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-02, RF-03, RF-07, RF-10, RF-11, RF-12_

  - [x] 1.3 Agregar definición del servicio TimescaleDB
    - Agregar bloque de servicio para timescaledb con imagen timescale/timescaledb:2.14.0-pg15
    - Configurar variables de entorno usando sintaxis ${VAR:-default} (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)
    - Agregar health check usando el comando pg_isready
    - Montar volumen: timescaledb-data
    - Montar script de inicialización: ./backend/go/scripts/init-timescaledb.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
    - Establecer política de reinicio: unless-stopped
    - Configurar mapeo de puertos: ${TIMESCALEDB_PORT:-5432}:5432
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-05, RF-03, RF-06, RF-07, RF-08, RF-10, RF-12_

  - [x] 1.4 Agregar sección de volúmenes
    - Definir volumen zookeeper-data con nombre: fleet-zookeeper-data
    - Definir volumen zookeeper-logs con nombre: fleet-zookeeper-logs
    - Definir volumen kafka-data con nombre: fleet-kafka-data
    - Definir volumen timescaledb-data con nombre: fleet-timescaledb-data
    - Agregar comentarios de documentación para los propósitos de los volúmenes
    - _Requirements: RF-07, RF-10_

  - [x] 1.5 Actualizar configuración de red
    - Verificar que fleet-network esté definida correctamente (ya existe en el archivo actual)
    - Agregar comentarios de documentación sobre resolución DNS
    - Asegurar que todos los servicios se conecten a fleet-network (comportamiento por defecto)
    - _Requirements: RF-04, RF-10_

  - [x] 1.6 Actualizar servicios existentes (ingestion, worker)
    - Actualizar depends_on del worker para incluir kafka y timescaledb con condition: service_healthy
    - Verificar referencias de depends_on del servicio ingestion si es necesario
    - Asegurar formato consistente con los nuevos servicios
    - _Requirements: RF-11_

- [x] 2. Checkpoint - Verificar sintaxis de docker-compose.yml
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [x] 3. Crear archivo de plantilla .env.example
  - [x] 3.1 Crear .env.example en la raíz del proyecto
    - Agregar encabezado de archivo con instrucciones para copiar a .env
    - Agregar HTTP_PORT con valor por defecto 8080 y comentario
    - Agregar HTTP_READ_TIMEOUT con valor por defecto 15s y comentario
    - Agregar HTTP_WRITE_TIMEOUT con valor por defecto 15s y comentario
    - Agregar HTTP_SHUTDOWN_TIMEOUT con valor por defecto 30s y comentario
    - Agregar LOG_LEVEL con valor por defecto info y comentario (opciones: debug, info, warn, error)
    - Agregar LOG_FORMAT con valor por defecto json y comentario (opciones: json, text)
    - Agregar WORKER_SHUTDOWN_TIMEOUT con valor por defecto 30s y comentario
    - Agregar POSTGRES_USER con valor por defecto fleet_user y comentario
    - Agregar POSTGRES_PASSWORD con valor de ejemplo "changeme" y comentario de advertencia de seguridad
    - Agregar POSTGRES_DB con valor por defecto fleet_db y comentario
    - Agregar TIMESCALEDB_PORT con valor por defecto 5432 y comentario sobre resolución de conflictos
    - Organizar en secciones con encabezados (HTTP API, Logging, Worker, TimescaleDB)
    - _Requirements: RF-08, RF-09, RF-16_

  - [x] 3.2 Actualizar .gitignore
    - Agregar .env a .gitignore si no está presente
    - Agregar .env.local a .gitignore
    - Agregar .env.*.local a .gitignore
    - Agregar comentario explicando por qué .env se ignora
    - _Requirements: RF-16_

- [x] 4. Crear script de inicialización init-timescaledb.sql
  - [x] 4.1 Crear backend/go/scripts/init-timescaledb.sql
    - Agregar encabezado de archivo con descripción y notas de diseño del esquema
    - Agregar CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE
    - Agregar CREATE SCHEMA IF NOT EXISTS telemetry
    - Crear tabla telemetry.positions con todos los campos:
      - captured_at TIMESTAMPTZ NOT NULL (columna de particionamiento)
      - vehicle_id VARCHAR(50) NOT NULL
      - latitude NUMERIC(10, 8) NOT NULL
      - longitude NUMERIC(11, 8) NOT NULL
      - accuracy NUMERIC(10, 2)
      - received_at TIMESTAMPTZ NOT NULL
      - processed_at TIMESTAMPTZ NOT NULL
      - stored_at TIMESTAMPTZ DEFAULT NOW()
      - event_id UUID NOT NULL UNIQUE
      - trace_id UUID NOT NULL
      - idempotency_key UUID
      - PRIMARY KEY (captured_at, vehicle_id)
    - Agregar SELECT create_hypertable con by_range('captured_at', INTERVAL '1 hour')
    - Crear índice idx_vehicle_captured ON telemetry.positions (vehicle_id, captured_at DESC)
    - Crear índice idx_event_id ON telemetry.positions (event_id)
    - Crear índice idx_trace_id ON telemetry.positions (trace_id)
    - Agregar SELECT add_compression_policy para datos mayores a 7 días
    - Agregar SELECT add_retention_policy para datos mayores a 30 días
    - Agregar sentencias GRANT para fleet_user sobre el esquema y las tablas
    - Agregar ALTER DEFAULT PRIVILEGES para tablas futuras
    - Agregar comentarios en línea explicando cada sección
    - _Requirements: RF-05, RF-08, RF-16_

- [x] 5. Crear script de inicialización create-kafka-topic.sh
  - [x] 5.1 Crear backend/go/scripts/create-kafka-topic.sh
    - Agregar shebang: #!/bin/bash
    - Agregar encabezado de archivo con descripción
    - Agregar sleep 15 para esperar a que Kafka esté completamente listo
    - Agregar comando kafka-topics --create con:
      - --bootstrap-server localhost:9092
      - --topic telemetry.positions
      - --partitions 3
      - --replication-factor 1
      - --config cleanup.policy=delete
      - --config retention.ms=86400000 (24 horas)
      - --if-not-exists
      - Redirigir stderr a /dev/null para suprimir advertencias de "ya existe"
    - Agregar sentencia echo confirmando la creación del topic
    - _Requirements: RF-02_

  - [x] 5.2 Hacer el script ejecutable
    - Ejecutar chmod +x backend/go/scripts/create-kafka-topic.sh
    - _Requirements: RF-02_

- [x] 6. Checkpoint - Verificar sintaxis de scripts de inicialización
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [x] 7. Crear Makefile a nivel raíz
  - [x] 7.1 Crear Makefile en la raíz del proyecto
    - Agregar encabezado de archivo con descripción e instrucciones de uso
    - Declarar .PHONY con todos los targets: help up down logs logs-timescaledb logs-kafka status clean clean-hard restart build
    - Establecer .DEFAULT_GOAL := help
    - Agregar target help con salida formateada listando todos los targets y descripciones
    - Agregar target up: docker compose up -d con mensaje de éxito mostrando URLs
    - Agregar target down: docker compose down con mensaje sobre volúmenes preservados
    - Agregar target restart: docker compose restart
    - Agregar target build: docker compose build
    - Agregar target status: docker compose ps con encabezado
    - Agregar target logs: docker compose logs -f con instrucciones para salir
    - Agregar target logs-kafka: docker compose logs -f kafka
    - Agregar target logs-timescaledb: docker compose logs -f timescaledb
    - Agregar target clean: docker compose down con mensaje
    - Agregar target clean-hard con prompt de confirmación, docker compose down -v, docker system prune -f
    - Agregar comentarios explicando cada sección
    - _Requirements: RF-17_

- [x] 9. Actualizar README.md con instrucciones de infraestructura
  - [x] 9.1 Agregar sección Quick Start
    - Agregar encabezado de sección: "## Quick Start"
    - Agregar prerrequisitos: Docker, Docker Compose
    - Agregar paso 1: Clonar repositorio
    - Agregar paso 2: Copiar .env.example a .env y personalizar
    - Agregar paso 3: Ejecutar make up (o docker compose up -d)
    - Agregar paso 4: Esperar 30 segundos para que los servicios estén healthy
    - Agregar paso 5: Verificar con make status
    - Agregar URLs para acceder a los servicios:
      - Ingestion API: http://localhost:8080
      - TimescaleDB: localhost:5432 (o ${TIMESCALEDB_PORT})
    - _Requirements: RF-09, RF-17_

  - [x] 9.2 Agregar sección Makefile Commands
    - Agregar encabezado de sección: "## Makefile Commands"
    - Agregar tabla con tres columnas: Command | Description | Usage
    - Documentar todos los 12 targets de make del Makefile
    - Agregar ejemplos para flujos de trabajo comunes (iniciar, ver logs, detener, limpiar)
    - _Requirements: RF-17_

  - [x] 9.3 Agregar sección Troubleshooting
    - Agregar encabezado de sección: "## Troubleshooting"
    - Agregar subsección: "Conflicto de puerto en 5432"
      - Explicar cómo establecer TIMESCALEDB_PORT en .env
      - Ejemplo: TIMESCALEDB_PORT=5433
    - Agregar subsección: "Servicios no healthy"
      - Comando para revisar logs: make logs
      - Comando para revisar estado: make status
      - Sugerencia: Aumentar límites de recursos de Docker
    - Agregar subsección: "Reinicio desde cero"
      - Comando: make clean-hard
      - Advertencia sobre pérdida de datos
    - Agregar subsección: "Error de topic ya existe"
      - Explicar comportamiento idempotente del script
      - Seguro ignorar si el topic existe
    - _Requirements: RF-06, RF-13_

  - [x] 9.4 Agregar sección Architecture
    - Agregar encabezado de sección: "## Architecture"
    - Agregar subsección: "Servicios de Infraestructura"
    - Describir rol de Zookeeper (coordinador de Kafka)
    - Describir rol de Kafka (bus de eventos, topic telemetry.positions)
    - Describir rol de TimescaleDB (persistencia de series temporales)
    - Agregar subsección: "Servicios de Aplicación"
    - Describir Ingestion API (receptor HTTP, publica en Kafka)
    - Describir Worker (consumidor de Kafka, escribe en TimescaleDB)
    - Agregar diagrama de flujo de datos (arte ASCII o referencia a design.md)
    - _Requirements: RF-10_

- [x] 10. Validación manual con smoke tests
  - [x] 10.1 Ejecutar inicio del stack completo
    - Desde la raíz del proyecto, ejecutar: make up
    - Esperar 30 segundos
    - Ejecutar: make status
    - Verificar que todos los servicios muestren "Up" o "Up (healthy)"
    - _Requirements: RF-01, RF-02, RF-05, RF-11, RNF-03_

  - [x] 10.2 Probar conectividad de TimescaleDB desde host
    - Ejecutar: psql -h localhost -p 5432 -U fleet_user -d fleet_db
    - Ingresar contraseña de POSTGRES_PASSWORD (por defecto: fleet_password)
    - Ejecutar: \dt telemetry.*
    - Verificar que la tabla telemetry.positions existe
    - Ejecutar: \d telemetry.positions
    - Verificar que todas las columnas coincidan con el diseño (captured_at, vehicle_id, latitude, longitude, etc.)
    - Salir de psql
    - _Requirements: RF-05, RF-06_

  - [x] 10.3 Probar existencia del topic de Kafka
    - Ejecutar: docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092
    - Verificar que la salida contiene "telemetry.positions"
    - Ejecutar: docker exec fleet-kafka kafka-topics --describe --topic telemetry.positions --bootstrap-server localhost:9092
    - Verificar: PartitionCount: 3
    - Verificar: ReplicationFactor: 1
    - _Requirements: RF-02_

  - [x] 10.4 Probar endpoint de health de Ingestion API
    - Ejecutar: curl http://localhost:8080/health
    - Verificar respuesta HTTP 200
    - _Requirements: RF-11_

  - [x] 10.5 Probar flujo end-to-end (manual)
    - Ejecutar: curl -X POST http://localhost:8080/positions -H "Content-Type: application/json" -d '{"vehicleId":"VH-SMOKE","latitude":40.7128,"longitude":-74.0060,"capturedAt":"2024-01-15T10:00:00Z"}'
    - Verificar respuesta HTTP 202 Accepted
    - Esperar 5 segundos para procesamiento del worker
    - Ejecutar: psql -h localhost -p 5432 -U fleet_user -d fleet_db -c "SELECT * FROM telemetry.positions WHERE vehicle_id='VH-SMOKE' ORDER BY captured_at DESC LIMIT 1"
    - Verificar que se devuelve una fila con latitud/longitud correctas
    - _Requirements: RF-02, RF-05, RF-11_

  - [x] 10.6 Probar acceso a logs
    - Ejecutar: make logs-timescaledb
    - Verificar que los logs se transmiten en tiempo real (Ctrl+C para salir)
    - Ejecutar: make logs-kafka
    - Verificar que los logs se transmiten en tiempo real (Ctrl+C para salir)
    - Ejecutar: make logs (sin filtro)
    - Verificar que los logs de todos los servicios se transmiten (Ctrl+C para salir)
    - _Requirements: RF-13_

  - [x] 10.7 Probar limpieza
    - Ejecutar: make down
    - Ejecutar: docker compose ps
    - Verificar que no hay contenedores ejecutándose
    - Ejecutar: docker volume ls | grep fleet
    - Verificar que los volúmenes aún existen (preservados)
    - Ejecutar: make up
    - Verificar que los servicios reinician con datos existentes
    - Ejecutar: make clean-hard (confirmar con 'y')
    - Ejecutar: docker volume ls | grep fleet
    - Verificar que los volúmenes se eliminaron
    - _Requirements: RF-14_

- [x] 11. Checkpoint final - Validación completa
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [x] 12. Reorganización de estructura de infraestructura y configuración
  - [x] 12.1 Crear carpeta `infrastructure/docker/`
    - Crear directorio `infrastructure/` en la raíz del proyecto
    - Crear subdirectorio `infrastructure/docker/`
    - Crear subdirectorios `infrastructure/docker/services/` (para configs específicas de servicios)
    - Crear subdirectorios `infrastructure/docker/scripts/` (para scripts de inicialización)
    - _Requirements: RNF-05_

  - [x] 12.2 Mover docker-compose.yml a infrastructure/docker/
    - Copiar actual `docker-compose.yml` (raíz) → `infrastructure/docker/docker-compose.yml`
    - 
    - Eliminar `docker-compose.yml` de la raíz (o dejar symlink)
    - Verificar que rutas en docker-compose.yml son relativas o correctas desde infrastructure/docker/
    - _Requirements: RNF-05_

  - [x] 12.3 Reorganizar archivos .env - Infraestructura Docker
    - Copiar actual `.env` (raíz) → `infrastructure/docker/.env` (mantener valores actuales)
    - Copiar actual `.env.example` (raíz) → `infrastructure/docker/.env.example`
    - Crear `infrastructure/docker/.env.example` con valores para desarrollo local:
      - ZOOKEEPER_CLIENT_PORT=2181
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_TOPIC=telemetry.positions
      - POSTGRES_USER=fleet_user
      - POSTGRES_PASSWORD=changeme
      - POSTGRES_DB=fleet_db
      - TIMESCALEDB_PORT=5433
      - DB_HOST=localhost
      - KAFKA_HOST=localhost
      - Agregar comentarios explicativos
    - _Requirements: RF-08_

  - [x] 12.4 Reorganizar archivos .env - Backend Go (Producción)
    - Crear `backend/go/.env.example` con variables de PRODUCCIÓN:
      - DB_HOST=timescaledb.rds.amazonaws.com (ejemplo)
      - DB_PORT=5432
      - DB_USER=fleet_prod_user
      - DB_PASSWORD=<change_me>
      - DB_NAME=fleet_prod_db
      - DB_SSL=true
      - KAFKA_BROKERS=kafka-prod-broker:9092
      - KAFKA_TOPIC=telemetry.positions.prod
      - HTTP_PORT=8080
      - HTTP_READ_TIMEOUT=15s
      - HTTP_WRITE_TIMEOUT=15s
      - HTTP_SHUTDOWN_TIMEOUT=30s
      - LOG_LEVEL=warn
      - LOG_FORMAT=json
      - Agregar comentarios explicativos (NO valores reales)
    - _Requirements: RF-08, RF-16_

  - [x] 12.5 Reorganizar archivos .env - Backend Go (Local)
    - Crear `backend/go/.env.local` (template comentado para desarrollo local):
      - Comentarios explicando que es para override local
      - Ejemplo comentado:
        # DB_HOST=localhost
        # DB_PORT=5433
        # DB_USER=fleet_user
        # DB_PASSWORD=fleet_password
        # DB_NAME=fleet_db
        # KAFKA_BROKERS=localhost:9092
        # LOG_LEVEL=debug
        # LOG_FORMAT=text
      - _Requirements: RF-16_

  - [x] 12.6 Crear .env.prod template en Backend Go
    - Crear `backend/go/.env.prod` (vacío pero gitignored):
      - Este archivo contendrá valores reales de producción
      - NO debe subirse a git (ya gitignored)
      - Será inyectado por CI/CD desde Secrets Manager
    - _Requirements: RF-16_

  - [x] 12.7 Actualizar .gitignore
    - Verificar que `.env` está en .gitignore (no debería estar)
    - Agregar `.env` a .gitignore si no está presente
    - Agregar `backend/go/.env.prod` a .gitignore
    - Agregar `backend/go/.env.local` a .gitignore
    - Agregar comentarios explicativos:
      # Environment-specific files (not tracked)
      .env
      .env.local
      backend/go/.env.prod
      backend/go/.env.local
    - _Requirements: RF-16_

  - [x] 12.8 Crear README en infrastructure/docker/
    - Crear `infrastructure/docker/README.md` con:
      - Sección: "Docker Compose Development Infrastructure"
      - Sección: "Quick Start" (copiar from Quick Start anterior)
      - Sección: "Environment Variables" explicando .env
      - Sección: "Services":
        - Zookeeper: role, port, health check
        - Kafka: role, brokers, topic telemetry.positions, partitions
        - TimescaleDB: role, port mapping, schema initialization
      - Sección: "Troubleshooting" (copiar del README principal)
      - Sección: "Scripts" (referencias a init-db.sh, create-topics.sh)
    - _Requirements: RF-10_

  - [x] 12.9 Crear README en infrastructure/
    - Crear `infrastructure/README.md` con:
      - Sección: "Infrastructure Organization"
      - Explicar que infrastructure/ contiene:
        - docker/: Development infrastructure (local Docker Compose)
        - (future) terraform/: Production infrastructure (AWS)
      - Sección: "Environments":
        - Development (Docker Compose local)
        - Production (AWS with Terraform - future)
      - Sección: "Configuration":
        - infrastructure/docker/.env: Local development variables
        - backend/go/.env.prod: Production variables (Secrets Manager)
        - backend/go/.env.local: Local overrides
      - Sección: "Getting Started":
        - Referencia a infrastructure/docker/README.md
    - _Requirements: RF-10_

  - [x] 12.10 Actualizar Makefile en raíz
    - Si existe, actualizar referencias a docker-compose.yml:
      - Cambiar `docker-compose.yml` → `infrastructure/docker/docker-compose.yml`
      - Cambiar `docker compose -f docker-compose.yml up` → `cd infrastructure/docker && docker compose up`
      - O usar `docker-compose -f infrastructure/docker/docker-compose.yml`
    - Actualizar referencias a .env:
      - Cambiar `.env` → `infrastructure/docker/.env`
    - Verificar que todos los targets (up, down, logs, etc.) funcionan desde raíz
    - _Requirements: RF-17_

  - [x] 12.11 Actualizar Makefile en backend/go/
    - Si existen referencias a .env de raíz, actualizar:
      - Cambiar rutas de .env a ../../../infrastructure/docker/.env si es necesario
    - Verificar que todos los targets funcionan desde backend/go/
    - _Requirements: RF-17_

  - [x] 12.12 Actualizar documentación principal (README.md raíz)
    - Actualizar sección Quick Start:
      - Cambiar "Copiar .env.example a .env" → "Copiar infrastructure/docker/.env.example a infrastructure/docker/.env"
      - Cambiar "docker compose up" → "make up (desde raíz)"
    - Actualizar sección Architecture:
      - Agregar nota sobre structure: "Ver infrastructure/ para detalles"
      - Agregar referencia a infrastructure/README.md
    - Mantener todo lo demás igual
    - _Requirements: RF-10_

  - [x] 12.13 Verificación final de estructura
    - Listar árbol de directorios:
      - Confirmar que infrastructure/docker/ existe con docker-compose.yml
      - Confirmar que backend/go/ tiene .env.example, .env.local, .env.prod
    - Confirmar que .gitignore ignora correctamente:
      - .env
      - backend/go/.env.prod
      - backend/go/.env.local
    - Confirmar que Makefile en raíz puede ejecutar: make up, make down, make logs
    - ~~Confirmar que tests pueden ejecutarse: make test-infrastructure~~ (tests removidos)
    - Confirmar que archivos no contienen secretos reales
    - _Requirements: RNF-05, RF-16_

## Notas

- **Sin property-based testing**: Esta funcionalidad es configuración IaC (Infrastructure as Code). Property-based testing no es aplicable según testing.md sección 13.
- **Tests de infraestructura**: Removidos (infraestructura es efímera). Validación mediante Makefile y smoke tests manuales.
- **Smoke tests manuales**: Las tareas 10.1-10.7 se realizan manualmente como validación final antes de declarar la funcionalidad completa.
- **Flujo de dependencias**: Las tareas se ejecutan en waves secuenciales: 1 (docker-compose) → 2 (checkpoint) → 3-4 (archivos) → 5 (scripts) → 6 (checkpoint) → 7 (Makefile) → 9 (README) → 10 (smoke tests) → 11 (checkpoint) → 12 (reorganización)
- **Seguridad de volúmenes**: docker compose down preserva volúmenes por defecto. Usar clean-hard solo cuando se desee reiniciar intencionalmente.
- **Conflictos de puerto**: Si TIMESCALEDB_PORT=5432 genera conflicto, cambiar a 5433 u otro puerto disponible en .env.
- **Timing de health check**: El primer inicio toma más tiempo (descargas de imágenes). Los inicios subsecuentes deberían ser <30s.
- **Creación de topic de Kafka**: El script es idempotente. Si el topic existe, --if-not-exists previene errores.
- **Inicialización de TimescaleDB**: El script SQL se ejecuta una vez en el primer inicio del contenedor. Los inicios subsecuentes se saltan (datos en volumen).
- **Estrategia de validación**: Smoke tests manuales (10.1-10.7) verifican el flujo end-to-end y comportamiento de la infraestructura.

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "1.5", "1.6"] },
    { "id": 2, "tasks": ["2"] },
    { "id": 3, "tasks": ["3.1", "3.2", "4.1"] },
    { "id": 4, "tasks": ["5.1"] },
    { "id": 5, "tasks": ["5.2"] },
    { "id": 6, "tasks": ["6"] },
    { "id": 7, "tasks": ["7.1"] },
    { "id": 8, "tasks": ["9.1"] },
    { "id": 9, "tasks": ["9.2", "9.3", "9.4"] },
    { "id": 10, "tasks": ["10.1", "10.2", "10.3", "10.4", "10.5", "10.6", "10.7"] },
    { "id": 11, "tasks": ["11"] },
    { "id": 12, "tasks": ["12.1", "12.2", "12.3", "12.4", "12.5", "12.6", "12.7", "12.8", "12.9", "12.10", "12.11", "12.12", "12.13"] }
  ]
}
```
