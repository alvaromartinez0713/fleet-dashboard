import { ILocationProvider, GPSData } from '../../../src/application/ports/LocationProvider';

/**
 * Fake location provider for testing.
 * Returns deterministic coordinates.
 */
export class FakeLocationProvider implements ILocationProvider {
  private permissionGranted = false;
  private gpsData: GPSData = {
    latitude: 40.7128,
    longitude: -74.006,
    accuracy: 5,
    timestamp: Date.now(),
  };

  async requestPermissions(): Promise<'granted' | 'denied'> {
    this.permissionGranted = true;
    return 'granted';
  }

  isPermissionGranted(): boolean {
    return this.permissionGranted;
  }

  async getCurrentLocation(): Promise<GPSData> {
    return this.gpsData;
  }

  // Test helpers
  setGranted(granted: boolean): void {
    this.permissionGranted = granted;
  }

  setLocation(lat: number, lon: number, accuracy: number = 5): void {
    this.gpsData = { latitude: lat, longitude: lon, accuracy, timestamp: Date.now() };
  }
}
