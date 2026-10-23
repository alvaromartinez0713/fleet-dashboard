import { Position } from '../../domain/models/Position';
import { IApiClient } from '../ports/IApiClient';
import { IPersistenceProvider } from '../ports/IPersistenceProvider';
import { IConnectivityProvider } from '../ports/IConnectivityProvider';
import { Clock } from '../ports/Clock';

/**
 * Response from SyncPositionUseCase
 */
export interface SyncPositionResult {
  /**
   * Whether the sync attempt was successful
   * - true: HTTP 202 Accepted received, position marked as 'sent'
   * - false: Error occurred (4xx validation error or 5xx transient error)
   */
  success: boolean;

  /**
   * Error message if sync failed
   * - undefined if success=true
   * - Contains reason from API response or network error
   */
  error?: string;

  /**
   * The position after sync attempt
   * - For success: status='sent'
   * - For validation error: status='validation_error'
   * - For transient error: status='error' with nextRetryAt
   */
  position?: Position;
}

/**
 * SyncPositionUseCase
 *
 * Syncs a single Position record to the backend, with intelligent error differentiation:
 * - 4xx validation errors: marked as 'validation_error', no retry scheduled
 * - 5xx server errors or network failures: marked as 'error', retry with exponential backoff
 * - 202 Accepted: marked as 'sent'
 *
 * Flow:
 * 1. Check if device is online
 * 2. If online: call IApiClient.sendPositions() with the position
 * 3. Handle response based on HTTP status:
 *    - 202: Call position.markAsSent(), update in SQLite, return success
 *    - 4xx: Call position.markAsError(..., true), update in SQLite, return failure (no retry)
 *    - 5xx or network error: Call position.markAsError(..., false), update in SQLite, return failure (will retry)
 * 4. If offline: throw error
 *
 * Requirements covered:
 * - RF-03: Sincronización por lotes (this use case handles single position sync)
 * - RF-04: Ingesta asíncrona (communicates with backend API)
 * - RNF-04: Integridad e idempotencia (4xx vs 5xx differentiation)
 * - RNF-06: Resiliencia (error handling, exponential backoff)
 * - RNF-07: Operación offline-first (checks connectivity)
 */
export class SyncPositionUseCase {
  constructor(
    private apiClient: IApiClient,
    private persistenceProvider: IPersistenceProvider,
    private connectivityProvider: IConnectivityProvider,
    private clock: Clock
  ) {}

  /**
   * Execute sync of a single position to the backend.
   *
   * @param position The Position to sync
   * @returns Promise that resolves to SyncPositionResult
   * @throws Never throws; all outcomes are returned in the result
   */
  async execute(position: Position): Promise<SyncPositionResult> {
    try {
      // 1. Check connectivity
      const connectivity = await this.connectivityProvider.getStatus();
      if (connectivity === 'offline') {
        return {
          success: false,
          error: 'Cannot sync while offline',
          position
        };
      }

      // 2. Prepare payload
      const payload = position.toPayload();

      // 3. Send to backend
      const response = await this.apiClient.sendPositions([payload]);

      // 4. Handle response based on status code
      if (response.status === 202 || response.status === 200 || response.status === 201) {
        // Success: backend accepted the position
        const sentPosition = position.markAsSent();
        await this.persistenceProvider.updatePosition(sentPosition);
        return {
          success: true,
          position: sentPosition
        };
      } else if (response.status >= 400 && response.status < 500) {
        // Validation error: don't retry
        const errorReason = response.error?.message || `HTTP ${response.status}`;
        const errorPosition = position.markAsError(errorReason, true, this.clock);
        await this.persistenceProvider.updatePosition(errorPosition);
        return {
          success: false,
          error: errorReason,
          position: errorPosition
        };
      } else {
        // Server error or other: retry with backoff
        const errorReason = response.error?.message || `HTTP ${response.status}`;
        const errorPosition = position.markAsError(errorReason, false, this.clock);
        await this.persistenceProvider.updatePosition(errorPosition);
        return {
          success: false,
          error: errorReason,
          position: errorPosition
        };
      }
    } catch (error) {
      // Network error or other exception: treat as transient
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      const errorPosition = position.markAsError(errorMessage, false, this.clock);
      await this.persistenceProvider.updatePosition(errorPosition);
      return {
        success: false,
        error: errorMessage,
        position: errorPosition
      };
    }
  }
}
