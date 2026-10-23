# Design: Captura móvil de GPS y gestión de ubicaciones

## 1. Visión arquitectónica

La aplicación móvil sigue **Clean Architecture** con cuatro capas principales:

```
┌─────────────────────────────────────────┐
│         PRESENTATION LAYER              │
│  (React Native Components, Screens)     │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│       APPLICATION LAYER                 │
│  (Use Cases, Coordinators)              │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         DOMAIN LAYER                    │
│  (Entities, Value Objects, Rules)       │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│    INFRASTRUCTURE LAYER                 │
│  (Adapters: Location, Persistence,      │
│   Connectivity, API, Clock, IDGen)      │
└─────────────────────────────────────────┘
```

Las dependencias **apuntan siempre hacia el dominio**, nunca desde el dominio hacia las capas externas.

## 2. Estructura de directorios

```
mobile/
└── src/
    ├── domain/
    │   ├── models/
    │   │   ├── Position.ts              # Entidad: ubicación capturada
    │   │   ├── Coordinate.ts            # Objeto de valor: lat/lon/accuracy
    │   │   ├── CaptureSession.ts        # Agregado: sesión de captura
    │   │   └── TraceContext.ts          # Objeto de valor: trace_id, idempotency_key
    │   ├── errors/
    │   │   ├── DomainError.ts           # Base para errores de dominio
    │   │   ├── InvalidCoordinateError.ts
    │   │   ├── PersistenceError.ts
    │   │   └── LocationError.ts
    │   └── repositories/
    │       └── PositionRepository.ts   # Contrato sin implementación
    │
    ├── application/
    │   ├── ports/
    │   │   ├── LocationProvider.ts     # Abstracción: GPS
    │   │   ├── PersistenceProvider.ts  # Abstracción: SQLite
    │   │   ├── ConnectivityProvider.ts # Abstracción: Red
    │   │   ├── ApiClient.ts            # Abstracción: HTTP
    │   │   ├── Clock.ts                # Abstracción: Tiempo
    │   │   └── IDGenerator.ts          # Abstracción: UUIDs
    │   │
    │   └── usecases/
    │       ├── CapturePositionUseCase.ts  # Capturar GPS → guardar local
    │       ├── GetCapturedPositionsUseCase.ts
    │       ├── StartCaptureUseCase.ts
    │       ├── StopCaptureUseCase.ts
    │       ├── GetConnectivityStatusUseCase.ts
    │       └── PreparePayloadUseCase.ts  # Formatear para envío
    │
    ├── infrastructure/
    │   ├── location/
    │   │   ├── RealLocationProvider.ts   # Implementación: GPS real
    │   │   ├── FakeLocationProvider.ts   # Para testing
    │   │   └── LocationPermissions.ts    # Manejo de permisos
    │   │
    │   ├── persistence/
    │   │   ├── SQLitePersistenceProvider.ts
    │   │   ├── database/
    │   │   │   ├── migrations/
    │   │   │   │   └── v1_initial_schema.sql
    │   │   │   └── dao/
    │   │   │       └── PositionDAO.ts
    │   │   └── FakePersistenceProvider.ts
    │   │
    │   ├── connectivity/
    │   │   ├── RealConnectivityProvider.ts
    │   │   └── FakeConnectivityProvider.ts
    │   │
    │   ├── api/
    │   │   ├── ApiClient.ts              # HTTP client
    │   │   └── types/
    │   │       └── PositionPayload.ts
    │   │
    │   ├── time/
    │   │   ├── RealClock.ts
    │   │   └── FakeClock.ts
    │   │
    │   └── idgen/
    │       ├── RealIDGenerator.ts        # Genera UUIDs reales
    │       └── FakeIDGenerator.ts        # UUIDs deterministas
    │
    ├── presentation/
    │   ├── screens/
    │   │   └── CaptureScreen.tsx         # Pantalla principal
    │   │
    │   ├── components/
    │   │   ├── CaptureControls.tsx       # Botones start/stop
    │   │   ├── ConnectivityIndicator.tsx
    │   │   ├── PositionsList.tsx         # Listado reactivo
    │   │   └── PositionItem.tsx          # Item individual
    │   │
    │   ├── hooks/
    │   │   ├── useCapture.ts             # Orquesta captura
    │   │   ├── usePositions.ts           # Orquesta listado
    │   │   ├── useConnectivity.ts        # Orquesta conectividad
    │   │   └── useLocationPermissions.ts
    │   │
    │   └── App.tsx                       # Punto de entrada
    │
    ├── services/
    │   ├── DependencyContainer.ts        # Inyección de dependencias
    │   └── ServiceFactory.ts             # Composición de servicios
    │
    └── config/
        └── environment.ts                # Configuración por entorno

tests/
├── support/
│   ├── mothers/
│   │   ├── PositionMother.ts           # Object Mother
│   │   ├── CoordinateMother.ts
│   │   └── TraceContextMother.ts
│   │
│   ├── fakes/
│   │   ├── FakeLocationProvider.ts
│   │   ├── FakePersistenceProvider.ts
│   │   ├── FakeConnectivityProvider.ts
│   │   ├── FakeClock.ts
│   │   └── FakeIDGenerator.ts
│   │
│   └── setup.ts                        # Configuración de test
│
├── unit/
│   ├── domain/
│   │   ├── Position.test.ts
│   │   ├── Coordinate.test.ts
│   │   └── TraceContext.test.ts
│   │
│   └── application/
│       ├── CapturePositionUseCase.test.ts
│       ├── GetCapturedPositionsUseCase.test.ts
│       └── ...
│
└── integration/
    ├── sqlite-capture.test.ts          # Captura + persistencia
    └── connectivity-detection.test.ts  # Detección de red
```

## 3. Modelos de dominio

### Position (Entidad - EXTENDIDO con Backoff)

```typescript
class Position {
  readonly eventId: EventId;
  readonly vehicleId: VehicleId;
  readonly coordinate: Coordinate;
  readonly capturedAt: Date;
  readonly traceContext: TraceContext;
  readonly status: PositionStatus; // 'pending' | 'sent' | 'error' | 'validation_error'
  readonly errorReason?: string;
  readonly isValidationError: boolean; // true si error es 4xx de validación
  readonly updatedAt: Date;
  readonly lastSyncAttempt?: Date; // Cuándo fue el último intento
  readonly syncAttemptCount: number; // Número de reintentos
  readonly nextRetryAt?: Date; // Cuándo reintentar (exponential backoff)
  
  constructor(
    eventId: EventId,
    vehicleId: VehicleId,
    coordinate: Coordinate,
    capturedAt: Date,
    traceContext: TraceContext,
    status: PositionStatus = 'pending',
    errorReason?: string,
    updatedAt?: Date,
    isValidationError: boolean = false,
    lastSyncAttempt?: Date,
    syncAttemptCount: number = 0,
    nextRetryAt?: Date
  ) { 
    this.eventId = eventId;
    this.vehicleId = vehicleId;
    this.coordinate = coordinate;
    this.capturedAt = capturedAt;
    this.traceContext = traceContext;
    this.status = status;
    this.errorReason = errorReason;
    this.updatedAt = updatedAt || new Date();
    this.isValidationError = isValidationError;
    this.lastSyncAttempt = lastSyncAttempt;
    this.syncAttemptCount = syncAttemptCount;
    this.nextRetryAt = nextRetryAt;
  }
  
  isValid(): boolean { ... }
  toPayload(): PositionPayload { ... }
  
  /**
   * Marca esta posición como enviada al backend (202 Accepted).
   * El registro NO se borra de SQLite, solo cambia su status.
   * Limpia intentos y nextRetryAt.
   * 
   * @returns Nueva instancia de Position con status='sent'
   */
  markAsSent(): Position {
    return new Position(
      this.eventId,
      this.vehicleId,
      this.coordinate,
      this.capturedAt,
      this.traceContext,
      'sent',
      undefined, // Limpiar errorReason
      new Date(), // Actualizar updatedAt
      false, // No hay error de validación
      new Date(), // lastSyncAttempt actual
      0, // Resetear contador de intentos
      undefined // Limpiar nextRetryAt
    );
  }
  
  /**
   * Marca esta posición como fallida al sincronizar.
   * Calcula nextRetryAt usando exponential backoff: min(2s * 2^(attempts-1), 300s).
   * 
   * @param reason Descripción del error
   * @param isValidationError true si es error 4xx (no reintentar)
   * @param clock Proveedor de hora (para calcular nextRetryAt)
   * @returns Nueva instancia de Position con status='error' y backoff calculado
   */
  markAsError(reason: string, isValidationError: boolean, clock: Clock): Position {
    const newSyncAttemptCount = this.syncAttemptCount + 1;
    
    let nextRetryAt: Date | undefined;
    
    // Solo calcular nextRetryAt si NO es error de validación
    if (!isValidationError) {
      // Exponential backoff: next_retry_at = now + min(2s * 2^(attempts-1), 300s)
      const baseDelayMs = 2000; // 2 segundos
      const maxDelayMs = 300000; // 300 segundos = 5 minutos
      const exponentialDelayMs = baseDelayMs * Math.pow(2, newSyncAttemptCount - 1);
      const delayMs = Math.min(exponentialDelayMs, maxDelayMs);
      
      nextRetryAt = new Date(clock.now().getTime() + delayMs);
    }

    return new Position(
      this.eventId,
      this.vehicleId,
      this.coordinate,
      this.capturedAt,
      this.traceContext,
      isValidationError ? 'validation_error' : 'error',
      reason,
      new Date(), // updatedAt actual
      isValidationError,
      new Date(), // lastSyncAttempt actual
      newSyncAttemptCount,
      nextRetryAt
    );
  }

  /**
   * Verifica si esta posición está lista para ser sincronizada AHORA.
   * 
   * Retorna true si:
   * - status='pending' (nunca se intentó)
   * - status='error' Y nextRetryAt <= ahora (es hora de reintentar)
   * 
   * Retorna false si:
   * - status='sent' (ya fue enviado)
   * - status='validation_error' (error de datos, no reintentar)
   * - status='error' pero nextRetryAt > ahora (esperar más)
   */
  canBeSynced(now: Date = new Date()): boolean {
    if (this.status === 'pending') return true;
    
    if (this.status === 'error' && this.nextRetryAt) {
      return now >= this.nextRetryAt;
    }
    
    return false;
  }

  /**
   * Verifica si esta posición ya fue enviada exitosamente.
   */
  isAlreadySent(): boolean {
    return this.status === 'sent';
  }

  /**
   * Verifica si esta posición tiene error de validación (no reintentar).
   */
  hasValidationError(): boolean {
    return this.status === 'validation_error';
  }

  /**
   * Obtiene el status visual legible para UI.
   */
  getStatusLabel(): string {
    const labels: Record<PositionStatus, string> = {
      'pending': 'Pendiente',
      'sent': 'Enviado',
      'error': 'Error',
      'validation_error': 'Error de datos',
    };
    return labels[this.status];
  }

  /**
   * Obtiene el ícono visual para el status.
   */
  getStatusIcon(): string {
    const icons: Record<PositionStatus, string> = {
      'pending': '🔵',
      'sent': '✓',
      'error': '⚠️',
      'validation_error': '❌',
    };
    return icons[this.status];
  }

  /**
   * Obtiene el próximo momento en el que se intentará sincronizar (para UI).
   */
  getNextRetryText(now: Date = new Date()): string | null {
    if (!this.nextRetryAt) return null;
    
    const diffMs = this.nextRetryAt.getTime() - now.getTime();
    if (diffMs <= 0) return 'Listo para reintentar';
    
    const diffSec = Math.ceil(diffMs / 1000);
    return `Reintentar en ${diffSec}s`;
  }
}

type PositionStatus = 'pending' | 'sent' | 'error' | 'validation_error';
```

**Cambios principales:**
- Cuatro estados: `'pending'`, `'sent'`, `'error'`, `'validation_error'`
- `isValidationError`: Indicador de si es error 4xx (no reintentar)
- `lastSyncAttempt`: Timestamp del último intento
- `syncAttemptCount`: Contador de reintentos (0 al inicio)
- `nextRetryAt`: Momento para próximo intento (calculado con backoff exponencial)
- `markAsError()` ahora calcula automáticamente `nextRetryAt` usando backoff
- `canBeSynced()` ahora verifica también si es hora de reintentar
- `getNextRetryText()`: Texto legible para mostrar en UI ("Reintentar en 32s")
- Inmutabilidad preservada: todos los cambios retornan nuevas instancias

**Cambios principales:**
- Campo `status` con tres valores: `pending` (recién capturado), `sent` (202 Accepted recibido), `error` (fallo en envío)
- Campo `errorReason` para almacenar detalles del error
- Campo `updatedAt` para rastrear cuándo fue el último cambio
- Métodos para transicionar estados: `markAsSent()` y `markAsError(reason)` retornan nuevas instancias (inmutabilidad)
- Métodos auxiliares: `canBeSynced()`, `isAlreadySent()`, `getStatusLabel()`, `getStatusIcon()`
- El status inicial es siempre `pending`
- La inmutabilidad asegura que cambios de estado son explícitos y trazables

### Coordinate (Objeto de valor)

```typescript
class Coordinate {
  readonly latitude: number;   // -90 a 90
  readonly longitude: number;  // -180 a 180
  readonly accuracy?: number;  // metros, > 0
  
  constructor(lat: number, lon: number, accuracy?: number) {
    if (!this.isValidLatitude(lat)) throw new InvalidCoordinateError(...);
    if (!this.isValidLongitude(lon)) throw new InvalidCoordinateError(...);
    this.latitude = lat;
    this.longitude = lon;
    this.accuracy = accuracy;
  }
  
  static fromGPS(gpsData: GPSData): Coordinate { ... }
}
```

### TraceContext (Objeto de valor)

```typescript
class TraceContext {
  readonly traceId: UUID;
  readonly idempotencyKey: UUID;
  
  constructor(traceId: UUID, idempotencyKey: UUID) { ... }
  
  static generate(idGen: IDGenerator): TraceContext { ... }
}
```

### CaptureSession (Agregado)

```typescript
class CaptureSession {
  readonly startedAt: Date;
  isActive: boolean;
  positionCount: number;
  
  start(clock: Clock): void { ... }
  stop(): void { ... }
}
```

## 4. Casos de uso

### CapturePositionUseCase

**Responsabilidad:** Capturar GPS una vez y guardar localmente.

```typescript
interface CapturePositionInput {
  vehicleId: string;
  clock: Clock;
  idGenerator: IDGenerator;
}

class CapturePositionUseCase {
  constructor(
    private locationProvider: LocationProvider,
    private persistenceProvider: PersistenceProvider
  ) {}

  async execute(input: CapturePositionInput): Promise<Position> {
    // 1. Obtener GPS
    const gpsData = await this.locationProvider.getCurrentLocation();
    
    // 2. Crear dominio
    const position = Position.create({
      vehicleId: input.vehicleId,
      coordinate: Coordinate.fromGPS(gpsData),
      capturedAt: input.clock.now(),
      traceContext: TraceContext.generate(input.idGenerator)
    });
    
    // 3. Validar
    if (!position.isValid()) throw new DomainError(...);
    
    // 4. Guardar en SQLite
    await this.persistenceProvider.savePosition(position);
    
    return position;
  }
}
```

### GetCapturedPositionsUseCase

**Responsabilidad:** Recuperar todas las posiciones locales.

```typescript
class GetCapturedPositionsUseCase {
  constructor(private persistenceProvider: PersistenceProvider) {}

  async execute(): Promise<Position[]> {
    return this.persistenceProvider.getAllPositions();
  }
}
```

### StartCaptureUseCase

**Responsabilidad:** Iniciar captura periódica (cada 30s).

```typescript
class StartCaptureUseCase {
  private intervalId?: NodeJS.Timeout;
  
  constructor(
    private capturePositionUseCase: CapturePositionUseCase
  ) {}

  execute(vehicleId: string): void {
    if (this.intervalId) return; // Ya activo
    
    this.intervalId = setInterval(async () => {
      await this.capturePositionUseCase.execute({ vehicleId });
    }, 30000);
  }
}
```

### StopCaptureUseCase

**Responsabilidad:** Detener captura periódica.

```typescript
class StopCaptureUseCase {
  execute(startCaptureUseCase: StartCaptureUseCase): void {
    startCaptureUseCase.stopInterval();
  }
}
```

### GetConnectivityStatusUseCase

**Responsabilidad:** Obtener estado actual de conectividad.

```typescript
class GetConnectivityStatusUseCase {
  constructor(private connectivityProvider: ConnectivityProvider) {}

  async execute(): Promise<ConnectivityStatus> {
    return this.connectivityProvider.getStatus();
  }
}
```

### PreparePayloadUseCase

**Responsabilidad:** Formatear Position para envío al backend.

```typescript
class PreparePayloadUseCase {
  async execute(positions: Position[]): Promise<PositionPayload[]> {
    // Convertir a formato esperado por backend
    return positions.map(p => p.toPayload());
  }
}
```

### SyncPositionUseCase (NUEVO)

**Responsabilidad:** Enviar un registro individual al backend y manejar respuesta 202 Accepted.

```typescript
interface SyncPositionInput {
  position: Position;
}

interface SyncPositionOutput {
  success: boolean;
  status: PositionStatus; // 'sent' si 202, 'error' o 'validation_error' si fallo
  eventId: string;
}

class SyncPositionUseCase {
  constructor(
    private apiClient: ApiClient,
    private persistenceProvider: PersistenceProvider,
    private connectivityProvider: ConnectivityProvider,
    private clock: Clock
  ) {}

  async execute(input: SyncPositionInput): Promise<SyncPositionOutput> {
    // 1. Verificar conectividad
    const connectivity = await this.connectivityProvider.getStatus();
    if (connectivity === 'offline') {
      throw new ConnectivityError('No hay conectividad disponible');
    }

    try {
      // 2. Preparar payload
      const payload = input.position.toPayload();

      // 3. Enviar al backend
      const response = await this.apiClient.sendPositions([payload]);

      // 4. Manejar respuesta 202 Accepted
      if (response.status === 202) {
        // Cambiar status de 'pending' a 'sent' sin borrar de SQLite
        const updatedPosition = input.position.markAsSent();
        await this.persistenceProvider.updatePosition(updatedPosition);

        return {
          success: true,
          status: 'sent',
          eventId: input.position.eventId.toString(),
        };
      }

      // 5. Manejar otros status exitosos
      if (response.status === 200 || response.status === 201) {
        const updatedPosition = input.position.markAsSent();
        await this.persistenceProvider.updatePosition(updatedPosition);

        return {
          success: true,
          status: 'sent',
          eventId: input.position.eventId.toString(),
        };
      }

      // 6. Diferenciar entre errores de validación (4xx) y errores transitorios (5xx, timeout)
      const isValidationError = response.status && response.status >= 400 && response.status < 500;
      
      // Error: cambiar status a 'error' o 'validation_error'
      const errorPosition = input.position.markAsError(
        response.error?.message || 'Unknown error',
        isValidationError,
        this.clock
      );
      await this.persistenceProvider.updatePosition(errorPosition);

      return {
        success: false,
        status: errorPosition.status,
        eventId: input.position.eventId.toString(),
      };
    } catch (error) {
      // Manejo de excepciones (timeout, network error, etc.)
      // Son errores transitorios, NO de validación
      const errorPosition = input.position.markAsError(
        error.message,
        false, // No es error de validación
        this.clock
      );
      await this.persistenceProvider.updatePosition(errorPosition);

      return {
        success: false,
        status: 'error',
        eventId: input.position.eventId.toString(),
      };
    }
  }
}
```

### SyncPendingPositionsUseCase (NUEVO)

**Responsabilidad:** Reintentar posiciones con status='error' cuyo nextRetryAt ha llegado.

```typescript
interface SyncPendingInput {
  maxConcurrent?: number; // Máximo de sincronizaciones simultáneas
}

interface SyncPendingOutput {
  synced: number;
  failed: number;
  updated: Position[];
}

class SyncPendingPositionsUseCase {
  constructor(
    private persistenceProvider: PersistenceProvider,
    private syncPositionUseCase: SyncPositionUseCase,
    private clock: Clock
  ) {}

  async execute(input: SyncPendingInput = {}): Promise<SyncPendingOutput> {
    const maxConcurrent = input.maxConcurrent || 3;
    const now = this.clock.now();

    // 1. Obtener posiciones listas para reintentar (pending + error con nextRetryAt <= now)
    const pendingPositions = await this.persistenceProvider.getPositionsByStatus('pending');
    const errorPositions = await this.persistenceProvider.getPositionsByStatus('error');
    
    // Filtrar errores que ya pueden reintentarse
    const retryableErrors = errorPositions.filter(
      p => p.nextRetryAt === undefined || p.nextRetryAt <= now
    );

    const allRetryable = [...pendingPositions, ...retryableErrors];

    if (allRetryable.length === 0) {
      return { synced: 0, failed: 0, updated: [] };
    }

    // 2. Sincronizar en paralelo (máximo maxConcurrent)
    const updated: Position[] = [];
    let synced = 0;
    let failed = 0;

    for (let i = 0; i < allRetryable.length; i += maxConcurrent) {
      const batch = allRetryable.slice(i, i + maxConcurrent);
      
      const results = await Promise.allSettled(
        batch.map(pos => this.syncPositionUseCase.execute({ position: pos }))
      );

      for (const result of results) {
        if (result.status === 'fulfilled' && result.value.success) {
          synced++;
        } else {
          failed++;
        }
      }
    }

    // 3. Recargar estado actual desde persistencia
    const allPositions = await this.persistenceProvider.getAllPositions();
    const updatedPositions = allPositions.filter(
      p => allRetryable.some(retry => retry.eventId === p.eventId)
    );

    return {
      synced,
      failed,
      updated: updatedPositions,
    };
  }
}
```

### RetryFailedSyncUseCase (NUEVO - Alias de SyncPendingPositionsUseCase)

Caso de uso especial que expone `SyncPendingPositionsUseCase` con un nombre explícito para reintentos fallidos en UI (botón "Reintentar todo").

```typescript
/**
 * Reintentar registros con status='error' o 'pending'.
 * Es un alias conceptual de SyncPendingPositionsUseCase.
 * 
 * Flujo:
 * 1. Obtener posiciones con error cuyo nextRetryAt <= now
 * 2. Obtener todas las posiciones pending
 * 3. Sincronizar hasta 3 en paralelo (para no sobrecargar)
 * 4. Retornar cuántas se sincronizaron y cuántas fallaron
 * 5. UI muestra badge con el total: "Reintentar (N)"
 */
class RetryFailedSyncUseCase extends SyncPendingPositionsUseCase {
  // Hereda toda la lógica de SyncPendingPositionsUseCase
}
```

### CaptureAndSyncPositionUseCase (NUEVO - Flujo Completo)

**Responsabilidad:** Capturar GPS, guardar localmente como 'pending', e intentar enviar inmediatamente.

```typescript
interface CaptureAndSyncInput {
  vehicleId: string;
}

interface CaptureAndSyncOutput {
  position: Position;
  syncAttempted: boolean;
  syncSuccess?: boolean;
}

class CaptureAndSyncPositionUseCase {
  constructor(
    private capturePositionUseCase: CapturePositionUseCase,
    private syncPositionUseCase: SyncPositionUseCase,
    private connectivityProvider: ConnectivityProvider
  ) {}

  async execute(input: CaptureAndSyncInput): Promise<CaptureAndSyncOutput> {
    // 1. Capturar GPS y guardar como 'pending' en SQLite
    const position = await this.capturePositionUseCase.execute({
      vehicleId: input.vehicleId,
    });

    // 2. Verificar conectividad
    const connectivity = await this.connectivityProvider.getStatus();
    const isOnline = connectivity === 'online';

    // 3. Si hay conectividad, intentar enviar inmediatamente
    if (isOnline) {
      try {
        const result = await this.syncPositionUseCase.execute({ position });
        return {
          position,
          syncAttempted: true,
          syncSuccess: result.success,
        };
      } catch (error) {
        // Si falla, la posición quedó guardada como 'error' o 'pending'
        // El usuario puede reintentar manualmente después
        console.error('Sync failed immediately:', error);
        return {
          position,
          syncAttempted: true,
          syncSuccess: false,
        };
      }
    }

    // 4. Si no hay conectividad, solo guardar (para sincronizar después)
    return {
      position,
      syncAttempted: false,
      syncSuccess: undefined,
    };
  }
}
```

**Cambios principales:**
- Nuevo caso de uso para sincronización de un registro individual
- Maneja 202 Accepted marcando como 'sent' sin borrar
- Actualiza status a 'error' si fallo
- Retorna información clara del resultado

## 5. Interfaces de infraestructura (Ports)

### LocationProvider

```typescript
interface LocationProvider {
  getCurrentLocation(): Promise<GPSData>;
  requestPermissions(): Promise<'granted' | 'denied'>;
  isPermissionGranted(): boolean;
  startBackgroundTracking?(interval: number): void;
  stopBackgroundTracking?(): void;
}

interface GPSData {
  latitude: number;
  longitude: number;
  accuracy: number;
  timestamp: number;
}
```

### PersistenceProvider (ACTUALIZADO - Con Backoff y Conteos)

```typescript
/**
 * Abstracción para acceso a persistencia local (SQLite).
 * 
 * Responsabilidades:
 * - Guardar y recuperar posiciones
 * - Cambiar estado de posiciones (status)
 * - Filtrar por estado para UI y sincronización
 * - Notificar a listeners sobre cambios
 */
interface PersistenceProvider {
  /**
   * Guardar una nueva posición en SQLite.
   * La posición siempre inicia con status='pending'.
   * 
   * @param position Entidad Position a guardar
   */
  savePosition(position: Position): Promise<void>;

  /**
   * Actualizar una posición existente (especialmente su status).
   * Utilizado cuando SyncPositionUseCase cambia status a 'sent', 'error' o 'validation_error'.
   * 
   * @param position Entidad Position con status actualizado
   */
  updatePosition(position: Position): Promise<void>;

  /**
   * Obtener todas las posiciones almacenadas en orden descendente por captured_at.
   * 
   * @returns Array de todas las posiciones
   */
  getAllPositions(): Promise<Position[]>;

  /**
   * Obtener posiciones de un vehículo específico.
   * 
   * @param vehicleId ID del vehículo
   * @returns Array de posiciones filtradas
   */
  getPositionsByVehicle(vehicleId: string): Promise<Position[]>;

  /**
   * Obtener posiciones filtradas por status.
   * Utilizado por SyncPositionUseCase para encontrar registros pendientes y con error.
   * 
   * @param status Status a filtrar: 'pending', 'sent', 'error', 'validation_error'
   * @returns Array de posiciones con ese status (sin filtrar por nextRetryAt)
   */
  getPositionsByStatus(status: PositionStatus): Promise<Position[]>;

  /**
   * Obtener posiciones que están listas para sincronizarse AHORA.
   * 
   * Retorna:
   * - Todas las posiciones con status='pending'
   * - Posiciones con status='error' donde nextRetryAt <= now
   * 
   * Utilizado por SyncPendingPositionsUseCase.
   * 
   * @param now Hora actual (para comparar con nextRetryAt)
   * @returns Array de posiciones listas para sincronizar
   */
  getPositionsPendingRetry(now?: Date): Promise<Position[]>;

  /**
   * Obtener una posición específica por event_id.
   * 
   * @param eventId ID único del evento
   * @returns Position o undefined si no existe
   */
  getPositionById(eventId: string): Promise<Position | undefined>;

  /**
   * Borrar una posición específica.
   * Consideración futura: usar cuando se confirme persistencia final en backend.
   * 
   * @param eventId ID único del evento
   */
  deletePosition(eventId: string): Promise<void>;

  /**
   * Borrar todas las posiciones.
   * 
   */
  deleteAllPositions(): Promise<void>;

  /**
   * Obtener el total de posiciones almacenadas.
   * 
   * @returns Número de posiciones
   */
  getPositionCount(): Promise<number>;

  /**
   * Obtener el total de posiciones por status.
   * Utilizado por UI para mostrar desglose: "N Pendiente | M Enviado | P Error | L Datos inválidos"
   * 
   * @param status Status a contar
   * @returns Número de posiciones con ese status
   */
  getPositionCountByStatus(status: PositionStatus): Promise<number>;

  /**
   * Obtener todos los conteos por status en una sola query.
   * Optimización: evita 4 queries separadas.
   * 
   * @returns Objeto { pending: N, sent: M, error: P, validation_error: L }
   */
  getPositionCountsByAllStatus(): Promise<Record<PositionStatus, number>>;

  /**
   * Registrar un listener que se dispara cuando se captura una nueva posición.
   * Utilizado por usePositions hook para actualizar UI.
   * 
   * @param callback Función a ejecutar cuando se capture un registro
   * @returns Función para desuscribirse
   */
  onPositionCaptured(callback: (position: Position) => void): () => void;

  /**
   * Registrar un listener que se dispara cuando cambia el status de una posición.
   * Utilizado por usePositions hook para actualizar item específico reactivamente.
   * 
   * @param callback Función a ejecutar cuando cambie status
   * @returns Función para desuscribirse
   */
  onPositionStatusChanged(
    callback: (position: Position) => void
  ): () => void;

  /**
   * Inicializar o verificar integridad de la base de datos.
   */
  initialize(): Promise<void>;

  /**
   * Cerrar la conexión a la base de datos.
   */
  close(): Promise<void>;
}

type PositionStatus = 'pending' | 'sent' | 'error' | 'validation_error';
```

**Cambios principales:**
- `getPositionsPendingRetry(now)`: Obtener registros listos para sincronizarse AHORA (pending + error con nextRetryAt <= now)
- `getPositionCountsByAllStatus()`: Obtener todos los conteos en una query para optimizar UI
- Soporte para cuatro estados: `'pending'`, `'sent'`, `'error'`, `'validation_error'`
- Documentación clara de cuándo usar cada método

**Cambios principales:**
- `updatePosition()`: Actualizar un registro existente (para cambios de status después de sincronización)
- `getPositionsByStatus()`: Filtrar por status ('pending', 'sent', 'error') para encontrar registros a sincronizar
- `getPositionCountByStatus()`: Contar registros por status para mostrar desglose en UI: "N Pendiente | M Enviado | P Error"
- `onPositionCaptured()`: Evento cuando se captura nuevo registro (actualiza listado)
- `onPositionStatusChanged()`: Evento cuando cambia status de un registro (actualiza UI reactivamente)
- `getPositionById()`: Recuperar un registro específico
- Documentación detallada de cada método y su propósito
- Los eventos son fundamentales para la reactividad del listado SIN polling cada 5 segundos

### ConnectivityProvider

```typescript
interface ConnectivityProvider {
  getStatus(): Promise<ConnectivityStatus>;
  onStatusChange(callback: (status: ConnectivityStatus) => void): void;
}

type ConnectivityStatus = 'online' | 'offline' | 'unknown';
```

### ApiClient

```typescript
interface ApiClient {
  sendPositions(payload: PositionPayload[]): Promise<SendPositionsResponse>;
}

interface PositionPayload {
  event_id: string;
  vehicle_id: string;
  latitude: number;
  longitude: number;
  accuracy?: number;
  captured_at: string; // ISO 8601
  trace_id: string;
  idempotency_key: string;
}
```

### Clock y IDGenerator

```typescript
interface Clock {
  now(): Date;
}

interface IDGenerator {
  generate(): string; // UUID v4
}
```

## 6. Componentes de presentación

### CaptureScreen

Pantalla principal que orquesta:
- CaptureControls (botones start/stop)
- ConnectivityIndicator (estado de red)
- PositionsList (listado de ubicaciones con botones de sincronización individual)
- Contador de registros desglosado por status: "N Pendiente | M Enviado | P Error"

### PositionItem (ACTUALIZADO - Sincronización Individual)

Componente para cada item del listado que muestra información del registro y permite sincronización individual:

```typescript
interface PositionItemProps {
  position: Position;
  onSync: (position: Position) => Promise<void>;
  isLoading?: boolean;
  isConnected: boolean; // Desde ConnectivityIndicator
}

const PositionItem: React.FC<PositionItemProps> = ({
  position,
  onSync,
  isLoading = false,
  isConnected,
}) => {
  const [syncError, setSyncError] = React.useState<string | null>(null);
  const [isSyncing, setIsSyncing] = React.useState(false);

  const handleSync = async () => {
    if (!isConnected) {
      setSyncError('Sin conectividad disponible');
      return;
    }

    if (position.status === 'sent') {
      setSyncError('Ya fue enviado correctamente');
      return;
    }

    try {
      setIsSyncing(true);
      setSyncError(null);
      await onSync(position);
    } catch (error) {
      setSyncError(error.message || 'Error al sincronizar');
    } finally {
      setIsSyncing(false);
    }
  };

  const statusIcon = position.getStatusIcon(); // 🔵, ✓, ❌
  const statusLabel = position.getStatusLabel(); // "Pendiente", "Enviado", "Error"

  const isSyncDisabled = !isConnected || position.status === 'sent' || isLoading || isSyncing;

  return (
    <View style={styles.container}>
      {/* Encabezado con status visual */}
      <View style={styles.header}>
        <Text style={styles.statusIndicator}>
          {statusIcon} {statusLabel}
        </Text>
        {position.status === 'error' && position.errorReason && (
          <Text style={styles.errorReason}>{position.errorReason}</Text>
        )}
      </View>

      {/* Datos de la ubicación */}
      <View style={styles.details}>
        <Text style={styles.vehicleId}>Vehículo: {position.vehicleId}</Text>
        <Text style={styles.coordinates}>
          📍 {position.coordinate.latitude.toFixed(6)}, {position.coordinate.longitude.toFixed(6)}
        </Text>
        {position.coordinate.accuracy && (
          <Text style={styles.accuracy}>
            Precisión: ±{position.coordinate.accuracy.toFixed(1)}m
          </Text>
        )}
        <Text style={styles.timestamp}>
          {position.capturedAt.toLocaleString()}
        </Text>
      </View>

      {/* Botón de sincronización individual */}
      <View style={styles.actionContainer}>
        <Button
          title={isSyncing ? 'Sincronizando...' : 'Sincronizar'}
          onPress={handleSync}
          disabled={isSyncDisabled}
        />
      </View>

      {/* Mostrar errores de sincronización */}
      {syncError && (
        <Text style={styles.syncError}>{syncError}</Text>
      )}
    </View>
  );
};

const styles = {
  container: {
    padding: 12,
    marginVertical: 8,
    backgroundColor: '#f5f5f5',
    borderRadius: 8,
    borderLeftWidth: 4,
    borderLeftColor: '#2196F3', // Color dinámico según status
  },
  header: {
    marginBottom: 8,
  },
  statusIndicator: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#333',
  },
  errorReason: {
    fontSize: 12,
    color: '#d32f2f',
    marginTop: 4,
  },
  details: {
    marginBottom: 12,
  },
  vehicleId: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
    marginBottom: 4,
  },
  coordinates: {
    fontSize: 13,
    color: '#666',
    fontFamily: 'monospace',
    marginBottom: 4,
  },
  accuracy: {
    fontSize: 12,
    color: '#999',
    marginBottom: 4,
  },
  timestamp: {
    fontSize: 12,
    color: '#999',
  },
  actionContainer: {
    marginBottom: 8,
  },
  syncError: {
    fontSize: 12,
    color: '#d32f2f',
    marginTop: 8,
  },
};

export default PositionItem;
```

**Cambios principales:**
- Muestra status visual con ícono y etiqueta: "🔵 Pendiente", "✓ Enviado", "❌ Error"
- Botón "Sincronizar" individual para ese registro específico
- Botón deshabilitado si:
  - No hay conectividad (`isConnected === false`)
  - Status es 'sent' (ya fue enviado)
  - Ya está sincronizando (`isSyncing === true`)
- Manejo de errores de sincronización con retroalimentación visual
- Muestra `errorReason` si status es 'error'
- Color de borde dinámico según status
- Diseño responsive y accesible

### usePositions Hook (ACTUALIZADO - Reactividad por Eventos)

```typescript
/**
 * Hook que gestiona el listado de posiciones almacenadas localmente.
 * 
 * Cambios clave:
 * - NO actualiza cada 5 segundos (optimización)
 * - Se actualiza ÚNICAMENTE cuando:
 *   1. Se captura nueva posición (evento onPositionCaptured)
 *   2. Cambia status de una posición (evento onPositionStatusChanged)
 *   3. La app vuelve del background (AppState.change)
 * 
 * @returns {Object} positions, countByStatus, loadPositions
 */
export const usePositions = (persistenceProvider: PersistenceProvider) => {
  const [positions, setPositions] = React.useState<Position[]>([]);
  const [countByStatus, setCountByStatus] = React.useState({
    pending: 0,
    sent: 0,
    error: 0,
  });
  const [isLoading, setIsLoading] = React.useState(true);

  /**
   * Cargar todas las posiciones y actualizar conteo por status.
   */
  const loadPositions = React.useCallback(async () => {
    try {
      setIsLoading(true);
      const loaded = await persistenceProvider.getAllPositions();
      setPositions(loaded);

      // Actualizar conteos
      const counts = {
        pending: await persistenceProvider.getPositionCountByStatus('pending'),
        sent: await persistenceProvider.getPositionCountByStatus('sent'),
        error: await persistenceProvider.getPositionCountByStatus('error'),
      };
      setCountByStatus(counts);
    } catch (error) {
      console.error('Error loading positions:', error);
    } finally {
      setIsLoading(false);
    }
  }, [persistenceProvider]);

  React.useEffect(() => {
    // 1. Cargar posiciones al montar el hook
    loadPositions();

    // 2. Escuchar evento de nueva posición capturada
    const unsubscribeCapture = persistenceProvider.onPositionCaptured((newPosition) => {
      setPositions((prev) => [newPosition, ...prev]); // Insertar al inicio
      setCountByStatus((prev) => ({
        ...prev,
        pending: prev.pending + 1,
      }));
    });

    // 3. Escuchar cambios de status (sincronización)
    const unsubscribeStatusChange = persistenceProvider.onPositionStatusChanged(
      (updatedPosition) => {
        setPositions((prev) =>
          prev.map((p) => (p.eventId === updatedPosition.eventId ? updatedPosition : p))
        );

        // Actualizar conteos: decrementar status anterior, incrementar nuevo
        setCountByStatus((prev) => {
          const updated = { ...prev };

          // Buscar status anterior
          const oldPosition = positions.find((p) => p.eventId === updatedPosition.eventId);
          if (oldPosition && oldPosition.status !== updatedPosition.status) {
            updated[oldPosition.status]--;
            updated[updatedPosition.status]++;
          }

          return updated;
        });
      }
    );

    // 4. Escuchar cambios de estado de la app (background/foreground)
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') {
        // Recargar cuando la app vuelve del background
        loadPositions();
      }
    });

    // 5. Limpiar subscripciones al desmontar
    return () => {
      unsubscribeCapture();
      unsubscribeStatusChange();
      subscription.remove();
    };
  }, [persistenceProvider, loadPositions]);

  return {
    positions, // Array de todas las posiciones ordenadas desc por captured_at
    countByStatus, // { pending: N, sent: M, error: P }
    totalCount: positions.length,
    isLoading,
    reload: loadPositions, // Función para recargar manualmente si es necesario
  };
};

/**
 * Hook para sincronizar una posición individual al backend.
 * 
 * @param syncPositionUseCase Caso de uso inyectado
 * @returns {Object} syncOne, isSyncing, error
 */
export const useSyncPosition = (syncPositionUseCase: SyncPositionUseCase) => {
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const syncOne = React.useCallback(
    async (position: Position): Promise<void> => {
      try {
        setIsSyncing(true);
        setError(null);

        const result = await syncPositionUseCase.execute({ position });

        if (!result.success) {
          throw new Error(result.statusReason || 'Sincronización fallida');
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Error desconocido';
        setError(errorMessage);
        throw err; // Propagar para que el componente maneje el error
      } finally {
        setIsSyncing(false);
      }
    },
    [syncPositionUseCase]
  );

  return { syncOne, isSyncing, error };
};

/**
 * Hook para conectividad.
 * 
 * @param connectivityProvider Proveedor inyectado
 * @returns {Object} isConnected, status
 */
export const useConnectivity = (connectivityProvider: ConnectivityProvider) => {
  const [status, setStatus] = React.useState<ConnectivityStatus>('unknown');

  React.useEffect(() => {
    // Obtener estado inicial
    connectivityProvider.getStatus().then(setStatus);

    // Escuchar cambios
    const unsubscribe = connectivityProvider.onStatusChange((newStatus) => {
      setStatus(newStatus);
    });

    return unsubscribe;
  }, [connectivityProvider]);

  return {
    isConnected: status === 'online',
    status,
  };
};
```

**Cambios principales:**
- `usePositions` NO actualiza cada 5 segundos (ahorro de renders y batería)
- Se actualiza ÚNICAMENTE en eventos concretos:
  1. `onPositionCaptured`: Nueva posición capturada
  2. `onPositionStatusChanged`: Cambió el status (ej: 'pending' → 'sent')
  3. `AppState.change`: La app vuelve del background
- Hook `useSyncPosition` encapsula SyncPositionUseCase con manejo de estado
- Hook `useConnectivity` reactivo a cambios de conectividad
- Conteo desglosado por status: `countByStatus = { pending: N, sent: M, error: P }`
- Retorna `reload()` para recargar manualmente si es necesario

## 7. Flujos principales

### Flujo 1: Captura periódica de GPS

```
Usuario toca "Iniciar Captura"
  ↓
StartCaptureUseCase.execute()
  ├─ Validar que no hay captura activa
  └─ Iniciar intervalo de 30 segundos
  
Cada 30 segundos:
  ↓
  CapturePositionUseCase.execute({ vehicleId, clock, idGenerator })
    ├─ LocationProvider.getCurrentLocation()
    │  └─ Si denegado: throw LocationError
    │
    ├─ Crear Position (dominio) con:
    │  ├─ eventId: UUID único
    │  ├─ vehicleId: string
    │  ├─ coordinate: Coordinate (lat, lon, accuracy)
    │  ├─ capturedAt: clock.now() (UTC)
    │  ├─ traceContext: TraceContext (traceId, idempotencyKey)
    │  └─ status: 'pending' (inicial siempre)
    │
    ├─ Validar Position.isValid()
    │  └─ Si inválida: throw ValidationError
    │
    ├─ PersistenceProvider.savePosition(position)
    │  └─ INSERT INTO positions (..., status='pending', ...)
    │
    └─ Emitir evento onPositionCaptured(position)
       └─ usePositions hook actualiza UI (agrega al inicio de listado)
       └─ countByStatus.pending++

Usuario toca "Parar Captura"
  ↓
StopCaptureUseCase.execute()
  └─ Detener intervalo
```

### Flujo 2: Sincronización individual de un registro (NUEVO)

```
Usuario visualiza PositionItem con status='pending'
  │
Usuario toca botón "Sincronizar"
  ↓
useSyncPosition hook → SyncPositionUseCase.execute({ position })
  │
  ├─ Validar conectividad
  │  ├─ ConnectivityProvider.getStatus()
  │  └─ Si offline: throw ConnectivityError "No hay conectividad"
  │
  ├─ Verificar status del registro
  │  └─ Si status='sent': throw DomainError "Ya fue enviado"
  │
  ├─ Preparar payload
  │  └─ position.toPayload() → PositionPayload (JSON)
  │
  ├─ Enviar al backend
  │  └─ ApiClient.sendPositions([payload])
  │     └─ POST /api/positions (HTTP)
  │
  └─ Manejar respuesta:
     │
     ├─ SI 202 Accepted (procesamiento asíncrono):
     │  ├─ Crear Position.markAsSent()
     │  │  └─ Nueva instancia con status='sent', errorReason=undefined
     │  │
     │  ├─ PersistenceProvider.updatePosition(updatedPosition)
     │  │  └─ UPDATE positions SET status='sent', updated_at=NOW() WHERE event_id=...
     │  │  └─ NO DELETE (registro permanece en SQLite)
     │  │
     │  ├─ Emitir evento onPositionStatusChanged(updatedPosition)
     │  │  └─ usePositions hook actualiza ese item en el listado
     │  │  └─ PositionItem muestra status '✓ Enviado'
     │  │  └─ countByStatus.pending--, countByStatus.sent++
     │  │
     │  └─ Retornar { success: true, status: 'sent', eventId }
     │
     ├─ SI 200/201 OK (confirmación inmediata):
     │  └─ (Futuro: considerar si borrar o no)
     │  └─ Por ahora: mismo flujo que 202 (marcar 'sent', no borrar)
     │
     └─ SI error (4xx/5xx/timeout):
        ├─ Crear Position.markAsError(errorMessage)
        │  └─ Nueva instancia con status='error', errorReason='...'
        │
        ├─ PersistenceProvider.updatePosition(errorPosition)
        │  └─ UPDATE positions SET status='error', error_reason='...', updated_at=NOW() WHERE event_id=...
        │
        ├─ Emitir evento onPositionStatusChanged(errorPosition)
        │  └─ usePositions hook actualiza ese item
        │  └─ PositionItem muestra status '❌ Error' con razón
        │  └─ countByStatus.pending--, countByStatus.error++
        │
        └─ Retornar { success: false, status: 'error', eventId }
           └─ PositionItem muestra botón "Reintentar" (vuelve a estar habilitado)

Usuario puede hacer click en "Sincronizar" nuevamente
  └─ Repite el flujo para registros con status='error'
```

**Cambios principales en este flujo:**
- Sincronización es INDIVIDUAL (un registro a la vez, no todos)
- 202 Accepted marca como 'sent' pero NO borra de SQLite
- El registro permanece visible en el listado con status '✓ Enviado'
- Errores marcan como 'error' con razón del fallo
- Usuario puede reintentar registros con error
- Actualización de UI es reactiva (eventos), no polling

### Flujo 3: Actualización reactiva de UI (ACTUALIZADO - SIN POLLING)

```
usePositions hook inicializa:
  ├─ Cargar todos los registros: getAllPositions()
  ├─ Calcular conteos: getPositionCountByStatus('pending', 'sent', 'error')
  └─ Mostrar: "N Pendiente | M Enviado | P Error"

Luego escucha exactamente TRES eventos:

1️⃣ EVENTO: Nueva posición capturada
   ├─ persistenceProvider.onPositionCaptured(position)
   ├─ setPositions(prev => [position, ...prev])
   └─ countByStatus.pending++

2️⃣ EVENTO: Status de posición cambió
   ├─ persistenceProvider.onPositionStatusChanged(updatedPosition)
   ├─ setPositions(prev => prev.map(p => 
   │    p.eventId === updatedPosition.eventId ? updatedPosition : p
   │  ))
   └─ Actualizar countByStatus (decrementar status viejo, incrementar nuevo)

3️⃣ EVENTO: App vuelve del background
   ├─ AppState.addEventListener('change', state => {
   │    if (state === 'active') loadPositions()
   │  })
   └─ Recargar desde persistencia para asegurar consistencia

❌ NO HAY polling cada 5 segundos
   └─ Ahorro de renders, CPU, batería

PositionsList actualiza:
  ├─ Mostrar listado en orden descendente por captured_at
  ├─ Cada PositionItem muestra status visual
  ├─ Cada PositionItem tiene botón "Sincronizar"
  └─ Mostrar contador desglosado por status en header
```

### Flujo 4: Funcionamiento sin conectividad (OFFLINE-FIRST)

```
Sin conectividad (offline=true):
  │
  ├─ Captura continúa normalmente
  │  └─ Registros se guardan en SQLite con status='pending'
  │
  ├─ ConnectivityProvider emite onStatusChange('offline')
  │  └─ useConnectivity hook actualiza isConnected=false
  │  └─ UI muestra indicador "🔴 SIN CONEXIÓN"
  │
  ├─ PositionItem.onSync() es llamado
  │  └─ Verifica isConnected === false
  │  └─ Muestra error "Sin conectividad disponible"
  │  └─ Botón "Sincronizar" deshabilitado
  │
  └─ Registros se acumulan localmente sin intentar envío

Se recupera conectividad (online=true):
  │
  ├─ ConnectivityProvider emite onStatusChange('online')
  │  └─ useConnectivity hook actualiza isConnected=true
  │  └─ UI muestra indicador "🟢 CONECTADO"
  │
  ├─ PositionItem.onSync() nuevamente habilitado
  │  └─ Botón "Sincronizar" clickeable para registros pending/error
  │
  └─ Usuario manualmente sincroniza registros pendientes (o batch automático en futuro)

Ningún registro se pierde en offline
  └─ Todos permanecen en SQLite hasta ser sincronizados exitosamente
```

## 8. Separación de dependencias (inyección)

Se usa **DependencyContainer** para composición:

```typescript
// En production
const container = new DependencyContainer({
  clock: new RealClock(),
  idGenerator: new RealIDGenerator(),
  locationProvider: new RealLocationProvider(),
  persistenceProvider: new SQLitePersistenceProvider(),
  connectivityProvider: new RealConnectivityProvider(),
  apiClient: new ApiClient(),
});

// En tests
const testContainer = new DependencyContainer({
  clock: new FakeClock(fixedDate),
  idGenerator: new FakeIDGenerator(deterministicUUIDs),
  locationProvider: new FakeLocationProvider(mockGPS),
  persistenceProvider: new FakePersistenceProvider(),
  connectivityProvider: new FakeConnectivityProvider('online'),
  apiClient: mockApiClient,
});
```

## 9. Esquema SQLite (ACTUALIZADO - Con Backoff Exponencial)

```sql
-- Tabla principal de posiciones capturadas
CREATE TABLE positions (
  -- Identificadores únicos (inmutables)
  event_id TEXT PRIMARY KEY NOT NULL,
  vehicle_id VARCHAR(50) NOT NULL,
  
  -- Coordenadas capturadas
  latitude NUMERIC(9, 6) NOT NULL,
  longitude NUMERIC(9, 6) NOT NULL,
  accuracy NUMERIC(5, 2),
  
  -- Timestamps
  captured_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  
  -- Trazabilidad
  trace_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  
  -- Status de sincronización (EXTENDIDO)
  -- Ciclo de vida: pending → sent (o error o validation_error)
  status TEXT NOT NULL DEFAULT 'pending' 
    CHECK(status IN ('pending', 'sent', 'error', 'validation_error')),
  
  -- Detalles de error si status='error' o 'validation_error'
  error_reason TEXT,
  
  -- Exponential backoff (NUEVO)
  -- Número de intentos fallidos (incrementa cada vez que falla)
  sync_attempt_count INTEGER DEFAULT 0,
  
  -- Timestamp del último intento de sincronización
  last_sync_attempt TIMESTAMPTZ,
  
  -- Timestamp del próximo intento permitido (exponential backoff)
  -- Calculado como: now + min(2s * 2^(attempts-1), 300s)
  -- Si es NULL o <= now, está listo para sincronizar
  -- Si es validation_error, siempre NULL (no reintentar)
  next_retry_at TIMESTAMPTZ,
  
  -- Flag para no reintentar errores de validación
  is_validation_error BOOLEAN DEFAULT FALSE
);

-- Índices para queries eficientes
CREATE INDEX idx_positions_vehicle_id ON positions(vehicle_id);
CREATE INDEX idx_positions_captured_at ON positions(captured_at DESC);
CREATE INDEX idx_positions_status ON positions(status);
CREATE UNIQUE INDEX idx_positions_idempotency ON positions(idempotency_key);
CREATE INDEX idx_positions_trace_id ON positions(trace_id);

-- Índice compuesto para queries comunes: "obtener pending del vehículo X"
CREATE INDEX idx_positions_vehicle_status ON positions(vehicle_id, status);

-- NUEVO: Índice para query de reintentos: "obtener registros con error listos para reintentar"
-- next_retry_at <= now AND status='error'
CREATE INDEX idx_positions_next_retry ON positions(next_retry_at) 
  WHERE status = 'error' AND next_retry_at IS NOT NULL;

-- NUEVO: Índice para conteos rápidos por status
CREATE INDEX idx_positions_status_created ON positions(status, created_at DESC);
```

**Cambios principales:**
- `sync_attempt_count`: Contador de intentos (0 al inicio)
- `last_sync_attempt`: Timestamp del último intento
- `next_retry_at`: Timestamp calculado con backoff exponencial (NULL para validation_error)
- `is_validation_error`: Flag para identificar errores que no se deben reintentar
- Índice `idx_positions_next_retry`: Optimiza queries para encontrar registros listos para reintentar

**Ciclo de vida de un registro con backoff:**

```
1. Se captura GPS
   ↓
   INSERT INTO positions (..., status='pending', sync_attempt_count=0, next_retry_at=NULL, ...)

2. Usuario toca "Sincronizar"
   ↓
   POST /api/positions
   
   ✓ SI 202 Accepted:
   UPDATE positions 
   SET status='sent', 
       updated_at=NOW(), 
       sync_attempt_count=0, 
       next_retry_at=NULL
   WHERE event_id=...
   
   ✗ SI error transitorios (5xx, timeout):
   sync_attempt_count_new = 1
   next_retry_at = NOW() + min(2s * 2^(1-1), 300s) = NOW() + 2s
   UPDATE positions 
   SET status='error', 
       error_reason='...',
       updated_at=NOW(),
       sync_attempt_count=1,
       last_sync_attempt=NOW(),
       next_retry_at=NOW() + 2s
   WHERE event_id=...
   
   ✗ SI error de validación (4xx):
   UPDATE positions 
   SET status='validation_error', 
       error_reason='...',
       updated_at=NOW(),
       sync_attempt_count=1,
       last_sync_attempt=NOW(),
       next_retry_at=NULL,
       is_validation_error=TRUE
   WHERE event_id=...

3. Background task (useAutoSync) revisa cada 10 segundos
   ↓
   SELECT * FROM positions 
   WHERE status='error' 
   AND (next_retry_at IS NULL OR next_retry_at <= NOW())
   LIMIT 3
   
   Para cada uno:
   ├─ Si falla nuevamente:
   │  sync_attempt_count_new = 2
   │  next_retry_at = NOW() + min(2s * 2^(2-1), 300s) = NOW() + 4s
   │  UPDATE ... SET sync_attempt_count=2, next_retry_at=NOW() + 4s
   │
   └─ Si falla 3 veces:
      next_retry_at = NOW() + min(2s * 2^(3-1), 300s) = NOW() + 8s
      (Sigue reintentando indefinidamente con backoff)

4. Tiempo máximo entre reintentos: 300s = 5 minutos
   ├─ Intento 1 fallido: esperar 2s
   ├─ Intento 2 fallido: esperar 4s
   ├─ Intento 3 fallido: esperar 8s
   ├─ Intento 4 fallido: esperar 16s
   ├─ Intento 5 fallido: esperar 32s
   ├─ Intento 6 fallido: esperar 64s
   ├─ Intento 7 fallido: esperar 128s
   ├─ Intento 8 fallido: esperar 256s
   ├─ Intento 9+ fallido: esperar 300s (techo máximo)
```

**Queries típicas con backoff:**

```sql
-- Obtener registros listos para reintentar AHORA
SELECT * FROM positions 
WHERE status='error' 
AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
ORDER BY next_retry_at ASC NULLS FIRST
LIMIT 3;

-- Contar registros por status para UI
SELECT 
  status, 
  COUNT(*) as count 
FROM positions 
GROUP BY status;

-- Obtener próximo reintento en X segundos
SELECT 
  event_id, 
  status, 
  (next_retry_at - CURRENT_TIMESTAMP) as seconds_until_retry
FROM positions 
WHERE status='error' 
ORDER BY next_retry_at ASC
LIMIT 1;

-- Estadísticas de reintentos
SELECT 
  COUNT(*) as total_errors,
  AVG(sync_attempt_count) as avg_attempts,
  MAX(sync_attempt_count) as max_attempts
FROM positions 
WHERE status='error';
```

## 11. Hooks de sincronización con backoff (NUEVO)

### useAutoSync - Background Retry Task

```typescript
/**
 * Hook que ejecuta reintentos automáticos en background.
 * 
 * Cada 10 segundos revisa si hay posiciones listas para reintentar:
 * - status='pending' (nunca se intentó)
 * - status='error' Y nextRetryAt <= now (es hora de reintentar)
 * 
 * Sincroniza hasta 3 registros en paralelo para no sobrecargar.
 * 
 * @param syncPendingUseCase Caso de uso inyectado
 * @param isEnabled true para activar, false para pausar
 * @returns {Object} isSyncing, lastError, totalSynced
 */
export const useAutoSync = (
  syncPendingUseCase: SyncPendingPositionsUseCase,
  isEnabled: boolean = true
) => {
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [lastError, setLastError] = React.useState<string | null>(null);
  const [totalSynced, setTotalSynced] = React.useState(0);

  React.useEffect(() => {
    if (!isEnabled) return;

    // Ejecutar cada 10 segundos
    const intervalId = setInterval(async () => {
      try {
        setIsSyncing(true);
        setLastError(null);

        const result = await syncPendingUseCase.execute({ maxConcurrent: 3 });

        if (result.synced > 0) {
          setTotalSynced(prev => prev + result.synced);
          console.log(`Auto-sync: ${result.synced} synced, ${result.failed} failed`);
        }
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error';
        setLastError(errorMessage);
        console.error('Auto-sync failed:', error);
      } finally {
        setIsSyncing(false);
      }
    }, 10000); // 10 segundos

    return () => clearInterval(intervalId);
  }, [syncPendingUseCase, isEnabled]);

  return { isSyncing, lastError, totalSynced };
};
```

**Comportamiento:**
- Se ejecuta cada 10 segundos si `isEnabled=true`
- Obtiene posiciones listas para sincronizar (pending + error con nextRetryAt <= now)
- Sincroniza máximo 3 en paralelo para no sobrecargar
- Si una falla, calcula nuevo nextRetryAt con backoff
- Si una tiene success, marca como 'sent'
- Se puede pausar/reanudar según estado de conectividad

### useSyncOnReconnect - Manual Sync Trigger

```typescript
/**
 * Hook que dispara sincronización manual cuando el usuario reconecta.
 * 
 * Flujo:
 * 1. Escucha cambios de conectividad (onStatusChange)
 * 2. Cuando pasa a 'online', intenta sincronizar hasta 3 registros
 * 3. UI muestra indicador de que está sincronizando
 * 4. Una vez terminado, UI actualiza conteos
 * 
 * @param connectivityProvider Proveedor de conectividad inyectado
 * @param syncPendingUseCase Caso de uso de sincronización
 * @returns {Object} isConnected, isSyncing, lastError
 */
export const useSyncOnReconnect = (
  connectivityProvider: ConnectivityProvider,
  syncPendingUseCase: SyncPendingPositionsUseCase
) => {
  const [isConnected, setIsConnected] = React.useState(false);
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [lastError, setLastError] = React.useState<string | null>(null);

  React.useEffect(() => {
    // Obtener estado inicial
    connectivityProvider.getStatus().then(status => {
      setIsConnected(status === 'online');
    });

    // Escuchar cambios de conectividad
    const unsubscribe = connectivityProvider.onStatusChange(async (newStatus) => {
      setIsConnected(newStatus === 'online');
      setLastError(null);

      // Si reconectó y hay registros pendientes, intentar sincronizar
      if (newStatus === 'online') {
        try {
          setIsSyncing(true);
          
          // Intentar sincronizar hasta 3 registros pendientes
          const result = await syncPendingUseCase.execute({ maxConcurrent: 3 });
          
          if (result.synced > 0) {
            console.log(`Sync on reconnect: ${result.synced} synced, ${result.failed} failed`);
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error';
          setLastError(errorMessage);
          console.error('Sync on reconnect failed:', error);
        } finally {
          setIsSyncing(false);
        }
      }
    });

    return unsubscribe;
  }, [connectivityProvider, syncPendingUseCase]);

  return { isConnected, isSyncing, lastError };
};
```

**Comportamiento:**
- Escucha cambios de conectividad
- Cuando pasa a 'online', intenta sincronizar inmediatamente
- Sincroniza hasta 3 registros en paralelo
- No interfiere con sincronización manual del usuario
- Útil para app que pasó de offline a online

### useGlobalSync - Manual Sync Botón

```typescript
/**
 * Hook que expone botón "Sincronizar" para sincronización manual del usuario.
 * 
 * El usuario toca el botón para:
 * - Sincronizar todos los registros pendientes (status='pending')
 * - Reintentar registros con error (status='error' con nextRetryAt <= now)
 * - Especialmente útil después de reconectar manualmente o para forzar sincronización
 * 
 * @param syncPendingUseCase Caso de uso inyectado
 * @returns {Object} syncAll, isSyncing, result, error
 */
export const useGlobalSync = (
  syncPendingUseCase: SyncPendingPositionsUseCase
) => {
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [result, setResult] = React.useState<SyncPendingOutput | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const syncAll = React.useCallback(async () => {
    try {
      setIsSyncing(true);
      setError(null);
      setResult(null);

      const syncResult = await syncPendingUseCase.execute({ maxConcurrent: 3 });
      
      setResult(syncResult);
      console.log(
        `Global sync complete: ${syncResult.synced} synced, ${syncResult.failed} failed`
      );
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Error desconocido';
      setError(errorMessage);
      console.error('Global sync failed:', err);
    } finally {
      setIsSyncing(false);
    }
  }, [syncPendingUseCase]);

  return { syncAll, isSyncing, result, error };
};
```

**Comportamiento:**
- Hook simple que expone función `syncAll()`
- Usuario toca botón "Sincronizar" en UI
- Sincroniza todos los registros listas (pending + error con nextRetryAt <= now)
- UI muestra indicador de carga mientras sincroniza
- Al terminar, UI actualiza automáticamente conteos (vía listeners de persistencia)
- Retorna conteo de sincronizados y fallidos

### useSyncPositionManual - Para PositionItem

```typescript
/**
 * Hook que sincroniza un registro individual.
 * 
 * Utilizado por PositionItem para botón "Sincronizar" individual.
 * Maneja conectividad, validación de estado y errores.
 * 
 * @param syncPositionUseCase Caso de uso inyectado
 * @returns {Object} syncOne, isSyncing, error
 */
export const useSyncPositionManual = (
  syncPositionUseCase: SyncPositionUseCase
) => {
  const [isSyncing, setIsSyncing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const syncOne = React.useCallback(
    async (position: Position): Promise<void> => {
      try {
        setIsSyncing(true);
        setError(null);

        const result = await syncPositionUseCase.execute({ position });

        if (!result.success) {
          // Obtener razón del error desde el objeto position actualizado
          setError(`Sincronización fallida: ${position.errorReason || 'error desconocido'}`);
          throw new Error(`Sync failed: ${position.errorReason}`);
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Error desconocido';
        setError(errorMessage);
        throw err;
      } finally {
        setIsSyncing(false);
      }
    },
    [syncPositionUseCase]
  );

  return { syncOne, isSyncing, error };
};
```

## 12. Componentes UI mejorados con backoff

### PositionItem (ACTUALIZADO - Con Backoff Visual)

```typescript
interface PositionItemProps {
  position: Position;
  onSync: (position: Position) => Promise<void>;
  isLoading?: boolean;
  isConnected: boolean; // Desde ConnectivityIndicator
}

const PositionItem: React.FC<PositionItemProps> = ({
  position,
  onSync,
  isLoading = false,
  isConnected,
}) => {
  const [syncError, setSyncError] = React.useState<string | null>(null);
  const [isSyncing, setIsSyncing] = React.useState(false);

  const handleSync = async () => {
    if (!isConnected) {
      setSyncError('Sin conectividad disponible');
      return;
    }

    if (position.status === 'sent') {
      setSyncError('Ya fue enviado correctamente');
      return;
    }

    if (position.status === 'validation_error') {
      setSyncError('No se puede reintentar: datos inválidos');
      return;
    }

    try {
      setIsSyncing(true);
      setSyncError(null);
      await onSync(position);
    } catch (error) {
      setSyncError(error.message || 'Error al sincronizar');
    } finally {
      setIsSyncing(false);
    }
  };

  const statusIcon = position.getStatusIcon();
  const statusLabel = position.getStatusLabel();
  const retryText = position.getNextRetryText(); // "Reintentar en 32s" o null

  // Botón deshabilitado si:
  // - Sin conectividad
  // - status='sent' (ya enviado)
  // - status='validation_error' (error permanente)
  // - status='error' pero nextRetryAt > now (esperar más)
  const isSyncDisabled =
    !isConnected ||
    position.status === 'sent' ||
    position.status === 'validation_error' ||
    (position.status === 'error' && !position.canBeSynced()) ||
    isLoading ||
    isSyncing;

  // Color de borde según status
  const getBorderColor = (): string => {
    switch (position.status) {
      case 'pending':
        return '#FFC107'; // Amarillo
      case 'sent':
        return '#4CAF50'; // Verde
      case 'error':
        return '#FF9800'; // Naranja
      case 'validation_error':
        return '#F44336'; // Rojo
      default:
        return '#2196F3'; // Azul
    }
  };

  return (
    <View style={[styles.container, { borderLeftColor: getBorderColor() }]}>
      {/* Encabezado con status visual */}
      <View style={styles.header}>
        <Text style={styles.statusIndicator}>
          {statusIcon} {statusLabel}
        </Text>

        {/* Mostrar contador de intentos si hay error */}
        {position.status === 'error' && position.syncAttemptCount > 0 && (
          <Text style={styles.attemptCount}>
            Intento {position.syncAttemptCount}
          </Text>
        )}

        {/* Mostrar próximo reintento si está calculado */}
        {position.status === 'error' && retryText && (
          <Text style={styles.retryScheduled}>{retryText}</Text>
        )}

        {/* Mostrar razón del error si existe */}
        {(position.status === 'error' || position.status === 'validation_error') &&
          position.errorReason && (
            <Text style={styles.errorReason}>{position.errorReason}</Text>
          )}
      </View>

      {/* Datos de la ubicación */}
      <View style={styles.details}>
        <Text style={styles.vehicleId}>Vehículo: {position.vehicleId}</Text>
        <Text style={styles.coordinates}>
          📍 {position.coordinate.latitude.toFixed(6)}, {position.coordinate.longitude.toFixed(6)}
        </Text>
        {position.coordinate.accuracy && (
          <Text style={styles.accuracy}>
            Precisión: ±{position.coordinate.accuracy.toFixed(1)}m
          </Text>
        )}
        <Text style={styles.timestamp}>
          {position.capturedAt.toLocaleString()}
        </Text>

        {/* Mostrar last_sync_attempt si hay */}
        {position.lastSyncAttempt && (
          <Text style={styles.lastSyncAttempt}>
            Último intento: {position.lastSyncAttempt.toLocaleString()}
          </Text>
        )}
      </View>

      {/* Botón de sincronización individual */}
      <View style={styles.actionContainer}>
        <Button
          title={isSyncing ? 'Sincronizando...' : 'Sincronizar'}
          onPress={handleSync}
          disabled={isSyncDisabled}
        />

        {/* Tooltip para validation_error */}
        {position.status === 'validation_error' && (
          <Text style={styles.tooltip}>
            ℹ️ No se puede reintentar: los datos del registro son inválidos. Revise la razón del error.
          </Text>
        )}

        {/* Tooltip para error esperando nextRetryAt */}
        {position.status === 'error' &&
          position.nextRetryAt &&
          !position.canBeSynced() && (
            <Text style={styles.tooltip}>
              ℹ️ Reintentar automático en {retryText}
            </Text>
          )}
      </View>

      {/* Mostrar errores de sincronización inmediata */}
      {syncError && (
        <Text style={styles.syncError}>{syncError}</Text>
      )}
    </View>
  );
};

const styles = {
  container: {
    padding: 12,
    marginVertical: 8,
    backgroundColor: '#f5f5f5',
    borderRadius: 8,
    borderLeftWidth: 4,
  },
  header: {
    marginBottom: 8,
  },
  statusIndicator: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#333',
  },
  attemptCount: {
    fontSize: 12,
    color: '#FF9800',
    marginTop: 4,
  },
  retryScheduled: {
    fontSize: 12,
    color: '#1976D2',
    marginTop: 2,
    fontStyle: 'italic',
  },
  errorReason: {
    fontSize: 12,
    color: '#d32f2f',
    marginTop: 4,
  },
  details: {
    marginBottom: 12,
  },
  vehicleId: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
    marginBottom: 4,
  },
  coordinates: {
    fontSize: 13,
    color: '#666',
    fontFamily: 'monospace',
    marginBottom: 4,
  },
  accuracy: {
    fontSize: 12,
    color: '#999',
    marginBottom: 4,
  },
  timestamp: {
    fontSize: 12,
    color: '#999',
  },
  lastSyncAttempt: {
    fontSize: 11,
    color: '#999',
    marginTop: 4,
    fontStyle: 'italic',
  },
  actionContainer: {
    marginBottom: 8,
  },
  tooltip: {
    fontSize: 11,
    color: '#1976D2',
    marginTop: 8,
    fontStyle: 'italic',
  },
  syncError: {
    fontSize: 12,
    color: '#d32f2f',
    marginTop: 8,
  },
};

export default PositionItem;
```

**Cambios principales:**
- Muestra contador de intentos: "Intento 3"
- Muestra próximo reintento: "Reintentar en 32s"
- Diferencia visual entre 'error' (naranja, reintentable) y 'validation_error' (rojo, no reintentable)
- Tooltip explicativo para validation_error: "No se puede reintentar: datos inválidos"
- Tooltip para error esperando backoff: "Reintentar automático en 32s"
- Muestra `lastSyncAttempt` para auditoría
- Botón deshabilitado si aún no es hora de reintentar

### PositionListHeader (NUEVO - Con Desglose Completo)

```typescript
interface PositionListHeaderProps {
  countByStatus: Record<PositionStatus, number>;
  onGlobalSync: () => Promise<void>;
  isSyncing: boolean;
  isConnected: boolean;
}

const PositionListHeader: React.FC<PositionListHeaderProps> = ({
  countByStatus,
  onGlobalSync,
  isSyncing,
  isConnected,
}) => {
  const totalPending = countByStatus.pending + countByStatus.error;
  const pendingText =
    totalPending > 0 ? ` (${totalPending})` : '';

  return (
    <View style={styles.container}>
      <View style={styles.titleRow}>
        <Text style={styles.title}>Ubicaciones capturadas</Text>
        {!isConnected && (
          <Text style={styles.offlineIndicator}>🔴 SIN CONEXIÓN</Text>
        )}
      </View>

      {/* Desglose de estados */}
      <View style={styles.statsRow}>
        <View style={styles.stat}>
          <Text style={styles.statLabel}>Pendiente</Text>
          <Text style={styles.statValue}>{countByStatus.pending}</Text>
        </View>
        <View style={styles.statDivider} />
        <View style={styles.stat}>
          <Text style={styles.statLabel}>Enviado</Text>
          <Text style={styles.statValue}>{countByStatus.sent}</Text>
        </View>
        <View style={styles.statDivider} />
        <View style={styles.stat}>
          <Text style={styles.statLabel}>Error</Text>
          <Text style={styles.statValue}>{countByStatus.error}</Text>
        </View>
        <View style={styles.statDivider} />
        <View style={styles.stat}>
          <Text style={styles.statLabel}>Datos inválidos</Text>
          <Text style={styles.statValue}>{countByStatus.validation_error}</Text>
        </View>
      </View>

      {/* Botón de sincronización global */}
      <View style={styles.buttonRow}>
        <Button
          title={isSyncing ? `Sincronizando${pendingText}...` : `Sincronizar todo${pendingText}`}
          onPress={onGlobalSync}
          disabled={!isConnected || isSyncing || totalPending === 0}
        />
      </View>
    </View>
  );
};

const styles = {
  container: {
    backgroundColor: '#fff',
    padding: 12,
    marginBottom: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  titleRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  title: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#333',
  },
  offlineIndicator: {
    fontSize: 12,
    color: '#d32f2f',
    fontWeight: 'bold',
  },
  statsRow: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    marginBottom: 12,
    padding: 8,
    backgroundColor: '#f5f5f5',
    borderRadius: 8,
  },
  stat: {
    alignItems: 'center',
    flex: 1,
  },
  statLabel: {
    fontSize: 11,
    color: '#666',
    marginBottom: 4,
  },
  statValue: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#333',
  },
  statDivider: {
    width: 1,
    backgroundColor: '#ddd',
  },
  buttonRow: {
    marginTop: 8,
  },
};

export default PositionListHeader;
```

## 10. Decisiones de diseño (ACTUALIZADO)

1. **Captura periódica cada 30s**: Balance entre precisión de ubicación y consumo de batería del dispositivo

2. **SQLite local con status**: Almacenamiento offline garantizado con rastreo de sincronización mediante campo `status`

3. **Offline-first architecture**: Los datos se guardan localmente PRIMERO, envío al backend es consideración secundaria

4. **UUIDs e idempotency_key**: Previenen duplicados en reintentos y garantizan que cada evento es único

5. **TraceContext (trace_id + idempotency_key)**: Facilita rastreo extremo a extremo desde cliente hasta backend

6. **Adapters reemplazables (Fake providers)**: Infraestructura completa abstracción permite testing sin hardware real o API real

7. **Cuatro estados en Position**: Ciclo de vida explícito:
   - `pending`: Recién capturado, listo para sincronizar
   - `sent`: Backend respondió 202 Accepted, procesando
   - `error`: Falló al sincronizar, puede reintentarse (transitorios)
   - `validation_error`: Error 4xx de validación, NO reintentar

8. **202 Accepted → status='sent' SIN borrado**: 
   - La respuesta 202 NO significa persistencia final en backend
   - Registro permanece en SQLite para auditoría y consideración futura de confirmación remota
   - Diferencia clara entre "aceptado para procesar" vs "almacenado definitivamente"

9. **Exponential backoff para reintentos**: 
   - Fórmula: `nextRetryAt = now + min(2s * 2^(attempts-1), 300s)`
   - Comienza con 2 segundos, dobla cada reintento hasta 300s máximo
   - Previene sobrecargar servidor y permite recuperación gradual
   - Campo `syncAttemptCount` rastrea número de intentos

10. **Sincronización individual**: Usuario elige qué registro sincronizar, no todos a la vez
    - Mayor control y feedback granular
    - Permite reintentar solo registros con error
    - Mejor experiencia offline-first

11. **Background auto-sync cada 10 segundos**: 
    - Reintentos automáticos en background (sin intervención del usuario)
    - Sincroniza hasta 3 registros en paralelo para no sobrecargar
    - Respeta backoff exponencial: no intenta si nextRetryAt > now
    - Se puede pausar según estado de conectividad

12. **Sync on reconnect**: 
    - Cuando la app detecta reconexión, intenta sincronizar inmediatamente
    - Aprovecha ventana de conectividad sin esperar 10 segundos siguientes
    - Optimiza experiencia offline-first

13. **Global sync botón**: 
    - Usuario puede forzar sincronización de todos los pendientes
    - Útil para "empujar ahora" sin esperar backoff automático
    - Mostrar conteo de registros listos: "Sincronizar todo (5)"

14. **Diferenciación de errores en UI**:
    - `validation_error` (rojo): No reintentar, revisar datos
    - `error` (naranja): Reintentable, con countdown visible
    - Visual claro en PositionItem con color de borde dinámico

15. **Actualización UI reactiva por eventos SIN polling**:
    - `onPositionCaptured`: Nueva posición capturada
    - `onPositionStatusChanged`: Cambió status de un registro
    - `AppState.change`: App vuelve del background
    - Ahorro significativo de renders, CPU y batería (NO cada 5s)

16. **Desglose de registros por status en UI**:
    - Muestra "N Pendiente | M Enviado | P Error | L Datos inválidos"
    - Visibilidad operativa clara del estado de sincronización
    - Badge de "Sincronizar todo (N)" muestra solo pendientes y con error

17. **Índices SQLite optimizados con backoff**:
    - Índice en `status` para queries rápidas por estado
    - Índice compuesto `(vehicle_id, status)` para queries frecuentes
    - Índice especial en `next_retry_at` para encontrar listas para reintentar
    - Garantiza performance incluso con miles de registros

18. **Auditoria completa de reintentos**: 
    - `sync_attempt_count`: Número de intentos
    - `last_sync_attempt`: Cuándo fue el último
    - `next_retry_at`: Cuándo será el próximo
    - `error_reason`: Qué falló
    - Facilita debugging y análisis post-mortem

19. **Inmutabilidad de Position en dominio**:
    - `markAsSent()` y `markAsError()` retornan nuevas instancias
    - Cambios de estado son explícitos y trazables
    - Evita bugs de mutación accidental
    - `canBeSynced()` verifica también nextRetryAt

20. **Separación clara de responsabilidades**:
    - Domain: Reglas de negocio puras (Position, validaciones, backoff)
    - Application: Casos de uso (CapturePosition, SyncPosition, SyncPendingPositions)
    - Infrastructure: Detalles técnicos (LocationProvider, SQLite, API)
    - Presentation: UI reactiva (componentes, hooks, listeners)
    - Cada capa conoce solo lo que necesita conocer

21. **Maximal concurrency = 3**: 
    - No sincronizar más de 3 simultáneamente para no sobrecargar
    - Balance entre velocidad y carga en red/servidor
    - Configurable según necesidades

22. **No reintentar indefinidamente, pero sí indefinidamente con backoff máximo**:
    - Registros con error siguen reintentando cada 300s = 5 minutos
    - El usuario puede ver estado: "Reintentando en 5 min" o "Listo para reintentar"
    - Considerar futuro: máximo de intentos absoluto (ej: después de 7 días sin éxito)

