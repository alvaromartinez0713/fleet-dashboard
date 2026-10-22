# Documento de Diseño: Configuración del Frontend con Docker Compose y Makefile

## 1. Visión General

Este documento define la arquitectura técnica, estructura de archivos y decisiones de diseño para integrar el frontend (React + TypeScript + Vite) en el entorno local containerizado del portal corporativo de monitoreo de flotas.

La solución proporciona:
- **Desarrollo ágil**: Hot reload (<3s) sin reinicio de contenedores
- **Simplicidad operacional**: Configuración de frontend sigue el patrón existente de otros servicios
- **Operación unificada**: Makefile centralizado para todas las operaciones
- **Documentación clara**: Configuración bien comentada y referenciable

## 2. Arquitectura de Componentes

El frontend se integra en la infraestructura existente manteniendo los patrones ya establecidos:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Host Machine (Local Dev)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                   Docker Network: fleet-network                  │   │
│  │                                                                  │   │
│  │  ┌────────────────────┐         ┌──────────────────────┐        │   │
│  │  │ frontend           │ HTTP    │ ingestion:8080       │        │   │
│  │  │ port: 3000         │────────→│ (API & SSE)          │        │   │
│  │  │ Vite dev server    │         └──────────────────────┘        │   │
│  │  │                    │                                          │   │
│  │  │ Volumes (RW):      │         ┌──────────────────────┐        │   │
│  │  │ • src/             │         │ kafka, timescaledb   │        │   │
│  │  │ • vite.config.ts   │         │ (existing services)  │        │   │
│  │  │ • package.json     │         └──────────────────────┘        │   │
│  │  └────────────────────┘                                          │   │
│  │                                                                  │   │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Makefile Targets                             │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │ make up              → docker-compose up -d                     │   │
│  │ make down            → docker-compose down                      │   │
│  │ make logs-frontend   → docker-compose logs -f frontend          │   │
│  │ make frontend-test   → npm run test (from ./frontend)           │   │
│  │ make frontend-build  → npm run build                            │   │
│  │ make frontend-dev    → npm run dev (host machine, rapid)        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Decisiones de Diseño de Infraestructura

1. **Sin Health Checks**: El frontend no incluye healthcheck en el Dockerfile ni en docker-compose.yml, coherente con la simplicidad de otros servicios
2. **Patrón de Configuración Simple**: build, ports, environment, volumes, depends_on, networks, restart
3. **Variables de Entorno**: VITE_* inyectadas desde `infrastructure/docker/.env`
4. **Volúmenes RW**: Permiten hot reload sin rebuild
5. **Comunicación por Red**: Usa service names internos (fleet-ingestion:8080)

## 3. Estructura de Archivos a Crear/Modificar

```
fleet-dashboard/
├── frontend/
│   ├── .env.example                    ← Documentación de variables (commiteado)
│   ├── .gitignore                      ← Incluir .env.local, dist/, node_modules/
│   ├── Dockerfile                      ← CREAR: dev/prod targets, SIN HEALTHCHECK
│   ├── package.json                    ← Existente, sin cambios
│   ├── vite.config.ts                  ← Verificar/actualizar config
│   ├── tsconfig.json
│   └── src/
│       └── (aplicación React)
│
├── infrastructure/
│   └── docker/
│       ├── docker-compose.yml          ← MODIFICAR: agregar frontend
│       └── .env                        ← MODIFICAR: agregar VITE_*
│
├── Makefile                            ← MODIFICAR: agregar frontend targets
│
└── README.md                           ← MODIFICAR: agregar Getting Started
```

**Notas sobre .env.local:**
- NO incluido en git (en .gitignore)
- Creado localmente por cada desarrollador
- Usado por `npm run dev` local, NO por Docker Compose

## 4. Dockerfile del Frontend (Sin HEALTHCHECK)

```dockerfile
# ============================================================================
# Build stage: Install dependencies and prepare base environment
# ============================================================================
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY package*.json ./

# Install all dependencies (including dev)
RUN npm ci

# Copy source code
COPY . .

# ============================================================================
# Development target: Vite dev server with hot reload
# ============================================================================
FROM builder AS dev

# Expose port for Vite dev server
EXPOSE 3000

# Default command: run dev server
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# ============================================================================
# Production build stage: Compile optimized artifacts
# ============================================================================
FROM builder AS prod-build

# Compile TypeScript and create optimized bundle
RUN npm run build

# ============================================================================
# Production target: Serve optimized artifacts
# ============================================================================
FROM nginx:latest-alpine AS prod

WORKDIR /app

# Copy only pre-built artifacts from builder
COPY --from=prod-build /app/dist /usr/share/nginx/html

# Expose standard port
EXPOSE 80

# Nginx start in foreground
CMD ["nginx", "-g", "daemon off;"]
```

**Decisiones de Diseño:**

- **Sin HEALTHCHECK**: Cada contenedor es simple, verificación externa si es necesaria
- **Node 20 Alpine**: LTS, pequeño, seguro para dev
- **Multietapa**: dev reutiliza builder, prod solo tiene dist/
- **npm ci**: Reproducible, respeta package-lock.json
- **--host 0.0.0.0**: Permite conexiones desde host (Docker)

## 5. Configuración en docker-compose.yml

```yaml
# ============================================================================
# Frontend Service - Add to existing docker-compose.yml
# ============================================================================

services:
  # ... existing services (zookeeper, kafka, timescaledb, ingestion, worker)

  # --------------------------------------------------------------------------
  # Frontend Service
  # --------------------------------------------------------------------------
  # React + Vite dashboard for fleet monitoring portal
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      target: dev              # Use dev target for local development
    container_name: fleet-frontend
    ports:
      - "${FRONTEND_PORT:-3000}:3000"
    environment:
      # API Configuration - inject from .env or use defaults
      VITE_API_BASE_URL: ${VITE_API_BASE_URL:-http://fleet-ingestion:8080}
      VITE_SSE_ENDPOINT: ${VITE_SSE_ENDPOINT:-http://fleet-ingestion:8080/api/vehicles/updates}
      VITE_DATA_MODE: ${VITE_DATA_MODE:-mock}
      VITE_MAPBOX_TOKEN: ${VITE_MAPBOX_TOKEN:-}
    volumes:
      # Source code mount for hot reload (read-write)
      - ./frontend/src:/app/src:rw
      # Configuration files (read-write for flexibility)
      - ./frontend/vite.config.ts:/app/vite.config.ts:rw
      - ./frontend/tsconfig.json:/app/tsconfig.json:rw
      - ./frontend/package.json:/app/package.json:ro
      # Exclude node_modules (use image copy, not host mount)
    depends_on:
      ingestion:
        condition: service_started
    networks:
      - fleet-network
    restart: unless-stopped
```

**Decisiones de Diseño:**

- **Build target dev**: Desarrollo con hot reload
- **Depends_on condition: service_started**: No bloquea, frontend funciona incluso si backend está iniciando
- **Port mapping variable**: ${FRONTEND_PORT:-3000}
- **Volumes RW**: Cambios sin rebuild
- **Sin HEALTHCHECK**: Simple, coherente con patrón existente
- **Sin labels**: No necesarios, simplificación

## 6. Variables de Entorno: infrastructure/docker/.env

```bash
# ============================================================================
# Fleet Platform - Docker Compose Environment Configuration
# ============================================================================

# === Frontend Service ===
FRONTEND_PORT=3000

# Frontend Environment Variables (VITE_ prefix per Vite conventions)
VITE_API_BASE_URL=http://localhost:8080
VITE_SSE_ENDPOINT=http://localhost:8080/api/vehicles/updates
VITE_DATA_MODE=mock
VITE_MAPBOX_TOKEN=

# === Backend Services ===
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=15s
HTTP_SHUTDOWN_TIMEOUT=30s

# === Kafka Configuration ===
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC=telemetry.positions
KAFKA_GROUP_ID=fleet-worker-group

# === TimescaleDB Configuration ===
TIMESCALEDB_PORT=5433
POSTGRES_USER=fleet_user
POSTGRES_PASSWORD=fleet_password
POSTGRES_DB=fleet_db
DB_HOST=timescaledb
DB_PORT=5432
DB_USER=fleet_user
DB_PASSWORD=fleet_password
DB_NAME=fleet_db
DB_SSLMODE=disable

# === Logging ===
LOG_LEVEL=info
LOG_FORMAT=json

# === Worker Configuration ===
WORKER_SHUTDOWN_TIMEOUT=30s
```

**Decisiones de Diseño:**

- **VITE_* en .env global**: Docker Compose inyecta al contenedor
- **Defaults coherentes**: localhost:8080 para desarrollo local
- **Variables documentadas**: Comentarios guían a desarrolladores

## 7. Frontend: frontend/.env.example

```bash
# ============================================================================
# Fleet Frontend - Environment Variables Documentation
# ============================================================================
# This file documents all available Vite frontend variables.
# DO NOT COPY OR USE THIS FILE IN PRODUCTION.
# Developers create .env.local from this template.

# === Data Source Configuration ===
# VITE_DATA_MODE: Where frontend gets vehicle data
# Values: mock (local fixtures) | api (backend HTTP)
VITE_DATA_MODE=mock

# === Backend HTTP API Configuration ===
# VITE_API_BASE_URL: Base URL for REST API requests
# Example: http://fleet-ingestion:8080 (Docker), http://localhost:8080 (host dev)
VITE_API_BASE_URL=http://localhost:8080

# === Server-Sent Events Configuration ===
# VITE_SSE_ENDPOINT: Full URL for real-time vehicle updates
VITE_SSE_ENDPOINT=http://localhost:8080/api/vehicles/updates

# === Mapbox Configuration ===
# VITE_MAPBOX_TOKEN: Mapbox API token for interactive maps
# Get from: https://account.mapbox.com/tokens/
# Leave empty to disable map features
VITE_MAPBOX_TOKEN=
```

## 8. Extensiones al Makefile

```makefile
# ============================================================================
# Frontend Service Targets (add to existing Makefile)
# ============================================================================

.PHONY: frontend-test frontend-lint frontend-format frontend-build \
        frontend-dev logs-frontend build-frontend

# Build Frontend Docker image
build-frontend:
	@echo "Building Frontend Docker image..."
	docker build -t fleet-frontend:dev --target dev ./frontend
	@echo "✓ Frontend image built: fleet-frontend:dev"

# Stream logs from frontend service only
logs-frontend:
	@echo "Streaming logs from Frontend (Ctrl+C to exit)..."
	docker compose -f $(DOCKER_COMPOSE_FILE) --env-file $(DOCKER_COMPOSE_ENV) logs -f frontend

# Run frontend tests in working directory
frontend-test:
	@echo "Running Frontend tests..."
	cd frontend && npm run test
	@echo "✓ Frontend tests completed"

# Run ESLint on frontend code
frontend-lint:
	@echo "Linting Frontend code..."
	cd frontend && npm run lint
	@echo "✓ Frontend linting completed"

# Format frontend code with Prettier
frontend-format:
	@echo "Formatting Frontend code..."
	cd frontend && npm run format
	@echo "✓ Frontend formatting completed"

# Build frontend distribution (production artifacts)
frontend-build:
	@echo "Building Frontend distribution..."
	cd frontend && npm run build
	@echo "✓ Frontend build completed: dist/"

# Run Vite dev server on host (rapid iteration, no container)
frontend-dev:
	@echo "Starting Frontend dev server on host..."
	@echo "Tip: Run 'make up' in another terminal to have backend services"
	cd frontend && npm run dev
```

**Decisiones de Diseño:**

- **Targets ejecutan localmente, no en Docker**: `cd frontend && npm ...` es más rápido
- **logs-frontend**: Acceso a logs específicos del frontend
- **build-frontend**: Opción de rebuild manual sin docker-compose

## 9. Configuración en vite.config.ts

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    // Allow connections from any host (important for Docker)
    host: '0.0.0.0',
    port: 3000,
    // HMR configuration for Docker
    hmr: {
      host: 'localhost',
      port: 3000,
    },
  },
  define: {
    // Default values for Vite environment variables
    'import.meta.env.VITE_API_BASE_URL': JSON.stringify('http://localhost:8080'),
    'import.meta.env.VITE_SSE_ENDPOINT': JSON.stringify('http://localhost:8080/api/vehicles/updates'),
    'import.meta.env.VITE_DATA_MODE': JSON.stringify('mock'),
    'import.meta.env.VITE_MAPBOX_TOKEN': JSON.stringify(''),
  },
  build: {
    target: 'ES2020',
    sourcemap: true,
    minify: 'terser',
  },
})
```

**Decisiones de Diseño:**

- **host: 0.0.0.0**: Permite conexiones desde Docker host
- **HMR config**: Asegura que hot reload funciona en Docker
- **Define defaults**: Fallback si .env no define valores

## 10. Patrón de Hot Reload

```
Developer saves file → Volume propagates (~100ms) → 
Vite detects change (~200ms) → Compila módulo (~1s) → 
Envía HMR WebSocket (~50ms) → Browser aplica cambios (~300ms) → 
TOTAL: ~1.65s (target: <3s)
```

**Sin HEALTHCHECK**, el flujo es más simple:
- Contenedor inicia normalmente
- Si Vite falla, docker logs muestra el error
- Restart policy reinicia si el proceso sale

## 11. Alineación con Patrones Existentes

La configuración del frontend **NO introduce nuevos patrones**:

| Aspecto | Patrón Existente | Frontend |
|--------|-----------------|----------|
| Docker Build | Multi-stage (ingestion, worker) | Multi-stage (dev/prod) ✓ |
| Ports | ${PORT:-default} | ${FRONTEND_PORT:-3000} ✓ |
| Environment | .env en infrastructure/docker | VITE_* en .env ✓ |
| Volumes | Montajes simples | ./src, ./*.ts en .rw ✓ |
| Network | fleet-network | fleet-network ✓ |
| Restart Policy | unless-stopped | unless-stopped ✓ |
| Healthcheck | NO | NO ✓ |
| Labels | NO | NO ✓ |

## 12. Criterios de Validación

- ✓ Dockerfile compila targets dev y prod sin error
- ✓ Docker Compose integra frontend sin romper servicios existentes
- ✓ `make up` inicia todos servicios incluyendo frontend en ~30s
- ✓ Frontend accesible en http://localhost:3000
- ✓ Hot reload: cambio en src/ → navegador actualiza en <3s, sin refresh
- ✓ `make logs-frontend` muestra logs de Vite
- ✓ `docker compose ps` muestra frontend como "Up"
- ✓ Variables VITE_* inyectadas correctamente desde .env
- ✓ Volúmenes montados, cambios reflejados sin rebuild
- ✓ Sin Health Checks en Dockerfile ni docker-compose.yml
