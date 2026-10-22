import { Position } from '@domain/entities/position';
import { Vehicle } from '@domain/entities/vehicle';
import { PositionValidator } from '@domain/validators/position-validator';
import { PositionUpdateEvent } from '@application/ports/vehicle-realtime-port';
import { VehicleStorage } from '@application/ports/vehicle-storage';

/**
 * HandlePositionUpdateUseCase - Procesador de actualizaciones individuales de posición.
 *
 * Responsabilidades:
 * - Validar coordenadas del evento
 * - Construir nueva posición a partir del evento
 * - Cargar estado actual de vehículos desde almacenamiento
 * - Encontrar vehículo por ID y actualizar su posición
 * - Guardar estado actualizado
 * - Notificar a la capa de presentación
 * - Capturar errores y evitar que propaguen (eventos inválidos se ignoran)
 *
 * Verifica: Requisito 3.2, Requisito 7.5, Requisito 7.6, Property 5 (idempotencia)
 */
export class HandlePositionUpdateUseCase {
  constructor(
    private vehicleStorage: VehicleStorage,
    private positionValidator: PositionValidator = PositionValidator,
  ) {}

  /**
   * Ejecuta el caso de uso de procesamiento de actualización de posición.
   *
   * @param event - Evento de actualización de posición recibido por SSE
   * @param onUpdate - Callback invocado con vehículos actualizados
   */
  async execute(
    event: PositionUpdateEvent,
    onUpdate: (vehicles: Vehicle[]) => void,
  ): Promise<void> {
    try {
      // Validar coordenadas
      (this.positionValidator as typeof PositionValidator).validate(
        event.latitude,
        event.longitude,
      );

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
      // Capturar excepciones y registrarlas sin lanzar
      // Los eventos inválidos se ignoran para evitar que rompan la app
      console.error('Failed to handle position update:', error);
    }
  }
}
