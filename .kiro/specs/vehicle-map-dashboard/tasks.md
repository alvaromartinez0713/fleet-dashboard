# Plan de Implementación: Dashboard de Mapa de Vehículos

## Overview

Este plan descompone el Dashboard de Mapa de Vehículos en tareas implementables, siguiendo una arquitectura de 4 capas:
1. **Dominio**: Entidades (Vehicle, Position, VehicleId) con validación
2. **Aplicación**: Casos de uso y coordinadores de estado
3. **Infraestructura**: Adaptadores para SSE, datos mock, HTTP, mapas
4. **Presentación**: Componentes React y hooks reutilizables
5. **Testing**: Pruebas de integración siguiendo `testing.md`

Cada tarea es completable en 1-2 horas y tiene criterios de aceptación explícitos.

---

## Nota importante: MVP con licensePlate = vehicleId

En el MVP, **`licensePlate` tiene exactamente el mismo valor que `vehicleId`**:

```typescript
// MockDataProvider genera:
const vehicleId = new VehicleId('MOCK-VEHICLE-01');
const licensePlate = vehicleId.value; // ← IDÉNTICO

// Evento SSE contiene ambos (igual):
{ "vehicleId": "MOCK-VEHICLE-01", "licensePlate": "MOCK-VEHICLE-01", ... }
```

**Por qué:** Backend aún no expone servicio separado de vehículos. En futuras versiones, serán independientes.

**Implicaciones:**
- Tarea 6.1 (MockDataProvider): `licensePlate = vehicleId.value`
- Tarea 6.2 (ApiDataProvider): Usa `licensePlate` del DTO cuando backend lo exponga
- Entity Vehicle: Mantiene ambos campos para flexibilidad
- UI: Muestra ambos; en MVP serán idénticos

---


## Fase 1: Configuración Inicial

- [x] 1.1 Crear estructura de directorios del frontend
  - Crear `frontend/src/domain/`, `frontend/src/application/`, `frontend/src/infrastructure/`, `frontend/src/presentation/`
  - Crear subdirectorios: `domain/entities/`, `domain/errors/`, `domain/validators/`
  - Crear subdirectorios: `application/ports/`, `application/usecases/`, `application/state-manager/`
  - Crear subdirectorios: `infrastructure/http/`, `infrastructure/sse/`, `infrastructure/maps/`, `infrastructure/persistence/`, `infrastructure/data-providers/`
  - Crear subdirectorios: `presentation/hooks/`, `presentation/components/`, `presentation/pages/`, `presentation/styles/`
  - _Verifies: Requisito 5 (separación arquitectónica)_

- [x] 1.2 Configurar Vite, TypeScript y dependencias del frontend
  - Instalar: `react`, `react-dom`, `typescript`, `leaflet`, `react-leaflet`, `vitest`, `@testing-library/react`, `@testing-library/user-event`, `msw`
  - Crear `frontend/tsconfig.json` con `strict: true`
  - Crear `frontend/vite.config.ts` con variables de entorno (`VITE_DATA_MODE`, `VITE_API_BASE_URL`, `VITE_SSE_ENDPOINT`)
  - Crear `frontend/.env.local` (mock) y `frontend/.env.production` (api)
  - _Acceptance: Dependencias instaladas, TypeScript configurado, variables de entorno disponibles_

- [x] 1.3 Crear eslint y prettier para consistencia de código
  - Crear `.eslintrc.json` (TypeScript + React rules)
  - Crear `.prettierrc`
  - Configurar script `lint` y `format` en `frontend/package.json`
  - _Acceptance: Linting funciona sin errores_

---

## Fase 2: Dominio

- [x] 2.1 Implementar objeto de valor VehicleId
  - Crear `frontend/src/domain/entities/vehicle-id.ts`
  - Validar que el valor no esté vacío
  - Implementar método `equals()` para comparación
  - Implementar método `toString()`
  - _Verifies: Requisito 5 (dominio independiente), Requisito 2_
  - _Acceptance: VehicleId no permite valores vacíos, equals() funciona correctamente_

- [x] 2.2 Implementar objeto de valor Position
  - Crear `frontend/src/domain/entities/position.ts`
  - Validar rango: latitud ∈ [-90, 90], longitud ∈ [-180, 180]
  - Almacenar timestamp (Date object)
  - Implementar método `equals()` con tolerancia de 1e-6 decimales
  - Implementar método `toString()` con 4 decimales de precisión
  - _Verifies: Requisito 2.2, Requisito 2.6, Property 3 (precisión preservada)_
  - _Acceptance: Position rechaza coords fuera de rango, equals() funciona, toString() devuelve 4 decimales_

- [x] 2.3 Implementar entidad Vehicle
  - Crear `frontend/src/domain/entities/vehicle.ts`
  - Contener: vehicleId (VehicleId), licensePlate (string), position (Position)
  - Validar invariante: vehicleId no nulo, licensePlate no vacío
  - Implementar método `updatePosition(newPosition: Position): Vehicle` (retorna nueva instancia)
  - Implementar método `isStalePosition(maxAgeMs?: number): boolean`
  - Implementar método `equals()` basado en vehicleId
  - _Verifies: Requisito 2, Requisito 5 (entidad del dominio)_
  - _Acceptance: Vehicle valida invariantes, updatePosition() retorna nueva instancia_

- [x] 2.4 Validación de coordenadas en Position
  - La validación se implementa directamente en `frontend/src/domain/entities/position.ts`
  - El constructor de Position valida: latitud ∈ [-90, 90], longitud ∈ [-180, 180]
  - Lanza `InvalidCoordinateError` si coordenadas fuera de rango
  - _Verifies: Requisito 2.6, Property 4 (rechazo de coords inválidas)_
  - No hay archivo separado `position-validator.ts` (patrón innecesario)
  - _Acceptance: Position rechaza coords inválidas en el constructor

- [x] 2.5 Implementar errores de dominio
  - Crear `frontend/src/domain/errors/vehicle-errors.ts`
  - Definir: `InvalidVehicleIdError`, `InvalidCoordinateError`, `InvalidVehicleError`
  - Todos heredan de `Error`
  - _Acceptance: Errores pueden lanzarse y capturarse_

---

## Fase 3: Aplicación - Puertos

- [x] 3.1 Definir puerto VehicleDataProvider
  - Crear `frontend/src/application/ports/vehicle-data-provider.ts`
  - Exportar interfaz: `loadVehicles(): Promise<Vehicle[]>`
  - _Verifies: Requisito 5 (puertos desacoplados)_
  - _Acceptance: Interfaz compilable con TypeScript_

- [x] 3.2 Definir puerto VehicleRealtimePort
  - Crear `frontend/src/application/ports/vehicle-realtime-port.ts`
  - Exportar interfaz con métodos:
    - `connect(): Promise<void>`
    - `onPositionUpdate(callback: (event) => void): Unsubscribe`
    - `onConnectionStatusChange(callback: (status) => void): Unsubscribe`
    - `disconnect(): Promise<void>`
    - `getStatus(): ConnectionStatus`
  - Exportar tipos: `PositionUpdateEvent`, `ConnectionStatus`, `Unsubscribe`
  - _Verifies: Requisito 3, Requisito 5 (puerto SSE)_
  - _Acceptance: Interfaz compilable, tipos exportados_

- [x] 3.3 Definir puerto VehicleStorage
  - Crear `frontend/src/application/ports/vehicle-storage.ts`
  - Exportar interfaz:
    - `saveVehicles(vehicles: Vehicle[]): Promise<void>`
    - `loadVehicles(): Promise<Vehicle[]>`
    - `clear(): Promise<void>`
  - _Verifies: Requisito 1.6 (cache local offline)_
  - _Acceptance: Interfaz compilable_

---

## Fase 4: Aplicación - Casos de Uso

- [x] 4.1 Implementar caso de uso LoadVehicles
  - Crear `frontend/src/application/usecases/load-vehicles.ts`
  - Exportar clase `LoadVehiclesUseCase`
  - Constructor recibe: `VehicleDataProvider`, `VehicleStorage`
  - Método `execute(): Promise<Vehicle[]>`
    - Llama `vehicleDataProvider.loadVehicles()`
    - Valida que todos los vehículos son válidos (vehicleId, position no nulos)
    - Guarda en almacenamiento local con `vehicleStorage.saveVehicles()`
    - Si API falla, intenta cargar desde almacenamiento local
    - Retorna array de vehículos
  - _Verifies: Requisito 1.6, Requisito 4.2_
  - _Acceptance: Carga vehículos exitosamente, recupera de cache si API falla_

- [x] 4.2 Implementar caso de uso SubscribeToPositionUpdates
  - Crear `frontend/src/application/usecases/subscribe-to-updates.ts`
  - Exportar clase `SubscribeToPositionUpdatesUseCase`
  - Constructor recibe: `VehicleRealtimePort`, `HandlePositionUpdateUseCase`
  - Método `execute(onUpdate: callback, onStatusChange: callback): Promise<Unsubscribe>`
    - Conecta a puerto realtime
    - Registra callbacks para actualizaciones y cambios de estado
    - Retorna función que desconecta y limpia recursos
  - _Verifies: Requisito 3.1, Requisito 3.7_
  - _Acceptance: Se suscribe correctamente, limpia al desuscribirse_

- [x] 4.3 Implementar caso de uso HandlePositionUpdate
  - Crear `frontend/src/application/usecases/handle-position-update.ts`
  - Exportar clase `HandlePositionUpdateUseCase`
  - Constructor recibe: `VehicleStorage`, `PositionValidator`
  - Método `async execute(event: PositionUpdateEvent, onUpdate: callback): Promise<void>`
    - Valida coordenadas con `positionValidator.validate()`
    - Construye nueva `Position` con event.latitude, event.longitude, new Date(event.timestamp)
    - Carga vehículos actuales de storage
    - Encuentra vehículo por vehicleId y llama `updatePosition()`
    - Guarda estado actualizado en storage
    - Llama `onUpdate(updatedVehicles)`
    - Captura excepciones y las registra sin lanzar (eventos inválidos se ignoran)
  - _Verifies: Requisito 3.2, Requisito 7.5, Requisito 7.6, Property 5 (idempotencia)_
  - _Acceptance: Actualiza posición correctamente, rechaza coords inválidas, no rompe la app_

---

## Fase 5: Aplicación - State Manager

- [x] 5.1 Implementar VehicleStateManager
  - Crear `frontend/src/application/state-manager/vehicle-state-manager.ts`
  - Exportar clase `VehicleStateManager`
  - Constructor recibe: `LoadVehiclesUseCase`, `SubscribeToPositionUpdatesUseCase`, `HandlePositionUpdateUseCase`
  - Métodos públicos:
    - `loadVehicles(): Promise<Vehicle[]>` - carga vehículos iniciales
    - `startRealtimeUpdates(onStatusChange: callback): Promise<Unsubscribe>` - inicia suscripción SSE
    - `stopRealtimeUpdates(): Promise<void>` - detiene suscripción
    - `subscribeToUpdates(callback: (vehicles: Vehicle[]) => void): Unsubscribe` - suscripción granular
    - `getVehicleById(vehicleId: string): Promise<Vehicle | null>` - busca vehículo
    - `subscribeToVehicleUpdates(vehicleId: string, callback: (vehicle: Vehicle) => void): Unsubscribe`
  - Mantiene referencia a lista de vehículos actual
  - Mantiene lista de callbacks de actualización
  - _Verifies: Requisito 5 (coordinador de aplicación), Requisito 6_
  - _Acceptance: Coordina casos de uso correctamente_

---

## Fase 6: Infraestructura - Adaptadores de Datos

- [x] 6.1 Implementar MockDataProvider
  - Crear `frontend/src/infrastructure/data-providers/mock-data-provider.ts`
  - Exportar clase `MockDataProvider implements VehicleDataProvider`
  - Método `loadVehicles(): Promise<Vehicle[]>`
    - Genera 20 vehículos con IDs: `MOCK-VEHICLE-01` a `MOCK-VEHICLE-20`
    - **licensePlate = vehicleId.value** (MVP: ambos campos son idénticos, sin formato "MAD-0001")
    - Coordenadas: zona 50km² centrada en Madrid (40.4168, -3.7038) con rango ±0.2
    - Usa seed determinista para reproducibilidad (mismo array siempre)
    - Todas las coordenadas válidas (lat ∈ [-90,90], lon ∈ [-180,180])
  - _Verifies: Requisito 4.1, Requisito 4.3, Property 7 (mock genera coords válidas), Property 8 (round-trip)_
  - _Acceptance: Devuelve 20 vehículos válidos, determinista, reproducible_
  
  **Nota MVP:** En esta fase, `licensePlate` no tiene formato de placa real. Es idéntico a `vehicleId.value`. Esto simplifica el MVP sin comprometer la arquitectura.

- [x] 6.2 Implementar ApiDataProvider
  - Crear `frontend/src/infrastructure/data-providers/api-data-provider.ts`
  - Exportar clase `ApiDataProvider implements VehicleDataProvider`
  - Constructor recibe: `apiClient: ApiClient`
  - Método `loadVehicles(): Promise<Vehicle[]>`
    - Llama `apiClient.get<VehicleDTO[]>('/api/vehicles')`
    - Transforma DTOs a entidades Vehicle
    - Lanza `VehicleLoadError` si API falla
    - Retorna array de vehículos
  - Interfaz `VehicleDTO` con campos: vehicleId, licensePlate, latitude, longitude, timestamp (ISO-8601)
  - _Verifies: Requisito 8.1, Requisito 8.2_
  - _Acceptance: Carga de API, transforma correctamente_

- [x] 6.3 Implementar DataProviderFactory
  - Crear `frontend/src/infrastructure/data-providers/data-provider-factory.ts`
  - Exportar función `createDataProvider(mode?: 'mock' | 'api'): VehicleDataProvider`
    - Usa `import.meta.env.VITE_DATA_MODE` si no se proporciona modo
    - Retorna `new MockDataProvider()` si mode === 'mock'
    - Retorna `new ApiDataProvider(apiClient)` si mode === 'api'
  - _Verifies: Requisito 4.6, Requisito 10_
  - _Acceptance: Selecciona provider correcto según configuración_

---

## Fase 7: Infraestructura - Adaptador HTTP

- [x] 7.1 Implementar ApiClient
  - Crear `frontend/src/infrastructure/http/api-client.ts`
  - Exportar clase `ApiClient`
  - Constructor recibe: `baseUrl?: string` (default desde `import.meta.env.VITE_API_BASE_URL`)
  - Métodos:
    - `get<T>(endpoint: string): Promise<T>`
    - `post<T>(endpoint: string, body: unknown): Promise<T>`
  - Maneja errores HTTP y los transforma a excepciones legibles
  - _Acceptance: Realiza peticiones HTTP, maneja errores_

---

## Fase 8: Infraestructura - Adaptador SSE

- [x] 8.1 Implementar SSEConnection
  - Crear `frontend/src/infrastructure/sse/sse-connection.ts`
  - Exportar clase `SSEConnection implements VehicleRealtimePort`
  - Constructor recibe: `sseUrl?: string` (default desde `import.meta.env.VITE_SSE_ENDPOINT`)
  - Propiedades privadas:
    - `eventSource: EventSource | null`
    - `status: ConnectionStatus` = 'DISCONNECTED'
    - `updateCallbacks: ((event) => void)[]` array
    - `statusCallbacks: ((status) => void)[]` array
    - `reconnectAttempts: number` = 0
    - `maxReconnectAttempts: number` = 5
    - `baseBackoffMs: number` = 1000
    - `maxBackoffMs: number` = 30000
  - Método `async connect(): Promise<void>`
    - Si ya está CONNECTED o CONNECTING, retorna sin hacer nada
    - Cambia status a CONNECTING
    - Crea new EventSource(sseUrl)
    - Registra listener para evento 'position_update'
    - Parsea JSON del evento y llama todos los updateCallbacks
    - Registra listener para 'error'
    - Cambio de status a CONNECTED
    - Si error, llama `handleConnectionError()`
  - Método privado `handleConnectionError()`
    - Cambia status a ERROR
    - Si reconnectAttempts < maxReconnectAttempts:
      - Calcula backoff: `Math.min(baseBackoffMs * 2^attempts, maxBackoffMs)`
      - Incrementa reconnectAttempts
      - Programa setTimeout que llama `connect()` de nuevo
    - Si maxAttempts alcanzado, status a DISCONNECTED
  - Método `onPositionUpdate(callback): Unsubscribe`
    - Añade callback a updateCallbacks
    - Retorna función que lo remueve
  - Método `onConnectionStatusChange(callback): Unsubscribe`
    - Añade callback a statusCallbacks
    - Retorna función que lo remueve
  - Método `async disconnect(): Promise<void>`
    - Si eventSource, ciérralo
    - Cambia status a DISCONNECTED
  - Método `getStatus(): ConnectionStatus`
    - Retorna status actual
  - Método privado `setStatus(status: ConnectionStatus)`
    - Cambia this.status
    - Llama todos los statusCallbacks
  - _Verifies: Requisito 3.1, Requisito 3.5, Requisito 3.6, Requisito 7.1, Requisito 7.2, Property 6 (reconexión)_
  - _Acceptance: Conecta a SSE, reconecta con backoff exponencial, limpia recursos_

- [x] 8.2 Implementar SSESimulator para desarrollo/testing
  - Crear `frontend/src/infrastructure/sse/sse-simulator.ts`
  - Exportar clase `SSESimulator implements VehicleRealtimePort`
  - Constructor recibe: `vehicles: Vehicle[]`, `updateIntervalMs?: number` = 2000
  - Propiedades privadas similares a SSEConnection
  - Método `async connect(): Promise<void>`
    - Simula latencia (await 500ms)
    - Status a CONNECTING, luego CONNECTED
    - Inicia interval que genera eventos de actualización cada `updateIntervalMs`
    - Para cada vehículo, genera pequeño cambio de coordenadas
    - Llama updateCallbacks con nuevo evento
  - Métodos `onPositionUpdate()`, `onConnectionStatusChange()`, `disconnect()`, `getStatus()` similares a SSEConnection
  - _Verifies: Requisito 4 (datos mock), desarrollo sin backend_
  - _Acceptance: Simula movimiento realista de vehículos_

- [x] 8.3 Implementar SSEConnectionFactory
  - Crear `frontend/src/infrastructure/sse/sse-connection-factory.ts`
  - Exportar función `createRealtimePort(mode?: 'real' | 'simulator'): VehicleRealtimePort`
    - Usa `import.meta.env.VITE_DATA_MODE` si no se proporciona
    - Retorna `new SSEConnection()` si mode === 'real'
    - Retorna `new SSESimulator(mockVehicles)` si mode === 'simulator'
  - _Acceptance: Selecciona puerto realtime correcto_

---

## Fase 9: Infraestructura - Persistencia Local

- [x] 9.1 Implementar LocalStorageAdapter
  - Crear `frontend/src/infrastructure/persistence/local-storage-adapter.ts`
  - Exportar clase `LocalStorageAdapter implements VehicleStorage`
  - Constante: `STORAGE_KEY = 'fleet-dashboard-vehicles'`
  - Método `async saveVehicles(vehicles: Vehicle[]): Promise<void>`
    - Serializa vehículos a JSON (convertir objetos de valor a plain objects)
    - Guarda en localStorage[STORAGE_KEY]
  - Método `async loadVehicles(): Promise<Vehicle[]>`
    - Recupera de localStorage[STORAGE_KEY]
    - Deserializa y reconstruye entidades Vehicle
    - Retorna array vacío si no existe
  - Método `async clear(): Promise<void>`
    - Limpia localStorage[STORAGE_KEY]
  - _Verifies: Requisito 1.6 (offline-first), Requisito 7.7_
  - _Acceptance: Persiste y recupera vehículos correctamente_

---

## Fase 10: Presentación - Hooks

- [x] 10.1 Implementar useVehicles hook
  - Crear `frontend/src/presentation/hooks/useVehicles.ts`
  - Exportar interfaz `UseVehiclesResult { vehicles, loading, error, refetch }`
  - Exportar hook `useVehicles(): UseVehiclesResult`
    - Estados: `vehicles: Vehicle[]`, `loading: boolean`, `error: Error | null`
    - useEffect al montar: carga vehículos con `vehicleStateManager.loadVehicles()`
    - useEffect suscripción: `vehicleStateManager.subscribeToUpdates(handleUpdate)`
    - Función `refetch()` que reintenta carga
    - Retorna `{ vehicles, loading, error, refetch }`
  - _Verifies: Requisito 6_
  - _Acceptance: Carga vehículos, se suscribe a actualizaciones, limpia al desmontar_

- [x] 10.2 Implementar useSSEConnection hook
  - Crear `frontend/src/presentation/hooks/useSSEConnection.ts`
  - Exportar interfaz `UseSSEConnectionResult { status, error }`
  - Exportar hook `useSSEConnection(): UseSSEConnectionResult`
    - Estados: `status: ConnectionStatus`, `error: Error | null`
    - useEffect al montar: `vehicleStateManager.startRealtimeUpdates(handleStatusChange)`
    - Cleanup: `vehicleStateManager.stopRealtimeUpdates()`
    - Si status es ERROR, guarda error en estado
    - Retorna `{ status, error }`
  - _Verifies: Requisito 6, Requisito 3.7_
  - _Acceptance: Gestiona ciclo de vida SSE, limpia recursos_

- [x] 10.3 Implementar useVehicleUpdates hook (granular)
  - Crear `frontend/src/presentation/hooks/useVehicleUpdates.ts`
  - Exportar interfaz `UseVehicleUpdatesResult { vehicle, loading }`
  - Exportar hook `useVehicleUpdates(vehicleId: string): UseVehicleUpdatesResult`
    - Estados: `vehicle: Vehicle | null`, `loading: boolean`
    - useEffect al montar: carga vehículo inicial con `vehicleStateManager.getVehicleById()`
    - useEffect: suscripción granular con `vehicleStateManager.subscribeToVehicleUpdates(vehicleId)`
    - Cleanup: desuscripción
    - Retorna `{ vehicle, loading }`
  - _Verifies: Requisito 6_
  - _Acceptance: Suscripción granular funciona_

- [x] 10.4 Implementar useReconnectionManager hook
  - Crear `frontend/src/presentation/hooks/useReconnectionManager.ts`
  - Exportar interfaz `UseReconnectionManagerResult { isReconnecting, nextRetryIn, manualRetry }`
  - Exportar hook `useReconnectionManager(): UseReconnectionManagerResult`
    - Estados: `isReconnecting: boolean`, `nextRetryIn: number` (ms)
    - Gestiona countdown de próximo reintento con setInterval
    - Función `manualRetry()` que dispara reconexión inmediata
    - Retorna `{ isReconnecting, nextRetryIn, manualRetry }`
  - _Verifies: Requisito 7.4_
  - _Acceptance: Muestra countdown, permite reintento manual_

---

## Fase 11: Presentación - Componentes

- [x] 11.1 Implementar MapContainer
  - Crear `frontend/src/presentation/components/MapContainer.tsx`
  - Props: `vehicles: Vehicle[]`
  - Renderiza `<MapContainer>` (de react-leaflet) centrado en Madrid (40.4168, -3.7038) con zoom 13
  - Renderiza `<TileLayer>` (OpenStreetMap)
  - Por cada vehículo, renderiza `<VehicleMarker vehicle={v} key={v.vehicleId.value} />`
  - _Verifies: Requisito 1.2, Requisito 1.4_
  - _Acceptance: Mapa interactivo renderizado_

- [x] 11.2 Implementar VehicleMarker
  - Crear `frontend/src/presentation/components/VehicleMarker.tsx`
  - Props: `vehicle: Vehicle`
  - Renderiza `<Marker>` en posición [lat, lon]
  - Renderiza `<Popup>` con `<VehicleTooltip vehicle={v} />`
  - Aplica clase CSS 'stale-marker' si posición es antigüa (> 60 min)
  - test-id: 'vehicle-marker'
  - _Verifies: Requisito 2.3, Requisito 2.4, Requisito 2.5_
  - _Acceptance: Marcador en posición correcta, tooltip visible, clase stale si antigüo_

- [x] 11.3 Implementar VehicleTooltip
  - Crear `frontend/src/presentation/components/VehicleTooltip.tsx`
  - Props: `vehicle: Vehicle`
  - Muestra:
    - Identificador del vehículo
    - Placa del vehículo
    - Latitud/Longitud (4 decimales)
    - Marca de tiempo relativa ("Última actualización hace X minutos")
  - _Verifies: Requisito 2.5_
  - _Acceptance: Información visible en tooltip_

- [x] 11.4 Implementar ConnectionStatus
  - Crear `frontend/src/presentation/components/ConnectionStatus.tsx`
  - Props: `status: ConnectionStatus`
  - Muestra indicador visual:
    - 🟢 "Conectado" si CONNECTED
    - 🟡 "Conectando..." si CONNECTING
    - 🔴 "Desconectado" si DISCONNECTED o ERROR
  - test-id: 'connection-status'
  - _Verifies: Requisito 7.1_
  - _Acceptance: Indicador se actualiza según status_

- [x] 11.5 Implementar DemoModeBanner
  - Crear `frontend/src/presentation/components/DemoModeBanner.tsx`
  - Renderiza banner: "🎭 MODO DEMOSTRACIÓN - Usando datos ficticios"
  - Solo se muestra si `import.meta.env.VITE_DATA_MODE === 'mock'`
  - test-id: 'demo-mode-banner'
  - _Verifies: Requisito 4.5_
  - _Acceptance: Banner visible en modo mock, no visible en modo api_

- [x] 11.6 Implementar ErrorAlert
  - Crear `frontend/src/presentation/components/ErrorAlert.tsx`
  - Props: `error: Error | null`, `onDismiss?: () => void`
  - Renderiza alerta con mensaje de error
  - test-id: 'error-alert'
  - _Acceptance: Alerta muestra error_

- [x] 11.7 Implementar LoadingSpinner
  - Crear `frontend/src/presentation/components/LoadingSpinner.tsx`
  - Renderiza spinner de carga
  - test-id: 'loading-spinner'
  - _Acceptance: Spinner renderizado durante carga_

---

## Fase 12: Presentación - Página

- [x] 12.1 Implementar VehicleDashboard page
  - Crear `frontend/src/presentation/pages/VehicleDashboard.tsx`
  - Llama hooks: `useVehicles()`, `useSSEConnection()`
  - Renderiza:
    - `<ConnectionStatus status={connectionStatus} />`
    - `{VITE_DATA_MODE === 'mock' && <DemoModeBanner />}`
    - `{error && <ErrorAlert error={error} />}`
    - `{loading ? <LoadingSpinner /> : <MapContainer vehicles={vehicles} />}`
  - test-id: 'vehicle-dashboard'
  - _Verifies: Requisito 1, Requisito 3, Requisito 4, Requisito 6, Requisito 7_
  - _Acceptance: Dashboard completo funcional_

---

## Fase 13: Integración y Contexto

- [x] 13.1 Crear proveedor de contexto ApplicationContext
  - Crear `frontend/src/presentation/context/ApplicationContext.tsx`
  - Exportar `ApplicationProvider` que instancia:
    - `MockDataProvider` o `ApiDataProvider` según config
    - `SSEConnection` o `SSESimulator` según config
    - `LocalStorageAdapter`
    - Todos los casos de uso (LoadVehicles, SubscribeToPositionUpdates, HandlePositionUpdate)
    - `VehicleStateManager`
  - Exponer todo mediante `useApplication()` hook
  - _Verifies: Requisito 5 (inyección de dependencias)_
  - _Acceptance: Todas las dependencias disponibles en contexto_

- [x] 13.2 Crear App.tsx
  - Crear `frontend/src/App.tsx`
  - Envuelve `<ApplicationProvider>`
    - `<VehicleDashboard />`
  - _Acceptance: App monta correctamente_

- [x] 13.3 Crear main.tsx
  - Crear `frontend/src/main.tsx`
  - Renderiza `<App />` en div#root
  - _Acceptance: Punto de entrada del frontend configurado_

---

## Fase 14: Checkpoint y Testing Setup

- [x] 14.1 Ensure TypeScript compilation passes
  - Ejecutar `npm run build` en `frontend/`
  - Verificar no hay errores de tipo
  - _Acceptance: Build exitoso sin errores TypeScript_

- [x] 14.2 Configure MSW (Mock Service Worker) para testing
  - Crear `frontend/src/tests/mocks/handlers.ts`
  - Definir handlers MSW para:
    - `GET /api/vehicles` - devuelve vehículos mock con estructura OpenAPI
    - `GET /api/vehicles/positions/stream` - simula SSE
  - Crear `frontend/src/tests/mocks/server.ts` que configura MSW con handlers
  - Crear `frontend/src/tests/setup.ts` que inicia/para MSW antes/después de tests
  - _Verifies: Requisito 8 (contrato OpenAPI), Requisito 4_
  - _Acceptance: MSW configurado, handlers disponibles para tests_

- [x] 14.3 Configure Vitest setup
  - Crear `frontend/vitest.config.ts`
  - Configurar setupFiles: `['src/tests/setup.ts']`
  - Configurar globals: true
  - Configurar environment: 'jsdom'
  - _Acceptance: Vitest configurado para testing_

---

## Fase 15: Pruebas de Integración (Consolidadas)

Según `testing.md`: "pocas pruebas, valiosas y difíciles de romper"
Se priorizan **flujos de usuario críticos**, no casos de borde exhaustivos.

- [x] 15.1 Test 1: Dashboard carga vehículos y muestra mapa
  - Crear `frontend/src/tests/integration/VehicleDashboard.integration.test.tsx`
  - GIVEN el dashboard en modo mock
  - WHEN la página se carga
  - THEN se renderiza un mapa interactivo
  - AND aparecen marcadores para los 8 vehículos mock
  - AND el estado de conexión muestra "Conectado"
  - Verifica: Requisito 1 (mapa + vehículos), Requisito 3 (SSE conectado)
  - Entrada: Usuario abre dashboard
  - Salida observable: Mapa + 8 marcadores visibles + status conectado
  - Suite: Rápida (< 2s)
  - Herramientas: Vitest + RTL
  - Propiedad: El número de marcadores = número de vehículos cargados

- [x] 15.2 Test 2: SSE actualiza posiciones sin recargar mapa
  - Crear `frontend/src/tests/integration/VehicleDashboard.integration.test.tsx` (continúa)
  - GIVEN el dashboard cargado con 8 vehículos
  - WHEN se emite evento SSE con nueva posición para un vehículo
  - THEN el marcador se actualiza en la nueva ubicación
  - AND el mapa NO se recarga (performance)
  - AND otros marcadores no se mueven
  - Verifica: Requisito 3 (SSE + actualización reactiva)
  - Entrada: Evento SSE simulado con nuevo lat/lon
  - Salida observable: Marcador reposicionado, otros inmóviles
  - Suite: Rápida (< 2s)
  - Herramientas: Vitest + RTL
  - Propiedad: Idempotencia - procesar el mismo evento 2 veces = mismo resultado final

- [x] 15.3 Test 3: Coordenadas inválidas se rechazan, app sigue funcionando
  - Crear `frontend/src/tests/integration/VehicleDashboard.integration.test.tsx` (continúa)
  - GIVEN el dashboard con SSE conectada
  - WHEN se emite evento con coordenadas fuera de rango (lat=91, lon=181)
  - THEN el evento se rechaza (no se lanza excepción)
  - AND el marcador anterior se conserva en su última posición válida
  - AND la app no se bloquea, sigue funcionando
  - AND evento válido siguiente se procesa correctamente
  - Verifica: Requisito 2.6 (validación coords), Requisito 7.5 (robustez)
  - Entrada: Evento SSE con datos inválidos, luego evento válido
  - Salida observable: Evento rechazado silenciosamente, app funciona, evento siguiente procesa
  - Suite: Rápida (< 2s)
  - Herramientas: Vitest + RTL
  - Propiedad: Robustez - eventos inválidos no rompen la app ni el flujo posterior



---

## Fase 16: Final Checkpoint

- [x] 16.1 Verify all tests pass
  - Ejecutar `npm run test` en `frontend/`
  - Todos los tests pasan
  - Coverage > 70% para dominio y aplicación
  - _Acceptance: Test suite pasa completamente_

- [x] 16.2 Verify dashboard runs in development
  - Ejecutar `npm run dev` en `frontend/`
  - Dashboard accesible en http://localhost:5173
  - Mapa interactivo visible
  - 20 vehículos mock mostrados
  - SSE simulado genera actualizaciones
  - _Acceptance: Dashboard funciona en desarrollo_

- [x] 16.3 Verify integration with OpenAPI contract
  - Con modo API, MSW valida responses contra contrato
  - No hay violaciones de contrato
  - _Verifies: Requisito 8_
  - _Acceptance: Contrato validado_

---

## Notes

### Dependency Management

- **Fase 2 depende de**: Fase 1 (estructura, TS config)
- **Fase 3,4,5 dependen de**: Fase 2 (dominio)
- **Fase 6,7,8,9 dependen de**: Fase 3 (puertos)
- **Fase 10,11,12 dependen de**: Fase 4,5 (aplicación)
- **Fase 13 depende de**: Fases 6-12
- **Fase 14,15,16 dependen de**: Fase 13 (integración)

### Testing Notes

- Enfoque: Pruebas críticas de **flujos de usuario**, no cobertura exhaustiva
- Según `testing.md`: "pocas pruebas, valiosas y difíciles de romper"
- 3 tests integración que cubren: carga inicial, actualización SSE, validación datos
- Tests siguen: caja negra, entrada por UI, salida observable
- No se mockean módulos internos (solo MSW en bordes externos)
- Todos los tests son deterministas y rápidos (< 2s cada uno)
- Trazabilidad: cada test referencia criterios de aceptación específicos

### Propiedades Verificadas

- **Propiedad 1**: El número de marcadores = número de vehículos cargados
- **Propiedad 2**: Idempotencia SSE - procesar mismo evento 2 veces = mismo resultado
- **Propiedad 3**: Robustez - eventos inválidos no rompen la app ni interrumpen flujo posterior

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["2.1", "2.2", "2.3", "2.4", "2.5"] },
    { "id": 2, "tasks": ["3.1", "3.2", "3.3"] },
    { "id": 3, "tasks": ["4.1", "4.2", "4.3"] },
    { "id": 4, "tasks": ["5.1"] },
    { "id": 5, "tasks": ["6.1", "6.2", "6.3"] },
    { "id": 6, "tasks": ["7.1"] },
    { "id": 7, "tasks": ["8.1", "8.2", "8.3"] },
    { "id": 8, "tasks": ["9.1"] },
    { "id": 9, "tasks": ["10.1", "10.2", "10.3", "10.4"] },
    { "id": 10, "tasks": ["11.1", "11.2", "11.3", "11.4", "11.5", "11.6", "11.7"] },
    { "id": 11, "tasks": ["12.1"] },
    { "id": 12, "tasks": ["13.1", "13.2", "13.3"] },
    { "id": 13, "tasks": ["14.1", "14.2", "14.3"] },
    { "id": 14, "tasks": ["15.1", "15.2", "15.3"] },
    { "id": 15, "tasks": ["16.1", "16.2", "16.3"] }
  ]
}
```
