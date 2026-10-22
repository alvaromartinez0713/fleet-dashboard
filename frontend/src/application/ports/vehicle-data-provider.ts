import { Vehicle } from '@domain/entities/vehicle';

/**
 * VehicleDataProvider - Puerto (interfaz) para cargar datos de vehículos.
 * Las implementaciones pueden ser mock o API real.
 */
export interface VehicleDataProvider {
  /**
   * Carga la lista de vehículos.
   */
  loadVehicles(): Promise<Vehicle[]>;
}
