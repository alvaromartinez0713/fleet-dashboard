import { describe, it, expect, beforeEach } from 'vitest';
import { Vehicle } from '@domain/entities/vehicle';
import { VehicleId } from '@domain/entities/vehicle-id';
import { Position } from '@domain/entities/position';
import { InvalidVehicleError } from '@domain/errors/vehicle-errors';

/**
 * Test suite para la entidad Vehicle
 * Verifies: Requisito 2 (mostrar posiciones últimas conocidas), Requisito 5 (entidad del dominio)
 *
 * Pruebas de caja negra que verifican:
 * - Invariantes de validación
 * - Immutabilidad (updatePosition retorna nueva instancia)
 * - Comparación con equals()
 * - Detección de posiciones antiguas
 */

describe('Vehicle Entity', () => {
  let vehicleId: VehicleId;
  let licensePlate: string;
  let position: Position;
  let vehicle: Vehicle;

  beforeEach(() => {
    vehicleId = new VehicleId('VEHICLE-001');
    licensePlate = 'VEHICLE-001'; // En MVP, placa = vehicleId
    position = new Position(40.4168, -3.7038, new Date('2024-01-15T10:00:00Z'));
    vehicle = new Vehicle(vehicleId, licensePlate, position);
  });

  describe('Constructor and Validation', () => {
    // Verifies: Requisito 5 (validación de invariantes)
    it('creates a valid vehicle with valid inputs', () => {
      expect(vehicle.vehicleId).toBe(vehicleId);
      expect(vehicle.licensePlate).toBe(licensePlate);
      expect(vehicle.position).toBe(position);
    });

    // Verifies: Requisito 5 (invariante: vehicleId no nulo)
    it('throws InvalidVehicleError when vehicleId is null', () => {
      expect(() => {
        new Vehicle(null as any, licensePlate, position);
      }).toThrow(InvalidVehicleError);
      expect(() => {
        new Vehicle(null as any, licensePlate, position);
      }).toThrow('VehicleId is required');
    });

    // Verifies: Requisito 5 (invariante: vehicleId no nulo)
    it('throws InvalidVehicleError when vehicleId is undefined', () => {
      expect(() => {
        new Vehicle(undefined as any, licensePlate, position);
      }).toThrow(InvalidVehicleError);
    });

    // Verifies: Requisito 5 (invariante: licensePlate no vacío)
    it('throws InvalidVehicleError when licensePlate is empty', () => {
      expect(() => {
        new Vehicle(vehicleId, '', position);
      }).toThrow(InvalidVehicleError);
      expect(() => {
        new Vehicle(vehicleId, '', position);
      }).toThrow('LicensePlate is required');
    });

    // Verifies: Requisito 5 (invariante: licensePlate no vacío)
    it('throws InvalidVehicleError when licensePlate is whitespace only', () => {
      expect(() => {
        new Vehicle(vehicleId, '   ', position);
      }).toThrow(InvalidVehicleError);
    });

    // Verifies: Requisito 5 (invariante: licensePlate no nulo)
    it('throws InvalidVehicleError when licensePlate is null', () => {
      expect(() => {
        new Vehicle(vehicleId, null as any, position);
      }).toThrow(InvalidVehicleError);
    });

    // Verifies: Requisito 5 (invariante: position no nulo)
    it('throws InvalidVehicleError when position is null', () => {
      expect(() => {
        new Vehicle(vehicleId, licensePlate, null as any);
      }).toThrow(InvalidVehicleError);
      expect(() => {
        new Vehicle(vehicleId, licensePlate, null as any);
      }).toThrow('Position is required');
    });

    // Verifies: Requisito 5 (invariante: position no nulo)
    it('throws InvalidVehicleError when position is undefined', () => {
      expect(() => {
        new Vehicle(vehicleId, licensePlate, undefined as any);
      }).toThrow(InvalidVehicleError);
    });
  });

  describe('updatePosition() - Immutability', () => {
    // Verifies: Requisito 5 (updatePosition retorna nueva instancia)
    it('returns a new Vehicle instance with updated position', () => {
      const newPosition = new Position(41.3851, 2.1734, new Date('2024-01-15T11:00:00Z'));
      const updatedVehicle = vehicle.updatePosition(newPosition);

      expect(updatedVehicle).not.toBe(vehicle);
      expect(updatedVehicle).toBeInstanceOf(Vehicle);
      expect(updatedVehicle.position).toBe(newPosition);
    });

    // Verifies: Requisito 5 (updatePosition es immutable)
    it('preserves original vehicleId in updated instance', () => {
      const newPosition = new Position(41.3851, 2.1734);
      const updatedVehicle = vehicle.updatePosition(newPosition);

      expect(updatedVehicle.vehicleId).toBe(vehicle.vehicleId);
      expect(updatedVehicle.vehicleId.equals(vehicle.vehicleId)).toBe(true);
    });

    // Verifies: Requisito 5 (updatePosition es immutable)
    it('preserves original licensePlate in updated instance', () => {
      const newPosition = new Position(41.3851, 2.1734);
      const updatedVehicle = vehicle.updatePosition(newPosition);

      expect(updatedVehicle.licensePlate).toBe(vehicle.licensePlate);
    });

    // Verifies: Requisito 5 (updatePosition no modifica original)
    it('does not modify the original vehicle', () => {
      const originalPosition = vehicle.position;
      const newPosition = new Position(41.3851, 2.1734);

      vehicle.updatePosition(newPosition);

      // Original vehicle's position should be unchanged
      expect(vehicle.position).toBe(originalPosition);
      expect(vehicle.position.latitude).toBe(40.4168);
    });

    // Verifies: Requisito 5 (updatePosition puede encadenarse)
    it('supports chaining multiple updatePosition calls', () => {
      const pos1 = new Position(41.3851, 2.1734, new Date('2024-01-15T11:00:00Z'));
      const pos2 = new Position(42.3601, 2.1743, new Date('2024-01-15T12:00:00Z'));

      const updated1 = vehicle.updatePosition(pos1);
      const updated2 = updated1.updatePosition(pos2);

      // All instances are different
      expect(vehicle).not.toBe(updated1);
      expect(updated1).not.toBe(updated2);

      // Final instance has correct position
      expect(updated2.position).toBe(pos2);

      // Original is unchanged
      expect(vehicle.position).toBe(position);
    });

    // Verifies: Requisito 5 (updatePosition retorna instancia válida)
    it('returns a valid Vehicle that passes validation', () => {
      const newPosition = new Position(41.3851, 2.1734);
      const updatedVehicle = vehicle.updatePosition(newPosition);

      // Should not throw when accessed
      expect(updatedVehicle.vehicleId).toBeDefined();
      expect(updatedVehicle.licensePlate).toBeDefined();
      expect(updatedVehicle.position).toBeDefined();
    });
  });

  describe('equals() - Comparison', () => {
    // Verifies: Requisito 5 (equals basado en vehicleId)
    it('returns true when comparing vehicles with same vehicleId', () => {
      const vehicleId2 = new VehicleId('VEHICLE-001');
      const vehicle2 = new Vehicle(vehicleId2, 'VEHICLE-001', position);

      expect(vehicle.equals(vehicle2)).toBe(true);
    });

    // Verifies: Requisito 5 (equals basado en vehicleId)
    it('returns false when comparing vehicles with different vehicleId', () => {
      const vehicleId2 = new VehicleId('VEHICLE-002');
      const vehicle2 = new Vehicle(vehicleId2, 'VEHICLE-002', position);

      expect(vehicle.equals(vehicle2)).toBe(false);
    });

    // Verifies: Requisito 5 (equals ignora posición)
    it('returns true even if positions differ, when vehicleIds are same', () => {
      const differentPosition = new Position(50.0, 10.0);
      const vehicle2 = new Vehicle(vehicleId, licensePlate, differentPosition);

      expect(vehicle.equals(vehicle2)).toBe(true);
      expect(vehicle.position).not.toEqual(vehicle2.position);
    });

    // Verifies: Requisito 5 (equals ignora placa)
    it('returns true even if licensePlates differ, when vehicleIds are same', () => {
      const vehicleId2 = new VehicleId('VEHICLE-001');
      const vehicle2 = new Vehicle(vehicleId2, 'DIFFERENT-PLATE', position);

      expect(vehicle.equals(vehicle2)).toBe(true);
    });

    // Verifies: Requisito 5 (equals reflexivo)
    it('returns true when comparing vehicle with itself', () => {
      expect(vehicle.equals(vehicle)).toBe(true);
    });

    // Verifies: Requisito 5 (equals transitivo)
    it('is transitive: if v1.equals(v2) and v2.equals(v3), then v1.equals(v3)', () => {
      const vehicleId2 = new VehicleId('VEHICLE-001');
      const vehicle2 = new Vehicle(vehicleId2, licensePlate, position);

      const vehicleId3 = new VehicleId('VEHICLE-001');
      const vehicle3 = new Vehicle(vehicleId3, licensePlate, position);

      expect(vehicle.equals(vehicle2)).toBe(true);
      expect(vehicle2.equals(vehicle3)).toBe(true);
      expect(vehicle.equals(vehicle3)).toBe(true);
    });
  });

  describe('isStalePosition() - Position Age Detection', () => {
    // Verifies: Requisito 2.4 (detectar posiciones antiguas)
    it('returns false for recent position with default maxAgeMs', () => {
      const recentPosition = new Position(40.4168, -3.7038, new Date());
      const vehicleWithRecent = new Vehicle(vehicleId, licensePlate, recentPosition);

      expect(vehicleWithRecent.isStalePosition()).toBe(false);
    });

    // Verifies: Requisito 2.4 (detectar posiciones antiguas > 60 min)
    it('returns true for position older than 60 minutes', () => {
      const oldDate = new Date(Date.now() - 61 * 60 * 1000); // 61 minutes ago
      const oldPosition = new Position(40.4168, -3.7038, oldDate);
      const vehicleWithOld = new Vehicle(vehicleId, licensePlate, oldPosition);

      expect(vehicleWithOld.isStalePosition()).toBe(true);
    });

    // Verifies: Requisito 2.4 (detectar posiciones exactamente 60 min)
    it('returns false for position exactly 60 minutes old (boundary)', () => {
      const boundaryDate = new Date(Date.now() - 60 * 60 * 1000); // exactly 60 minutes
      const boundaryPosition = new Position(40.4168, -3.7038, boundaryDate);
      const vehicleWithBoundary = new Vehicle(vehicleId, licensePlate, boundaryPosition);

      expect(vehicleWithBoundary.isStalePosition()).toBe(false);
    });

    // Verifies: Requisito 2.4 (detectar posiciones con maxAgeMs custom)
    it('respects custom maxAgeMs parameter', () => {
      const twoHourOldDate = new Date(Date.now() - 2 * 60 * 60 * 1000); // 2 hours ago
      const vehicleWithOldPos = new Vehicle(vehicleId, licensePlate, 
        new Position(40.4168, -3.7038, twoHourOldDate)
      );

      // Default (60 min): should be stale
      expect(vehicleWithOldPos.isStalePosition()).toBe(true);

      // Custom (3 hours): should not be stale
      expect(vehicleWithOldPos.isStalePosition(3 * 60 * 60 * 1000)).toBe(false);

      // Custom (1 hour): should be stale
      expect(vehicleWithOldPos.isStalePosition(60 * 60 * 1000)).toBe(true);

      // Custom (90 min): should be stale
      expect(vehicleWithOldPos.isStalePosition(90 * 60 * 1000)).toBe(true);
    });

    // Verifies: Requisito 2.4 (0 maxAgeMs = posición debe ser exactamente NOW)
    it('treats position as stale when maxAgeMs is 0 and position has any age', () => {
      // Create position at a known past time
      const pastDate = new Date(Date.now() - 1); // 1ms ago
      const vehicleWithPastPos = new Vehicle(vehicleId, licensePlate, 
        new Position(40.4168, -3.7038, pastDate)
      );

      expect(vehicleWithPastPos.isStalePosition(0)).toBe(true);
    });
  });

  describe('toString() - String Representation', () => {
    it('returns string representation containing vehicle info', () => {
      const str = vehicle.toString();

      expect(str).toContain('Vehicle');
      expect(str).toContain('VEHICLE-001');
      expect(str).toContain((40.4168).toFixed(4));
      expect(str).toContain((-3.7038).toFixed(4));
    });
  });

  describe('Property: Idempotence of updatePosition', () => {
    // Property-based: Processing the same position update twice yields same result
    // Validates: RNF-04 (Integridad e idempotencia)
    it('processing same position twice produces identical result', () => {
      const newPosition = new Position(41.3851, 2.1734, new Date('2024-01-15T11:00:00Z'));

      const updated1 = vehicle.updatePosition(newPosition);
      const updated2 = vehicle.updatePosition(newPosition);

      expect(updated1.vehicleId.equals(updated2.vehicleId)).toBe(true);
      expect(updated1.licensePlate).toBe(updated2.licensePlate);
      expect(updated1.position.latitude).toBe(updated2.position.latitude);
      expect(updated1.position.longitude).toBe(updated2.position.longitude);
    });
  });
});
