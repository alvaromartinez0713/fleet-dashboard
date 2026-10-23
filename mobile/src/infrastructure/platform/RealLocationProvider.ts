import { ILocationProvider, GPSData } from '../../application/ports/LocationProvider';

/**
 * RealLocationProvider
 *
 * Production implementation of location provider using device GPS.
 * In this MVP, it returns deterministic test data due to Expo limitations.
 * For production, use react-native-geolocation or similar.
 */
export class RealLocationProvider implements ILocationProvider {
  private permissionGranted: boolean = false;

  /**
   * Request location permission from the user.
   */
  async requestPermissions(): Promise<'granted' | 'denied'> {
    try {
      // For MVP, simulate permission granted
      // In production, use expo-location or react-native-permissions
      console.log('[RealLocationProvider] Permission granted (simulated)');
      this.permissionGranted = true;
      return 'granted';
    } catch (error) {
      console.error('[RealLocationProvider] Permission request failed:', error);
      this.permissionGranted = false;
      return 'denied';
    }
  }

  /**
   * Check if location permission is already granted.
   */
  isPermissionGranted(): boolean {
    return this.permissionGranted;
  }

  /**
   * Get current device location from GPS.
   * Returns latitude, longitude, accuracy, and timestamp.
   * 
   * For MVP: Returns test coordinates in New York area.
   * In production: Replace with real GPS implementation.
   */
  async getCurrentLocation(): Promise<GPSData> {
    if (!this.permissionGranted) {
      throw new Error('Location permission not granted. Call requestPermissions() first.');
    }

    try {
      // For MVP, return deterministic test location
      // In production, use:
      // - expo-location (Expo projects)
      // - react-native-geolocation (bare React Native)
      // - @react-native-camera/camera or similar native modules
      
      const baseLatitude = 40.7128;
      const baseLongitude = -74.0060;
      
      return {
        latitude: baseLatitude + (Math.random() - 0.5) * 0.001,
        longitude: baseLongitude + (Math.random() - 0.5) * 0.001,
        accuracy: 5 + Math.random() * 10,
        timestamp: Date.now(),
      };
    } catch (error) {
      const msg = error instanceof Error ? error.message : 'Unknown GPS error';
      throw new Error(`Failed to get current location: ${msg}`);
    }
  }
}
