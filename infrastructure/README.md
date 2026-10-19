# Infrastructure

Este directorio contiene toda la configuración de infraestructura para el Portal de Monitoreo de Flotas, incluyendo ambientes de desarrollo y producción.

## Organización

```
infrastructure/
├── docker/               # Desarrollo local (Docker Compose)
│   ├── docker-compose.yml
│   ├── .env.example
│   ├── services/
│   ├── scripts/
│   └── README.md
├── terraform/            # Producción (AWS) - Futuro
└── README.md
```

## Ambientes

### Development (Docker Compose)

**Ubicación**: `infrastructure/docker/`

Ejecuta el stack completo localmente:
- Zookeeper (coordinador de Kafka)
- Kafka (bus de eventos)
- TimescaleDB (persistencia de telemetría)
- Ingestion API (backend Go)
- Worker (consumer de Kafka, procesador de eventos)

**Inicio rápido**:
```bash
cd infrastructure/docker
cp .env.example .env
docker compose up -d
```

Ver detalles en `infrastructure/docker/README.md`

### Production (AWS)

**Ubicación**: `infrastructure/terraform/` (futuro)

Desplegará en AWS:
- RDS PostgreSQL + TimescaleDB (administrado)
- EC2 o Fargate para API e ingesta
- MSK (Managed Streaming for Apache Kafka)
- S3, CloudWatch, VPC, etc.

Pendiente de implementación.

## Configuración

### Archivos de entorno

La configuración se divide por entorno y componente:

#### Docker Infrastructure (`infrastructure/docker/.env`)
- Variables para Docker Compose
- Puertos, credenciales locales, retención de datos
- **Gitignored**: Sí (excepto `.env.example`)

#### Backend Go - Production (`backend/go/.env.prod`)
- Variables de PRODUCCIÓN (RDS, MSK, etc.)
- **Gitignored**: Sí (nunca commitear secretos)
- **Inyección**: Via AWS Secrets Manager en CI/CD

#### Backend Go - Development (`backend/go/.env.local`)
- Overrides locales para desarrollo
- **Gitignored**: Sí (local-only)
- Ejemplo: DB_HOST=localhost, KAFKA_BROKERS=localhost:9092

#### Plantilla de producción (`backend/go/.env.example`)
- Template documentativo (sin valores reales)
- Muestra variables requeridas en producción
- Gittracked: Sí (documentación)

### Jerarquía de configuración

Backend Go carga variables en este orden:

1. `backend/go/.env.prod` (producción, Secrets Manager)
2. `backend/go/.env.local` (development overrides, local)
3. Archivo `.env` de Docker (via docker-compose env_file)
4. Variables del sistema operativo

## Inicio rápido

### Local (Docker Compose)

```bash
# 1. Desde raíz del proyecto
make up

# 2. Esperar 30 segundos
sleep 30

# 3. Verificar estado
make status

# 4. Ver logs
make logs-timescaledb
make logs-kafka

# 5. Probar endpoint
curl http://localhost:8080/health

# 6. Parar
make down
```

### Limpieza

```bash
# Parar y preservar volúmenes (datos se mantienen)
make down

# Parar y eliminar volúmenes (PÉRDIDA DE DATOS)
make clean-hard
```

## Seguridad

- **Nunca commitear archivos .env** (están gitignored)
- **Nunca commitear .env.prod** (contiene credenciales reales)
- **Usar .env.example como template** para documentación
- **En producción**: Inyectar via Secrets Manager, no en archivos
- **En local**: Usar .env.local para overrides sin tracking

## Referencias

- [Docker Development](./docker/README.md)
- [Backend Go Configuration](../backend/go/.env.example)

