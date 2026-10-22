/**
 * Object Mother Pattern for Vehicle Test Data
 *
 * Following testing.md Section 7: Object Mother + Test Data Builder
 *
 * Provides reusable, valid test scenarios for Vehicle entities.
 * Each scenario returns a builder that allows customization.
 *
 * Usage:
 *   const vehicle = VehicleMother.valid().withVehicleId('VH-001').build();
 *   const vehicles = VehicleMother.validMultiple(10);
 */

import { Vehicle } from '@domain/entities/vehicle';
import { VehicleId } from '@domain/entities/vehicle-id';
import { Position } from '@domain/entities/position';

/**
 * VehicleBuilder - Fluent builder for creating Vehicle instances
 * Allows overriding specific fields while keeping others at sensible defaults
 */
export class VehicleBuilder {
  private vehicleId: string = 'MOCK-VEHICLE-01';
  private licensePlate: string = 'MOCK-VEHICLE-01';
  private latitude: number = 40.4168; // Madrid
  private longitude: number = -3.7038; // Madrid
  private timestamp: Date = new Date();

  withVehicleId(id: string): VehicleBuilder {
    this.vehicleId = id;
    return this;
  }

  withLicensePlate(plate: string): VehicleBuilder {
    this.licensePlate = plate;
    return this;
  }

  withLatitude(lat: number): VehicleBuilder {
    this.latitude = lat;
    return this;
  }

  withLongitude(lon: number): VehicleBuilder {
    this.longitude = lon;
    return this;
  }

  withPosition(latitude: number, longitude: number, timestamp?: Date): VehicleBuilder {
    this.latitude = latitude;
    this.longitude = longitude;
    if (timestamp) {
      this.timestamp = timestamp;
    }
    return this;
  }

  withTimestamp(timestamp: Date): VehicleBuilder {
    this.timestamp = timestamp;
    return this;
  }

  build(): Vehicle {
    return new Vehicle(
      new VehicleId(this.vehicleId),
      this.licensePlate,
      new Position(this.latitude, this.longitude, this.timestamp),
    );
  }
}

/**
 * VehicleMother - Factory for creating test Vehicle scenarios
 */
export const VehicleMother = {
  /**
   * valid() - Returns a builder with sensible defaults
   * Creates a vehicle in Madrid area with current timestamp
   */
  valid: (): VehicleBuilder => new VehicleBuilder(),

  /**
   * validMultiple(count) - Creates multiple valid vehicles
   * Each vehicle has a unique ID and slightly different coordinates
   * All within valid ranges
   */
  validMultiple: (count: number): Vehicle[] => {
    return Array.from({ length: count }, (_, i) => {
      const id = `MOCK-VEHICLE-${String(i + 1).padStart(2, '0')}`;
      return new VehicleBuilder()
        .withVehicleId(id)
        .withLicensePlate(id)
        .withPosition(
          40.4168 + (Math.random() - 0.5) * 0.2, // Within ±0.1 degrees of Madrid
          -3.7038 + (Math.random() - 0.5) * 0.2,
        )
        .build();
    });
  },

  /**
   * outOfRange() - Returns a builder with invalid coordinates
   * Used for testing validation of invalid positions
   */
  outOfRange: (): VehicleBuilder =>
    new VehicleBuilder()
      .withVehicleId('INVALID-VEHICLE')
      .withLicensePlate('INVALID-VEHICLE')
      .withLatitude(91) // Invalid: > 90
      .withLongitude(181), // Invalid: > 180

  /**
   * stalePosition() - Returns a builder with old timestamp
   * Position is more than 60 minutes old
   */
  stalePosition: (): VehicleBuilder => {
    const oldTime = new Date(Date.now() - 61 * 60 * 1000); // 61 minutes ago
    return new VehicleBuilder().withTimestamp(oldTime);
  },

  /**
   * recentPosition() - Returns a builder with fresh timestamp
   * Position was updated less than 1 minute ago
   */
  recentPosition: (): VehicleBuilder => {
    const recentTime = new Date(Date.now() - 30 * 1000); // 30 seconds ago
    return new VehicleBuilder().withTimestamp(recentTime);
  },

  /**
   * atCoordinates(lat, lon) - Returns a builder with specific position
   * Useful for testing specific geographic scenarios
   */
  atCoordinates: (latitude: number, longitude: number): VehicleBuilder =>
    new VehicleBuilder().withPosition(latitude, longitude),

  /**
   * inCriticalZone() - Returns a builder for a vehicle in a defined critical zone
   * Example critical zone: downtown Madrid
   */
  inCriticalZone: (): VehicleBuilder =>
    new VehicleBuilder()
      .withVehicleId('CRITICAL-01')
      .withLicensePlate('CRITICAL-01')
      .withPosition(40.415, -3.704), // Puerta del Sol

  /**
   * stopped() - Returns a builder for a stationary vehicle
   * Multiple readings at same location (simulates stopped vehicle)
   */
  stopped: (): VehicleBuilder =>
    new VehicleBuilder()
      .withVehicleId('STOPPED-01')
      .withLicensePlate('STOPPED-01')
      .withPosition(40.4168, -3.7038),
};
