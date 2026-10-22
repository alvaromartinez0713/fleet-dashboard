import { Vehicle } from '@domain/entities/vehicle';
import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { VehicleMother } from '@tests/support/mothers/vehicle-mother';

/**
 * FakeVehicleDataProvider - Implementación fake de VehicleDataProvider para testing
 *
 * Permite controlar exactamente qué vehículos se retornan o qué error se lanza,
 * sin dependencia de red o datos reales.
 *
 * Verifica: testing.md Sección 5 (Fakes con estado)
 */
export class FakeVehicleDataProvider implements VehicleDataProvider {
  private error: Error | null = null;
  private vehicles: Vehicle[] = [];
  private callCount: number = 0;

  withError(error: Error): this {
    this.error = error;
    return this;
  }

  withVehicles(vehicles: Vehicle[]): this {
    this.vehicles = vehicles;
    return this;
  }

  withValidVehicles(count: number = 20): this {
    this.vehicles = VehicleMother.validMultiple(count);
    return this;
  }

  async loadVehicles(): Promise<Vehicle[]> {
    this.callCount++;

    if (this.error) {
      throw this.error;
    }

    return [...this.vehicles]; // Retorna copia para evitar mutación
  }

  getCallCount(): number {
    return this.callCount;
  }

  reset(): void {
    this.error = null;
    this.vehicles = [];
    this.callCount = 0;
  }
}
