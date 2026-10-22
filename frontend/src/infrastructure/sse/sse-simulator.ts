import {
  VehicleRealtimePort,
  PositionUpdateEvent,
  ConnectionStatus,
  Unsubscribe,
} from '@application/ports/vehicle-realtime-port';
import { Vehicle } from '@domain/entities/vehicle';
import { Position } from '@domain/entities/position';

/**
 * SSESimulator - Simulador de SSE para desarrollo y testing
 *
 * Implementa VehicleRealtimePort generando eventos de actualización de posición
 * simulados para múltiples vehículos. No requiere conexión de red ni backend real.
 *
 * Características:
 * - Simula actualizaciones periódicas de posición con movimiento realista
 * - Maneja conexión/desconexión con latencia simulada
 * - Soporta múltiples suscriptores a eventos y cambios de estado
 * - Determinista: comportamiento reproducible para pruebas
 * - Compatible con la interfaz VehicleRealtimePort
 *
 * Verifica: testing.md Sección 8 (Interceptación a nivel de red)
 */
export class SSESimulator implements VehicleRealtimePort {
  private status: ConnectionStatus = ConnectionStatus.DISCONNECTED;
  private updateCallbacks: ((event: PositionUpdateEvent) => void)[] = [];
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private simulationInterval: NodeJS.Timeout | null = null;
  private vehiclePositions: Map<string, Position> = new Map();
  private random: () => number;

  /**
   * Constructor del SSESimulator
   *
   * @param vehicles - Array de vehículos a simular
   * @param updateIntervalMs - Intervalo entre actualizaciones en ms (default: 2000)
   * @param seed - Semilla para aleatoriedad determinista (opcional, para reproducibilidad)
   */
  constructor(
    private vehicles: Vehicle[],
    private updateIntervalMs: number = 2000,
    seed?: number,
  ) {
    // Inicializar posiciones de vehículos desde sus datos iniciales
    vehicles.forEach((v) => {
      this.vehiclePositions.set(v.vehicleId.value, v.position);
    });

    // Usar generador de números aleatorios determinista si se proporciona seed
    this.random = seed !== undefined ? this.seededRandom(seed) : Math.random;
  }

  /**
   * Conecta al simulador de SSE
   *
   * Simula una latencia de conexión y empieza a generar eventos de actualización
   * de posición periódicamente.
   */
  async connect(): Promise<void> {
    if (this.status !== ConnectionStatus.DISCONNECTED) {
      return;
    }

    this.setStatus(ConnectionStatus.RECONNECTING);

    // Simular latencia de conexión realista (~500ms)
    await new Promise((resolve) => setTimeout(resolve, 500));

    this.setStatus(ConnectionStatus.CONNECTED);

    // Comenzar a simular actualizaciones periódicas
    this.simulationInterval = setInterval(() => {
      this.generatePositionUpdates();
    }, this.updateIntervalMs);
  }

  /**
   * Se suscribe a actualizaciones de posición
   *
   * @param callback - Función a ejecutar cuando haya una actualización
   * @returns Función para desuscribirse
   */
  onPositionUpdate(
    callback: (event: PositionUpdateEvent) => void,
  ): Unsubscribe {
    this.updateCallbacks.push(callback);
    return () => {
      this.updateCallbacks = this.updateCallbacks.filter((cb) => cb !== callback);
    };
  }

  /**
   * Se suscribe a cambios de estado de conexión
   *
   * @param callback - Función a ejecutar cuando cambie el estado
   * @returns Función para desuscribirse
   */
  onConnectionStatusChange(
    callback: (status: ConnectionStatus) => void,
  ): Unsubscribe {
    this.statusCallbacks.push(callback);
    return () => {
      this.statusCallbacks = this.statusCallbacks.filter((cb) => cb !== callback);
    };
  }

  /**
   * Desconecta del simulador
   *
   * Detiene la generación de eventos y cambia el estado a DISCONNECTED.
   */
  async disconnect(): Promise<void> {
    if (this.simulationInterval) {
      clearInterval(this.simulationInterval);
      this.simulationInterval = null;
    }
    this.setStatus(ConnectionStatus.DISCONNECTED);
  }

  /**
   * Obtiene el estado actual de la conexión
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Genera actualizaciones de posición para todos los vehículos
   *
   * Simula movimiento realista:
   * - Pequeños cambios de posición en cada actualización
   * - Variación aleatoria en latitud y longitud
   * - Timestamp actualizado en cada evento
   *
   * @private
   */
  private generatePositionUpdates(): void {
    this.vehicles.forEach((vehicle) => {
      const currentPos = this.vehiclePositions.get(vehicle.vehicleId.value);
      if (!currentPos) {
        return;
      }

      // Generar cambio pequeño y realista de posición
      // Rango: ±0.0005 grados (~55 metros en latitud)
      const latChange = (this.random() - 0.5) * 0.001;
      const lonChange = (this.random() - 0.5) * 0.001;

      const newLat = currentPos.latitude + latChange;
      const newLon = currentPos.longitude + lonChange;

      // Crear evento de actualización
      const event: PositionUpdateEvent = {
        vehicleId: vehicle.vehicleId.value,
        latitude: newLat,
        longitude: newLon,
        timestamp: new Date().toISOString(),
      };

      // Actualizar posición en el simulador
      try {
        const newPosition = new Position(newLat, newLon);
        this.vehiclePositions.set(vehicle.vehicleId.value, newPosition);

        // Notificar a todos los suscriptores
        this.updateCallbacks.forEach((cb) => cb(event));
      } catch (error) {
        // Si la posición es inválida (ej: fuera de rangos), ignorar
        // Esto nunca debería suceder con los cambios pequeños usados
        console.warn(`Invalid position generated for vehicle ${vehicle.vehicleId.value}`, error);
      }
    });
  }

  /**
   * Cambia el estado de conexión y notifica a los suscriptores
   *
   * @private
   */
  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    this.statusCallbacks.forEach((cb) => cb(status));
  }

  /**
   * Crea un generador de números aleatorios determinista usando una semilla
   *
   * Implementa el algoritmo de congruencia lineal para reproducibilidad.
   * Todos los números generados con la misma semilla serán idénticos,
   * lo que permite pruebas deterministas.
   *
   * @private
   */
  private seededRandom(seed: number): () => number {
    // Linear congruential generator - valores de POSIX.1
    const a = 1103515245;
    const c = 12345;
    const m = 2147483648;

    let current = seed;

    return () => {
      current = (a * current + c) % m;
      return current / m;
    };
  }
}
