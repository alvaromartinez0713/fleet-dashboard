import { Coordinate } from '../../../src/domain/models';

/**
 * Object Mother for creating Coordinate test data.
 * Provides named scenarios with valid default values.
 */
export class CoordinateMother {
  /**
   * Valid coordinate with default values (center of Madrid).
   */
  static valid(): CoordinateBuilder {
    return new CoordinateBuilder(40.4168, -3.7038, 5);
  }

  /**
   * Coordinate at the equator, prime meridian.
   */
  static atOrigin(): CoordinateBuilder {
    return new CoordinateBuilder(0, 0, 0);
  }

  /**
   * Coordinate at north pole boundary.
   */
  static atNorthPole(): CoordinateBuilder {
    return new CoordinateBuilder(90, 0, 0);
  }

  /**
   * Coordinate at south pole boundary.
   */
  static atSouthPole(): CoordinateBuilder {
    return new CoordinateBuilder(-90, 0, 0);
  }

  /**
   * Coordinate at international date line (east).
   */
  static atDateLineEast(): CoordinateBuilder {
    return new CoordinateBuilder(0, 180, 0);
  }

  /**
   * Coordinate at international date line (west).
   */
  static atDateLineWest(): CoordinateBuilder {
    return new CoordinateBuilder(0, -180, 0);
  }
}

/**
 * Test Data Builder for Coordinate.
 * Fluent API to override specific fields while keeping others as defaults.
 */
export class CoordinateBuilder {
  private latitude: number;
  private longitude: number;
  private accuracy?: number;

  constructor(latitude: number, longitude: number, accuracy?: number) {
    this.latitude = latitude;
    this.longitude = longitude;
    this.accuracy = accuracy;
  }

  withLatitude(latitude: number): CoordinateBuilder {
    this.latitude = latitude;
    return this;
  }

  withLongitude(longitude: number): CoordinateBuilder {
    this.longitude = longitude;
    return this;
  }

  withAccuracy(accuracy: number | undefined): CoordinateBuilder {
    this.accuracy = accuracy;
    return this;
  }

  build(): Coordinate {
    return new Coordinate(this.latitude, this.longitude, this.accuracy);
  }
}
