# Implementation Plan: Backend Setup (Bootstrap/Scaffolding)

## Overview

Este plan cubre la implementación de la base fundacional del backend aplicando dos principios arquitectónicos complementarios:

1. **Vertical Slice Architecture** a nivel de `internal/`: cada servicio (`ingestion`, `worker`) posee su propio slice autónomo; no existe código compartido entre slices; `cmd/ingestion/main.go` importa únicamente `internal/ingestion/...` y `cmd/worker/main.go` importa únicamente `internal/worker/...`.

2. **Clean Architecture** dentro de cada slice: las capas se organizan como `domain → ports → adapters → config`, con las dependencias orientadas hacia el dominio; los tipos de dominio no conocen frameworks; los ports definen contratos; los adaptadores los implementan; y la infraestructura conecta las piezas.

El plan incluye: inicialización del módulo Go, creación de los dos slices (`internal/ingestion/`, `internal/worker/`), definición de tipos de dominio, ports, adaptadores HTTP del slice ingestion, infraestructura de configuración y logging para ambos slices, puntos de entrada, Makefile y configuración de ejecución local con Docker Compose.

Los tipos `TraceID`, `AppError`, `Logger` y `Config` están duplicados intencionalmente en cada slice, priorizando la autonomía y capacidad de extracción futura. No se implementan pruebas, casos de uso ni adaptadores de salida en esta Spec.

## Tasks

### Inicialización del módulo y estructura

- [x] 1. Inicializar el módulo Go y crear la estructura de directorios
  - [x] 1.1 Crear `backend/go/go.mod` con `module` path, versión `go 1.26.0` y dependencias directas: `github.com/gin-gonic/gin v1.9.1`, `go.uber.org/zap v1.27.0`, `github.com/google/uuid v1.6.0`; ejecutar `go mod tidy` para generar `go.sum`
    - Versiones semánticas exactas, sin rangos ni comodines
    - _Requirements: 1.2_
  - [x] 1.2 Crear los directorios de los dos slices bajo `backend/go/internal/`: `ingestion/domain`, `ingestion/ports`, `ingestion/http/middleware`, `ingestion/http/health`, `ingestion/config`, `worker/domain`, `worker/ports`, `worker/config`; y `cmd/ingestion`, `cmd/worker`
    - No crear directorios sin archivos; crear solo lo que tendrá archivos en esta Spec
    - _Requirements: 1.1_

### Slice ingestion — dominio y ports

- [x] 2. Definir dominio y ports del slice ingestion
  - [x] 2.1 Crear `internal/ingestion/domain/errors.go` — `package domain`; `AppError` con campos `Code string`, `Message string`, `Inner error`, `IsRecoverable bool`; funciones `NewAppError` y `NewRecoverableError`; implementar `Error()` y `Unwrap()`; sin imports de Gin, Kafka ni drivers SQL
    - _Requirements: 1.4_
  - [x] 2.2 Crear `internal/ingestion/domain/trace.go` — `package domain`; tipo `TraceID string`; `NewTraceID()` usando `github.com/google/uuid` (UUID v4); `ContextWithTraceID(ctx, tid)`; `TraceIDFromContext(ctx)` devuelve `("", false)` si no existe; sin imports de Gin ni zap
    - _Requirements: 8.2, 8.3_
  - [x] 2.3 Crear `internal/ingestion/ports/config.go` — `package ports`; `Config{HTTP{Port int, ReadTimeout, WriteTimeout, ShutdownTimeout time.Duration}, Logging{Level, Format string}}`; sin campos Kafka ni PostgreSQL; `ConfigLoader`; `Validator`; `ValidationError{Field, Rule, Message string}`
    - _Requirements: 2.1_
  - [x] 2.4 Crear `internal/ingestion/ports/logging.go` — `package ports`; interfaz `Logger` con métodos `Info`, `Warn`, `Error`, `Fatal`, `DPanic`, `Panic`, `WithTraceID(ctx, domain.TraceID)`, `GetTraceID(ctx)`, `WithFields(...Field)`; `Field{Key string; Value interface{}}`; función `BuildFields(pairs ...interface{}) []Field`
    - _Requirements: 8.1_
  - [x] 2.5 Crear `internal/ingestion/ports/server.go` — `package ports`; `HTTPServer` con `Start() error` y `Stop(ctx context.Context) error`; y `internal/ingestion/ports/health.go` — `HealthChecker` con `Check(ctx context.Context) (bool, error)`
    - _Requirements: 4.1, 6.1_

### Slice ingestion — adaptadores HTTP

- [x] 3. Implementar los adaptadores de entrada HTTP del slice ingestion
  - [x] 3.1 Crear `internal/ingestion/http/router.go` — `package ingestionhttp`; `NewRouter(cfg ports.Config, logger ports.Logger, healthChecker ports.HealthChecker) *Router`; `gin.New()`; `HandleMethodNotAllowed=true`; `ReadTimeout` y `WriteTimeout` de `cfg.HTTP`; cadena de middleware en orden: `trace_id → logging → recovery/error`; handlers `NoRoute` (404) y `NoMethod` (405) con Problem Details; `Start()` vincula el puerto e inicia en goroutine; `Stop(ctx)` llama a `http.Server.Shutdown`; `registerRoutes()` registra solo `GET /health`
    - _Requirements: 4.1, 4.2, 4.3, 4.4_
  - [x] 3.2 Crear `internal/ingestion/http/middleware/trace_id.go` — middleware Gin que extrae `X-Trace-ID`; si es UUID v4 válido (36 chars, formato `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) lo reutiliza; en cualquier otro caso genera uno nuevo con `domain.NewTraceID()`; almacena en contexto con `domain.ContextWithTraceID`; escribe header de respuesta `X-Trace-ID` antes de `c.Next()`
    - _Requirements: 8.2, 8.3, 8.4_
  - [x] 3.3 Crear `internal/ingestion/http/middleware/logging.go` — middleware Gin que registra método, path, status, `latency_ms`, `trace_id` por solicitud; lee TraceID del contexto con `domain.TraceIDFromContext` después de `c.Next()`; no registra query string, `Authorization`, `Cookie` ni cuerpo
    - _Requirements: 8.1_
  - [x] 3.4 Crear `internal/ingestion/http/middleware/error.go` — struct `ProblemDetails` con campos `Type string`, `Title string`, `Status int`, `Detail string`, `Instance string` y tags JSON en inglés (`json:"type"` etc.); recovery de panic → 500 Problem Details (stack trace solo en logs, nunca en respuesta); `NoRoute` → 404; `NoMethod` → 405; `Content-Type: application/problem+json` en todas las respuestas de error
    - _Requirements: 5.1, 5.2, 5.3, 5.4_
  - [x] 3.5 Crear `internal/ingestion/http/health/handler.go` — `HealthHandler` con `NewHealthHandler(checker ports.HealthChecker)`; `Handle(c *gin.Context)` llama a `checker.Check` con timeout de 1 segundo; responde 200 `{"status":"healthy"}` si `true`; 503 `{"status":"unhealthy"}` si `false` o error; sin incluir el texto del error en la respuesta
    - _Requirements: 6.1, 6.2, 6.3_

### Slice ingestion — infraestructura

- [x] 4. Implementar la infraestructura de configuración del slice ingestion
  - [x] 4.1 Crear `internal/ingestion/config/loader.go` — `package config`; `EnvConfigLoader`; lee con `os.LookupEnv` únicamente `HTTP_PORT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_SHUTDOWN_TIMEOUT`, `LOG_LEVEL`, `LOG_FORMAT`; sin campos Kafka ni PostgreSQL; `getEnvRequired` falla con nombre del campo si la variable no está definida; `parseInt`/`parseDuration` fallan con nombre del campo si el formato es incorrecto; el error identifica el campo sin incluir su valor
    - _Requirements: 2.1, 2.2, 2.3_
  - [x] 4.2 Crear `internal/ingestion/config/validator.go` — `package config`; `ConfigValidator` con `Validate(cfg ports.Config) []ports.ValidationError`; valida `HTTP.Port` en rango 1–65535; `HTTP.ReadTimeout` y `HTTP.WriteTimeout` ≥ 1s; `HTTP.ShutdownTimeout` entre 1s y 300s; cada `ValidationError` incluye `Field` y `Message` sin el valor del campo
    - _Requirements: 2.2, 2.3_
  - [x] 4.3 Crear `internal/ingestion/config/logger.go` — `package config`; `NewLogger(cfg ports.Config) (ports.Logger, func(), error)`; `ZapLogger` que envuelve `*zap.Logger` con codificación JSON y campos base `timestamp`, `level`, `message`; `WithTraceID` guarda TraceID en contexto; `GetTraceID` lo recupera e inyecta como campo `trace_id` en logs; cleanup llama `logger.Sync()`; sin tokens ni secretos en logs
    - _Requirements: 8.1, 8.5_
  - [x] 4.4 Crear `internal/ingestion/config/composer.go` — `package config`; `DependencyComposer`; `NewDependencyComposer(cfg ports.Config, logger ports.Logger)`; `ComposeAPI() (ports.HTTPServer, error)` instancia `HealthHandler`, los middleware y el `Router` y los conecta; si la construcción de una dependencia falla libera los recursos ya inicializados y retorna error
    - _Requirements: 3.1, 3.2_

### Slice worker — dominio, ports e infraestructura (copias independientes)

- [x] 5. Implementar el slice worker con copias independientes de dominio, ports e infraestructura
  - [x] 5.1 Crear `internal/worker/domain/errors.go` y `internal/worker/domain/trace.go` — `package domain`; copias independientes de `AppError` y `TraceID` del slice ingestion; el paquete no importa ni referencia `internal/ingestion`
    - _Requirements: 1.4, 1.6_
  - [x] 5.2 Crear `internal/worker/ports/config.go` — `package ports`; `Config{Worker{ShutdownTimeout time.Duration}, Logging{Level, Format string}}`; sin campos HTTP, Kafka ni PostgreSQL; `ConfigLoader`; `Validator`; `ValidationError`; y `internal/worker/ports/logging.go` — copia independiente de `Logger`, `Field`, `BuildFields`
    - _Requirements: 2.1, 8.1_
  - [x] 5.3 Crear `internal/worker/config/loader.go`, `validator.go` y `logger.go` — `package config`; loader lee únicamente `WORKER_SHUTDOWN_TIMEOUT`, `LOG_LEVEL`, `LOG_FORMAT`; validator valida `Worker.ShutdownTimeout` en rango 1–300s; logger idéntico a `internal/ingestion/config/logger.go` pero copia independiente; ningún archivo importa `internal/ingestion`
    - _Requirements: 2.1, 2.2, 2.3, 8.1_
  - [x] 5.4 Crear `internal/worker/config/composer.go` — `package config`; `DependencyComposer`; `NewDependencyComposer(cfg ports.Config, logger ports.Logger)`; `ComposeWorker() (func(), error)` devuelve función de limpieza de recursos (esqueleto); incluir comentario en inglés indicando que la lógica de consumo de mensajes se implementa en la Spec de ingesta
    - _Requirements: 3.4_

### Puntos de entrada y ejecución local

- [x] 6. Crear los puntos de entrada y la configuración de ejecución local
  - [x] 6.1 Crear `backend/go/cmd/ingestion/main.go`, `bootstrap.go`, `server.go` y `signals.go` — importa solo `internal/ingestion/...`; `main()` llama a `bootstrap.Run()` y maneja el código de salida; `bootstrap.go` coordina: load config → validate → NewLogger → ComposeAPI; `server.go` maneja: Start → SIGTERM → Stop con ShutdownTimeout → segunda señal fuerza os.Exit(1); `signals.go` encapsula el manejo de señales (setupSignalHandler, waitForSignal, waitForSecondSignal); sin reglas de negocio en `main.go`
    - _Requirements: 2.4, 3.2, 4.4, 7.1, 7.2, 7.3, 7.5_
  - [x] 6.2 Crear `backend/go/cmd/worker/main.go`, `bootstrap.go`, `lifecycle.go` y `signals.go` — importa solo `internal/worker/...`; `main()` llama a `bootstrap.Run()` y maneja el código de salida; `bootstrap.go` coordina: load config → validate → NewLogger → ComposeWorker; `lifecycle.go` maneja: worker ready → SIGTERM → cleanup con timeout de 10s → segunda señal fuerza os.Exit(1); `signals.go` encapsula el manejo de señales; comentario en inglés indicando que la lógica de consumo de mensajes se implementa en la Spec de ingesta
    - _Requirements: 2.4, 3.4, 7.4, 7.5_
  - [x] 6.3 Crear `backend/go/Makefile` — `build-ingestion` compila `cmd/ingestion` a `bin/ingestion` con `-mod=readonly`; `build-worker` compila `cmd/worker` a `bin/worker` con `-mod=readonly`; `run` levanta `docker compose up -d --build --wait`; `clean` elimina `bin/`
    - _Requirements: (base del proyecto)_
  - [x] 6.4 Crear `backend/go/Dockerfile` multi-stage con targets `ingestion` y `worker` (cada target construye su binario independiente desde `cmd/<service>` con `-mod=readonly` y lo empaqueta en imagen Alpine con usuario no-root); crear `docker-compose.yml` en la raíz con dos servicios independientes: `ingestion` (target: ingestion, puerto `${HTTP_PORT:-8080}:8080`, healthcheck en `/health`) y `worker` (target: worker, depends_on ingestion); comentario en inglés indicando que Kafka y TimescaleDB no se incluyen en este compose porque sus adaptadores se implementan en la Spec de ingesta; crear `backend/go/.env.example` (no en la raíz) organizado por secciones con comentarios: "Ingestion Service", "Worker Service" y secciones comentadas para "Future: Kafka Configuration" y "Future: PostgreSQL Configuration"; incluir las 7 variables del setup base con valores de marcador; sin secretos reales; crear `.gitignore` en la raíz del repositorio con patrones generales del monorepo (archivos de entorno, IDE, OS, logs, node_modules, build artifacts generales) y `backend/go/.gitignore` con patrones específicos de Go (binarios en `bin/`, `*.exe`, `*.test`, `*.out`, `vendor/`, archivos `.env` reales, binarios compilados `ingestion` y `worker`, coverage reports); ambos archivos deben conservar `.env.example` y `docker-compose.yml` (son infraestructura como código); incluir comentarios indicando que cada proyecto tiene su propio `.gitignore` para patrones específicos de su tecnología
    - _Requirements: 2.5, 6.3, 12.1, 12.2_

## Notes

- Vertical Slice Architecture: `internal/ingestion/` e `internal/worker/` son slices independientes; `cmd/ingestion` importa únicamente `internal/ingestion/`, `cmd/worker` importa únicamente `internal/worker/`.
- Los tipos `TraceID`, `AppError`, `Logger` y `Config` están duplicados intencionalmente en cada slice; esto es preferible a una abstracción compartida prematura.
- Kafka y PostgreSQL no están en los `Config` de esta Spec; se añaden en la Spec de ingesta cuando cada servicio los necesite.
- No se crean pruebas en esta Spec; se definirán por pilares en una Spec separada.
- `docker-compose.yml` arranca ambos servicios `ingestion` y `worker`; Kafka y TimescaleDB se añaden en la Spec de ingesta.

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2"] },
    { "id": 2, "tasks": ["2.1", "2.2", "2.3", "2.4", "2.5"] },
    { "id": 3, "tasks": ["3.1", "3.2", "3.3", "3.4", "3.5"] },
    { "id": 4, "tasks": ["4.1", "4.2", "4.3", "4.4", "5.1", "5.2", "5.3", "5.4"] },
    { "id": 5, "tasks": ["6.1", "6.2", "6.3", "6.4"] }
  ]
}
```
