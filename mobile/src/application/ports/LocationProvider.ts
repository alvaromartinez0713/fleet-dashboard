/**
 * GPSData represents raw location data from the device's GPS sensor.
 *
 * This is the output contract from the LocationProvider adapter. Values are raw
 * GPS readings without validation - validation of coordinate ranges, accuracy, and
 * timestamp occurs when converting to domain value objects (Coordinate, etc.).
 *
 * The interface accepts any numeric values; domain validation will enforce:
 * - Latitude: [-90, 90]
 * - Longitude: [-180, 180]
 * - Accuracy: >= 0 (if provided)
 * - Timestamp: milliseconds since epoch
 */
export interface GPSData {
  /**
   * Latitude coordinate in degrees (-90.0 to +90.0).
   * Raw value from GPS sensor without validation.
   */
  latitude: number;

  /**
   * Longitude coordinate in degrees (-180.0 to +180.0).
   * Raw value from GPS sensor without validation.
   */
  longitude: number;

  /**
   * Estimated accuracy (confidence) of the GPS reading in meters.
   * Must be greater than or equal to 0. Raw value from GPS sensor without validation.
   */
  accuracy: number;

  /**
   * Timestamp when the GPS reading was captured.
   * Standard JavaScript timestamp: milliseconds since Unix epoch (January 1, 1970, 00:00:00 UTC).
   * Example: Date.now() returns a valid value in this format.
   */
  timestamp: number;
}

/**
 * Port interface for location (GPS) data access.
 *
 * ILocationProvider abstracts platform-specific GPS functionality and permission management.
 * This allows the application layer to capture location data without depending on
 * specific platform implementations (iOS, Android, web, etc.).
 *
 * This interface supports RF-01 (Captura móvil) by enabling:
 * 1. Permission request and state checking
 * 2. Single GPS reading capture
 * 3. Platform-agnostic location acquisition
 *
 * Concrete implementations include:
 * - RealLocationProvider: Uses device GPS (React Native Geolocation API or native modules)
 * - FakeLocationProvider: Returns deterministic GPS data (for testing)
 *
 * Important: The provider does NOT validate coordinate ranges or accuracy - that validation
 * happens in the domain layer when converting to Coordinate value object. The interface
 * accepts any numeric values, trusting domain validation to enforce constraints.
 *
 * Implementation details:
 * - Permission requests should use platform-specific dialogs (iOS/Android)
 * - Permission state should be cached/remembered for isPermissionGranted() checks
 * - getCurrentLocation() should wait for a GPS fix and return accurate readings
 * - Errors should be thrown for failures; null/undefined returns are not acceptable
 */
export interface ILocationProvider {
  /**
   * Request location permission from the user.
   *
   * Prompts the user via a platform-specific dialog to grant or deny permission
   * to access device location. This method should persist the result so that
   * isPermissionGranted() can later check it without prompting again.
   *
   * @returns Promise resolving to 'granted' if user allows access, 'denied' if refused
   *
   * @example
   * const result = await provider.requestPermissions();
   * if (result === 'granted') {
   *   const location = await provider.getCurrentLocation();
   * }
   */
  requestPermissions(): Promise<'granted' | 'denied'>;

  /**
   * Check if location permission has already been granted.
   *
   * Synchronous check of the current permission state, reflecting the result of the
   * most recent requestPermissions() call or the current system permission state if
   * permissions were modified externally (e.g., via Settings app).
   *
   * @returns true if user has granted location permission, false otherwise
   *
   * @example
   * if (provider.isPermissionGranted()) {
   *   const location = await provider.getCurrentLocation();
   * } else {
   *   const result = await provider.requestPermissions();
   * }
   */
  isPermissionGranted(): boolean;

  /**
   * Get the current device location from GPS.
   *
   * Captures a single GPS reading with latitude, longitude, accuracy, and timestamp.
   * This method waits for the GPS sensor to return a reading and should return accurate
   * and current data. Raw GPS data is returned without validation - domain validation
   * occurs when converting to Coordinate value object.
   *
   * Preconditions:
   * - Permission should have been requested via requestPermissions()
   * - isPermissionGranted() should return true before calling this
   *
   * @returns Promise resolving to GPSData with the current location reading
   * @throws LocationError or subclass if GPS unavailable, permission denied, sensor error, or timeout
   *
   * @example
   * try {
   *   const gpsData = await provider.getCurrentLocation();
   *   const coordinate = Coordinate.fromGPSData(gpsData);
   * } catch (error) {
   *   console.error('Failed to get location:', error);
   * }
   */
  getCurrentLocation(): Promise<GPSData>;
}
