# Plan de Implementación: Configuración del Frontend con Docker Compose y Makefile

## Resumen

Este plan integra el frontend (React + TypeScript + Vite) en la orquestación local mediante Docker Compose y Makefile. La implementación se construye de forma incremental: Dockerfile → variables de entorno → servicio Docker Compose → comandos Makefile → validación → documentación.

## Tareas

- [x] 1. Crear frontend/Dockerfile (multietapa, sin HEALTHCHECK)
  - [x] 1.1 Crear archivo frontend/Dockerfile con stages dev y prod
    - Crear archivo frontend/Dockerfile
    - Stage builder: Instala deps con npm ci en /app
    - Stage dev: Expone puerto 3000, NO healthcheck, CMD = npm run dev --host 0.0.0.0
    - Stage prod-build: Ejecuta npm run build generando dist/
    - Stage prod: Usa nginx:alpine, copia solo dist/, sirve en puerto 80
    - Agregar comentarios en línea documentando cada stage
    - _Requisitos: 1, 3, 12_

  - [x] 1.2 Verificar que el Dockerfile dev target compila sin error
    - Ejecutar: docker build -t fleet-frontend:dev --target dev ./frontend
    - Verificar: Imagen creada exitosamente (sin errores)
    - Hecho cuando: docker build completa sin error y devuelve SHA de imagen

---

- [x] 2. Crear frontend/.env.example (documentación de variables)
  - [x] 2.1 Crear archivo frontend/.env.example
    - Crear archivo frontend/.env.example
    - Documentar VITE_DATA_MODE (con explicación: mock o api)
    - Documentar VITE_API_BASE_URL (con explicación y ejemplos para dev/prod)
    - Documentar VITE_SSE_ENDPOINT (con explicación)
    - Documentar VITE_MAPBOX_TOKEN (con enlace a procedimiento de obtención)
    - Agregar comentarios explicativos para cada variable
    - Agregar encabezado indicando propósito del archivo
    - _Requisitos: 4, 9_

  - [x] 2.2 Verificar que frontend/.env.example es válido
    - Verificar: Archivo contiene todas 4 variables VITE_*
    - Verificar: Comentarios claros y en inglés
    - Hecho cuando: Archivo existe con formato correcto, puede ser leído sin errores

---

- [x] 3. Verificar frontend/.gitignore
  - [x] 3.1 Revisar contenido de frontend/.gitignore
    - Revisar: ¿Está presente .env.local? (variable local NO commiteada)
    - Revisar: ¿Está presente dist/? (artefactos de build NO commiteados)
    - Revisar: ¿Está presente node_modules/? (dependencias NO commiteadas)
    - Si faltan líneas: Agregar las entradas faltantes
    - _Requisitos: 9_

  - [x] 3.2 Validar que .gitignore es correcto
    - Ejecutar: cat frontend/.gitignore | grep -E "(\.env\.local|dist|node_modules)"
    - Verificar: Comando devuelve 3+ coincidencias
    - Hecho cuando: grep devuelve todas las entradas esperadas

---

- [x] 4. Actualizar frontend/vite.config.ts
  - [x] 4.1 Revisar y actualizar vite.config.ts
    - Verificar: server.host = '0.0.0.0' (permite conexiones desde Docker)
    - Verificar: server.port = 3000
    - Verificar: HMR config con host: 'localhost', port: 3000 (para hot reload en Docker)
    - Verificar: Sección define con defaults para VITE_API_BASE_URL, VITE_SSE_ENDPOINT, VITE_DATA_MODE, VITE_MAPBOX_TOKEN
    - Agregar comentarios en línea documentando propósito de cada sección
    - _Requisitos: 4_

  - [x] 4.2 Validar que vite.config.ts es correcto
    - Ejecutar: npm run dev desde frontend/ y verificar que inicia sin error
    - Verificar: Mensaje de éxito "VITE v... ready in ... ms"
    - Hecho cuando: npm run dev inicia sin errores y acepta conexiones en puerto 3000

---

- [x] 5. Actualizar infrastructure/docker/.env (agregar VITE_*)
  - [x] 5.1 Editar infrastructure/docker/.env
    - Editar: infrastructure/docker/.env existente
    - Mantener: Todas variables backend existentes (NO borrar)
    - Agregar al final sección # === Frontend Service ===
    - Agregar FRONTEND_PORT=3000
    - Agregar VITE_API_BASE_URL=http://localhost:8080
    - Agregar VITE_SSE_ENDPOINT=http://localhost:8080/api/vehicles/updates
    - Agregar VITE_DATA_MODE=mock
    - Agregar VITE_MAPBOX_TOKEN= (vacío por defecto)
    - Agregar comentarios explicativos para cada variable
    - _Requisitos: 2, 4, 5_

  - [x] 5.2 Validar que infrastructure/docker/.env es válido
    - Ejecutar: docker compose config 2>&1 | grep -i error
    - Verificar: Comando devuelve vacío (sin errores de sintaxis)
    - Verificar: Archivo tiene ambas secciones (Backend Services y Frontend Service)
    - Verificar: No hay duplicados de variables
    - Hecho cuando: docker compose config ejecuta sin errores

---

- [x] 6. Actualizar infrastructure/docker/docker-compose.yml (agregar frontend service)
  - [x] 6.1 Agregar sección frontend bajo services
    - Editar: infrastructure/docker/docker-compose.yml
    - Insertar nuevo bloque frontend: (después de worker o ingestion)
    - Contenido del bloque:
      - build: { context: ./frontend, dockerfile: Dockerfile, target: dev }
      - container_name: fleet-frontend
      - ports: ["${FRONTEND_PORT:-3000}:3000"]
      - environment: Variables VITE_* desde .env
      - volumes: ./frontend/src:/app/src:rw, vite.config.ts:/app/vite.config.ts:rw, tsconfig.json, package.json
      - depends_on: ingestion (condition: service_started)
      - networks: [fleet-network]
      - restart: unless-stopped
    - SIN healthcheck, SIN labels, patrón simple
    - _Requisitos: 2, 5, 10, 12_

  - [x] 6.2 Validar que docker-compose.yml es correcto
    - Ejecutar: docker compose -f infrastructure/docker/docker-compose.yml config 2>&1 | tail -20
    - Verificar: docker compose config no devuelve errores
    - Verificar: Sección frontend: aparece en output con todas claves correctas
    - Verificar: Servicios existentes (zookeeper, kafka, timescaledb, ingestion, worker) NO se modificaron
    - Hecho cuando: docker compose config ejecuta sin errores y muestra sección frontend correctamente

---

- [x] 7. Agregar Makefile target: build-frontend
  - [x] 7.1 Agregar target build-frontend al Makefile
    - Editar: Makefile en raíz
    - Agregar: .PHONY: build-frontend
    - Agregar target con:
      - echo "Building Frontend Docker image..."
      - docker build -t fleet-frontend:dev --target dev ./frontend
      - echo "✓ Frontend image built: fleet-frontend:dev"
    - Ubicación: Agregar después de otros targets de construcción (cerca de build-api, etc.)
    - Agregar comentarios en línea
    - _Requisitos: 6_

  - [x] 7.2 Validar que make build-frontend funciona
    - Ejecutar: make build-frontend
    - Verificar: docker build completa sin error
    - Verificar: Imagen fleet-frontend:dev se crea exitosamente
    - Hecho cuando: make build-frontend ejecuta sin error y genera imagen

---

- [x] 8. Agregar Makefile target: logs-frontend
  - [x] 8.1 Agregar target logs-frontend al Makefile
    - Editar: Makefile en raíz
    - Agregar: .PHONY: logs-frontend
    - Agregar target con:
      - echo "Streaming logs from Frontend (Ctrl+C to exit)..."
      - docker compose -f $(DOCKER_COMPOSE_FILE) --env-file $(DOCKER_COMPOSE_ENV) logs -f frontend
    - Verificar: Makefile ya define DOCKER_COMPOSE_FILE y DOCKER_COMPOSE_ENV
    - Ubicación: Agregar después de otros targets de logs (cerca de logs-timescaledb, etc.)
    - _Requisitos: 6_

  - [x] 8.2 Validar que make logs-frontend funciona
    - Ejecutar: make up (esperar ~30s para que frontend inicie)
    - Ejecutar: make logs-frontend
    - Verificar: Muestra logs de Vite en tiempo real sin error
    - Presionar Ctrl+C para salir
    - Hecho cuando: make logs-frontend transmite logs sin errores

---

- [x] 9. Agregar Makefile targets: frontend-test, frontend-lint, frontend-format, frontend-build
  - [x] 9.1 Agregar target frontend-test
    - Editar: Makefile en raíz
    - Agregar: .PHONY: frontend-test
    - Contenido:
      - echo "Running Frontend tests..."
      - cd frontend && npm run test
      - echo "✓ Frontend tests completed"
    - _Requisitos: 7_

  - [x] 9.2 Agregar target frontend-lint
    - Editar: Makefile en raíz
    - Agregar: .PHONY: frontend-lint
    - Contenido:
      - echo "Linting Frontend code..."
      - cd frontend && npm run lint
      - echo "✓ Frontend linting completed"
    - _Requisitos: 7_

  - [x] 9.3 Agregar target frontend-format
    - Editar: Makefile en raíz
    - Agregar: .PHONY: frontend-format
    - Contenido:
      - echo "Formatting Frontend code..."
      - cd frontend && npm run format
      - echo "✓ Frontend formatting completed"
    - _Requisitos: 7_

  - [x] 9.4 Agregar target frontend-build
    - Editar: Makefile en raíz
    - Agregar: .PHONY: frontend-build
    - Contenido:
      - echo "Building Frontend distribution..."
      - cd frontend && npm run build
      - echo "✓ Frontend build completed: dist/"
    - Ubicación: Agregar en sección Frontend (nueva)
    - _Requisitos: 7_

  - [x] 9.5 Validar que Makefile targets funcionan
    - Ejecutar: make frontend-lint
    - Verificar: Ejecuta sin error (o reporta errores válidos de código)
    - Ejecutar: make frontend-build
    - Verificar: Genera directorio frontend/dist/ sin error
    - Hecho cuando: Todos targets ejecutan sin error fatal

---

- [x] 10. Agregar Makefile target: frontend-dev
  - [x] 10.1 Agregar target frontend-dev al Makefile
    - Editar: Makefile en raíz
    - Agregar: .PHONY: frontend-dev
    - Contenido:
      - echo "Starting Frontend dev server on host..."
      - echo "Tip: Run 'make up' in another terminal to have backend services"
      - cd frontend && npm run dev
    - Nota: Este target corre en host (NO Docker), para desarrollo rápido
    - Ubicación: Agregar en sección Frontend (nueva)
    - _Requisitos: 7_

  - [x] 10.2 Validar que make frontend-dev funciona
    - Ejecutar: make frontend-dev
    - Verificar: Inicia servidor Vite en http://localhost:3000
    - Presionar Ctrl+C para detener
    - Hecho cuando: make frontend-dev inicia servidor Vite sin error

---

- [x] 11. Actualizar Makefile target: help
  - [x] 11.1 Actualizar target help en Makefile
    - Editar: target help en Makefile
    - Agregar antes de final del target help:
      - Sección "Frontend Development:" con make frontend-dev, logs-frontend, build-frontend
      - Sección "Frontend Build & Test:" con make frontend-test, frontend-lint, frontend-format, frontend-build
    - Agregar comentarios explicativos para cada target
    - _Requisitos: 6_

  - [x] 11.2 Validar que make help muestra nuevas secciones
    - Ejecutar: make help
    - Verificar: Muestra nuevas secciones Frontend Development y Frontend Build & Test
    - Verificar: Todos targets listados correctamente
    - Hecho cuando: make help incluye nuevas secciones con formato correcto

---

- [x] 12. Validar hot reload funciona (<3 segundos)
  - [x] 12.1 Iniciar stack con Docker Compose
    - Ejecutar: make up
    - Esperar: ~30 segundos para que frontend esté "Up"
    - _Requisitos: 2, 8_

  - [x] 12.2 Verificar hot reload en navegador
    - Abrir: http://localhost:3000 en navegador
    - Editar: archivo frontend/src/App.tsx (cambiar texto visible)
    - Guardar: archivo
    - Observar: cambios deben reflejarse en navegador en <3 segundos sin refresh manual
    - Verificar: NO hay reinicio de contenedor, solo hot reload
    - Hecho cuando: Cambio visible en navegador en <3s sin refresh, sin error en console del navegador

---

- [x] 13. Validar integración Docker Compose
  - [x] 13.1 Verificar que all services inician correctamente
    - Ejecutar: make up
    - Verificar: Todos servicios inician sin errores (~30s)
    - _Requisitos: 2, 5, 6_

  - [x] 13.2 Verificar que frontend aparece en docker compose ps
    - Ejecutar: docker compose ps
    - Verificar: Frontend aparece en lista con estado "Up"
    - Verificar: Puerto 3000 mapeado correctamente
    - _Requisitos: 2, 5_

  - [x] 13.3 Verificar acceso HTTP a frontend
    - Ejecutar: curl http://localhost:3000/
    - Verificar: Devuelve HTML (no error)
    - _Requisitos: 2_

  - [x] 13.4 Verificar logs-frontend funciona
    - Ejecutar: make logs-frontend
    - Verificar: Muestra logs de Vite normalmente
    - Presionar: Ctrl+C para salir
    - _Requisitos: 6_

  - [x] 13.5 Detener servicios cleanly
    - Ejecutar: make down
    - Verificar: Todos servicios se detienen sin error
    - Ejecutar: docker ps
    - Verificar: No hay contenedores corriendo (vacío)
    - Hecho cuando: Todos comandos ejecutan sin error, servicios inician/detienen cleanly

---

- [x] 14. Validar variables de entorno inyectadas en container
  - [x] 14.1 Verificar variables VITE_* en container
    - Ejecutar: make up
    - Ejecutar: docker exec fleet-frontend env | grep VITE_
    - Verificar: Devuelve exactamente estas líneas:
      - VITE_API_BASE_URL=http://localhost:8080
      - VITE_SSE_ENDPOINT=http://localhost:8080/api/vehicles/updates
      - VITE_DATA_MODE=mock
      - VITE_MAPBOX_TOKEN= (vacío o no aparece)
    - _Requisitos: 4, 5_

  - [x] 14.2 Validar que variables son correctas
    - Verificar: 4 variables presentes y con valores correctos de .env
    - Hecho cuando: docker exec devuelve todas variables esperadas con valores correctos

---

- [x] 15. Actualizar README.md (Getting Started)
  - [x] 15.1 Agregar sección Getting Started - Local Development
    - Editar: README.md en raíz
    - Agregar sección "## Getting Started - Local Development" después de introducción
    - Incluir subsección "Prerequisites" con Docker Desktop, Node.js
    - Incluir subsección "Quick Start" con:
      - Comando: make up
      - URLs: Frontend http://localhost:3000, Backend API http://localhost:8080
      - Comando: make logs
      - Comando: make down
    - Incluir subsección "Makefile Commands" con:
      - Tabla o lista de targets principales: up, down, logs, logs-frontend, status, frontend-dev, frontend-test, frontend-lint, frontend-format, frontend-build
    - Incluir subsección "Environment Configuration" con:
      - Variables VITE_* explicadas
      - Cómo crear frontend/.env.local
    - Incluir subsección "Hot Reload" con:
      - Explicación de cómo funciona (<3s)
      - Instrucciones para usar make frontend-dev
    - Agregar referencias a .kiro/steering/ docs
    - _Requisitos: 11_

  - [x] 15.2 Validar que README.md es correcto
    - Verificar: Archivo tiene sección "Getting Started - Local Development"
    - Verificar: Todos subsecciones presentes
    - Verificar: Comandos documentados con ejemplos
    - Hecho cuando: README contiene documentación completa y clara

---

- [x] 16. Validación final integrada
  - [x] 16.1 Ejecutar secuencia completa de validación
    1. Ejecutar: make up → inicia todos servicios sin error (~30s)
    2. Ejecutar: make status → muestra frontend "Up" junto a otros servicios
    3. Ejecutar: make logs → muestra logs de todos servicios, incluyendo frontend
    4. Ejecutar: make logs-frontend → muestra solo logs de frontend (Vite)
    5. Ejecutar: make frontend-lint → ejecuta linting sin error (o reporta errores válidos)
    6. Ejecutar: make frontend-test → ejecuta tests (o skip si no hay tests)
    7. Ejecutar: make frontend-build → genera frontend/dist/ sin error
    8. Cambiar archivo en frontend/src/ → verificar hot reload en <3s
    9. Ejecutar: make down → detiene todos servicios sin error
    10. Verificar: docker ps devuelve vacío (ningún contenedor corre)
    11. Ejecutar: make help → muestra todos targets incluyendo frontend
    - _Requisitos: 1-12_

  - [x] 16.2 Verificar no hay secrets en logs/console
    - Verificar: Make logs NO exponen variables sensibles
    - Verificar: Console del navegador NO tiene errores críticos
    - _Requisitos: 1-12_

  - [x] 16.3 Checkpoint final - Validación completa
    - Verificar: Todos los comandos ejecutan sin error
    - Verificar: Frontend accesible en http://localhost:3000
    - Verificar: Hot reload funciona <3s
    - Verificar: docker-compose.yml válido (sin advertencias)
    - Verificar: Makefile targets documentados en help
    - Hecho cuando: Todos checks pass

---

## Notas

### Resumen de Cambios

| Archivo | Acción | Tarea |
|---------|--------|-------|
| `frontend/Dockerfile` | Crear | 1 |
| `frontend/.env.example` | Crear | 2 |
| `frontend/.gitignore` | Verificar | 3 |
| `frontend/vite.config.ts` | Actualizar | 4 |
| `infrastructure/docker/.env` | Modificar | 5 |
| `infrastructure/docker/docker-compose.yml` | Modificar | 6 |
| `Makefile` | Agregar 7 targets | 7-11 |
| `README.md` | Agregar sección | 15 |

### Decisiones de Diseño (Referencias a design.md)

1. **Sin HEALTHCHECK** (design.md Section 2.1, Section 4):
   - Simple, coherente con otros servicios backend
   - Verificación externa si se requiere
   - No se complica configuración por verificación internal

2. **Patrón Simple de Configuración** (design.md Section 2.1, Section 5):
   - build, ports, environment, volumes, depends_on, networks, restart
   - SIN labels, SIN features innecesarios
   - Mismo patrón que ingestion y worker

3. **Hot Reload <3 segundos** (design.md Section 10):
   - Volúmenes RW para src/, vite.config.ts, tsconfig.json, package.json
   - HMR configurado en vite.config.ts
   - WebSocket HMR activo

4. **Variables de Entorno Centralizadas** (design.md Section 6):
   - VITE_* en infrastructure/docker/.env (desarrollo local)
   - frontend/.env.local NO commiteado (local override)
   - frontend/.env.example SÍ commiteado (documentación)

5. **Makefile Unificado** (design.md Section 11):
   - Targets frontend integrados en Makefile raíz
   - Consistencia con targets backend existentes
   - help target actualizado

### Requisitos a Requisitos de Aceptación (requirements.md)

- Req 1 → Tasks 1.1, 1.2 (Imagen Docker del Frontend)
- Req 2 → Tasks 6.1, 6.2 (Servicio Frontend en Docker Compose)
- Req 3 → Tasks 1.1, 1.2 (Dockerfile del Frontend)
- Req 4 → Tasks 4.1, 4.2, 5.1, 5.2 (Variables de Entorno para Vite)
- Req 5 → Tasks 5.1, 5.2, 6.1, 6.2 (Integración con Docker Compose Existente)
- Req 6 → Tasks 7.1, 7.2, 8.1, 8.2, 11.1, 11.2 (Makefile: Comandos Unificados de Operación)
- Req 7 → Tasks 9.1-9.5, 10.1, 10.2 (Makefile: Comandos Frontend Build y Test)
- Req 8 → Tasks 6.1, 6.2, 12.1, 12.2 (Volume Mounts para Hot Reload)
- Req 9 → Tasks 2.1, 2.2, 3.1, 3.2 (Configuración de Variables de Entorno)
- Req 10 → Tasks 6.1, 6.2 (Configuración Simplificada del Frontend en Docker Compose)
- Req 11 → Tasks 15.1, 15.2 (Documentación de Ejecución)
- Req 12 → Tasks 1.1, 1.2, 6.1, 6.2 (Alineación con Patrones Existentes de Infraestructura)

---

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["2.2", "3.1"] },
    { "id": 3, "tasks": ["3.2", "4.1"] },
    { "id": 4, "tasks": ["4.2", "5.1"] },
    { "id": 5, "tasks": ["5.2", "6.1"] },
    { "id": 6, "tasks": ["6.2", "7.1"] },
    { "id": 7, "tasks": ["7.2", "8.1"] },
    { "id": 8, "tasks": ["8.2", "9.1", "9.2"] },
    { "id": 9, "tasks": ["9.3", "9.4", "9.5"] },
    { "id": 10, "tasks": ["10.1", "10.2", "11.1"] },
    { "id": 11, "tasks": ["11.2", "12.1"] },
    { "id": 12, "tasks": ["12.2", "13.1"] },
    { "id": 13, "tasks": ["13.2", "13.3", "13.4"] },
    { "id": 14, "tasks": ["13.5", "14.1"] },
    { "id": 15, "tasks": ["14.2", "15.1"] },
    { "id": 16, "tasks": ["15.2", "16.1"] },
    { "id": 17, "tasks": ["16.2", "16.3"] }
  ]
}
```
