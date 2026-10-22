import { Vehicle } from '@domain/entities/vehicle';
import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { VehicleMother } from '@tests/support/mothers/vehicle-mother';

/**
 * MockDataProvider - Implementación mock del puerto VehicleDataProvider
 *
 * Proporciona datos hardcoded de vehículos para testing y demostración
 * sin realizar llamadas de red.
 *
 * Genera 8 vehículos con diferentes características usando VehicleMother
 * para garantizar datos válidos y consistentes.
 *
 * Verifica: testing.md Sección 5 (Dobles de prueba - Fakes)
 */
export class MockDataProvider implements VehicleDataProvider {
  /**
   * Carga una lista hardcoded de vehículos de demostración
   *
   * @returns Promise que resuelve a una lista de 8 vehículos válidos
   */
  async loadVehicles(): Promise<Vehicle[]> {
    return this.getHardcodedVehicles();
  }

  /**
   * Genera vehículos de demostración con diferentes características
   *
   * @returns Array de 8 vehículos válidos con diferentes posiciones y estados
   */
  private getHardcodedVehicles(): Vehicle[] {
    return [
      // Vehículo 1: Centro de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-01')
        .withLicensePlate('MAD-001')
        .withPosition(40.4168, -3.7038, new Date())
        .build(),

      // Vehículo 2: Centro de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-02')
        .withLicensePlate('MAD-002')
        .withPosition(40.4180, -3.7050, new Date())
        .build(),

      // Vehículo 3: Zona norte de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-03')
        .withLicensePlate('MAD-003')
        .withPosition(40.4500, -3.7000, new Date())
        .build(),

      // Vehículo 4: Zona sur de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-04')
        .withLicensePlate('MAD-004')
        .withPosition(40.3900, -3.7100, new Date())
        .build(),

      // Vehículo 5: Zona este de Madrid - posición antigua (stale)
      VehicleMother.stalePosition()
        .withVehicleId('MOCK-VEHICLE-05')
        .withLicensePlate('MAD-005')
        .withPosition(40.4200, -3.6800, new Date(Date.now() - 2 * 60 * 60 * 1000))
        .build(),

      // Vehículo 6: Zona oeste de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-06')
        .withLicensePlate('MAD-006')
        .withPosition(40.4100, -3.7300, new Date())
        .build(),

      // Vehículo 7: Zona crítica - posición reciente
      VehicleMother.inCriticalZone()
        .withVehicleId('MOCK-VEHICLE-07')
        .withLicensePlate('MAD-007')
        .build(),

      // Vehículo 8: Otra zona de Madrid - posición reciente
      VehicleMother.valid()
        .withVehicleId('MOCK-VEHICLE-08')
        .withLicensePlate('MAD-008')
        .withPosition(40.4250, -3.6950, new Date())
        .build(),
    ];
  }
}
