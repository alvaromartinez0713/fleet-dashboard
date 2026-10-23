import { Position } from '../../domain/models/Position';
import { CapturePositionUseCase } from './CapturePositionUseCase';
import { SyncPositionUseCase } from './SyncPositionUseCase';
import { IConnectivityProvider } from '../ports/IConnectivityProvider';

/**
 * Response from CaptureAndSyncPositionUseCase
 */
export interface CaptureAndSyncResult {
  /**
   * The captured Position
   */
  position: Position;

  /**
   * Whether a sync attempt was made
   * - true: Device was online, so sync was attempted
   * - false: Device was offline, sync was skipped
   */
  syncAttempted: boolean;

  /**
   * Whether sync succeeded (only meaningful if syncAttempted=true)
   * - true: HTTP 202 Accepted, position marked as 'sent'
   * - false: Sync failed (either 4xx validation error or 5xx transient error)
   */
  syncSuccess: boolean;
}

/**
 * CaptureAndSyncPositionUseCase
 *
 * Captures a GPS position and immediately attempts to sync it to the backend if online.
 * Implements offline-first: if offline, captures locally only without attempting sync.
 *
 * Flow:
 * 1. Call CapturePositionUseCase to capture GPS location
 * 2. Check connectivity via IConnectivityProvider
 * 3. If online: immediately call SyncPositionUseCase to send to backend
 * 4. If offline: return with syncAttempted=false
 * 5. Return result with position, syncAttempted, syncSuccess
 *
 * Requirements covered:
 * - RF-01: Captura móvil de coordenadas
 * - RF-03: Sincronización por lotes (immediate sync if online)
 * - RNF-07: Operación offline-first (checks connectivity before sync)
 */
export class CaptureAndSyncPositionUseCase {
  constructor(
    private capturePosition: CapturePositionUseCase,
    private syncPosition: SyncPositionUseCase,
    private connectivityProvider: IConnectivityProvider
  ) {}

  /**
   * Execute capture with optional immediate sync if online.
   *
   * @param vehicleId The vehicle ID to associate with the capture
   * @returns Promise that resolves to CaptureAndSyncResult
   */
  async execute(vehicleId: string): Promise<CaptureAndSyncResult> {
    // 1. Capture GPS location
    const position = await this.capturePosition.execute(vehicleId);

    // 2. Check connectivity
    const connectivity = await this.connectivityProvider.getStatus();

    // 3. If offline, return without sync attempt
    if (connectivity === 'offline') {
      return {
        position,
        syncAttempted: false,
        syncSuccess: false
      };
    }

    // 4. If online, attempt sync
    const syncResult = await this.syncPosition.execute(position);

    return {
      position: syncResult.position || position,
      syncAttempted: true,
      syncSuccess: syncResult.success
    };
  }
}
