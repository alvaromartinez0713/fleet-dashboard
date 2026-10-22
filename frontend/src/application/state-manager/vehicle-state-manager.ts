import { Vehicle } from '@domain/entities/vehicle';
import { ConnectionStatus, Unsubscribe } from '@application/ports/vehicle-realtime-port';
import { LoadVehiclesUseCase } from '@application/usecases/load-vehicles';
import { SubscribeToPositionUpdatesUseCase } from '@application/usecases/subscribe-to-position-updates';
import { HandlePositionUpdateUseCase } from '@application/usecases/handle-position-update';

/**
 * Position update event from SSE
 */
interface PositionUpdateEvent {
  vehicleId: string;
  latitude: number;
  longitude: number;
  timestamp: string;
}

/**
 * VehicleStateManager - Coordinador de aplicación que gestiona el estado de vehículos.
 *
 * Responsabilidades:
 * - Orquestar todos los casos de uso relacionados con vehículos
 * - Mantener referencia a la lista actual de vehículos
 * - Mantener lista de callbacks para notificación de actualizaciones
 * - Coordinar carga inicial, suscripción a SSE y procesamiento de actualizaciones
 * - Proporcionar métodos públicos para carga, suscripción y actualización
 *
 * Verifica: Requisito 5 (coordinador de aplicación), Requisito 6 (hooks reutilizables)
 */
export class VehicleStateManager {
  // Propiedades privadas: estado
  private vehicles: Vehicle[] = [];
  private updateCallbacks: Array<(vehicles: Vehicle[]) => void> = [];
  private vehicleUpdateCallbacks: Map<string, Array<(vehicle: Vehicle) => void>> = new Map();
  private realtimeUnsubscribe: Unsubscribe | null = null;

  /**
   * Constructor - Inyecta casos de uso.
   *
   * @param loadVehiclesUseCase - Caso de uso para cargar vehículos iniciales
   * @param subscribeToPositionUpdatesUseCase - Caso de uso para suscribirse a SSE
   * @param handlePositionUpdateUseCase - Caso de uso para procesar actualizaciones individuales
   */
  constructor(
    private loadVehiclesUseCase: LoadVehiclesUseCase,
    private subscribeToPositionUpdatesUseCase: SubscribeToPositionUpdatesUseCase,
    private handlePositionUpdateUseCase: HandlePositionUpdateUseCase,
  ) {}

  /**
   * Carga vehículos iniciales y notifica todos los callbacks.
   *
   * Flujo:
   * 1. Ejecuta el caso de uso de carga de vehículos
   * 2. Actualiza el estado interno (this.vehicles)
   * 3. Notifica a todos los callbacks suscritos que los vehículos han cambiado
   * 4. Retorna la lista cargada
   *
   * @returns Promise que se resuelve con la lista de vehículos cargados
   */
  async loadVehicles(): Promise<Vehicle[]> {
    try {
      // Ejecutar caso de uso de carga
      const loaded = await this.loadVehiclesUseCase.execute();

      // Actualizar estado interno
      this.vehicles = loaded;

      // Notificar a todos los callbacks
      this.notifyAllCallbacks();

      return loaded;
    } catch (error) {
      console.error('Error loading vehicles:', error);
      // El caso de uso no lanza excepciones, pero en caso de fallos internos inesperados
      return [];
    }
  }

  /**
   * Inicia la suscripción a actualizaciones en tiempo real vía SSE.
   *
   * Flujo:
   * 1. Ejecuta el caso de uso de suscripción a actualizaciones
   * 2. Pasa callbacks internos para procesar actualizaciones y cambios de estado
   * 3. Almacena la función de desuscripción para limpieza posterior
   * 4. Retorna función de unsubscribe
   *
   * @param onStatusChange - Callback invocado cuando cambia el estado de conexión SSE
   * @returns Promise que se resuelve con función para detener la suscripción
   *
   * Verifica: Requisito 3.1, Requisito 6.2
   */
  async startRealtimeUpdates(
    onStatusChange: (status: ConnectionStatus) => void,
  ): Promise<Unsubscribe> {
    try {
      // Callback interno para procesar actualizaciones de posición
      const handleUpdate = (event: PositionUpdateEvent) => {
        // Coordinar con el caso de uso de procesamiento de actualización
        this.handlePositionUpdateUseCase.execute(event, (vehicles) => {
          // Actualizar estado interno
          this.vehicles = vehicles;

          // Notificar a todos los callbacks
          this.notifyAllCallbacks();

          // Notificar a callbacks específicos de vehículos actualizados
          if (event.vehicleId) {
            const updatedVehicle = vehicles.find((v) => v.vehicleId.value === event.vehicleId);
            if (updatedVehicle) {
              this.notifyVehicleCallback(event.vehicleId, updatedVehicle);
            }
          }
        });
      };

      // Callback interno para cambios de estado de conexión
      const handleStatusChange = (status: ConnectionStatus) => {
        onStatusChange(status);
      };

      // Ejecutar caso de uso de suscripción
      this.realtimeUnsubscribe = await this.subscribeToPositionUpdatesUseCase.execute(
        handleUpdate,
        handleStatusChange,
      );

      return this.realtimeUnsubscribe;
    } catch (error) {
      console.error('Error starting realtime updates:', error);
      throw new Error('Failed to start realtime updates');
    }
  }

  /**
   * Detiene la suscripción a actualizaciones en tiempo real.
   *
   * Flujo:
   * 1. Si hay una suscripción activa, la desinscribe (llama unsubscribe)
   * 2. Limpia la referencia
   *
   * @returns Promise que se resuelve cuando la desuscripción está completa
   */
  async stopRealtimeUpdates(): Promise<void> {
    if (this.realtimeUnsubscribe) {
      this.realtimeUnsubscribe();
      this.realtimeUnsubscribe = null;
    }
  }

  /**
   * Suscribe un callback a los cambios de estado global de vehículos.
   *
   * Cuando se actualiza la lista de vehículos (por carga inicial o SSE),
   * todos los callbacks suscritos son notificados con la lista actualizada.
   *
   * Flujo:
   * 1. Añade el callback a la lista de callbacks globales
   * 2. Retorna función que remueve el callback de la lista
   *
   * @param callback - Función invocada con lista de vehículos actualizada
   * @returns Función de desuscripción que remueve este callback
   *
   * Verifica: Requisito 6.1, Requisito 6.5
   */
  subscribeToUpdates(callback: (vehicles: Vehicle[]) => void): Unsubscribe {
    // Añadir callback a la lista
    this.updateCallbacks.push(callback);

    // Retornar función de desuscripción
    return () => {
      this.updateCallbacks = this.updateCallbacks.filter((cb) => cb !== callback);
    };
  }

  /**
   * Obtiene un vehículo por su ID.
   *
   * Busca en la lista actual de vehículos por vehicleId.value.
   *
   * @param vehicleId - ID del vehículo a buscar (string)
   * @returns Promise que se resuelve con el vehículo si existe, null en caso contrario
   */
  async getVehicleById(vehicleId: string): Promise<Vehicle | null> {
    const vehicle = this.vehicles.find((v) => v.vehicleId.value === vehicleId);
    return vehicle ?? null;
  }

  /**
   * Suscribe un callback a las actualizaciones de un vehículo específico.
   *
   * Cuando se actualiza la posición de un vehículo específico,
   * solo los callbacks suscritos a ese vehículo son notificados.
   *
   * Flujo:
   * 1. Obtiene o crea la lista de callbacks para ese vehicleId
   * 2. Añade el callback a la lista
   * 3. Retorna función que remueve el callback
   *
   * @param vehicleId - ID del vehículo a monitorear
   * @param callback - Función invocada con el vehículo actualizado
   * @returns Función de desuscripción que remueve este callback
   *
   * Verifica: Requisito 6.6
   */
  subscribeToVehicleUpdates(
    vehicleId: string,
    callback: (vehicle: Vehicle) => void,
  ): Unsubscribe {
    // Obtener o crear lista de callbacks para este vehículo
    if (!this.vehicleUpdateCallbacks.has(vehicleId)) {
      this.vehicleUpdateCallbacks.set(vehicleId, []);
    }

    const callbacks = this.vehicleUpdateCallbacks.get(vehicleId)!;
    callbacks.push(callback);

    // Retornar función de desuscripción
    return () => {
      this.vehicleUpdateCallbacks.set(
        vehicleId,
        callbacks.filter((cb) => cb !== callback),
      );

      // Limpiar lista vacía
      if (callbacks.length === 0) {
        this.vehicleUpdateCallbacks.delete(vehicleId);
      }
    };
  }

  /**
   * Notifica a todos los callbacks globales con la lista actual de vehículos.
   *
   * @private
   */
  private notifyAllCallbacks(): void {
    this.updateCallbacks.forEach((callback) => {
      try {
        callback(this.vehicles);
      } catch (error) {
        console.error('Error in update callback:', error);
      }
    });
  }

  /**
   * Notifica a los callbacks específicos de un vehículo.
   *
   * @private
   * @param vehicleId - ID del vehículo que fue actualizado
   * @param vehicle - Vehículo actualizado
   */
  private notifyVehicleCallback(vehicleId: string, vehicle: Vehicle): void {
    const callbacks = this.vehicleUpdateCallbacks.get(vehicleId);
    if (callbacks) {
      callbacks.forEach((callback) => {
        try {
          callback(vehicle);
        } catch (error) {
          console.error(`Error in vehicle update callback for ${vehicleId}:`, error);
        }
      });
    }
  }

  /**
   * Obtiene la lista actual de vehículos.
   *
   * @returns Array de vehículos actual
   */
  getVehicles(): Vehicle[] {
    return this.vehicles;
  }
}
