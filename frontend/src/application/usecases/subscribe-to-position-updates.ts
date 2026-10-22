import { VehicleRealtimePort, Unsubscribe, ConnectionStatus, PositionUpdateEvent } from '@application/ports/vehicle-realtime-port';
import { Vehicle } from '@domain/entities/vehicle';

/**
 * SubscribeToPositionUpdatesUseCase - Caso de uso para suscribirse a actualizaciones de posición en tiempo real.
 *
 * Responsabilidades:
 * - Conecta al puerto realtime (SSE)
 * - Registra callbacks para actualizaciones de posición
 * - Registra callbacks para cambios de estado de conexión
 * - Retorna función que desconecta y limpia recursos
 *
 * Verifica: Requisito 3.1 (SSE conecta), Requisito 3.7 (actualización reactiva)
 *
 * Nota: En la versión inicial, este caso de uso solo coordina la conexión y
 * registra los listeners. El procesamiento específico de cada evento será
 * delegado a HandlePositionUpdateUseCase (tarea 4.3).
 */
export class SubscribeToPositionUpdatesUseCase {
  constructor(private realtimePort: VehicleRealtimePort) {}

  /**
   * Ejecuta el caso de uso de suscripción a actualizaciones en tiempo real.
   *
   * Flujo:
   * 1. Conecta al servidor SSE a través del puerto realtime
   * 2. Se registra un callback para cada evento de actualización de posición
   * 3. Se registra un callback para cambios de estado de conexión
   * 4. Retorna una función que desconecta y limpia ambos listeners
   *
   * @param onUpdate - Callback invocado cuando llegan actualizaciones de posición con evento SSE
   * @param onStatusChange - Callback invocado cuando cambia el estado de conexión
   * @returns Función de desuscripción que limpia recursos
   */
  async execute(
    onUpdate: (event: PositionUpdateEvent) => void,
    onStatusChange: (status: ConnectionStatus) => void,
  ): Promise<Unsubscribe> {
    // Conectar al servidor SSE
    await this.realtimePort.connect();

    // Registrar callback para actualizaciones de posición
    const unsubscribeUpdates = this.realtimePort.onPositionUpdate((event: PositionUpdateEvent) => {
      // Pasar el evento al callback para que el state manager lo procese
      try {
        onUpdate(event);
      } catch (error) {
        console.error('Error in position update callback:', error);
      }
    });

    // Registrar callback para cambios de estado de conexión
    const unsubscribeStatus = this.realtimePort.onConnectionStatusChange((status: ConnectionStatus) => {
      try {
        onStatusChange(status);
      } catch (error) {
        console.error('Error in status change callback:', error);
      }
    });

    // Retornar función de desuscripción que libera ambos listeners
    return () => {
      unsubscribeUpdates();
      unsubscribeStatus();
      // Llamar disconnect de forma asincrónica sin esperar
      // El comportamiento específico dependerá de cómo se integre con el state manager
      this.realtimePort.disconnect().catch((error) => {
        console.error('Error disconnecting from realtime port:', error);
      });
    };
  }
}
