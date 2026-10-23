import { Position } from '../../domain/models/Position';
import { Coordinate } from '../../domain/models/Coordinate';
import { TraceContext } from '../../domain/models/TraceContext';
import { ILocationProvider } from '../ports/LocationProvider';
import { IPersistenceProvider } from '../ports/IPersistenceProvider';
import { Clock } from '../ports/Clock';
import { IdentifierGenerator } from '../ports/IdentifierGenerator';

/**
 * CapturePositionUseCase
 *
 * Captures current GPS location once and saves it locally to SQLite.
 * This is a single capture operation (not periodic).
 * For periodic capture, use StartCaptureUseCase.
 *
 * Flow:
 * 1. Request current GPS location from device via ILocationProvider
 * 2. Create a Position entity with status='pending'
 * 3. Save the position to SQLite via IPersistenceProvider
 * 4. Emit onPositionCaptured event
 * 5. Return the captured Position
 *
 * Requirements covered:
 * - RF-01: Automatic capture of GPS coordinates
 * - RF-02: Persistent storage locally
 * - RNF-01: Clean Architecture
 * - RNF-02: Offline-first operation
 */
export class CapturePositionUseCase {
  constructor(
    private locationProvider: ILocationProvider,
    private persistenceProvider: IPersistenceProvider,
    private clock: Clock,
    private idGenerator: IdentifierGenerator
  ) {}

  /**
   * Execute a single GPS capture and save locally.
   *
   * @param vehicleId The vehicle ID to associate with this capture
   * @returns Promise that resolves to the captured Position
   * @throws LocationError if GPS cannot be accessed or permission denied
   * @throws PersistenceError if SQLite operation fails
   */
  async execute(vehicleId: string): Promise<Position> {
    // 1. Validate input
    if (!vehicleId || vehicleId.trim().length === 0) {
      throw new Error('Vehicle ID is required and must not be empty');
    }

    // 2. Get current GPS location
    const gpsData = await this.locationProvider.getCurrentLocation();

    // 3. Create Coordinate value object from GPS data
    const coordinate = new Coordinate(
      gpsData.latitude,
      gpsData.longitude,
      gpsData.accuracy
    );

    // 4. Generate unique identifiers
    const eventId = this.idGenerator.generateUUID();
    const traceId = this.idGenerator.generateUUID();
    const idempotencyKey = this.idGenerator.generateUUID();

    // 5. Create Position entity with status='pending'
    const position = new Position(
      eventId,
      vehicleId,
      coordinate,
      this.clock.now(),
      new TraceContext(traceId, idempotencyKey),
      'pending', // Initial status
      undefined, // No error
      this.clock.now(),
      false, // Not a validation error
      undefined, // No sync attempts yet
      0, // syncAttemptCount = 0
      undefined // No retry scheduled
    );

    // 6. Validate the position
    if (!position.isValid()) {
      throw new Error('Invalid position data');
    }

    // 7. Save to SQLite
    await this.persistenceProvider.savePosition(position);

    // Note: Event listeners (onPositionCaptured) are called by the persistence provider

    return position;
  }
}
