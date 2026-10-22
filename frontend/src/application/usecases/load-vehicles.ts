import { Vehicle } from '@domain/entities/vehicle';
import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { VehicleStorage } from '@application/ports/vehicle-storage';
import { InvalidVehicleError } from '@domain/errors/vehicle-errors';

/**
 * LoadVehiclesUseCase - Caso de uso para cargar vehículos con fallback a almacenamiento local.
 *
 * Responsabilidades:
 * - Cargar vehículos desde el proveedor de datos (API o mock)
 * - Validar que todos los vehículos son válidos (campos requeridos no nulos)
 * - Guardar vehículos en almacenamiento local para operación offline
 * - Si la carga falla, intentar cargar desde almacenamiento local como fallback
 *
 * Verifica: Requisito 1.6 (offline-first), Requisito 4.2 (cargar vehículos)
 */
export class LoadVehiclesUseCase {
  constructor(
    private vehicleDataProvider: VehicleDataProvider,
    private vehicleStorage: VehicleStorage,
  ) {}

  /**
   * Ejecuta el caso de uso de carga de vehículos.
   *
   * Flujo:
   * 1. Intenta cargar vehículos desde el proveedor de datos
   * 2. Valida que todos los vehículos sean válidos (vehicleId y position no nulos)
   * 3. Guarda vehículos en almacenamiento local para operación offline
   * 4. Si cualquier paso falla, intenta cargar desde almacenamiento local
   * 5. Si todo falla, retorna array vacío sin lanzar excepción
   *
   * @returns Promise que se resuelve con un array de vehículos
   * @throws No lanza excepciones; siempre retorna datos (desde API o cache local, o vacío)
   */
  async execute(): Promise<Vehicle[]> {
    try {
      // Paso 1: Cargar vehículos desde el proveedor de datos
      const vehicles = await this.vehicleDataProvider.loadVehicles();

      // Paso 2: Validar que todos los vehículos son válidos
      vehicles.forEach((vehicle) => {
        if (!vehicle.vehicleId || !vehicle.position) {
          throw new InvalidVehicleError(
            `Vehicle is invalid: missing required fields. Vehicle: ${JSON.stringify(vehicle)}`,
          );
        }
      });

      // Paso 3: Guardar en almacenamiento local para operación offline
      await this.vehicleStorage.saveVehicles(vehicles);

      return vehicles;
    } catch (error) {
      // Fallback: Si la API falla, intentar cargar desde almacenamiento local (sin loguear)
      try {
        return await this.vehicleStorage.loadVehicles();
      } catch (storageError) {
        // Si el almacenamiento también falla, retornar array vacío sin lanzar excepción
        return [];
      }
    }
  }
}
