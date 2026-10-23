import { InvalidCoordinateError } from '../errors';

/**
 * Coordinate is an immutable value object representing a geographic location.
 * Validates that latitude is in [-90, 90] and longitude is in [-180, 180].
 * Accuracy is optional and represents precision in meters (must be >= 0 if provided).
 */
export class Coordinate {
  private static readonly MIN_LATITUDE = -90;
  private static readonly MAX_LATITUDE = 90;
  private static readonly MIN_LONGITUDE = -180;
  private static readonly MAX_LONGITUDE = 180;

  readonly latitude: number;
  readonly longitude: number;
  readonly accuracy?: number;

  constructor(latitude: number, longitude: number, accuracy?: number) {
    this.validateLatitude(latitude);
    this.validateLongitude(longitude);
    if (accuracy !== undefined) {
      this.validateAccuracy(accuracy);
    }

    this.latitude = latitude;
    this.longitude = longitude;
    this.accuracy = accuracy;
  }

  private validateLatitude(latitude: number): void {
    if (
      !Number.isFinite(latitude) ||
      latitude < Coordinate.MIN_LATITUDE ||
      latitude > Coordinate.MAX_LATITUDE
    ) {
      throw new InvalidCoordinateError(
        `Latitude must be between ${Coordinate.MIN_LATITUDE} and ${Coordinate.MAX_LATITUDE}, got ${latitude}`
      );
    }
  }

  private validateLongitude(longitude: number): void {
    if (
      !Number.isFinite(longitude) ||
      longitude < Coordinate.MIN_LONGITUDE ||
      longitude > Coordinate.MAX_LONGITUDE
    ) {
      throw new InvalidCoordinateError(
        `Longitude must be between ${Coordinate.MIN_LONGITUDE} and ${Coordinate.MAX_LONGITUDE}, got ${longitude}`
      );
    }
  }

  private validateAccuracy(accuracy: number): void {
    if (!Number.isFinite(accuracy) || accuracy < 0) {
      throw new InvalidCoordinateError(
        `Accuracy must be a non-negative finite number, got ${accuracy}`
      );
    }
  }

  /**
   * Create a Coordinate from GPS data.
   * Useful for converting raw GPS readings into domain value objects.
   */
  static fromGPSData(gpsData: {
    latitude: number;
    longitude: number;
    accuracy?: number;
  }): Coordinate {
    return new Coordinate(
      gpsData.latitude,
      gpsData.longitude,
      gpsData.accuracy
    );
  }

  /**
   * Value object equality: two Coordinates are equal if their lat/lon/accuracy are identical.
   */
  equals(other: Coordinate): boolean {
    return (
      this.latitude === other.latitude &&
      this.longitude === other.longitude &&
      this.accuracy === other.accuracy
    );
  }

  /**
   * String representation for logging.
   */
  toString(): string {
    return `Coordinate(lat=${this.latitude}, lon=${this.longitude}, accuracy=${this.accuracy ?? 'N/A'})`;
  }
}
