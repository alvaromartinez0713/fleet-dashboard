import { VehicleId } from './vehicle-id';
import { Position } from './position';
import { InvalidVehicleError } from '../errors/vehicle-errors';

/**
 * Vehicle - Entidad de dominio que representa un vehículo con su posición actual.
 *
 * Invariantes:
 * - vehicleId no puede ser nulo
 * - licensePlate no puede estar vacío
 * - position no puede ser nulo
 *
 * La entidad es inmutable: updatePosition() retorna una nueva instancia.
 */
export class Vehicle {
  constructor(
    readonly vehicleId: VehicleId,
    readonly licensePlate: string,
    readonly position: Position,
  ) {
    this.validate();
  }

  /**
   * Valida los invariantes de la entidad.
   * @throws InvalidVehicleError si algún invariante se viola
   */
  private validate(): void {
    if (!this.vehicleId) {
      throw new InvalidVehicleError('VehicleId is required');
    }
    if (!this.licensePlate || this.licensePlate.trim() === '') {
      throw new InvalidVehicleError('LicensePlate is required and cannot be empty');
    }
    if (!this.position) {
      throw new InvalidVehicleError('Position is required');
    }
  }

  /**
   * Actualiza la posición del vehículo y retorna una nueva instancia.
   * Implementa el patrón de inmutabilidad: el vehículo original no se modifica.
   *
   * @param newPosition - Nueva posición del vehículo
   * @returns Nueva instancia de Vehicle con la posición actualizada
   */
  public updatePosition(newPosition: Position): Vehicle {
    return new Vehicle(this.vehicleId, this.licensePlate, newPosition);
  }

  /**
   * Verifica si la posición actual es antigua (stale).
   *
   * @param maxAgeMs - Edad máxima permitida en milisegundos (por defecto 60 minutos)
   * @returns true si la posición es más antigua que maxAgeMs, false en caso contrario
   */
  public isStalePosition(maxAgeMs: number = 60 * 60 * 1000): boolean {
    const ageMs = Date.now() - this.position.timestamp.getTime();
    return ageMs > maxAgeMs;
  }

  /**
   * Compara este vehículo con otro, basado en el vehicleId.
   * Dos vehículos son iguales si tienen el mismo vehicleId, independientemente de su posición.
   *
   * @param other - Otro vehículo para comparar
   * @returns true si los vehicleIds son iguales, false en caso contrario
   */
  public equals(other: Vehicle): boolean {
    return this.vehicleId.equals(other.vehicleId);
  }

  /**
   * Representación en string de la entidad.
   */
  public toString(): string {
    return `Vehicle(id: ${this.vehicleId}, plate: ${this.licensePlate}, position: ${this.position})`;
  }
}
