# Plan de Implementación: Infraestructura Docker Compose (Kafka + TimescaleDB)

## Resumen

Este plan extiende el docker-compose.yml existente con servicios de infraestructura (Zookeeper, Kafka, TimescaleDB), agrega scripts de inicialización, un Makefile a nivel raíz para orquestación y tests de integración para verificar el stack completo.

La implementación se construye de forma incremental: configuración base → scripts → validación → documentación.

## Tareas

- [ ] 1. Extender docker-compose.yml con servicios de infraestructura
  - [ ] 1.1 Agregar definición del servicio Zookeeper
    - Agregar bloque de servicio para zookeeper con imagen confluentinc/cp-zookeeper:7.6.0
    - Configurar variables de entorno (ZOO_MY_ID, ZOO_PORT, ZOO_SERVERS)
    - Agregar health check usando el comando "ruok"
    - Montar volúmenes: zookeeper-data, zookeeper-logs
    - Establecer política de reinicio: unless-stopped
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-01, RF-03, RF-07, RF-10, RF-12_

  - [ ] 1.2 Agregar definición del servicio Kafka
    - Agregar bloque de servicio para kafka con imagen confluentinc/cp-kafka:7.6.0
    - Configurar variables de entorno (KAFKA_BROKER_ID, KAFKA_ZOOKEEPER_CONNECT, KAFKA_ADVERTISED_LISTENERS, replication, partitions, retention)
    - Agregar health check usando el comando kafka-broker-api-versions
    - Montar volumen: kafka-data
    - Establecer política de reinicio: unless-stopped
    - Agregar depends_on: zookeeper con condition: service_healthy
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-02, RF-03, RF-07, RF-10, RF-11, RF-12_

  - [ ] 1.3 Agregar definición del servicio TimescaleDB
    - Agregar bloque de servicio para timescaledb con imagen timescale/timescaledb:2.14.0-pg15
    - Configurar variables de entorno usando sintaxis ${VAR:-default} (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)
    - Agregar health check usando el comando pg_isready
    - Montar volumen: timescaledb-data
    - Montar script de inicialización: ./backend/go/scripts/init-timescaledb.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
    - Establecer política de reinicio: unless-stopped
    - Configurar mapeo de puertos: ${TIMESCALEDB_PORT:-5432}:5432
    - Agregar comentarios de documentación en línea
    - _Requirements: RF-05, RF-03, RF-06, RF-07, RF-08, RF-10, RF-12_

  - [ ] 1.4 Agregar sección de volúmenes
    - Definir volumen zookeeper-data con nombre: fleet-zookeeper-data
    - Definir volumen zookeeper-logs con nombre: fleet-zookeeper-logs
    - Definir volumen kafka-data con nombre: fleet-kafka-data
    - Definir volumen timescaledb-data con nombre: fleet-timescaledb-data
    - Agregar comentarios de documentación para los propósitos de los volúmenes
    - _Requirements: RF-07, RF-10_

  - [ ] 1.5 Actualizar configuración de red
    - Verificar que fleet-network esté definida correctamente (ya existe en el archivo actual)
    - Agregar comentarios de documentación sobre resolución DNS
    - Asegurar que todos los servicios se conecten a fleet-network (comportamiento por defecto)
    - _Requirements: RF-04, RF-10_

  - [ ] 1.6 Actualizar servicios existentes (ingestion, worker)
    - Actualizar depends_on del worker para incluir kafka y timescaledb con condition: service_healthy
    - Verificar referencias de depends_on del servicio ingestion si es necesario
    - Asegurar formato consistente con los nuevos servicios
    - _Requirements: RF-11_

- [ ] 2. Checkpoint - Verificar sintaxis de docker-compose.yml
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [ ] 3. Crear archivo de plantilla .env.example
  - [ ] 3.1 Crear .env.example en la raíz del proyecto
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

  - [ ] 3.2 Actualizar .gitignore
    - Agregar .env a .gitignore si no está presente
    - Agregar .env.local a .gitignore
    - Agregar .env.*.local a .gitignore
    - Agregar comentario explicando por qué .env se ignora
    - _Requirements: RF-16_

- [ ] 4. Crear script de inicialización init-timescaledb.sql
  - [ ] 4.1 Crear backend/go/scripts/init-timescaledb.sql
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

- [ ] 5. Crear script de inicialización create-kafka-topic.sh
  - [ ] 5.1 Crear backend/go/scripts/create-kafka-topic.sh
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

  - [ ] 5.2 Hacer el script ejecutable
    - Ejecutar chmod +x backend/go/scripts/create-kafka-topic.sh
    - _Requirements: RF-02_

- [ ] 6. Checkpoint - Verificar sintaxis de scripts de inicialización
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [ ] 7. Crear Makefile a nivel raíz
  - [ ] 7.1 Crear Makefile en la raíz del proyecto
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

- [ ] 8. Tests de integración para infraestructura
  - [ ] 8.1 Crear tests/integration/infrastructure_test.go
    - Agregar build tag: //go:build integration
    - Importar testcontainers-go o usar comandos docker compose exec
    - Test: Zookeeper arranca y el health check pasa
      - Ejecutar docker compose up -d zookeeper
      - Esperar estado healthy (máximo 30s)
      - Verificar que el estado del contenedor sea "healthy"
      - Limpiar: docker compose down
      - _Requirements: RF-01, RF-03, RNF-03_
    
    - Test: Kafka arranca, se conecta a Zookeeper y está healthy
      - Ejecutar docker compose up -d zookeeper kafka
      - Esperar estado healthy de kafka (máximo 45s)
      - Verificar que el estado del contenedor kafka sea "healthy"
      - Limpiar: docker compose down
      - _Requirements: RF-02, RF-03, RF-11, RNF-03_
    
    - Test: El topic de Kafka telemetry.positions existe con configuración correcta
      - Ejecutar docker compose up -d zookeeper kafka
      - Esperar kafka healthy
      - Ejecutar: docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092
      - Afirmar: la salida contiene "telemetry.positions"
      - Ejecutar: docker exec fleet-kafka kafka-topics --describe --topic telemetry.positions --bootstrap-server localhost:9092
      - Afirmar: la salida muestra 3 particiones
      - Afirmar: la salida muestra replication factor 1
      - Limpiar: docker compose down
      - _Requirements: RF-02_
    
    - Test: TimescaleDB arranca, inicializa el esquema y está healthy
      - Ejecutar docker compose up -d timescaledb
      - Esperar estado healthy de timescaledb (máximo 30s)
      - Verificar que el estado del contenedor sea "healthy"
      - Limpiar: docker compose down
      - _Requirements: RF-05, RF-03, RNF-03_
    
    - Test: Creación de esquema y tabla de TimescaleDB
      - Ejecutar docker compose up -d timescaledb
      - Esperar estado healthy
      - Ejecutar psql: docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "\dn"
      - Afirmar: la salida contiene el esquema "telemetry"
      - Ejecutar psql: docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "\dt telemetry.*"
      - Afirmar: la salida contiene la tabla "telemetry.positions"
      - Ejecutar psql: docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "SELECT * FROM timescaledb_information.hypertables WHERE hypertable_name='positions'"
      - Afirmar: la consulta devuelve 1 fila (hypertable existe)
      - Limpiar: docker compose down
      - _Requirements: RF-05_
    
    - Test: Resolución DNS de red entre servicios
      - Ejecutar docker compose up -d zookeeper kafka timescaledb
      - Esperar a que todos estén healthy
      - Ejecutar: docker exec fleet-kafka ping -c 1 zookeeper
      - Afirmar: ping tiene éxito
      - Ejecutar: docker exec fleet-kafka ping -c 1 timescaledb
      - Afirmar: ping tiene éxito
      - Limpiar: docker compose down
      - _Requirements: RF-04_
    
    - Test: Persistencia de volumen después de stop/start
      - Ejecutar docker compose up -d timescaledb
      - Esperar healthy
      - Ejecutar psql INSERT: docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -c "INSERT INTO telemetry.positions (captured_at, vehicle_id, latitude, longitude, received_at, processed_at, event_id, trace_id) VALUES (NOW(), 'VH-TEST', 40.0, -74.0, NOW(), NOW(), gen_random_uuid(), gen_random_uuid())"
      - Ejecutar docker compose stop
      - Ejecutar docker compose start
      - Esperar healthy
      - Ejecutar psql SELECT: docker exec fleet-timescaledb psql -U fleet_user -d fleet_db -t -c "SELECT COUNT(*) FROM telemetry.positions WHERE vehicle_id='VH-TEST'"
      - Afirmar: count = 1 (datos persistidos)
      - Limpiar: docker compose down -v
      - _Requirements: RF-07, NRF-01_
    
    - Test: Mapeo de puerto para TimescaleDB
      - Ejecutar docker compose up -d timescaledb
      - Esperar healthy
      - Ejecutar psql desde host: psql -h localhost -p 5432 -U fleet_user -d fleet_db -c "SELECT 1"
      - Afirmar: la consulta tiene éxito (conexión desde host funciona)
      - Limpiar: docker compose down
      - _Requirements: RF-06_
    
    - Test: Variables de entorno desde .env se aplican
      - Crear .env.test temporal con TIMESCALEDB_PORT=5433
      - Ejecutar docker compose --env-file .env.test up -d timescaledb
      - Esperar healthy
      - Ejecutar: docker compose --env-file .env.test ps
      - Afirmar: el mapeo de puerto muestra "0.0.0.0:5433->5432/tcp"
      - Limpiar: docker compose --env-file .env.test down
      - Eliminar .env.test
      - _Requirements: RF-08_

  - [ ] 8.2 Crear tests/integration/startup_order_test.go
    - Agregar build tag: //go:build integration
    - Test: El orden de inicio respeta las dependencias
      - Ejecutar docker compose up -d (todos los servicios)
      - Capturar tiempos de inicio de contenedores desde docker inspect
      - Afirmar: tiempo de inicio de zookeeper < tiempo de inicio de kafka
      - Afirmar: kafka alcanza healthy antes de que worker comience a consumir
      - Limpiar: docker compose down
      - _Requirements: RF-11_

  - [ ] 8.3 Crear tests/integration/cleanup_test.go
    - Agregar build tag: //go:build integration
    - Test: docker compose down preserva volúmenes
      - Ejecutar docker compose up -d
      - Esperar healthy
      - Ejecutar docker compose down (sin -v)
      - Ejecutar docker volume ls
      - Afirmar: el volumen fleet-kafka-data existe
      - Afirmar: el volumen fleet-timescaledb-data existe
      - Limpiar: docker volume rm fleet-kafka-data fleet-timescaledb-data (limpieza manual)
      - _Requirements: RF-14_
    
    - Test: docker compose down -v elimina volúmenes
      - Ejecutar docker compose up -d
      - Esperar healthy
      - Ejecutar docker compose down -v
      - Ejecutar docker volume ls
      - Afirmar: el volumen fleet-kafka-data NO existe
      - Afirmar: el volumen fleet-timescaledb-data NO existe
      - _Requirements: RF-14_

- [ ] 9. Checkpoint - Ejecutar tests de integración
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

- [ ] 10. Actualizar README.md con instrucciones de infraestructura
  - [ ] 10.1 Agregar sección Quick Start
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

  - [ ] 10.2 Agregar sección Makefile Commands
    - Agregar encabezado de sección: "## Makefile Commands"
    - Agregar tabla con tres columnas: Command | Description | Usage
    - Documentar todos los 12 targets de make del Makefile
    - Agregar ejemplos para flujos de trabajo comunes (iniciar, ver logs, detener, limpiar)
    - _Requirements: RF-17_

  - [ ] 10.3 Agregar sección Troubleshooting
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

  - [ ] 10.4 Agregar sección Architecture
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

- [ ] 11. Validación manual con smoke tests
  - [ ] 11.1 Ejecutar inicio del stack completo
    - Desde la raíz del proyecto, ejecutar: make up
    - Esperar 30 segundos
    - Ejecutar: make status
    - Verificar que todos los servicios muestren "Up" o "Up (healthy)"
    - _Requirements: RF-01, RF-02, RF-05, RF-11, RNF-03_

  - [ ] 11.2 Probar conectividad de TimescaleDB desde host
    - Ejecutar: psql -h localhost -p 5432 -U fleet_user -d fleet_db
    - Ingresar contraseña de POSTGRES_PASSWORD (por defecto: fleet_password)
    - Ejecutar: \dt telemetry.*
    - Verificar que la tabla telemetry.positions existe
    - Ejecutar: \d telemetry.positions
    - Verificar que todas las columnas coincidan con el diseño (captured_at, vehicle_id, latitude, longitude, etc.)
    - Salir de psql
    - _Requirements: RF-05, RF-06_

  - [ ] 11.3 Probar existencia del topic de Kafka
    - Ejecutar: docker exec fleet-kafka kafka-topics --list --bootstrap-server localhost:9092
    - Verificar que la salida contiene "telemetry.positions"
    - Ejecutar: docker exec fleet-kafka kafka-topics --describe --topic telemetry.positions --bootstrap-server localhost:9092
    - Verificar: PartitionCount: 3
    - Verificar: ReplicationFactor: 1
    - _Requirements: RF-02_

  - [ ] 11.4 Probar endpoint de health de Ingestion API
    - Ejecutar: curl http://localhost:8080/health
    - Verificar respuesta HTTP 200
    - _Requirements: RF-11_

  - [ ] 11.5 Probar flujo end-to-end (manual)
    - Ejecutar: curl -X POST http://localhost:8080/positions -H "Content-Type: application/json" -d '{"vehicleId":"VH-SMOKE","latitude":40.7128,"longitude":-74.0060,"capturedAt":"2024-01-15T10:00:00Z"}'
    - Verificar respuesta HTTP 202 Accepted
    - Esperar 5 segundos para procesamiento del worker
    - Ejecutar: psql -h localhost -p 5432 -U fleet_user -d fleet_db -c "SELECT * FROM telemetry.positions WHERE vehicle_id='VH-SMOKE' ORDER BY captured_at DESC LIMIT 1"
    - Verificar que se devuelve una fila con latitud/longitud correctas
    - _Requirements: RF-02, RF-05, RF-11_

  - [ ] 11.6 Probar acceso a logs
    - Ejecutar: make logs-timescaledb
    - Verificar que los logs se transmiten en tiempo real (Ctrl+C para salir)
    - Ejecutar: make logs-kafka
    - Verificar que los logs se transmiten en tiempo real (Ctrl+C para salir)
    - Ejecutar: make logs (sin filtro)
    - Verificar que los logs de todos los servicios se transmiten (Ctrl+C para salir)
    - _Requirements: RF-13_

  - [ ] 11.7 Probar limpieza
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

- [ ] 12. Checkpoint final - Validación completa
  - Asegurar que todos los tests pasen, preguntar al usuario si surgen dudas.

## Notas

- **Sin property-based testing**: Esta funcionalidad es configuración IaC (Infrastructure as Code). Property-based testing no es aplicable según testing.md sección 13.
- **Tests de integración requieren Docker**: Los tests en la suite "integration" requieren Docker Engine con el plugin compose instalado.
- **Smoke tests manuales**: Las tareas 11.1-11.7 se realizan manualmente como validación final antes de declarar la funcionalidad completa.
- **Flujo de dependencias**: Las tareas 1-2 deben completarse antes que 3-5. La tarea 6 (tests) se bloquea en 1-5. Las tareas 7 (Makefile) y 10 (README) pueden ejecutarse en paralelo con los tests.
- **Seguridad de volúmenes**: docker compose down preserva volúmenes por defecto. Usar clean-hard solo cuando se desee reiniciar intencionalmente.
- **Conflictos de puerto**: Si TIMESCALEDB_PORT=5432 genera conflicto, cambiar a 5433 u otro puerto disponible en .env.
- **Timing de health check**: El primer inicio toma más tiempo (descargas de imágenes). Los inicios subsecuentes deberían ser <30s.
- **Creación de topic de Kafka**: El script es idempotente. Si el topic existe, --if-not-exists previene errores.
- **Inicialización de TimescaleDB**: El script SQL se ejecuta una vez en el primer inicio del contenedor. Los inicios subsecuentes se saltan (datos en volumen).
- **Estrategia de testing**: Los tests de integración verifican comunicación entre servicios. Los smoke tests verifican el flujo end-to-end manualmente.

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "1.5", "1.6"] },
    { "id": 2, "tasks": ["3.1", "3.2", "4.1"] },
    { "id": 3, "tasks": ["5.1"] },
    { "id": 4, "tasks": ["5.2", "7.1"] },
    { "id": 5, "tasks": ["8.1", "8.2", "8.3", "10.1"] },
    { "id": 6, "tasks": ["10.2", "10.3", "10.4"] },
    { "id": 7, "tasks": ["11.1", "11.2", "11.3", "11.4"] },
    { "id": 8, "tasks": ["11.5", "11.6", "11.7"] }
  ]
}
```
