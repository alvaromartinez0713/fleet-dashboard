# Requirements Document

## Introduction

Este documento define los requisitos para la configuración base (bootstrap) del backend del Portal Corporativo de Monitoreo de Flotas y Telemetría. El alcance cubre la inicialización del módulo Go, la estructura de directorios, la carga y validación de configuración, la composición de dependencias, el servidor HTTP con Gin, el manejo de errores con Problem Details, el health endpoint, el apagado controlado y el registro estructurado con `trace_id`. No se incluye lógica de negocio, adaptadores de mensajería ni persistencia.

La solución aplica **Vertical Slice Architecture** dentro de `internal/`: cada servicio (`api`, `worker`) posee su propio slice autónomo con sus propios tipos de dominio, ports, adaptadores e infraestructura de configuración. No existe código compartido entre slices; cuando los tipos son idénticos en ambos servicios se duplican intencionalmente, priorizando la capacidad de extracción futura de cada servicio sobre la abstracción prematura.

Esta base fundacional permite que las especificaciones siguientes construyan sobre una estructura de dependencias bien definida, con ports e interfaces ya establecidos en cada slice. Los dos binarios independientes (`api` y `worker`) se inicializan aquí con su flujo de arranque, composición de dependencias y apagado controlado, dejando la integración con Kafka y TimescaleDB para la Spec de ingesta.

## Glossary

- **Backend**: conjunto de servicios de servidor que procesan solicitudes HTTP, mensajes de Kafka y datos de telemetría.
- **Ingestion_Service**: binario `cmd/ingestion` que expone el servidor HTTP y atiende las solicitudes del portal web y la aplicación móvil.
- **Worker_Service**: binario `cmd/worker` que consume mensajes del bus de eventos y coordina su procesamiento.
- **Config_Loader**: componente de cada slice que lee la configuración exclusivamente desde variables de entorno y la expone como una estructura tipada específica del servicio.
- **HTTP_Router**: adaptador de entrada del slice `ingestion` que configura el motor Gin, registra rutas y aplica la cadena de middleware; reside en `internal/ingestion/http/`.
- **Error_Handler**: componente del slice `ingestion` que captura errores inesperados, panics y rutas no encontradas, y los transforma en respuestas Problem Details conformes a RFC 9457; reside en `internal/ingestion/http/middleware/error.go`.
- **Problem_Details**: formato estándar de respuesta de error definido por RFC 9457, con `Content-Type: application/problem+json` y los miembros `type`, `title`, `status`, `detail` e `instance`.
- **Health_Endpoint**: endpoint `GET /health` que informa del estado operativo del servicio; el handler reside en `internal/ingestion/http/health/handler.go`.
- **Graceful_Shutdown**: proceso de apagado controlado que deja de aceptar nuevas solicitudes, finaliza las solicitudes en curso dentro de un timeout y libera todos los recursos adquiridos.
- **Go_Module**: unidad de versionado de código Go definida por `go.mod` con versión de Go fijada y dependencias directas con versiones exactas.
- **Clean_Architecture**: principio de organización del código con dependencias orientadas hacia el dominio, aplicado aquí mediante Vertical Slice Architecture: cada servicio posee su propio slice (`internal/ingestion/`, `internal/worker/`) con sus propias capas `domain`, `ports` e infraestructura; los slices no se importan entre sí.
- **TraceID**: identificador de correlación por solicitud, de tipo `string` alias, con formato UUID v4, generado en el middleware de la API o reutilizado desde el header `X-Trace-ID` del cliente; definido de forma independiente en el dominio de cada slice.
- **Structured_Logger**: componente de registro que emite entradas en formato JSON con campos estructurados, incluyendo `timestamp`, `level`, `message` y `trace_id`; implementado de forma independiente en la infraestructura de cada slice.
- **Service**: término genérico que abarca tanto al `API_Service` como al `Worker_Service`.

## Requirements

### Requisito 1 — Estructura del proyecto y módulo Go

**User Story:** Como desarrollador, quiero una estructura de proyecto con Vertical Slice Architecture, para que cada servicio sea autónomo e independientemente extractable sin refactorización de paquetes compartidos.

#### Criterios de aceptación

1. THE `Backend` SHALL organizar el código bajo `backend/go/` con: `cmd/ingestion/main.go`, `cmd/worker/main.go`, y dos slices en `internal/`: `internal/ingestion/` (con subdirectorios `domain`, `ports`, `http`, `http/middleware`, `http/health`, `config`) e `internal/worker/` (con subdirectorios `domain`, `ports`, `config`); no se crearán directorios sin archivos de código.
2. THE `Go_Module` SHALL definirse en un único `backend/go/go.mod` con versión de Go fijada a `go 1.26.0` y dependencias directas con versiones semánticas exactas, sin rangos ni comodines.
3. THE `cmd/ingestion/main.go` SHALL importar únicamente paquetes de `internal/ingestion/`; THE `cmd/worker/main.go` SHALL importar únicamente paquetes de `internal/worker/`; los slices no se importan entre sí.
4. THE paquete `internal/ingestion/domain` y THE paquete `internal/worker/domain` no SHALL importar Gin, Kafka, drivers SQL ni bibliotecas de infraestructura.
5. THE paquetes bajo `internal/ingestion/ports` y `internal/worker/ports` no SHALL importar implementaciones concretas de adaptadores ni bibliotecas de infraestructura.
6. WHERE código idéntico exista en ambos slices, THE `Backend` SHALL mantenerlo duplicado en cada slice en lugar de extraerlo a un paquete compartido; la duplicación es preferible a una abstracción prematura.

### Requisito 2 — Carga y validación de configuración

**User Story:** Como operador, quiero que la configuración se cargue exclusivamente desde variables de entorno y se valide al arranque, para que el servicio no se inicie con configuración incorrecta o incompleta.

#### Criterios de aceptación

1. WHEN el `Service` arranca, THE `Config_Loader` SHALL cargar la configuración exclusivamente desde variables de entorno antes de abrir puertos o establecer conexiones externas.
2. IF algún campo obligatorio falta, tiene tipo incorrecto o está fuera de rango, THEN THE `Config_Loader` SHALL rechazar la configuración completa como una unidad y retornar un error.
3. THE `Config_Loader` SHALL incluir en el error el nombre del campo afectado y la causa del rechazo sin incluir el valor del campo.
4. IF la validación de configuración falla al arrancar, THEN THE `Service` SHALL terminar sin abrir puertos, sin establecer conexiones externas y con un código de salida distinto de cero.
5. THE `Backend` SHALL incluir un archivo `.env.example` en la raíz del repositorio con valores de marcador de posición para las variables del setup base (`HTTP_PORT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_SHUTDOWN_TIMEOUT`, `WORKER_SHUTDOWN_TIMEOUT`, `LOG_LEVEL`, `LOG_FORMAT`), sin secretos reales; las variables de Kafka y PostgreSQL no figuran en este archivo porque sus adaptadores pertenecen a la Spec de ingesta.

### Requisito 3 — Composición de dependencias

**User Story:** Como desarrollador, quiero un punto de composición de dependencias por slice, para que los binarios de entrada no contengan lógica de negocio y las dependencias sean reemplazables.

#### Criterios de aceptación

1. THE `Ingestion_Service` SHALL disponer de un `DependencyComposer` en `internal/ingestion/config/composer.go` que construya e inyecte todos los adaptadores del slice `ingestion` antes de que el servicio empiece a aceptar solicitudes.
2. THE `Ingestion_Service` y THE `Worker_Service` SHALL contener en sus `main.go` únicamente carga de config, composición de dependencias y arranque del servicio; no contendrán reglas de negocio ni handlers directos.
3. IF una dependencia falla al construirse durante la composición, THEN THE `DependencyComposer` SHALL liberar los recursos ya inicializados y retornar un error que cause la terminación del `Service` con código de salida distinto de cero en un tiempo máximo de 10 segundos.
4. THE `Worker_Service` SHALL disponer de un `DependencyComposer` en `internal/worker/config/composer.go` que devuelva una función de limpieza (esqueleto); la lógica de consumo de mensajes se implementa en la Spec de ingesta.

### Requisito 4 — Arranque del servidor HTTP

**User Story:** Como consumidor de la API, quiero un servidor HTTP consistente y configurable, para que las solicitudes sean atendidas de forma predecible con timeouts controlados.

#### Criterios de aceptación

1. THE `Ingestion_Service` SHALL exponer el `HTTP_Router` en el puerto definido por la configuración del slice `ingestion`, dentro del rango válido 1–65535.
2. THE `HTTP_Router` SHALL registrar las rutas y la cadena de middleware a través de `internal/ingestion/http`, no en `main.go`.
3. THE `Ingestion_Service` SHALL aplicar un `ReadTimeout` y un `WriteTimeout` configurables, con valor por defecto de 15 segundos cuando no se especifiquen.
4. IF el puerto no puede vincularse al arrancar, THEN THE `Ingestion_Service` SHALL terminar con código de salida distinto de cero.

### Requisito 5 — Manejo de errores con Problem Details

**User Story:** Como consumidor de la API, quiero respuestas de error con formato estándar, para que los errores sean interpretables de forma consistente sin exponer detalles internos.

#### Criterios de aceptación

1. THE `Error_Handler` SHALL responder con `Content-Type: application/problem+json` y los miembros `type`, `title`, `status` conformes a RFC 9457 en todas las respuestas de error; todos los nombres de campo en inglés.
2. THE `Error_Handler` SHALL garantizar que el valor del campo `status` en el cuerpo JSON sea idéntico al código de estado HTTP de la respuesta.
3. THE `Error_Handler` SHALL responder con 404 si la ruta no existe, con 405 si el método HTTP no está admitido para esa ruta, y con 500 ante cualquier error o panic inesperado.
4. THE `Error_Handler` SHALL garantizar que las respuestas de error no incluyan trazas de ejecución, rutas internas del sistema, sentencias SQL ni valores de secretos.

### Requisito 6 — Health endpoint

**User Story:** Como operador, quiero un endpoint de health check, para que los sistemas de orquestación puedan verificar el estado del servicio sin necesidad de lógica adicional.

#### Criterios de aceptación

1. WHEN se recibe `GET /health` y el `Service` está listo para atender solicitudes, THE `Health_Endpoint` SHALL responder con código 200 y el cuerpo `{"status":"healthy"}` en un tiempo máximo de 1 segundo; el handler reside en `internal/ingestion/http/health/handler.go`.
2. IF una dependencia requerida del `Service` no está disponible, THEN THE `Health_Endpoint` SHALL responder con código 503 y el cuerpo `{"status":"unhealthy"}`.
3. THE `Health_Endpoint` SHALL garantizar que la respuesta no incluya secretos, cadenas de conexión ni detalles internos de infraestructura.

### Requisito 7 — Apagado controlado

**User Story:** Como operador, quiero que los servicios se apaguen de forma ordenada al recibir una señal de terminación, para que las solicitudes en curso finalicen sin pérdida de datos.

#### Criterios de aceptación

1. WHEN el `Ingestion_Service` recibe una señal `SIGINT` o `SIGTERM`, THE `Graceful_Shutdown` SHALL dejar de aceptar nuevas solicitudes e indicar no disponibilidad a los sistemas de orquestación.
2. THE `Graceful_Shutdown` SHALL permitir que las solicitudes en curso finalicen dentro de un timeout configurable, con valor por defecto de 30 segundos y rango válido de 1 a 300 segundos.
3. IF el timeout de apagado se agota antes de que todas las solicitudes en curso finalicen, THEN THE `Graceful_Shutdown` SHALL forzar la terminación y registrar el número de solicitudes no finalizadas.
4. WHEN el `Worker_Service` recibe una señal `SIGINT` o `SIGTERM`, THE `Worker_Service` SHALL liberar los recursos adquiridos y terminar con código de salida 0 en un tiempo máximo de 10 segundos.
5. IF el `Service` recibe una segunda señal de terminación durante el proceso de apagado, THEN THE `Service` SHALL terminar de inmediato con código de salida distinto de cero.

### Requisito 8 — Registro estructurado con trace_id

**User Story:** Como operador, quiero entradas de log estructuradas con un identificador de correlación por solicitud, para que pueda correlacionar todos los eventos de una misma solicitud en los sistemas de monitoreo.

#### Criterios de aceptación

1. THE `Structured_Logger` SHALL incluir como mínimo los campos `timestamp`, `level`, `message` y `trace_id` en cada entrada de log.
2. WHEN el `HTTP_Router` recibe una solicitud con el header `X-Trace-ID` cuyo valor es un UUID v4 válido de máximo 128 caracteres, THE `HTTP_Router` SHALL reutilizarlo como `TraceID` y adjuntarlo a todas las entradas de log de esa solicitud.
3. IF la solicitud no incluye el header `X-Trace-ID` o su valor no tiene el formato UUID v4 válido, THEN THE `HTTP_Router` SHALL generar un nuevo `TraceID` con formato UUID v4.
4. THE `HTTP_Router` SHALL devolver el `TraceID` activo en el header de respuesta `X-Trace-ID` para todas las solicitudes.
5. THE `Structured_Logger` SHALL garantizar que ninguna entrada de log incluya tokens de autenticación, contraseñas, claves privadas ni valores de secretos de configuración.
