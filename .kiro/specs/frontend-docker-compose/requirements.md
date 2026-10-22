# Documento de Requisitos: Configuración del Frontend con Docker Compose y Makefile

## Introducción

Este documento define los requisitos para integrar el frontend (React + TypeScript + Vite) en la orquestación local del proyecto mediante Docker Compose y Makefile.

El objetivo es proporcionar un entorno containerizado con hot reload para desarrollo, junto con comandos unificados de construcción, pruebas y formateo de código que se integren coherentemente con los servicios backend existentes.

Esta configuración constituye la **Fase 1** de la preparación del entorno de desarrollo local del portal corporativo de monitoreo de flotas.

## Glosario

- **Frontend**: Aplicación React web del portal corporativo
- **Docker Compose**: Herramienta de orquestación de contenedores para definir y ejecutar servicios
- **Vite**: Servidor de desarrollo y herramienta de construcción para aplicaciones web
- **Hot Reload**: Capacidad de reflejar cambios de código en el navegador sin recargar manualmente
- **Makefile**: Archivo que automatiza comandos de construcción y operación
- **Container Image**: Imagen de Docker que encapsula la aplicación y dependencias
- **Service**: Contenedor en ejecución dentro de Docker Compose
- **Network**: Red de Docker que permite comunicación entre contenedores
- **TypeScript**: Lenguaje tipado que compila a JavaScript
- **ESLint**: Herramienta de análisis estático de código JavaScript/TypeScript
- **Prettier**: Formateador automático de código
- **Vitest**: Framework de pruebas unitarias para aplicaciones Vite
- **Development Environment**: Entorno local para desarrolladores
- **Production Build**: Artefacto compilado y optimizado para despliegue
- **Health Check**: Mecanismo para verificar que un servicio está listo y funcionando

## Requisitos

### Requisito 1: Imagen de Docker del Frontend

**Historia de Usuario:** Como desarrollador, quiero una imagen de Docker para el frontend de modo que pueda ejecutar la aplicación en un contenedor aislado.

#### Criterios de Aceptación

1. WHEN the Frontend Service is built, THE Docker Image SHALL include Node.js runtime (LTS version), npm package manager, and all application dependencies installed from package.json.

2. THE Docker Image SHALL define a working directory at `/app` that contains the entire frontend source code and build artifacts.

3. WHERE the Build Target is development, THE Docker Image SHALL expose port 3000 for the Vite dev server and configure volume mounts for source code hot reload.

4. WHERE the Build Target is production, THE Docker Image SHALL contain only the pre-built distribution artifacts (dist/ directory) optimized for production deployment.

5. THE Docker Image SHALL use a minimal base image (Node.js Alpine or equivalent) to reduce image size and attack surface.

---

### Requisito 2: Servicio Frontend en Docker Compose

**Historia de Usuario:** Como operador de infraestructura, quiero que el frontend esté definido en Docker Compose de modo que se ejecute junto con los servicios backend.

#### Criterios de Aceptación

1. WHEN Docker Compose is executed from the project root, THE Frontend Service SHALL be started as a container named `fleet-frontend` and remain running in the background.

2. THE Frontend Service SHALL depend on the network `fleet-network` defined in the existing Docker Compose stack to enable service-to-service communication.

3. WHEN the Frontend Service is started in development mode, THE Vite Dev Server SHALL be accessible at `http://localhost:3000` from the host machine.

4. WHEN the Frontend Service is started, THE port 3000 SHALL be exposed and mapped to the host for external access.

5. THE Frontend Service configuration SHALL include environment variables for API endpoints (VITE_API_BASE_URL, VITE_SSE_ENDPOINT) injected from the `.env` file.

6. WHERE a source file in `/frontend/src` is modified, THE Hot Reload mechanism SHALL reflect the changes in the browser within 3 seconds without requiring container restart or manual refresh.

7. IF the Frontend Service fails to start or crashes, THE Docker Compose restart policy SHALL automatically restart it unless explicitly stopped.

---

### Requisito 3: Dockerfile del Frontend

**Historia de Usuario:** Como desarrollador, quiero un Dockerfile bien estructurado para el frontend de modo que pueda construir la imagen localmente y en CI/CD.

#### Criterios de Aceptación

1. THE Dockerfile SHALL define two build targets: `dev` (development image) and `prod` (production image).

2. WHERE the Build Target is dev, THE Dockerfile SHALL install dependencies using npm and configure the container to execute `npm run dev` as the default command.

3. WHERE the Build Target is prod, THE Dockerfile SHALL execute `npm run build` to produce optimized artifacts in the `dist/` directory, and configure a production web server (nginx or Node.js server) to serve the static files.

4. THE Dockerfile dev target SHALL mount the source code as a volume (`/app/src`, `/app/vite.config.ts`, `/app/package.json`) to enable live code changes.

5. THE Dockerfile prod target SHALL copy only the pre-built `dist/` directory (not the entire source) to minimize image size.

6. WHEN the production image is built, THE resulting image size SHALL not exceed 200 MB.

7. THE Dockerfile SHALL declare the application working directory as `/app`.

---

### Requisito 4: Variables de Entorno para Vite

**Historia de Usuario:** Como desarrollador, quiero que las variables de entorno del frontend se configuren automáticamente de modo que apunte a los servicios backend correctos.

#### Criterios de Aceptación

1. WHEN the Frontend Service is started, THE environment variable VITE_API_BASE_URL SHALL be set to the Backend Ingestion API endpoint (default: `http://localhost:8080`).

2. WHEN the Frontend Service is started, THE environment variable VITE_SSE_ENDPOINT SHALL be set to the Server-Sent Events endpoint for real-time updates (default: `http://localhost:8080/api/vehicles/updates`).

3. WHEN the Frontend Service is started, THE environment variable VITE_DATA_MODE SHALL be set to `mock` or `api` depending on the deployment context.

4. THE environment variables SHALL be injected into the container from the Docker Compose `.env` file located at `infrastructure/docker/.env`.

5. IF an environment variable is not explicitly set in the `.env` file, THE Frontend Service SHALL use sensible default values that allow it to connect to the Backend Services running in the local stack.

---

### Requisito 5: Integración con Docker Compose Existente

**Historia de Usuario:** Como desarrollador, quiero integrar el frontend en el Docker Compose existente de modo que se maneje de forma coherente con los otros servicios.

#### Criterios de Aceptación

1. WHEN Docker Compose up is executed, THE Frontend Service SHALL start after all infrastructure services (Zookeeper, Kafka, TimescaleDB) have declared their readiness.

2. THE Frontend Service configuration in docker-compose.yml SHALL reference the existing `fleet-network` bridge network for communication with Backend Services.

3. THE Frontend Service SHALL use the same environment file pattern as existing services (Docker Compose `.env` file at `infrastructure/docker/.env`).

4. WHEN the Frontend Service is running and a Backend Service dependency fails or restarts, THE Frontend Service SHALL remain operational but display connection errors in the UI when attempting to contact the Backend.

5. THE Docker Compose configuration SHALL clearly document the Frontend Service dependencies, ports, volumes, and environment variables for future maintenance.

---

### Requisito 6: Makefile: Comandos Unificados de Operación

**Historia de Usuario:** Como desarrollador, quiero comandos Makefile para operar el stack completo de modo que no tenga que recordar la sintaxis de Docker Compose.

#### Criterios de Aceptación

1. WHEN `make up` is executed, THE Makefile SHALL start all services including the Frontend Service and display status information.

2. WHEN `make down` is executed, THE Makefile SHALL stop all services and preserve volumes for the next start.

3. WHEN `make logs` is executed, THE Makefile SHALL stream logs from all services in real time, filtering by service name as needed.

4. WHERE a specific service is named (e.g., `make logs-frontend`), THE Makefile SHALL stream logs only from that service.

5. WHEN `make status` is executed, THE Makefile SHALL display the health status and port information of all running services.

6. THE Makefile SHALL document all targets with a help message displayed by `make help`.

7. WHEN `make clean` is executed, THE Makefile SHALL stop and remove all containers while preserving named volumes for data persistence.

8. WHEN `make clean-hard` is executed, THE Makefile SHALL remove all containers and volumes after user confirmation.

---

### Requisito 7: Makefile: Comandos Frontend Build y Test

**Historia de Usuario:** Como desarrollador, quiero comandos Makefile para ejecutar tareas del frontend (test, lint, format) de modo que tenga una interfaz consistente.

#### Criterios de Aceptación

1. WHEN `make frontend-test` is executed, THE Makefile SHALL execute `npm run test` inside the frontend directory and display test results.

2. WHEN `make frontend-lint` is executed, THE Makefile SHALL execute `npm run lint` to analyze code style and report violations without modifying files.

3. WHEN `make frontend-format` is executed, THE Makefile SHALL execute `npm run format` to automatically format all source files using Prettier.

4. WHEN `make frontend-build` is executed, THE Makefile SHALL execute `npm run build` to produce a production-optimized distribution in the frontend `dist/` directory.

5. WHEN `make frontend-dev` is executed, THE Makefile SHALL execute `npm run dev` to start the Vite dev server for local development (non-containerized, for rapid iteration).

6. THE Makefile SHALL display a clear success or failure message after each command execution.

7. THE Makefile commands for frontend SHALL be documented in the help output with descriptions.

---

### Requisito 8: Volume Mounts para Hot Reload

**Historia de Usuario:** Como desarrollador, quiero que los cambios de código se reflejen automáticamente en el navegador de modo que no tenga que reiniciar el contenedor.

#### Criterios de Aceptación

1. WHEN the Frontend Service is running in development mode, THE Docker Compose configuration SHALL mount the following paths as volumes:
   - `./frontend/src:/app/src` (application source code)
   - `./frontend/vite.config.ts:/app/vite.config.ts` (Vite configuration)
   - `./frontend/tsconfig.json:/app/tsconfig.json` (TypeScript configuration)
   - `./frontend/package.json:/app/package.json` (dependencies manifest)

2. WHEN a source file is modified on the host machine, THE mounted volume SHALL propagate the change to the container within 100ms.

3. WHEN the Vite Dev Server detects a source file change, THE browser client SHALL receive a hot module replacement (HMR) update within 3 seconds.

4. THE volume mounts SHALL be read-write to allow the dev server to generate temporary files and update source maps.

5. WHERE a file is deleted or renamed in the host source directory, THE Vite Dev Server SHALL detect the change and reload the application.

---

### Requisito 9: Configuración de Variables de Entorno

**Historia de Usuario:** Como desarrollador, quiero documentación clara de variables de entorno de modo que pueda customizar el frontend para diferentes escenarios.

#### Criterios de Aceptación

1. THE file `frontend/.env.example` SHALL be committed to version control and document all available VITE_* environment variables with clear comments.

2. THE environment variables for frontend (VITE_API_BASE_URL, VITE_SSE_ENDPOINT, VITE_DATA_MODE, VITE_MAPBOX_TOKEN) SHALL be configured in the Docker Compose `.env` file at `infrastructure/docker/.env`.

3. WHEN the Frontend Service container is started, THE environment variables from `infrastructure/docker/.env` SHALL be injected automatically into the container.

4. WHEN a developer needs to customize API endpoints locally, they MAY create `frontend/.env.local` which SHALL NOT be committed to version control (included in `.gitignore`).

5. IF `frontend/.env.local` does not exist, THE Vite dev server SHALL use defaults from `vite.config.ts`.

---

### Requisito 10: Configuración Simplificada del Frontend en Docker Compose

**Historia de Usuario:** Como desarrollador, quiero que la configuración del frontend en Docker Compose sea simple y coherente con otros servicios, sin complejidades innecesarias.

#### Criterios de Aceptación

1. THE Frontend Service configuration in docker-compose.yml SHALL follow the same pattern as existing services: build, ports, environment, volumes, depends_on, networks, restart.

2. WHERE the Frontend Service is started, THE Docker Compose configuration SHALL NOT include unnecessary features like health checks or labels that other services don't require.

3. WHEN the Frontend Service is running, port 3000 SHALL be accessible from the host at `http://localhost:3000`.

4. THE Frontend Service SHALL depend on the network `fleet-network` and be able to communicate with backend services using service names (e.g., `fleet-ingestion:8080`).

5. WHERE the Frontend Service fails or stops, THE restart policy SHALL automatically restart it unless explicitly stopped by the user.

---

### Requisito 11: Documentación de Ejecución

**Historia de Usuario:** Como nuevo desarrollador, quiero documentación clara de cómo ejecutar el frontend de modo que pueda comenzar rápidamente.

#### Criterios de Aceptación

1. THE project README.md file SHALL include a "Getting Started" section that documents the quickest way to run the entire stack locally.

2. THE README SHALL provide explicit commands for:
   - Building the Frontend Docker image
   - Starting all services with Docker Compose
   - Accessing the Frontend at http://localhost:3000
   - Stopping and cleaning up services

3. THE README SHALL explain the purpose and usage of key Makefile targets with examples.

4. THE README SHALL document the available environment variables and their default values.

5. WHERE a developer needs to customize API endpoints or data sources, THE README SHALL provide clear instructions for creating a `.env.local` file in the frontend directory.

6. THE README SHALL note any known limitations or prerequisites (Docker Desktop version, available memory, etc.).

---

### Requisito 12: Alineación con Patrones Existentes de Infraestructura

**Historia de Usuario:** Como arquitecto, quiero que Docker Compose y el Dockerfile del frontend sigan los patrones simples de los servicios backend, sin abstracciones innecesarias.

#### Criterios de Aceptación

1. THE Frontend Service configuration SHALL match the pattern of other services in docker-compose.yml: straightforward build, ports, environment, volumes, depends_on, networks, restart.

2. THE Frontend Dockerfile SHALL NOT include HEALTHCHECK directives; health verification happens externally if needed, not within the container.

3. WHEN the Frontend Dockerfile is built, it SHALL have dev and prod targets similar to backend services (ingestion, worker), without overcomplicating the setup.

4. THE Frontend Service SHALL avoid labels, advanced scheduling directives, or other Docker features that are not used by existing services.

5. WHEN integrating the Frontend with Docker Compose, no changes SHALL be made to existing backend service configurations (ingestion, worker, migrate, kafka, zookeeper, timescaledb).

---

### Requisito 13: Alcance Excluido

**Historia de Usuario:** Como gestor del proyecto, quiero claridad sobre qué NO incluye esta fase de modo que las expectativas estén alineadas.

#### Criterios de Aceptación

1. THIS Phase 1 shall NOT include CI/CD pipeline integration (GitHub Actions workflows). Phase 2 will address automated builds and deployments.

2. THIS Phase 1 shall NOT include production infrastructure deployments (AWS/Terraform). Infrastructure as Code is handled separately.

3. THIS Phase 1 shall NOT include mobile application containerization. Frontend web containerization is the scope.

4. THIS Phase 1 shall NOT include integration tests between Frontend and Backend services; Unit and component tests are in scope.

5. THIS Phase 1 shall NOT modify the existing Backend services, Kafka, or TimescaleDB configurations; those remain untouched.

6. THIS Phase 1 shall NOT include load testing or performance benchmarking; basic functionality and hot reload capability are verified.

---

## Aceptación General

- All Docker and Makefile configurations SHALL follow the naming conventions, patterns, and standards documented in `.kiro/steering/tech.md` and `.kiro/steering/structure.md`.
- All documentation and comments in configuration files SHALL be in English per project standards in `.kiro/steering/lenguaje.md` (code and technical documentation in English).
- All shell scripts and Makefile targets SHALL include error handling and clear status messages to aid troubleshooting.
