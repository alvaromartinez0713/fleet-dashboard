import { Vehicle } from '@domain/entities/vehicle';
import { VehicleStorage } from '@application/ports/vehicle-storage';

/**
 * FakeVehicleStorage - Implementación fake de VehicleStorage para testing
 *
 * Almacena vehículos en memoria para testing determinista sin dependencias.
 *
 * Verifica: testing.md Sección 5 (Fakes con estado)
 */
export class FakeVehicleStorage implements VehicleStorage {
  private vehicles: Vehicle[] = [];
  private error: Error | null = null;

  withError(error: Error): this {
    this.error = error;
    return this;
  }

  async saveVehicles(vehicles: Vehicle[]): Promise<void> {
    if (this.error) {
      throw this.error;
    }
    // Almacena copia para evitar mutación
    this.vehicles = vehicles.map((v) => v);
  }

  async loadVehicles(): Promise<Vehicle[]> {
    if (this.error) {
      throw this.error;
    }
    // Retorna copia para evitar mutación
    return [...this.vehicles];
  }

  async clear(): Promise<void> {
    if (this.error) {
      throw this.error;
    }
    this.vehicles = [];
  }

  getStoredVehicles(): Vehicle[] {
    return [...this.vehicles];
  }

  isEmpty(): boolean {
    return this.vehicles.length === 0;
  }

  reset(): void {
    this.vehicles = [];
    this.error = null;
  }
}
