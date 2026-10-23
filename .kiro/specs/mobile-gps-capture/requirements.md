# Requirements: Captura móvil de GPS con sincronización automática

## 1. Descripción general

Desarrollar una aplicación móvil en React Native que capture coordenadas de ubicación del dispositivo mediante GPS cada 30 segundos, almacene localmente en SQLite, intente enviar inmediatamente después de capturar, y maneje reintentos automáticos con exponential backoff para registros que fallan.

La aplicación debe operar bajo el principio offline-first, capturando registros en background sin perder datos, diferenciando entre errores de validación (no reintenta) y errores transitorios (reintenta con backoff), y proporcionando sincronización manual para forzar reenvío de registros pendientes u errados.

## 2. Glosario

- **Status ('pending', 'sent', 'error', 'validation_error'):** Estado del registro en SQLite que indica si aún no se ha enviado, se envió exitosamente (202), falló con error de validación o falló con error transitorio
- **Captura (capture):** Lectura de coordenadas de GPS en un momento específico
- **Envío (send):** Intento de transmitir un registro al backend mediante HTTP POST
- **202 Accepted:** Respuesta HTTP del backend indicando que el evento fue recibido y puesto en el bus de mensajería para procesamiento asíncrono
- **Exponential backoff:** Estrategia de reintento donde el intervalo entre intentos se duplica (ej: 2s → 4s → 8s → 16s)
- **Sincronización manual:** Acción del usuario para forzar reenvío inmediato de registros pendientes u errados

## 3. Requisitos funcionales

### RF-01-MOB. Captura automática de GPS cada 30 segundos

La aplicación deberá capturar automáticamente coordenadas de ubicación cada 30 segundos y guardar cada captura como un nuevo registro en SQLite.

**Criterios de aceptación:**
- RF-01-MOB.1: La app solicita permisos de ubicación al usuario en la primera ejecución
- RF-01-MOB.2: Una vez otorgados los permisos, la captura de GPS se ejecuta automáticamente en segundo plano cada 30 segundos
- RF-01-MOB.3: Cada lectura de GPS captura latitud (NUMERIC 9,6), longitud (NUMERIC 9,6) y precisión (NUMERIC 5,2)
- RF-01-MOB.4: Cada captura se registra con marca temporal UTC (captured_at) y se almacena inmediatamente con status='pending'
- RF-01-MOB.5: Se genera un event_id (UUID) y un idempotency_key (UUID) únicos para cada captura al momento de guardar en SQLite
- RF-01-MOB.6: La captura funciona incluso sin conectividad a internet

### RF-02-MOB. Almacenamiento local persistente en SQLite con estados de sincronización

La aplicación deberá almacenar las ubicaciones capturadas con un estado de sincronización que permita rastrear si fueron enviadas, fallaron o están pendientes.

**Criterios de aceptación:**
- RF-02-MOB.1: Se crea automáticamente una base de datos SQLite en el almacenamiento local del dispositivo
- RF-02-MOB.2: Cada registro almacena: event_id (UUID), vehicle_id, latitude, longitude, accuracy, captured_at (TIMESTAMPTZ), trace_id (UUID), idempotency_key (UUID), status (VARCHAR: 'pending'|'sent'|'error'|'validation_error'), last_sync_attempt (TIMESTAMPTZ nullable), sync_attempt_count (INTEGER default 0), next_retry_at (TIMESTAMPTZ nullable)
- RF-02-MOB.3: Los registros se persisten aunque la app se cierre o se pierda conectividad
- RF-02-MOB.4: Se implementa un esquema con índices para búsquedas eficientes por vehicle_id, status, captured_at y next_retry_at
- RF-02-MOB.5: La base de datos maneja migraciones de esquema de forma automática

### RF-03-MOB. Envío inmediato después de captura

Después de guardar cada registro de captura en SQLite con status='pending', la aplicación deberá intentar enviar inmediatamente ese registro al backend.

**Criterios de aceptación:**
- RF-03-MOB.1: Tan pronto como se guarda un registro de captura en SQLite, se intenta enviar mediante HTTP POST al endpoint del backend
- RF-03-MOB.2: Si la app está online, el envío debe completarse en aproximadamente 200ms (latencia observada en desarrollo local)
- RF-03-MOB.3: El envío ocurre de forma asíncrona y no bloquea futuras capturas
- RF-03-MOB.4: Si no hay conectividad, el registro permanece con status='pending' localmente sin intentar envío
- RF-03-MOB.5: El payload enviado incluye todos los campos requeridos: event_id, vehicle_id, latitude, longitude, accuracy, captured_at, trace_id, idempotency_key

### RF-04-MOB. Manejo de respuesta 202 Accepted del backend

Cuando el backend responde con 202 Accepted al envío de un registro, la aplicación deberá marcar el registro como enviado sin borrarlo de la base de datos local.

**Criterios de aceptación:**
- RF-04-MOB.1: Al recibir HTTP 202 Accepted para un registro, el status se cambia de 'pending' a 'sent' en SQLite
- RF-04-MOB.2: El registro NO se borra de la base de datos local (aún no hay garantía de persistencia final en el backend)
- RF-04-MOB.3: last_sync_attempt se actualiza con la marca temporal actual
- RF-04-MOB.4: sync_attempt_count se incrementa en 1
- RF-04-MOB.5: El registro permanece visible en el listado con status visual 'Enviado'

### RF-05-MOB. Diferenciación de errores de envío

La aplicación deberá distinguir entre errores de validación (no reintenta automáticamente) y errores transitorios (reintenta con exponential backoff).

**Criterios de aceptación:**
- RF-05-MOB.1: Errores HTTP 4xx (excepto 429) y 422 se clasifican como 'validation_error' → status se cambia a 'validation_error' en SQLite
- RF-05-MOB.2: Registros con status='validation_error' NO se reintentan automáticamente (son irrecuperables sin intervención manual)
- RF-05-MOB.3: Errores HTTP 5xx, timeouts, errores de red y 429 se clasifican como 'transient_error' → status se cambia a 'error' en SQLite
- RF-05-MOB.4: Registros con status='error' se reintentan automáticamente con exponential backoff (inicio 2s, máximo 300s, factor 2)
- RF-05-MOB.5: La respuesta HTTP de error se analiza para extraer detalles técnicos sin perder el registro en SQLite
- RF-05-MOB.6: last_sync_attempt, sync_attempt_count y next_retry_at se actualizan en cada intento fallido

### RF-06-MOB. Reintentos automáticos con exponential backoff

La aplicación deberá reintentar automáticamente el envío de registros con status='error' utilizando exponential backoff.

**Criterios de aceptación:**
- RF-06-MOB.1: Cuando ocurre un error transitorio, se calcula next_retry_at = now + (2s * 2^(sync_attempt_count - 1))
- RF-06-MOB.2: El máximo intervalo de reintento es 300 segundos (5 minutos)
- RF-06-MOB.3: Cada reintento se ejecuta asíncrono en background sin bloquear captura o UI
- RF-06-MOB.4: Se registran los reintentos en logs de diagnóstico (timestamp, evento_id, intento #, próximo retry)
- RF-06-MOB.5: La app mantiene una cola de reintentos ordenada por next_retry_at para eficiencia de recursos

### RF-07-MOB. Manejo de conectividad offline/online

La aplicación deberá acumular registros 'pending' cuando no hay conectividad y reanudar reintentos al reconectar.

**Criterios de aceptación:**
- RF-07-MOB.1: Cuando la app detecta conectividad online, verifica registros con status='pending' u 'error'
- RF-07-MOB.2: Registros con status='pending' se reintentan como nuevo envío (sin considerar sync_attempt_count)
- RF-07-MOB.3: Registros con status='error' cuyo next_retry_at <= now se reintentan inmediatamente
- RF-07-MOB.4: Los reintentos en reconexión no se bloquean entre sí; se ejecutan en paralelo hasta un límite razonable (ej: máximo 3 en paralelo)
- RF-07-MOB.5: No se reintenta registros con status='validation_error' (fueron ya clasificados como irrecuperables)

### RF-08-MOB. Interfaz de control de captura y sincronización manual

La aplicación deberá proporcionar controles en UI para gestionar la captura y permitir sincronización manual.

**Criterios de aceptación:**
- RF-08-MOB.1: Botón "Iniciar Captura" que activa la captura de GPS cada 30 segundos
- RF-08-MOB.2: Botón "Parar Captura" que desactiva la captura (registros pendientes se conservan)
- RF-08-MOB.3: Indicador visual mostrando si la captura está activa o inactiva
- RF-08-MOB.4: Botón "Sincronizar" global que:
  - Si la app está ONLINE: intenta reenviar todos los registros con status='pending' u 'error' (excepto 'validation_error')
  - Si la app está OFFLINE: muestra mensaje "Sin conectividad"
- RF-08-MOB.5: Al parar la captura, no se pierden registros existentes

### RF-09-MOB. Listado reactivo de ubicaciones con status detallado

La aplicación deberá mostrar un listado de ubicaciones capturadas que se actualice automáticamente y permita sincronización individual de registros.

**Criterios de aceptación:**
- RF-09-MOB.1: Se muestra un listado de todas las ubicaciones almacenadas localmente en orden descendente por captured_at
- RF-09-MOB.2: Cada item del listado muestra: vehicle_id, latitud, longitud, precisión, fecha/hora captura, status visual ('Pendiente' | 'Enviado' | 'Error' | 'Error de validación'), y última fecha de intento de sincronización
- RF-09-MOB.3: El listado se actualiza ÚNICAMENTE cuando:
  - Se captura un nuevo registro (evento de captura de GPS)
  - Cambia el status de un registro (de 'Pendiente' a 'Enviado', a 'Error', etc.)
  - Se ejecuta una sincronización manual
  - La app regresa del background
  - NO se actualiza cada 5 segundos automáticamente (optimización de renders)
- RF-09-MOB.4: Cada item del listado incluye un botón "Reintentar" que:
  - Si está OFFLINE: muestra mensaje "Sin conectividad"
  - Si está ONLINE y el registro es 'validation_error': muestra botón deshabilitado con tooltip "Error de validación: requiere corrección"
  - Si está ONLINE y el registro es 'pending' o 'error': intenta reenviar inmediatamente ese registro
- RF-09-MOB.5: Se muestra un resumen en cabecera mostrando total de registros y desglose por status (ej: "12 total: 8 Pendientes, 2 Enviados, 2 Errores, 0 Errores de validación")

### RF-10-MOB. Indicador de conectividad

La aplicación deberá mostrar el estado de conectividad del dispositivo en tiempo real.

**Criterios de aceptación:**
- RF-10-MOB.1: Se muestra un indicador visual en la barra superior que diferencia conectividad (online = verde, offline = rojo)
- RF-10-MOB.2: El indicador refleja cambios de conectividad en tiempo real (sin lag perceptible)
- RF-10-MOB.3: En modo offline, se desactiva el botón de sincronización global
- RF-10-MOB.4: En modo offline, los botones "Reintentar" de items individuales muestran estado deshabilitado

### RF-11-MOB. Preparación de payload para envío

La aplicación deberá preparar los registros locales en el formato requerido por el backend antes de cada intento de envío.

**Criterios de aceptación:**
- RF-11-MOB.1: El payload incluye: event_id (UUID NOT NULL), vehicle_id (VARCHAR 50 NOT NULL), latitude (NUMERIC 9,6 NOT NULL), longitude (NUMERIC 9,6 NOT NULL), accuracy (NUMERIC 5,2 opcional), captured_at (TIMESTAMPTZ NOT NULL), trace_id (UUID NOT NULL), idempotency_key (UUID NOT NULL)
- RF-11-MOB.2: Todos los campos requeridos se validan antes de envío; si falta un campo obligatorio, el registro se marca como 'validation_error'
- RF-11-MOB.3: Los registros sin conectividad permanecen locales sin intentar envío
- RF-11-MOB.4: Cada registro tiene event_id e idempotency_key únicos generados al momento de captura

## 4. Requisitos no funcionales

### RNF-01-MOB. Clean Architecture

La aplicación deberá aplicar Clean Architecture con separación clara entre capas.

**Criterios de aceptación:**
- RNF-01-MOB.1: Dominio (entidades, objetos de valor, reglas de negocio) sin dependencias de frameworks
- RNF-01-MOB.2: Aplicación (casos de uso, coordinación de captura y sincronización) sin dependencias de React Native ni infraestructura
- RNF-01-MOB.3: Infraestructura (adapters) para locationProvider, persistenceProvider, connectivityProvider y apiClient
- RNF-01-MOB.4: Presentación (componentes, pantallas, hooks) que usan el dominio y aplicación sin exponer detalles técnicos

### RNF-02-MOB. Operación offline-first

La aplicación debe funcionar correctamente sin conectividad a internet en todas las operaciones de captura y almacenamiento.

**Criterios de aceptación:**
- RNF-02-MOB.1: Captura de GPS funcionando sin conectividad
- RNF-02-MOB.2: Almacenamiento local funcionando sin conectividad
- RNF-02-MOB.3: Listado visualizable sin conectividad
- RNF-02-MOB.4: No se pierden registros al desconectar
- RNF-02-MOB.5: Reintentos se pausan en modo offline y reanudan al reconectar

### RNF-03-MOB. Determinismo en datos de prueba

La aplicación debe generar datos deterministas para facilitar pruebas y reproducibilidad de escenarios.

**Criterios de aceptación:**
- RNF-03-MOB.1: Timestamps (captured_at, last_sync_attempt, next_retry_at) son inyectables desde pruebas
- RNF-03-MOB.2: Clock provider es inyectable para fijar fecha/hora en tests
- RNF-03-MOB.3: IDGenerator es inyectable para UUIDs predecibles en tests
- RNF-03-MOB.4: LocationProvider es reemplazable por fake en tests que devuelve coordenadas deterministas

### RNF-04-MOB. Separación y reemplazabilidad de dependencias

Las dependencias externas deben estar claramente separadas detrás de puertos/interfaces y ser reemplazables para pruebas.

**Criterios de aceptación:**
- RNF-04-MOB.1: LocationProvider abstracto con implementación para hardware real e implementación fake para tests
- RNF-04-MOB.2: PersistenceProvider abstracto con SQLite real en producción e SQLite en memoria (:memory:) en tests
- RNF-04-MOB.3: ConnectivityProvider abstracto para estado de red, reemplazable en tests
- RNF-04-MOB.4: APIClient abstracto para comunicación HTTP con backend, sustituible por interceptación de red (MSW) en tests

### RNF-05-MOB. Idempotencia en sincronización

Los reintentos de envío deben ser idempotentes gracias al idempotency_key enviado al backend.

**Criterios de aceptación:**
- RNF-05-MOB.1: Cada registro tiene un idempotency_key único generado al capturar
- RNF-05-MOB.2: El idempotency_key se incluye en todos los intentos de envío (no cambia en reintentos)
- RNF-05-MOB.3: El backend valida el idempotency_key para evitar duplicados aunque la app reintente

### RNF-06-MOB. Resiliencia y recuperación

La aplicación debe recuperarse correctamente de fallos de red, timeouts y cambios de conectividad sin perder datos ni causar duplicados.

**Criterios de aceptación:**
- RNF-06-MOB.1: Un timeout en envío no causa corrupción de estado en SQLite
- RNF-06-MOB.2: Un cambio offline→online reinicia reintentos correctamente
- RNF-06-MOB.3: La app se recupera correctamente de crashes con registros en cualquier estado
- RNF-06-MOB.4: No se generan duplicados en casos de reintentos múltiples gracias al idempotency_key

### RNF-07-MOB. Observabilidad y diagnóstico

Logs y trazas deben permitir diagnosticar problemas de captura y sincronización en producción.

**Criterios de aceptación:**
- RNF-07-MOB.1: Se registran eventos clave: captura realizada, envío iniciado, respuesta recibida, cambio de status, error de validación, reintento planificado
- RNF-07-MOB.2: Los logs incluyen event_id y trace_id para correlación de eventos
- RNF-07-MOB.3: Errores de red y tiempos de espera se registran con suficiente contexto sin exponer credenciales
- RNF-07-MOB.4: Los logs de diagnóstico no ralentizan la ejecución significativamente

## 5. Criterios de aceptación globales

- [ ] La app captura GPS cada 30 segundos cuando está activa
- [ ] Cada captura genera un registro con status='pending' y campos únicos (event_id, idempotency_key)
- [ ] Inmediatamente después de capturar, se intenta enviar ese registro al backend
- [ ] Respuesta 202 Accepted cambia status a 'sent' sin borrar el registro de SQLite
- [ ] Errores 4xx/422 se clasifican como 'validation_error' (no se reintentan)
- [ ] Errores 5xx/timeout/red se clasifican como 'error' y se reintentan con exponential backoff (2s → 4s → 8s → ... → máx 300s)
- [ ] En offline, se acumulan registros 'pending' sin intentar envío
- [ ] Al reconectar, se reinician reintentos de registros 'pending' y 'error'
- [ ] Todos los registros se almacenan en SQLite con estado persistente
- [ ] El listado muestra registros con status visual actualizado solo en eventos clave
- [ ] Botón de sincronización global reenvía 'pending' y 'error' (excepto 'validation_error')
- [ ] Botón "Reintentar" individual permite reenvío de registros específicos
- [ ] No se pierden datos al desconectar, cerrar app o cambiar conectividad
- [ ] Indicador de conectividad muestra online/offline en tiempo real
- [ ] La arquitectura es Clean con capas bien separadas
- [ ] Todos los datos tienen UUID e idempotency_key únicos
- [ ] Funciona sin conectividad a internet

## 6. Dependencias y contexto

- **Backend:** API de ingesta HTTP en Go que recibe los registros y responde 202 Accepted (RF-04 del producto)
- **Base de datos:** SQLite en dispositivo con soporte para estados de sincronización
- **Framework:** React Native con TypeScript
- **Permisos:** Ubicación (iOS), Ubicación (Android)
- **Bibliotecas:** react-native-geolocation (ubicación), @react-native-community/netinfo (conectividad), react-native-sqlite-storage (persistencia)

## 7. Limitaciones conocidas y notas

- La frecuencia de captura (30s) es configurable en fases posteriores
- La precisión de GPS depende del hardware del dispositivo
- Geofencing y alertas de zonas críticas son opcionales para este MVP
- El máximo número de reintentos en paralelo se limita a 3 para optimizar recursos
- La exposición de detalles de error al usuario (UI) no debe revelar información técnica sensible

## 8. Trazabilidad a requisitos del producto

- **RF-01:** Captura móvil de coordenadas ✓
- **RF-02:** Persistencia local y funcionamiento sin conexión ✓
- **RF-03:** Sincronización por lotes (manual en esta fase) ✓
- **RNF-01:** Clean Architecture ✓
- **RNF-04:** Integridad e idempotencia ✓
- **RNF-06:** Resiliencia ✓
- **RNF-07:** Operación offline-first ✓

