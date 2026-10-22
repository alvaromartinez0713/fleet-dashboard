import { InvalidVehicleIdError } from '../errors/vehicle-errors';

// Verifies: Requisito 5 (dominio independiente), Requisito 2

/**
 * VehicleId - Objeto de valor que representa el identificador único de un vehículo.
 * En el MVP, el vehicleId es equivalente a la placa de licencia.
 */
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
