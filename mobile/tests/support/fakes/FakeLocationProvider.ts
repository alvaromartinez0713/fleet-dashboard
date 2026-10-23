import {
  ILocationProvider,
  GPSData,
} from '../../../src/application/ports/LocationProvider';

/**
 * FakeLocationProvider
 *
 * Configurable location provider for testing.
 * Allows tests to inject specific GPS coordinates and simulate permission scenarios.
 */
export class FakeLocationProvider implements ILocationProvider {
  private gpsData: GPSData;
  private permissionGranted: boolean = true;
  private shouldThrowError: boolean = false;
  private errorMessage: string = 'Location access denied';

  constructor(
    latitude: number = 40.7128,
    longitude: number = -74.006,
    accuracy: number = 10
  ) {
    this.gpsData = {
      latitude,
      longitude,
      accuracy,
      timestamp: Date.now(),
    };
  }

  async getCurrentLocation(): Promise<GPSData> {
    if (this.shouldThrowError) {
      throw new Error(this.errorMessage);
    }

    if (!this.permissionGranted) {
      throw new Error('Location permission not granted');
    }

    return {
      ...this.gpsData,
      timestamp: Date.now(),
    };
  }

  async requestPermissions(): Promise<'granted' | 'denied'> {
    if (this.shouldThrowError) {
      throw new Error(this.errorMessage);
    }
    return this.permissionGranted ? 'granted' : 'denied';
  }

  isPermissionGranted(): boolean {
    return this.permissionGranted;
  }

  /**
   * Set GPS coordinates for testing.
   */
  setCoordinates(latitude: number, longitude: number, accuracy: number): void {
    this.gpsData = {
      latitude,
      longitude,
      accuracy,
      timestamp: Date.now(),
    };
  }

  /**
   * Simulate permission grant/deny.
   */
  setPermissionGranted(granted: boolean): void {
    this.permissionGranted = granted;
  }

  /**
   * Simulate location access error.
   */
  setError(shouldThrow: boolean, message?: string): void {
    this.shouldThrowError = shouldThrow;
    if (message) {
      this.errorMessage = message;
    }
  }
}
