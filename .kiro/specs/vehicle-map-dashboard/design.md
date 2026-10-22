# Diseño técnico: Dashboard de Mapa de Vehículos

## 1. Overview

El Dashboard de Mapa de Vehículos es un componente frontend que visualiza en tiempo real la ubicación de una flota de vehículos. Implementa **Clean Architecture** separando claramente la lógica de negocio (dominio), la coordinación de operaciones (aplicación), la integración con sistemas externos (infraestructura) y la interfaz visual (presentación).

**Arquitectura propuesta:**
- **Dominio**: Entidades de negocio (Vehicle, Position, VehicleId) y objetos de valor con invariantes
- **Aplicación**: Casos de uso (LoadVehicles, SubscribeToPositionUpdates, HandlePositionUpdate) y puertos/interfaces
- **Infraestructura**: Adaptadores para SSE, datos mock, mapas, API HTTP
- **Presentación**: Componentes React, hooks y manejo de UI

El dashboard utiliza **Server-Sent Events (SSE)** para recibir actualizaciones de posición en tiempo real, aplicando reconexión automática con backoff exponencial. Puede operar con datos mock para demostración o conectarse a la API real del backend.

**Stack tecnológico seleccionado:**
- **Librería de mapa**: `react-leaflet` con `leaflet` (ligera, open-source, sin dependencia de API keys)
- **Gestión de SSE**: Hook personalizado `useSSEConnection` que encapsula EventSource
- **Gestión de estado**: React Context + hooks (sin Redux para mantener MVP simple)
- **Validación**: Funciones de dominio puro sin dependencias externas
- **Testing**: Vitest para unit tests, RTL para integración, MSW para interceptación de red

---

## 2. Arquitectura de capas

### 2.1 Estructura de directorios

```
frontend/src/
├── domain/
│   ├── entities/
│   │   ├── vehicle.ts           # Entidad Vehicle
│   │   ├── position.ts          # Objeto de valor Position
│   │   └── vehicle-id.ts        # Objeto de valor VehicleId
│   ├── errors/
│   │   └── vehicle-errors.ts    # Errores del dominio
│   └── validators/
│       └── position-validator.ts # Validación de coordenadas
├── application/
│   ├── ports/
│   │   ├── vehicle-data-provider.ts    # Puerto: cargar vehículos
│   │   ├── vehicle-realtime-port.ts    # Puerto: SSE y actualizaciones
│   │   └── vehicle-storage.ts          # Puerto: almacenamiento local
│   ├── usecases/
│   │   ├── load-vehicles.ts            # Caso de uso: cargar vehículos
│   │   ├── subscribe-to-updates.ts     # Caso de uso: suscribirse a SSE
│   │   └── handle-position-update.ts   # Caso de uso: procesar actualización
│   └── state-manager/
│       └── vehicle-state-manager.ts    # Coordinador de aplicación
├── infrastructure/
│   ├── http/
│   │   └── api-client.ts               # Cliente HTTP para backend
│   ├── sse/
│   │   ├── sse-connection.ts           # Adaptador SSE real
│   │   └── sse-simulator.ts            # Simulador SSE para mock
│   ├── maps/
│   │   └── leaflet-map-adapter.ts      # Adaptador para Leaflet
│   ├── persistence/
│   │   └── local-storage-adapter.ts    # Almacenamiento local del navegador
│   └── data-providers/
│       ├── api-data-provider.ts        # Proveedor de datos real
│       ├── mock-data-provider.ts       # Proveedor de datos mock
│       └── data-provider-factory.ts    # Factory para seleccionar proveedor
├── presentation/
│   ├── hooks/
│   │   ├── useVehicles.ts              # Hook para cargar vehículos
│   │   ├── useSSEConnection.ts         # Hook para gestionar SSE
│   │   ├── useVehicleUpdates.ts        # Hook para suscripciones granulares
│   │   └── useReconnectionManager.ts   # Hook para lógica de reconexión
│   ├── components/
│   │   ├── MapContainer.tsx            # Contenedor principal del mapa
│   │   ├── VehicleMarker.tsx           # Marcador de vehículo
│   │   ├── VehicleTooltip.tsx          # Tooltip con información
│   │   ├── ConnectionStatus.tsx        # Indicador de conexión
│   │   └── DemoModeBanner.tsx          # Banner de modo demo
│   ├── pages/
│   │   └── VehicleDashboard.tsx        # Página principal del dashboard
│   └── styles/
│       └── dashboard.css               # Estilos del dashboard
└── config/
    └── environment.ts                  # Configuración por entorno
```

### 2.2 Reglas de dependencia

1. **Dominio es independiente**: No importa de aplicación, infraestructura ni presentación
2. **Aplicación depende de dominio y puertos**: Nunca de infraestructura concreta
3. **Infraestructura depende de dominio y aplicación**: Implementa los puertos
4. **Presentación depende de aplicación y dominio**: Nunca directamente de infraestructura
5. **Flujo de datos**: Presentación → Aplicación → Dominio/Infraestructura

---

## 3. Arquitectura de componentes React

### 3.1 Estructura de componentes

```
VehicleDashboard (page)
├── ConnectionStatus (indicador de conexión SSE)
├── DemoModeBanner (banner si está en modo demo)
└── MapContainer (componente raíz del mapa)
    ├── useVehicles() hook
    ├── useSSEConnection() hook
    └── VehicleRenderer (renderiza marcadores)
        ├── VehicleMarker (por cada vehículo)
        │   └── VehicleTooltip (al pasar mouse)
        └── [Controles del mapa de Leaflet]
```

### 3.2 Componentes principales

**VehicleDashboard** (Presentación)
```typescript
// Punto de entrada del dashboard
// Responsabilidades:
// - Orquestar hooks de carga y SSE
// - Manejar estados de carga/error
// - Pasar datos a MapContainer
// - Mostrar indicadores de conexión

export const VehicleDashboard: React.FC = () => {
  const { vehicles, loading, error } = useVehicles();
  const { status: connectionStatus } = useSSEConnection();
  
  return (
    <div className="dashboard">
      <ConnectionStatus status={connectionStatus} />
      {import.meta.env.VITE_DATA_MODE === 'mock' && <DemoModeBanner />}
      {error && <ErrorAlert error={error} />}
      {loading ? <LoadingSpinner /> : <MapContainer vehicles={vehicles} />}
    </div>
  );
};
```

**MapContainer** (Presentación)
```typescript
// Contenedor del mapa
// Responsabilidades:
// - Renderizar mapa Leaflet
// - Renderizar marcadores de vehículos
// - Manejar interacción del usuario
// - Coordinar con hooks de actualización

interface MapContainerProps {
  vehicles: Vehicle[];
}

export const MapContainer: React.FC<MapContainerProps> = ({ vehicles }) => {
  const mapRef = useRef<L.Map | null>(null);
  
  return (
    <MapContainer center={[40, -3]} zoom={13}>
      <TileLayer url="..." />
      <VehicleRenderer vehicles={vehicles} />
    </MapContainer>
  );
};
```

**VehicleMarker** (Presentación)
```typescript
// Marcador individual de vehículo
// Responsabilidades:
// - Renderizar marcador en posición correcta
// - Mostrar etiqueta con ID del vehículo
// - Mostrar tooltip al hover
// - Aplicar estilo visual según antigüedad

interface VehicleMarkerProps {
  vehicle: Vehicle;
}

export const VehicleMarker: React.FC<VehicleMarkerProps> = ({ vehicle }) => {
  const isStale = isPositionStale(vehicle.position.timestamp);
  
  return (
    <Marker position={[vehicle.position.latitude, vehicle.position.longitude]}>
      <Popup className={isStale ? 'stale-marker' : ''}>
        <VehicleTooltip vehicle={vehicle} />
      </Popup>
    </Marker>
  );
};
```

---

## 4. Componentes y Interfaces

### 4.1 Dominio

**Vehicle** (Entidad)
```typescript
// Ubicación: domain/entities/vehicle.ts
// Responsabilidades:
// - Representar un vehículo de la flota
// - Validar invariantes (ID no nulo, placa válida, posición válida)
// - Métodos para actualizar posición

export class Vehicle {
  constructor(
    readonly vehicleId: VehicleId,
    readonly licensePlate: string,
    readonly position: Position,
  ) {
    this.validate();
  }

  private validate(): void {
    if (!this.vehicleId || this.licensePlate.trim() === '') {
      throw new InvalidVehicleError('VehicleId and licensePlate are required');
    }
  }

  updatePosition(newPosition: Position): Vehicle {
    return new Vehicle(this.vehicleId, this.licensePlate, newPosition);
  }

  isStalePosition(maxAgeMs: number = 60 * 60 * 1000): boolean {
    return Date.now() - this.position.timestamp.getTime() > maxAgeMs;
  }

  // Métodos para comparación e igualdad
  equals(other: Vehicle): boolean {
    return this.vehicleId.equals(other.vehicleId);
  }
}
```

**Position** (Objeto de valor)
```typescript
// Ubicación: domain/entities/position.ts
// Responsabilidades:
// - Representar ubicación geográfica con precisión
// - Validar coordenadas dentro de rango válido
// - Proporcionar métodos de comparación

export class Position {
  readonly latitude: number;
  readonly longitude: number;
  readonly timestamp: Date;

  constructor(latitude: number, longitude: number, timestamp?: Date) {
    this.validate(latitude, longitude);
    this.latitude = latitude;
    this.longitude = longitude;
    this.timestamp = timestamp ?? new Date();
  }

  private validate(lat: number, lon: number): void {
    if (lat < -90 || lat > 90) {
      throw new InvalidCoordinateError(`Latitude must be between -90 and 90, got ${lat}`);
    }
    if (lon < -180 || lon > 180) {
      throw new InvalidCoordinateError(`Longitude must be between -180 and 180, got ${lon}`);
    }
  }

  // Proporciona precisión de 4 decimales (~11 metros)
  toString(): string {
    return `${this.latitude.toFixed(4)}, ${this.longitude.toFixed(4)}`;
  }

  equals(other: Position): boolean {
    return (
      Math.abs(this.latitude - other.latitude) < 1e-6 &&
      Math.abs(this.longitude - other.longitude) < 1e-6
    );
  }
}
```

**VehicleId** (Objeto de valor)
```typescript
// Ubicación: domain/entities/vehicle-id.ts
// Responsabilidades:
// - Representar identificador único de vehículo
// - Validar formato (non-empty string)
// - Proporcionar métodos de comparación

export class VehicleId {
  readonly value: string;

  constructor(value: string) {
    if (!value || value.trim() === '') {
      throw new InvalidVehicleIdError('VehicleId must not be empty');
    }
    this.value = value;
  }

  equals(other: VehicleId): boolean {
    return this.value === other.value;
  }

  toString(): string {
    return this.value;
  }
}
```

### 4.2 Puertos (Interfaces de aplicación)

**VehicleDataProvider** (Puerto de entrada)
```typescript
// Ubicación: application/ports/vehicle-data-provider.ts
// Responsabilidades:
// - Definir contrato para carga de vehículos
// - No asumir implementación (API real, mock, etc.)

export interface VehicleDataProvider {
  // Cargar todos los vehículos disponibles
  loadVehicles(): Promise<Vehicle[]>;
}
```

**VehicleRealtimePort** (Puerto de entrada)
```typescript
// Ubicación: application/ports/vehicle-realtime-port.ts
// Responsabilidades:
// - Definir contrato para recepción de actualizaciones en tiempo real
// - Manejar ciclo de vida de conexión

export interface VehicleRealtimePort {
  // Establecer conexión con servidor de actualizaciones
  connect(): Promise<void>;

  // Registrar callback para cada actualización de posición
  onPositionUpdate(callback: (update: PositionUpdateEvent) => void): Unsubscribe;

  // Registrar callback para cambios de estado de conexión
  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe;

  // Desconectar y liberar recursos
  disconnect(): Promise<void>;

  // Obtener estado actual de conexión
  getStatus(): ConnectionStatus;
}

export interface PositionUpdateEvent {
  vehicleId: string;
  latitude: number;
  longitude: number;
  timestamp: string; // ISO-8601
}

export type ConnectionStatus = 'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR';

export type Unsubscribe = () => void;
```

**VehicleStorage** (Puerto de salida)
```typescript
// Ubicación: application/ports/vehicle-storage.ts
// Responsabilidades:
// - Definir contrato para almacenamiento local persistente

export interface VehicleStorage {
  // Guardar vehículos en almacenamiento local
  saveVehicles(vehicles: Vehicle[]): Promise<void>;

  // Cargar vehículos de almacenamiento local
  loadVehicles(): Promise<Vehicle[]>;

  // Limpiar almacenamiento local
  clear(): Promise<void>;
}
```

### 4.3 Casos de uso (Application Layer)

**LoadVehicles** (Use Case)
```typescript
// Ubicación: application/usecases/load-vehicles.ts
// Responsabilidades:
// - Orquestar carga inicial de vehículos
// - Validar datos retornados
// - Actualizar estado de aplicación

export class LoadVehiclesUseCase {
  constructor(
    private vehicleDataProvider: VehicleDataProvider,
    private vehicleStorage: VehicleStorage,
  ) {}

  async execute(): Promise<Vehicle[]> {
    try {
      const vehicles = await this.vehicleDataProvider.loadVehicles();
      
      // Validar que todos los vehículos son válidos
      vehicles.forEach(v => {
        if (!v.vehicleId || !v.position) {
          throw new InvalidVehicleError('Vehicle missing required fields');
        }
      });

      // Guardar en almacenamiento local para offline
      await this.vehicleStorage.saveVehicles(vehicles);
      
      return vehicles;
    } catch (error) {
      // Si la API falla, intentar cargar desde almacenamiento local
      return this.vehicleStorage.loadVehicles();
    }
  }
}
```

**SubscribeToPositionUpdates** (Use Case)
```typescript
// Ubicación: application/usecases/subscribe-to-updates.ts
// Responsabilidades:
// - Orquestar suscripción a actualizaciones en tiempo real
// - Manejar reconexión automática
// - Procesar eventos entrantes

export class SubscribeToPositionUpdatesUseCase {
  constructor(
    private realtimePort: VehicleRealtimePort,
    private handlePositionUpdate: HandlePositionUpdateUseCase,
  ) {}

  async execute(
    onUpdate: (vehicles: Vehicle[]) => void,
    onStatusChange: (status: ConnectionStatus) => void,
  ): Promise<Unsubscribe> {
    await this.realtimePort.connect();

    // Registrar callbacks para actualizaciones
    const unsubscribeUpdates = this.realtimePort.onPositionUpdate((event) => {
      this.handlePositionUpdate.execute(event, onUpdate);
    });

    const unsubscribeStatus = this.realtimePort.onConnectionStatusChange((status) => {
      onStatusChange(status);
    });

    // Retornar función de desuscripción que libera ambos listeners
    return () => {
      unsubscribeUpdates();
      unsubscribeStatus();
      this.realtimePort.disconnect();
    };
  }
}
```

**HandlePositionUpdate** (Use Case)
```typescript
// Ubicación: application/usecases/handle-position-update.ts
// Responsabilidades:
// - Procesar actualización individual de posición
// - Validar datos de evento
// - Actualizar estado de vehículo

export class HandlePositionUpdateUseCase {
  constructor(
    private vehicleStorage: VehicleStorage,
    private positionValidator: PositionValidator,
  ) {}

  async execute(
    event: PositionUpdateEvent,
    onUpdate: (vehicles: Vehicle[]) => void,
  ): Promise<void> {
    try {
      // Validar coordenadas
      this.positionValidator.validate(event.latitude, event.longitude);

      // Construir nueva posición
      const newPosition = new Position(
        event.latitude,
        event.longitude,
        new Date(event.timestamp),
      );

      // Cargar vehículos actuales
      const vehicles = await this.vehicleStorage.loadVehicles();

      // Encontrar y actualizar vehículo
      const updatedVehicles = vehicles.map((v) => {
        if (v.vehicleId.value === event.vehicleId) {
          return v.updatePosition(newPosition);
        }
        return v;
      });

      // Guardar estado actualizado
      await this.vehicleStorage.saveVehicles(updatedVehicles);

      // Notificar presentación de actualización
      onUpdate(updatedVehicles);
    } catch (error) {
      console.error('Failed to handle position update:', error);
      // No lanzar error; los eventos inválidos se ignoran
    }
  }
}
```

---

## 5. Infraestructura

### 5.1 Adaptador SSE

**SSEConnection** (Implementación real)
```typescript
// Ubicación: infrastructure/sse/sse-connection.ts
// Responsabilidades:
// - Conectar a servidor SSE real
// - Parsear eventos SSE
// - Manejar reconexión con backoff exponencial

export class SSEConnection implements VehicleRealtimePort {
  private eventSource: EventSource | null = null;
  private status: ConnectionStatus = 'DISCONNECTED';
  private updateCallbacks: ((event: PositionUpdateEvent) => void)[] = [];
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private baseBackoffMs = 1000;
  private maxBackoffMs = 30000;

  constructor(
    private sseUrl: string,
    private clock: Clock = new SystemClock(),
  ) {}

  async connect(): Promise<void> {
    if (this.status === 'CONNECTED' || this.status === 'CONNECTING') {
      return;
    }

    this.setStatus('CONNECTING');

    try {
      this.eventSource = new EventSource(this.sseUrl);

      this.eventSource.addEventListener('position_update', (event) => {
        try {
          const data = JSON.parse(event.data);
          this.updateCallbacks.forEach(cb => cb(data));
          this.reconnectAttempts = 0; // Reset backoff on success
        } catch (e) {
          console.error('Failed to parse SSE event:', e);
        }
      });

      this.eventSource.addEventListener('error', () => {
        this.handleConnectionError();
      });

      this.setStatus('CONNECTED');
    } catch (error) {
      this.handleConnectionError();
    }
  }

  private handleConnectionError(): void {
    this.setStatus('ERROR');

    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      const backoffMs = Math.min(
        this.baseBackoffMs * Math.pow(2, this.reconnectAttempts),
        this.maxBackoffMs,
      );

      this.reconnectAttempts++;

      setTimeout(() => {
        this.connect();
      }, backoffMs);
    } else {
      this.setStatus('DISCONNECTED');
    }
  }

  onPositionUpdate(callback: (event: PositionUpdateEvent) => void): Unsubscribe {
    this.updateCallbacks.push(callback);
    return () => {
      this.updateCallbacks = this.updateCallbacks.filter(cb => cb !== callback);
    };
  }

  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe {
    this.statusCallbacks.push(callback);
    return () => {
      this.statusCallbacks = this.statusCallbacks.filter(cb => cb !== callback);
    };
  }

  async disconnect(): Promise<void> {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.setStatus('DISCONNECTED');
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    this.statusCallbacks.forEach(cb => cb(status));
  }
}
```

**SSESimulator** (Para desarrollo/testing)
```typescript
// Ubicación: infrastructure/sse/sse-simulator.ts
// Responsabilidades:
// - Simular servidor SSE con datos mock
// - Generar actualizaciones de posición realistas
// - Simular desconexiones/errores para testing

export class SSESimulator implements VehicleRealtimePort {
  private status: ConnectionStatus = 'DISCONNECTED';
  private updateCallbacks: ((event: PositionUpdateEvent) => void)[] = [];
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private simulationInterval: NodeJS.Timeout | null = null;
  private vehiclePositions: Map<string, Position> = new Map();

  constructor(
    private vehicles: Vehicle[],
    private updateIntervalMs: number = 2000,
  ) {
    // Inicializar posiciones de vehículos
    vehicles.forEach(v => {
      this.vehiclePositions.set(v.vehicleId.value, v.position);
    });
  }

  async connect(): Promise<void> {
    if (this.status !== 'DISCONNECTED') return;

    this.setStatus('CONNECTING');
    
    // Simular latencia de conexión
    await new Promise(resolve => setTimeout(resolve, 500));

    this.setStatus('CONNECTED');

    // Comenzar a simular actualizaciones
    this.simulationInterval = setInterval(() => {
      this.vehicles.forEach(vehicle => {
        const currentPos = this.vehiclePositions.get(vehicle.vehicleId.value)!;
        
        // Generar pequeño cambio de posición (movimiento simulado)
        const newLat = currentPos.latitude + (Math.random() - 0.5) * 0.001;
        const newLon = currentPos.longitude + (Math.random() - 0.5) * 0.001;

        const event: PositionUpdateEvent = {
          vehicleId: vehicle.vehicleId.value,
          latitude: newLat,
          longitude: newLon,
          timestamp: new Date().toISOString(),
        };

        this.vehiclePositions.set(vehicle.vehicleId.value, new Position(newLat, newLon));
        this.updateCallbacks.forEach(cb => cb(event));
      });
    }, this.updateIntervalMs);
  }

  onPositionUpdate(callback: (event: PositionUpdateEvent) => void): Unsubscribe {
    this.updateCallbacks.push(callback);
    return () => {
      this.updateCallbacks = this.updateCallbacks.filter(cb => cb !== callback);
    };
  }

  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe {
    this.statusCallbacks.push(callback);
    return () => {
      this.statusCallbacks = this.statusCallbacks.filter(cb => cb !== callback);
    };
  }

  async disconnect(): Promise<void> {
    if (this.simulationInterval) {
      clearInterval(this.simulationInterval);
      this.simulationInterval = null;
    }
    this.setStatus('DISCONNECTED');
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    this.statusCallbacks.forEach(cb => cb(status));
  }
}
```

### 5.2 Proveedores de datos

**MockDataProvider**
```typescript
// Ubicación: infrastructure/data-providers/mock-data-provider.ts
// Responsabilidades:
// - Generar 20 vehículos deterministas con coordenadas realistas
// - Mantener los mismos datos entre llamadas (reproducibilidad)

export class MockDataProvider implements VehicleDataProvider {
  async loadVehicles(): Promise<Vehicle[]> {
    const vehicles: Vehicle[] = [];
    
    // Coordenadas realistas: Centro de Madrid, zona de 50km²
    const baseLat = 40.4168;
    const baseLon = -3.7038;
    const latRange = 0.2;
    const lonRange = 0.2;

    for (let i = 1; i <= 20; i++) {
      const vehicleId = new VehicleId(`MOCK-VEHICLE-${String(i).padStart(2, '0')}`);
      const licensePlate = `MAD-${String(i).padStart(4, '0')}`;
      
      // Generar posición pseudoaleatoria pero determinista (seed basado en ID)
      const seed = i * 12345; // Seed simple
      const lat = baseLat + (((seed * 7) % 10000) / 10000) * latRange - latRange / 2;
      const lon = baseLon + (((seed * 13) % 10000) / 10000) * lonRange - lonRange / 2;

      const position = new Position(lat, lon, new Date());
      const vehicle = new Vehicle(vehicleId, licensePlate, position);
      
      vehicles.push(vehicle);
    }

    return vehicles;
  }
}
```

**ApiDataProvider**
```typescript
// Ubicación: infrastructure/data-providers/api-data-provider.ts
// Responsabilidades:
// - Cargar vehículos desde API backend real
// - Transformar DTO a entidades de dominio
// - Manejar errores de API

export class ApiDataProvider implements VehicleDataProvider {
  constructor(private apiClient: ApiClient) {}

  async loadVehicles(): Promise<Vehicle[]> {
    try {
      const dtos = await this.apiClient.get<VehicleDTO[]>('/api/vehicles');
      
      return dtos.map(dto => {
        const vehicleId = new VehicleId(dto.vehicleId);
        const position = new Position(
          dto.latitude,
          dto.longitude,
          new Date(dto.timestamp),
        );
        return new Vehicle(vehicleId, dto.licensePlate, position);
      });
    } catch (error) {
      throw new VehicleLoadError('Failed to load vehicles from API', { cause: error });
    }
  }
}

interface VehicleDTO {
  vehicleId: string;
  licensePlate: string;
  latitude: number;
  longitude: number;
  timestamp: string; // ISO-8601
}
```

---

## 6. Presentación: Hooks reutilizables

### 6.1 useVehicles Hook

```typescript
// Ubicación: presentation/hooks/useVehicles.ts
// Responsabilidades:
// - Cargar vehículos inicialmente
// - Mantener estado de vehículos en componente
// - Manejar actualizaciones de posición
// - Proporcionar estados de carga/error

interface UseVehiclesResult {
  vehicles: Vehicle[];
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export const useVehicles = (): UseVehiclesResult => {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const vehicleStateManager = useVehicleStateManager(); // Inyectado desde contexto

  // Cargar vehículos inicialmente
  useEffect(() => {
    const loadVehicles = async () => {
      try {
        setLoading(true);
        setError(null);
        const loaded = await vehicleStateManager.loadVehicles();
        setVehicles(loaded);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setLoading(false);
      }
    };

    loadVehicles();
  }, [vehicleStateManager]);

  // Suscribirse a actualizaciones de posición
  useEffect(() => {
    const handleUpdate = (updated: Vehicle[]) => {
      setVehicles(updated);
    };

    const unsubscribe = vehicleStateManager.subscribeToUpdates(handleUpdate);

    return unsubscribe;
  }, [vehicleStateManager]);

  const refetch = async () => {
    try {
      setLoading(true);
      setError(null);
      const loaded = await vehicleStateManager.loadVehicles();
      setVehicles(loaded);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  };

  return { vehicles, loading, error, refetch };
};
```

### 6.2 useSSEConnection Hook

```typescript
// Ubicación: presentation/hooks/useSSEConnection.ts
// Responsabilidades:
// - Gestionar ciclo de vida de conexión SSE
// - Mantener estado de conexión
// - Limpiar recursos en desmontaje

interface UseSSEConnectionResult {
  status: ConnectionStatus;
  error: Error | null;
}

export const useSSEConnection = (): UseSSEConnectionResult => {
  const [status, setStatus] = useState<ConnectionStatus>('DISCONNECTED');
  const [error, setError] = useState<Error | null>(null);

  const vehicleStateManager = useVehicleStateManager();

  useEffect(() => {
    let unsubscribe: (() => void) | null = null;

    const setupConnection = async () => {
      try {
        setError(null);
        unsubscribe = await vehicleStateManager.startRealtimeUpdates(
          (connectionStatus) => {
            setStatus(connectionStatus);
            if (connectionStatus === 'ERROR') {
              setError(new Error('SSE connection failed'));
            }
          },
        );
      } catch (err) {
        const e = err instanceof Error ? err : new Error(String(err));
        setError(e);
        setStatus('DISCONNECTED');
      }
    };

    setupConnection();

    // Cleanup: desconectar cuando el componente se desmonta
    return () => {
      if (unsubscribe) {
        unsubscribe();
      }
      vehicleStateManager.stopRealtimeUpdates();
    };
  }, [vehicleStateManager]);

  return { status, error };
};
```

### 6.3 useVehicleUpdates Hook

```typescript
// Ubicación: presentation/hooks/useVehicleUpdates.ts
// Responsabilidades:
// - Permitir suscripción granular a actualizaciones de vehículos específicos
// - Facilitar optimización de componentes para listas grandes

interface UseVehicleUpdatesResult {
  vehicle: Vehicle | null;
  loading: boolean;
}

export const useVehicleUpdates = (vehicleId: string): UseVehicleUpdatesResult => {
  const [vehicle, setVehicle] = useState<Vehicle | null>(null);
  const [loading, setLoading] = useState(true);

  const vehicleStateManager = useVehicleStateManager();

  useEffect(() => {
    setLoading(true);
    
    // Cargar vehículo inicial
    vehicleStateManager.getVehicleById(vehicleId).then((v) => {
      setVehicle(v);
      setLoading(false);
    });

    // Suscribirse a actualizaciones de este vehículo específico
    const unsubscribe = vehicleStateManager.subscribeToVehicleUpdates(
      vehicleId,
      (updated) => {
        setVehicle(updated);
      },
    );

    return unsubscribe;
  }, [vehicleId, vehicleStateManager]);

  return { vehicle, loading };
};
```

### 6.4 useReconnectionManager Hook

```typescript
// Ubicación: presentation/hooks/useReconnectionManager.ts
// Responsabilidades:
// - Orquestar lógica de reconexión
// - Exponenciar backoff
// - Permitir reintentos manuales

interface UseReconnectionManagerResult {
  isReconnecting: boolean;
  nextRetryIn: number;
  manualRetry: () => void;
}

export const useReconnectionManager = (): UseReconnectionManagerResult => {
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [nextRetryIn, setNextRetryIn] = useState(0);
  const vehicleStateManager = useVehicleStateManager();

  useEffect(() => {
    const handleConnectionError = async () => {
      setIsReconnecting(true);
      
      const attempt = vehicleStateManager.getReconnectAttempts();
      const baseBackoffMs = 1000;
      const backoffMs = Math.min(
        baseBackoffMs * Math.pow(2, attempt),
        30000, // máx 30 segundos
      );

      // Mostrar countdown
      let remaining = backoffMs;
      const countdownInterval = setInterval(() => {
        remaining -= 100;
        setNextRetryIn(Math.max(0, remaining));
        if (remaining <= 0) {
          clearInterval(countdownInterval);
        }
      }, 100);

      // Esperar y reintentar
      await new Promise(resolve => setTimeout(resolve, backoffMs));
      await vehicleStateManager.startRealtimeUpdates(() => {});
      
      setIsReconnecting(false);
    };

    vehicleStateManager.onReconnectionNeeded(handleConnectionError);
  }, [vehicleStateManager]);

  const manualRetry = async () => {
    setIsReconnecting(true);
    try {
      await vehicleStateManager.startRealtimeUpdates(() => {});
    } finally {
      setIsReconnecting(false);
    }
  };

  return { isReconnecting, nextRetryIn, manualRetry };
};
```

---

## 7. Flujo de datos y actualización en tiempo real

### 7.1 Diagrama conceptual de flujo

```
┌─────────────────────────────────────────────────────────────────┐
│                    Presentación (React)                          │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │ VehicleDashboard                                          │   │
│ │  - useVehicles() → vehicles[], loading, error            │   │
│ │  - useSSEConnection() → status, error                    │   │
│ │                                                           │   │
│ │  └─ MapContainer(vehicles)                              │   │
│ │     ├─ useVehicles() → vehicles actualizadas            │   │
│ │     └─ VehicleMarker x 20                               │   │
│ │        └─ useVehicleUpdates(vehicleId)                  │   │
│ └──────────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
  ┌─────────────┐ ┌─────────────┐ ┌──────────────┐
  │  Aplicación │ │ VehicleState│ │   Dominio    │
  │  (Use Cases)│ │   Manager   │ │  (Entidades) │
  ├─────────────┤ ├─────────────┤ ├──────────────┤
  │LoadVehicles │ │ coordina    │ │Vehicle       │
  │Subscribe    │ │ casos uso   │ │Position      │
  │Handle       │ │ y estado    │ │VehicleId     │
  │ Update      │ │ compartido  │ │ PositionValidator
  └─────────────┘ └─────────────┘ └──────────────┘
       │               │
       └───────────────┼─────────────────────────────┐
                       ▼                             │
            ┌──────────────────────┐                 │
            │  Infraestructura      │                 │
            ├──────────────────────┤                 │
            │ SSEConnection        │ (Real Backend)  │
            │ ├─connect()          │                 │
            │ ├─onPositionUpdate() │                 │
            │ └─reconnect logic    │                 │
            │                      │                 │
            │ MockDataProvider     │ (Desarrollo)    │
            │ ApiDataProvider      │                 │
            │                      │                 │
            │ LocalStorageAdapter  │ (Cache local)   │
            └──────────────────────┘
                       │
                       ▼
        ┌──────────────────────────┐
        │   Backend API (SSE)       │
        │ GET /api/vehicles         │
        │ EventStream               │
        │ /api/vehicles/stream      │
        └──────────────────────────┘
```

### 7.2 Secuencia de inicialización

```
1. Usuario abre VehicleDashboard
   ↓
2. useVehicles() hook monta
   ├─ LoadVehiclesUseCase.execute()
   ├─ VehicleDataProvider.loadVehicles() (mock o API)
   ├─ VehicleStorage.saveVehicles() (cache local)
   └─ setVehicles([...]) → re-render con marcadores
   ↓
3. useSSEConnection() hook monta
   ├─ SSEConnection.connect()
   ├─ EventSource.addEventListener('position_update')
   └─ setStatus('CONNECTED') → mostrar indicador
   ↓
4. Usuario ve dashboard con 20 vehículos en mapa
```

### 7.3 Secuencia de actualización en tiempo real

```
1. Backend publica evento SSE: position_update
   ↓
2. SSEConnection recibe evento
   ├─ parse JSON
   └─ llama updateCallbacks
   ↓
3. VehicleStateManager.subscribeToUpdates callback se ejecuta
   ├─ SubscribeToPositionUpdatesUseCase ejecuta
   ├─ HandlePositionUpdateUseCase.execute()
   │  ├─ PositionValidator.validate(lat, lon)
   │  ├─ Position constructor crea nueva instancia
   │  ├─ Vehicle.updatePosition()
   │  └─ VehicleStorage.saveVehicles() (cache local)
   └─ llama onUpdate(updatedVehicles)
   ↓
4. Hook useVehicles se actualiza
   ├─ setVehicles(updated) → estado nuevo
   └─ re-render MapContainer
   ↓
5. MapContainer re-renderiza
   ├─ VehicleMarker del vehículo actualizado se mueve
   └─ Usuario ve movimiento en tiempo real
```

---

## 8. Manejo de errores y reconexión

### 8.1 Estrategia de reconexión con backoff exponencial

```typescript
// Pseudocódigo del ReconnectionManager
class ReconnectionManager {
  private reconnectAttempts = 0;
  private maxAttempts = 5;
  private baseBackoffMs = 1000;
  private maxBackoffMs = 30000;

  async attemptReconnect(): Promise<void> {
    while (this.reconnectAttempts < this.maxAttempts) {
      const backoffMs = Math.min(
        this.baseBackoffMs * Math.pow(2, this.reconnectAttempts),
        this.maxBackoffMs,
      );

      console.log(`Reconectando en ${backoffMs}ms (intento ${this.reconnectAttempts + 1})`);
      
      await sleep(backoffMs);

      try {
        await this.sseConnection.connect();
        this.reconnectAttempts = 0; // Reset on success
        return; // Conexión exitosa
      } catch (error) {
        this.reconnectAttempts++;
        console.error(`Reconexión falló (intento ${this.reconnectAttempts})`);
      }
    }

    // Máximo de intentos alcanzado
    console.error('Máximo de intentos de reconexión alcanzado');
    this.notifyMaxReconnectAttempts();
  }
}
```

### 8.2 Máquina de estados de conexión

```
DISCONNECTED
    ↓
  connect()
    ↓
CONNECTING → (timeout/error) → ERROR
    ↓                             ↓
  success              (reconexión automática)
    ↓                             ↓
CONNECTED ←──────────────────────┘
    ↓
disconnect()
    ↓
DISCONNECTED


Estados y transiciones:
- DISCONNECTED: No hay conexión activa
- CONNECTING: Intentando establecer conexión
- CONNECTED: Conexión activa, recibiendo eventos
- ERROR: Fallo de conexión, esperando reconectar
```

### 8.3 Manejo de eventos inválidos

```typescript
// En EventProcessor
try {
  validateEvent(event);
  const vehicle = updateVehicleWithEvent(event);
  saveVehicle(vehicle);
  notifyUpdate(vehicle);
} catch (ValidationError) {
  // Log el evento inválido
  logger.warn(`Evento inválido ignorado: ${event}`, { event });
  // NO lanzar error; continuar recibiendo eventos
  // NO corromper estado actual del vehículo
}
```

### 8.4 Cache local para offline

```typescript
// Estrategia:
// 1. Cada vez que llega una actualización, guardar en localStorage
// 2. Si SSE se desconecta, usar datos en cache
// 3. Cuando SSE se reconecta, revalidar con servidor
// 4. Marcar visualmente datos como "desactualizados" si > 60 minutos

// En VehicleMarker:
const isStaleData = isPositionStale(vehicle.position.timestamp);

return (
  <Marker
    className={isStaleData ? 'stale-marker' : ''}
    title="Última actualización: HH:MM:SS"
  >
    {/* ... */}
  </Marker>
);
```

---

## 9. Correctness Properties

*Una propiedad es una característica o comportamiento que debe mantenerse verdadera en todas las ejecuciones válidas del sistema. Las propiedades sirven como puente entre especificaciones legibles por humanos y garantías de correctitud verificables por máquina.*

### Property 1: Invariante - Número de marcadores = Número de vehículos

*Para cualquier* conjunto de vehículos cargados en el dashboard, el número de marcadores renderizados en el mapa debe ser exactamente igual al número de vehículos.

**Valida: Requisitos 1.2, 1.3**

**Implementación (pseudocódigo de test):**
```typescript
property('marker count equals vehicle count', () => {
  fc.assert(
    fc.property(
      vehicleGenerator(fc),
      (vehicles) => {
        // Arrange
        const dashboard = renderDashboard({ vehicles });
        
        // Act
        const markers = dashboard.queryAllByTestId('vehicle-marker');
        
        // Assert
        return markers.length === vehicles.length;
      },
    ),
  );
});
```

### Property 2: Invariante - Todos los IDs de vehículos son únicos

*Para cualquier* conjunto de vehículos mostrados, todos los identificadores (IDs) mostrados en los marcadores deben ser únicos (sin duplicados).

**Valida: Requisitos 2.3**

**Implementación:**
```typescript
property('all vehicle ids are unique', () => {
  fc.assert(
    fc.property(
      vehicleGenerator(fc).filter(v => v.length > 0),
      (vehicles) => {
        const ids = vehicles.map(v => v.vehicleId.value);
        const uniqueIds = new Set(ids);
        return ids.length === uniqueIds.size;
      },
    ),
  );
});
```

### Property 3: Invariante - Coordenadas válidas persisten desde fuente a representación

*Para cualquier* vehículo con coordenadas válidas, las coordenadas renderizadas en el marcador deben preservar al menos 4 decimales de precisión.

**Valida: Requisitos 2.2**

**Implementación:**
```typescript
property('position precision preserved to 4 decimals', () => {
  fc.assert(
    fc.property(
      positionGenerator(fc),
      (position) => {
        // Arrange
        const marker = renderVehicleMarker({ vehicle: createVehicle(position) });
        
        // Act
        const displayedCoords = marker.getByTestId('coords').textContent;
        const [displayedLat, displayedLon] = displayedCoords.split(', ').map(Number);
        
        // Assert - Verificar precisión de 4 decimales
        return (
          Math.abs(displayedLat - position.latitude) < 1e-4 &&
          Math.abs(displayedLon - position.longitude) < 1e-4
        );
      },
    ),
  );
});
```

### Property 4: Invariante - Validación rechaza coordenadas fuera de rango

*Para cualquier* coordenada con latitud fuera del rango [-90, 90] o longitud fuera de [-180, 180], el validador debe lanzar excepción y rechazar la creación de Position.

**Valida: Requisitos 2.6**

**Implementación:**
```typescript
property('invalid coordinates are rejected', () => {
  fc.assert(
    fc.property(
      fc.oneof(
        fc.float({ min: -180, max: -91 }),
        fc.float({ min: 91, max: 180 }),
      ),
      fc.float({ min: -180, max: 180 }),
      (invalidLat, validLon) => {
        // Assert
        expect(() => new Position(invalidLat, validLon))
          .toThrow(InvalidCoordinateError);
      },
    ),
  );
});
```

### Property 5: Idempotencia - Procesar la misma actualización múltiples veces = procesarla una vez

*Para cualquier* evento de actualización de posición, procesarlo N veces debe resultar en el mismo estado final que procesarlo una sola vez.

**Valida: Requisitos 3.2**

**Implementación:**
```typescript
property('processing same position update is idempotent', () => {
  fc.assert(
    fc.property(
      positionUpdateEventGenerator(fc),
      fc.integer({ min: 1, max: 10 }),
      async (event, repetitions) => {
        // Arrange
        const initialState = await getVehicleStorage().loadVehicles();
        
        // Act - Process event once
        await handlePositionUpdateUseCase.execute(event, () => {});
        const stateAfterOnce = await getVehicleStorage().loadVehicles();
        
        // Act - Process same event N times
        for (let i = 0; i < repetitions; i++) {
          await handlePositionUpdateUseCase.execute(event, () => {});
        }
        const stateAfterRepeat = await getVehicleStorage().loadVehicles();
        
        // Assert
        return statesAreEqual(stateAfterOnce, stateAfterRepeat);
      },
    ),
  );
});
```

### Property 6: Idempotencia - Reconectar tras fallo = transición correcta de estado

*Para cualquier* secuencia de fallos de conexión, reconectar tras cada fallo debe resultar en el mismo estado conectado final.

**Valida: Requisitos 3.5**

**Implementación:**
```typescript
property('reconnection after failure reaches connected state', () => {
  fc.assert(
    fc.property(
      fc.array(
        fc.oneof(
          fc.constant('timeout'),
          fc.constant('network_error'),
          fc.constant('server_error'),
        ),
        { minLength: 1, maxLength: 5 },
      ),
      async (failures) => {
        // Arrange
        const mockSSE = createMockSSE();
        mockSSE.injectFailures(failures);
        
        // Act
        const reconnectionManager = new ReconnectionManager(mockSSE);
        await reconnectionManager.attemptReconnect();
        
        // Assert
        return mockSSE.getStatus() === 'CONNECTED';
      },
    ),
  );
});
```

### Property 7: Invariante - Datos mock siempre generan coordenadas válidas

*Para cualquier* ejecución de MockDataProvider.loadVehicles(), todos los vehículos generados deben tener coordenadas dentro de rango válido (lat ∈ [-90, 90], lon ∈ [-180, 180]).

**Valida: Requisitos 4.3**

**Implementación:**
```typescript
property('mock data always generates valid coordinates', () => {
  fc.assert(
    fc.property(fc.anything(), async () => {
      // Arrange
      const provider = new MockDataProvider();
      
      // Act
      const vehicles = await provider.loadVehicles();
      
      // Assert
      return vehicles.every(v => {
        const lat = v.position.latitude;
        const lon = v.position.longitude;
        return (
          lat >= -90 && lat <= 90 &&
          lon >= -180 && lon <= 180
        );
      });
    }),
  );
});
```

### Property 8: Round-trip - Datos mock transformados a/desde DTO mantienen integridad

*Para cualquier* vehículo generado por MockDataProvider, convertirlo a DTO y de vuelta a Vehicle debe mantener los campos esenciales (ID, posición, placa).

**Valida: Requisitos 4.6**

**Implementación:**
```typescript
property('vehicle to dto and back preserves data', () => {
  fc.assert(
    fc.property(vehicleGenerator(fc), (original) => {
      // Act
      const dto = vehicleToDTO(original);
      const reconstructed = dtoToVehicle(dto);
      
      // Assert
      return (
        reconstructed.vehicleId.value === original.vehicleId.value &&
        reconstructed.licensePlate === original.licensePlate &&
        Math.abs(reconstructed.position.latitude - original.position.latitude) < 1e-6 &&
        Math.abs(reconstructed.position.longitude - original.position.longitude) < 1e-6
      );
    }),
  );
});
```

### Property 9: Metamórfica - Orden de actualizaciones SSE no afecta estado final de mapeo

*Para cualquier* secuencia de actualizaciones de posición para vehículos distintos, el orden en que se procesan no debe afectar el estado final del mapeo posición→vehículo (solo el estado del vehículo más reciente importa).

**Valida: Requisitos 3.2**

**Implementación:**
```typescript
property('update processing order independence', () => {
  fc.assert(
    fc.property(
      fc.array(positionUpdateEventGenerator(fc), { minLength: 2, maxLength: 10 }),
      async (events) => {
        // Ensure at least 2 different vehicles
        const uniqueVehicles = new Set(events.map(e => e.vehicleId));
        fc.pre(uniqueVehicles.size >= 2);
        
        // Arrange & Act - Process in original order
        const state1 = {};
        for (const event of events) {
          await handleUpdate(event, (v) => {
            state1[v.vehicleId.value] = v;
          });
        }
        
        // Arrange & Act - Process in random order
        const shuffled = fc.shuffled(events);
        const state2 = {};
        for (const event of shuffled) {
          await handleUpdate(event, (v) => {
            state2[v.vehicleId.value] = v;
          });
        }
        
        // Assert - Final state should be identical
        return statesAreEqual(state1, state2);
      },
    ),
  );
});
```

---

## 10. Estrategia de testing

### 10.1 Suite de pruebas rápidas (Unit + Domain)

**Ubicación**: `frontend/src/**/*.test.ts`

**Ejemplos:**
- Domain: Position, Vehicle, VehicleId validation
- Validators: posición out-of-range detection
- Use Cases: LoadVehicles, HandlePositionUpdate (con fakes)
- Presentación: Componentes simples sin SSE real

**Herramientas**: Vitest, @testing-library/react

**Tiempo**: < 2 segundos

### 10.2 Suite de integración

**Ubicación**: `frontend/src/**/*.integration.test.ts`

**Ejemplos:**
- SSE real con servidor mock (MSW)
- Flujo completo: cargar vehículos → abrir dashboard → recibir actualización → verificar mapa
- Reconexión y manejo de errores
- LocalStorage persisten y recuperan correctamente

**Herramientas**: Vitest, MSW (Mock Service Worker), @testing-library/react

**Tiempo**: < 10 segundos

### 10.3 E2E (opcional para MVP)

**Ubicación**: `e2e/dashboard.spec.ts`

**Ejemplos:**
- Abrir Playwright, cargar dashboard
- Verificar mapa interactivo
- Verificar SSE simulado
- Zoom, pan, marcadores visibles

**Herramientas**: Playwright

**Tiempo**: < 30 segundos

### 10.4 Cobertura esperada

- **Dominio**: 100% (crítico para reglas de negocio)
- **Aplicación**: 90%+ (casos de uso, state manager)
- **Infraestructura**: 60%+ (adaptadores, fakes probados)
- **Presentación**: 70%+ (flujos de usuario principales)

---

## 11. Integración con OpenAPI

### 11.1 Contrato esperado del backend

```yaml
# openapi.yaml (fragmento)
paths:
  /api/vehicles:
    get:
      summary: Get all vehicles
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Vehicle'
        '500':
          content:
            application/problem+json:
              schema:
                $ref: '#/components/schemas/ProblemDetails'

  /api/vehicles/positions/stream:
    get:
      summary: Subscribe to real-time position updates
      responses:
        '200':
          content:
            text/event-stream:
              schema:
                type: object
                items:
                  $ref: '#/components/schemas/PositionUpdateEvent'

components:
  schemas:
    Vehicle:
      type: object
      properties:
        vehicleId:
          type: string
        licensePlate:
          type: string
        latitude:
          type: number
          minimum: -90
          maximum: 90
        longitude:
          type: number
          minimum: -180
          maximum: 180
        timestamp:
          type: string
          format: date-time
      required: [vehicleId, licensePlate, latitude, longitude, timestamp]

    PositionUpdateEvent:
      type: object
      properties:
        type:
          type: string
          enum: [position_update]
        data:
          type: object
          properties:
            vehicleId:
              type: string
            latitude:
              type: number
            longitude:
              type: number
            timestamp:
              type: string
              format: date-time
          required: [vehicleId, latitude, longitude, timestamp]

    ProblemDetails:
      type: object
      properties:
        type:
          type: string
        title:
          type: string
        status:
          type: integer
        detail:
          type: string
```

### 11.2 Validación de respuestas

```typescript
// En test de integración:
// Usar MSW para interceptar requests
// Validar que handlers cumplen esquema OpenAPI

const handlers = [
  http.get('/api/vehicles', ({ request }) => {
    return HttpResponse.json(
      [
        {
          vehicleId: 'VH-001',
          licensePlate: 'MAD-0001',
          latitude: 40.4168,
          longitude: -3.7038,
          timestamp: new Date().toISOString(),
        },
      ],
      { status: 200 },
    );
  }),

  http.get('/api/vehicles/positions/stream', ({ request }) => {
    // EventSource requires text/event-stream
    return new HttpResponse('...', {
      headers: { 'Content-Type': 'text/event-stream' },
    });
  }),
];
```

---

## 12. Variables de entorno y configuración

```bash
# .env.local (desarrollo)
VITE_DATA_MODE=mock              # 'mock' o 'api'
VITE_API_BASE_URL=http://localhost:8080
VITE_SSE_ENDPOINT=/api/vehicles/positions/stream
VITE_MAP_TILE_URL=https://tile.openstreetmap.org/{z}/{x}/{y}.png

# .env.production
VITE_DATA_MODE=api
VITE_API_BASE_URL=https://api.example.com
VITE_SSE_ENDPOINT=/api/vehicles/positions/stream
VITE_MAP_TILE_URL=https://tile.openstreetmap.org/{z}/{x}/{y}.png
```

---

## 13. Notas de implementación

1. **Selección de librería de mapa**: `react-leaflet` es ligera y open-source. Si en el futuro se necesita Mapbox, solo cambiar el adaptador en infraestructura.

2. **SSE vs WebSocket**: SSE es unidireccional y suficiente para este MVP. No hay necesidad de bidireccional.

3. **Estado compartido**: React Context + hooks es suficiente para el MVP. No agregar Redux/Zustand sin demostrada necesidad.

4. **Dato mock determinista**: Mismo seed produce mismos datos entre ejecutiones, facilitando testing reproducible.

5. **Offline-first**: localStorage persiste vehículos; si SSE cae, mostrar datos cacheados marcados como "desactualizados".

6. **Reconexión**: Máx 5 intentos con backoff exponencial (1s, 2s, 4s, 8s, 16s, máx 30s).

7. **Performance**: Para 20 vehículos, el dashboard es rápido. Si en futuro hay 1000+, optimizar con clustering y virtual scroll.

---

## 14. Diagrama de interacción: Secuencia completa

```
┌──────────┐         ┌──────────────┐       ┌─────────────┐
│ Usuario  │         │  Dashboard   │       │  Backend    │
└──────────┘         └──────────────┘       └─────────────┘
     │                      │                      │
     │ Abre dashboard       │                      │
     │─────────────────────>│                      │
     │                      │                      │
     │                      │ LoadVehiclesUseCase  │
     │                      ├─ Mock o API          │
     │                      │  GET /api/vehicles   │
     │                      │────────────────────> │
     │                      │                      │
     │                      │            [20 vehicles]
     │                      │ <────────────────────│
     │                      │                      │
     │   [Mapa con 20       │                      │
     │    marcadores]       │                      │
     │ <─────────────────────                      │
     │                      │                      │
     │                      │ SSEConnection.connect│
     │                      ├─ EventSource         │
     │                      │  GET /api/stream     │
     │                      │────────────────────> │
     │                      │                      │
     │                      │    [EventSource open]
     │   [Status:           │                      │
     │    Conectado]        │                      │
     │ <─────────────────────                      │
     │                      │                      │
     │                      │ [Vehicle moves]      │
     │                      │                      │
     │                      │    position_update   │
     │                      │ <────────────────────│
     │                      │ { vehicleId, lat,    │
     │                      │   lon, timestamp }   │
     │                      │                      │
     │                      │ HandlePositionUpdate │
     │                      ├─ Validate            │
     │                      ├─ Update Vehicle      │
     │                      └─ Save in localStorage│
     │                      │                      │
     │   [Marcador 1 se     │                      │
     │    mueve en mapa]    │                      │
     │ <─────────────────────                      │
     │                      │                      │
     │                      │ [Más eventos SSE]   │
     │                      │ <────────────────────│
     │                      │                      │
     │   [Múltiples         │                      │
     │    marcadores        │                      │
     │    actualizados]     │                      │
     │ <─────────────────────                      │
     │                      │                      │
     │ [Cierra tab]         │                      │
     │─────────────────────>│                      │
     │                      │ SSEConnection.       │
     │                      │   disconnect()       │
     │                      │────────────────────> │
     │                      │                      │
     │                      │    [EventSource      │
     │                      │     cerrado]         │
     │                      │ <────────────────────│
```

---

## Resumen ejecutivo

El Dashboard de Mapa de Vehículos implementa una arquitectura modular y escalable que separa claramente las responsabilidades:

- **Dominio puro**: Entidades (Vehicle, Position, VehicleId) con validación de invariantes
- **Aplicación**: Casos de uso desacoplados que coordinan la lógica de negocio
- **Infraestructura**: Adaptadores intercambiables (SSE real, mock, localStorage, mapas)
- **Presentación**: Componentes React y hooks reutilizables

El flujo de datos es unidireccional: Presentación → Aplicación → Dominio/Infraestructura. Las pruebas se organizan en suites: rápidas (unit + domain), integración (con SSE/HTTP reales) y E2E (optional).

La reconexión es automática con backoff exponencial. Los datos se cachean localmente. El sistema es offline-first. El contrato con el backend se valida contra OpenAPI.

Este diseño es **implementable**, **testeable** y **mantenible**, siguiendo las restricciones de Clean Architecture y los estándares del proyecto.
