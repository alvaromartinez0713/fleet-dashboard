import { Position } from '../../domain/models/Position';
import { IPersistenceProvider } from '../ports/IPersistenceProvider';
import { Clock } from '../ports/Clock';
import { SyncPositionUseCase } from './SyncPositionUseCase';

/**
 * Response from SyncPendingPositionsUseCase
 */
export interface SyncPendingResult {
  /**
   * Number of positions successfully synced
   */
  synced: number;

  /**
   * Number of positions that failed to sync
   */
  failed: number;

  /**
   * Array of positions that were attempted in this sync batch
   * These are the updated positions after their sync attempt
   */
  updated: Position[];
}

/**
 * Maximum number of positions to sync concurrently.
 * This prevents overwhelming the device and backend.
 */
const MAX_CONCURRENT = 3;

/**
 * SyncPendingPositionsUseCase
 *
 * Syncs all pending and retryable positions to the backend in a batch with smart retry logic.
 * Uses exponential backoff to avoid hammering the backend.
 *
 * Flow:
 * 1. Get all positions that are ready to sync:
 *    - status='pending' (never sent)
 *    - status='error' AND nextRetryAt <= now (time to retry)
 * 2. Sync up to MAX_CONCURRENT positions in parallel using Promise.allSettled()
 * 3. For each position, call SyncPositionUseCase.execute()
 * 4. Track synced vs failed counts
 * 5. Return batch result with updated positions
 *
 * Requirements covered:
 * - RF-03: Sincronización por lotes
 * - RNF-04: Integridad e idempotencia (retry logic with backoff)
 * - RNF-06: Resiliencia (batch processing with partial failures)
 * - RNF-07: Operación offline-first (respects offline mode)
 */
export class SyncPendingPositionsUseCase {
  constructor(
    private syncPosition: SyncPositionUseCase,
    private persistenceProvider: IPersistenceProvider,
    private clock: Clock
  ) {}

  /**
   * Execute batch sync of all pending and retryable positions.
   *
   * @returns Promise that resolves to SyncPendingResult
   */
  async execute(): Promise<SyncPendingResult> {
    // 1. Get all positions ready for sync
    const pendingPositions = await this.persistenceProvider.getPositionsPendingRetry(
      this.clock.now()
    );

    if (pendingPositions.length === 0) {
      return {
        synced: 0,
        failed: 0,
        updated: []
      };
    }

    let synced = 0;
    let failed = 0;
    const updated: Position[] = [];

    // 2. Sync in batches to respect concurrency limit
    for (let i = 0; i < pendingPositions.length; i += MAX_CONCURRENT) {
      const batch = pendingPositions.slice(i, i + MAX_CONCURRENT);

      // 3. Sync batch concurrently using Promise.allSettled
      //    This ensures one failure doesn't break the entire batch
      const results = await Promise.allSettled(
        batch.map(position => this.syncPosition.execute(position))
      );

      // 4. Process results
      for (const result of results) {
        if (result.status === 'fulfilled') {
          const syncResult = result.value;
          if (syncResult.success) {
            synced++;
          } else {
            failed++;
          }
          if (syncResult.position) {
            updated.push(syncResult.position);
          }
        } else {
          // Unhandled exception from SyncPositionUseCase
          failed++;
        }
      }
    }

    return {
      synced,
      failed,
      updated
    };
  }
}
